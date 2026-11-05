package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/store"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCredentials = errors.New("ungültiger Benutzername oder Passwort")
	ErrUserDisabled       = errors.New("dieses Benutzerkonto ist deaktiviert")
	ErrSessionExpired     = errors.New("sitzung abgelaufen")
	ErrUnauthorized       = errors.New("nicht autorisiert")
	ErrForbidden          = errors.New("keine ausreichenden Berechtigungen (Admin erforderlich)")
	ErrInvalidCSRF        = errors.New("ungültiges oder fehlendes CSRF-Token")
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

type User struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Role        Role       `json:"role"`
	Disabled    bool       `json:"disabled"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
}

type Session struct {
	IDHash     string    `json:"-"`
	UserID     int64     `json:"userId"`
	Username   string    `json:"username"`
	Role       Role      `json:"role"`
	CSRFToken  string    `json:"csrfToken"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"userAgent"`
}

type Manager struct {
	cfg         *config.Config
	store       *store.Store
	log         *slog.Logger
	setupToken  string
	rateLimiter *RateLimiter
}

func NewManager(cfg *config.Config, st *store.Store, log *slog.Logger) (*Manager, error) {
	m := &Manager{
		cfg:         cfg,
		store:       st,
		log:         log,
		rateLimiter: NewRateLimiter(5, 1*time.Minute), // 5 failed attempts per min
	}

	// Check if any admin user exists. If not, generate and log a one-time setup token.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var adminCount int
	err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin' AND disabled = 0`).Scan(&adminCount)
	if err != nil {
		return nil, fmt.Errorf("check admin users: %w", err)
	}

	if adminCount == 0 {
		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err != nil {
			return nil, err
		}
		m.setupToken = hex.EncodeToString(tokenBytes)
		log.Warn("==========================================================================")
		log.Warn("  ORBIT SETUP ERFORDERLICH: Kein Administrator vorhanden!")
		log.Warn("  Einmaliger Setup-Token für Ersteinrichtung:")
		log.Warn(fmt.Sprintf("  SETUP TOKEN: %s", m.setupToken))
		log.Warn("  Verwende diesen Token in der Weboberfläche, um den ersten Admin anzulegen.")
		log.Warn("==========================================================================")
	}

	return m, nil
}

func (m *Manager) NeedsSetup() bool {
	return m.setupToken != ""
}

// CompleteSetup creates the first admin user using the one-time setup token.
func (m *Manager) CompleteSetup(ctx context.Context, token, username, password string) error {
	if m.setupToken == "" {
		return errors.New("das Setup wurde bereits abgeschlossen")
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(m.setupToken)) != 1 {
		return errors.New("ungültiger Setup-Token")
	}
	username = strings.TrimSpace(username)
	if len(username) < 3 {
		return errors.New("benutzername muss mindestens 3 Zeichen lang sein")
	}
	if len(password) < 8 {
		return errors.New("passwort muss mindestens 8 Zeichen lang sein")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	_, err = m.store.DB.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role, disabled, created_at)
		 VALUES (?, ?, 'admin', 0, ?)`,
		username, hash, now,
	)
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	m.setupToken = ""
	m.log.Info("initial admin account created successfully", "username", username)
	return nil
}

// Login verifies credentials, enforces rate limits, creates a session and CSRF token.
func (m *Manager) Login(ctx context.Context, username, password, ip, userAgent string) (*Session, string, error) {
	if !m.rateLimiter.Allow(ip) {
		return nil, "", errors.New("zu viele fehlgeschlagene Anmeldeversuche. Bitte versuche es in einer Minute erneut")
	}

	var user User
	var hash string
	var lastLogin sql.NullInt64
	var createdAt int64
	var roleStr string

	err := m.store.DB.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, disabled, created_at, last_login_at
		 FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &hash, &roleStr, &user.Disabled, &createdAt, &lastLogin)

	if err != nil {
		m.rateLimiter.RecordFailure(ip)
		m.RecordAudit(ctx, 0, username, "auth.login", "", "failed", ip, "user not found or db error")
		return nil, "", ErrInvalidCredentials
	}

	user.Role = Role(roleStr)
	user.CreatedAt = time.Unix(createdAt, 0)
	if lastLogin.Valid {
		t := time.Unix(lastLogin.Int64, 0)
		user.LastLoginAt = &t
	}

	if user.Disabled {
		return nil, "", ErrUserDisabled
	}

	if !VerifyPassword(password, hash) {
		m.rateLimiter.RecordFailure(ip)
		m.RecordAudit(ctx, user.ID, username, "auth.login", "", "failed", ip, "invalid password")
		return nil, "", ErrInvalidCredentials
	}

	m.rateLimiter.Reset(ip)

	// Generate raw session ID and CSRF token
	rawSessionID := make([]byte, 32)
	csrfBytes := make([]byte, 32)
	rand.Read(rawSessionID)
	rand.Read(csrfBytes)

	sessionToken := hex.EncodeToString(rawSessionID)
	csrfToken := hex.EncodeToString(csrfBytes)
	sessionHash := HashToken(sessionToken)

	now := time.Now()
	expiresAt := now.Add(m.cfg.SessionTTL)

	_, err = m.store.DB.ExecContext(ctx,
		`INSERT INTO sessions (id_hash, user_id, csrf_token, created_at, last_seen_at, expires_at, ip, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionHash, user.ID, csrfToken, now.Unix(), now.Unix(), expiresAt.Unix(), ip, userAgent,
	)
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	_, _ = m.store.DB.ExecContext(ctx, `UPDATE users SET last_login_at = ? WHERE id = ?`, now.Unix(), user.ID)
	m.RecordAudit(ctx, user.ID, username, "auth.login", "", "success", ip, "")

	sess := &Session{
		IDHash:    sessionHash,
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CSRFToken: csrfToken,
		CreatedAt: now,
		ExpiresAt: expiresAt,
		IP:        ip,
		UserAgent: userAgent,
	}

	return sess, sessionToken, nil
}

