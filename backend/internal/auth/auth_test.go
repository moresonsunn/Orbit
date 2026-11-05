package auth

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/store"
)

func TestPasswordHashAndVerify(t *testing.T) {
	password := "SecretP@ssw0rd!123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !VerifyPassword(password, hash) {
		t.Errorf("expected password verification to succeed")
	}

	if VerifyPassword("WrongPassword", hash) {
		t.Errorf("expected wrong password verification to fail")
	}
}

func TestInitialSetupAndLogin(t *testing.T) {
	tmpDir := t.TempDir()
	st, err := store.Open(filepath.Join(tmpDir, "data"))
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	cfg := &config.Config{
		SessionTTL:  1 * time.Hour,
		SessionIdle: 30 * time.Minute,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mgr, err := NewManager(cfg, st, logger)
	if err != nil {
		t.Fatalf("failed to create auth manager: %v", err)
	}

	if !mgr.NeedsSetup() {
		t.Fatalf("expected NeedsSetup=true when no admin exists")
	}

	setupToken := mgr.setupToken
	if len(setupToken) == 0 {
		t.Fatalf("expected non-empty setup token")
	}

	// Try completing setup with wrong token
	err = mgr.CompleteSetup(context.Background(), "wrong-token", "admin", "adminpassword123")
	if err == nil {
		t.Errorf("expected error with invalid token")
	}

	// Complete setup with correct token
	err = mgr.CompleteSetup(context.Background(), setupToken, "admin", "adminpassword123")
	if err != nil {
		t.Fatalf("failed to complete setup: %v", err)
	}

	if mgr.NeedsSetup() {
		t.Errorf("expected NeedsSetup=false after setup completion")
	}

	// Test login
	sess, sessionToken, err := mgr.Login(context.Background(), "admin", "adminpassword123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}
	if sess.Username != "admin" || sess.Role != RoleAdmin {
		t.Errorf("unexpected session data: %+v", sess)
	}

	// Validate session
	valSess, err := mgr.ValidateSession(context.Background(), sessionToken)
	if err != nil {
		t.Fatalf("failed to validate session: %v", err)
	}
	if valSess.Username != "admin" {
		t.Errorf("expected username admin, got %s", valSess.Username)
	}
}
