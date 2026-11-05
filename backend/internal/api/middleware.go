package api

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/phili/orbit/internal/auth"
	"github.com/phili/orbit/internal/config"
)

type contextKey string

const (
	sessionKey contextKey = "session"
)

type Middleware struct {
	cfg     *config.Config
	authMgr *auth.Manager
}

func NewMiddleware(cfg *config.Config, am *auth.Manager) *Middleware {
	return &Middleware{cfg: cfg, authMgr: am}
}

// SecurityHeaders sets modern defense-in-depth HTTP response headers.
func (m *Middleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// CORS handles origin checks for API requests.
func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			allowed := false
			if m.cfg.DevMode {
				allowed = true
			} else if len(m.cfg.TrustedOrigins) > 0 {
				for _, o := range m.cfg.TrustedOrigins {
					if strings.EqualFold(origin, o) {
						allowed = true
						break
					}
				}
			} else {
				// Same-host request
				allowed = true
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, Authorization")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Authenticate extracts session from cookie or Bearer header and attaches to context.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// Check cookie first
		if cookie, err := r.Cookie("orbit_session"); err == nil && cookie.Value != "" {
			token = cookie.Value
		}

		// Fallback to Authorization: Bearer
		if token == "" {
			authHdr := r.Header.Get("Authorization")
			if strings.HasPrefix(authHdr, "Bearer ") {
				token = strings.TrimPrefix(authHdr, "Bearer ")
			}
		}

		if token != "" {
			sess, err := m.authMgr.ValidateSession(r.Context(), token)
			if err == nil && sess != nil {
				ctx := context.WithValue(r.Context(), sessionKey, sess)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAuth enforces that a valid session exists.
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := r.Context().Value(sessionKey).(*auth.Session)
		if !ok || sess == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin enforces that the authenticated user is an administrator.
func (m *Middleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := r.Context().Value(sessionKey).(*auth.Session)
		if !ok || sess == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if sess.Role != auth.RoleAdmin {
			http.Error(w, `{"error":"forbidden: admin role required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// VerifyCSRF validates CSRF token for mutative state-changing methods.
func (m *Middleware) VerifyCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check unsafe HTTP methods
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
			sess, ok := r.Context().Value(sessionKey).(*auth.Session)
			if ok && sess != nil {
				csrfHeader := r.Header.Get("X-CSRF-Token")
				if csrfHeader == "" || subtle.ConstantTimeCompare([]byte(csrfHeader), []byte(sess.CSRFToken)) != 1 {
					http.Error(w, `{"error":"invalid or missing CSRF token"}`, http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func GetSession(ctx context.Context) (*auth.Session, bool) {
	sess, ok := ctx.Value(sessionKey).(*auth.Session)
	return sess, ok && sess != nil
}