// ValidateSession verifies session token and checks idle / expiration timeouts.
func (m *Manager) ValidateSession(ctx context.Context, sessionToken string) (*Session, error) {
	if sessionToken == "" {
		return nil, ErrUnauthorized
	}

	sessionHash := HashToken(sessionToken)
	now := time.Now().Unix()

	var sess Session
	var roleStr string
	var disabled bool
	var lastSeen int64
	var expiresAt int64
	var createdAt int64

	err := m.store.DB.QueryRowContext(ctx,
		`SELECT s.user_id, u.username, u.role, u.disabled, s.csrf_token, s.created_at, s.last_seen_at, s.expires_at, s.ip, s.user_agent
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.id_hash = ? AND s.expires_at > ?`,
		sessionHash, now,
	).Scan(&sess.UserID, &sess.Username, &roleStr, &disabled, &sess.CSRFToken, &createdAt, &lastSeen, &expiresAt, &sess.IP, &sess.UserAgent)

	if err != nil {
		return nil, ErrSessionExpired
	}

	if disabled {
		return nil, ErrUserDisabled
	}

	sess.Role = Role(roleStr)
	sess.IDHash = sessionHash
	sess.CreatedAt = time.Unix(createdAt, 0)
	sess.ExpiresAt = time.Unix(expiresAt, 0)

	// Check idle timeout
	if m.cfg.SessionIdle > 0 && time.Since(time.Unix(lastSeen, 0)) > m.cfg.SessionIdle {
		_, _ = m.store.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id_hash = ?`, sessionHash)
		return nil, ErrSessionExpired
	}

	// Bump last seen
	_, _ = m.store.DB.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE id_hash = ?`, now, sessionHash)

	return &sess, nil
}

func (m *Manager) Logout(ctx context.Context, sessionToken string) {
	if sessionToken != "" {
		sessionHash := HashToken(sessionToken)
		_, _ = m.store.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id_hash = ?`, sessionHash)
	}
}

func (m *Manager) RecordAudit(ctx context.Context, userID int64, username, action, target, result, ip, details string) {
	if m.store == nil {
		return
	}
	_, _ = m.store.DB.ExecContext(ctx,
		`INSERT INTO audit_log (ts, user_id, username, action, target, result, ip, details)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(), userID, username, action, target, result, ip, details,
	)
}

// Password hashing using Argon2id with 64MB memory, 3 iterations, 4 parallelism
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	encoded := fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=4$%s$%s",
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	)
	return encoded, nil
}

func VerifyPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expectedHash, err := hex.DecodeString(parts[5])
	if err != nil {
		return false
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}

func HashToken(token string) string {
	h := argon2.IDKey([]byte(token), []byte("orbit-token-salt"), 1, 16*1024, 2, 32)
	return hex.EncodeToString(h)
}

// RateLimiter for login attempts
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rl.window)

	var valid []time.Time
	for _, t := range rl.attempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.attempts[ip] = valid
	return len(valid) < rl.max
}

func (rl *RateLimiter) RecordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.attempts[ip] = append(rl.attempts[ip], time.Now())
}

func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}
