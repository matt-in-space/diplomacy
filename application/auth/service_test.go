package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
)

func newTestService() *auth.Service {
	return auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
}

func TestServiceSignupCreatesPlayer(t *testing.T) {
	svc := newTestService()

	player, err := svc.Signup(context.Background(), "a@example.com", "Alice", "password123")
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}
	if player.Email != "a@example.com" {
		t.Fatalf("Email = %q, want a@example.com", player.Email)
	}
	if player.DisplayName != "Alice" {
		t.Fatalf("DisplayName = %q, want Alice", player.DisplayName)
	}
	if len(player.PasswordHash) == 0 {
		t.Fatal("expected a non-empty PasswordHash")
	}
	if string(player.PasswordHash) == "password123" {
		t.Fatal("PasswordHash must not be the plaintext password")
	}
}

func TestServiceSignupRejectsDuplicateEmail(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	if _, err := svc.Signup(ctx, "a@example.com", "Alice", "password123"); err != nil {
		t.Fatalf("first Signup failed: %v", err)
	}
	_, err := svc.Signup(ctx, "a@example.com", "Someone Else", "password456")
	if !errors.Is(err, auth.ErrPlayerAlreadyExists) {
		t.Fatalf("second Signup error = %v, want ErrPlayerAlreadyExists", err)
	}
}

func TestServiceSignupRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		displayName string
		password    string
	}{
		{"empty email", "", "Alice", "password123"},
		{"email missing @", "not-an-email", "Alice", "password123"},
		{"empty display name", "a@example.com", "", "password123"},
		{"short password", "a@example.com", "Alice", "short"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService()
			_, err := svc.Signup(context.Background(), tt.email, tt.displayName, tt.password)
			if err == nil {
				t.Fatal("expected Signup to fail")
			}
		})
	}
}

func TestServiceLoginSucceeds(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	if _, err := svc.Signup(ctx, "a@example.com", "Alice", "password123"); err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	session, err := svc.Login(ctx, "a@example.com", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if session.Token == "" {
		t.Fatal("expected a non-empty session token")
	}
	if !session.ExpiresAt.After(time.Now()) {
		t.Fatalf("ExpiresAt = %v, want a time in the future", session.ExpiresAt)
	}
}

func TestServiceLoginRejectsWrongPassword(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	if _, err := svc.Signup(ctx, "a@example.com", "Alice", "password123"); err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	_, err := svc.Login(ctx, "a@example.com", "wrong-password")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("Login error = %v, want ErrInvalidCredentials", err)
	}
}

func TestServiceLoginRejectsUnknownEmailWithSameError(t *testing.T) {
	svc := newTestService()

	_, err := svc.Login(context.Background(), "missing@example.com", "password123")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("Login error = %v, want ErrInvalidCredentials", err)
	}
}

func TestServiceLogoutRemovesSession(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	if _, err := svc.Signup(ctx, "a@example.com", "Alice", "password123"); err != nil {
		t.Fatalf("Signup failed: %v", err)
	}
	session, err := svc.Login(ctx, "a@example.com", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if err := svc.Logout(ctx, session.Token); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	if _, err := svc.Authenticate(ctx, session.Token); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("Authenticate after Logout error = %v, want ErrSessionNotFound", err)
	}
}

func TestServiceAuthenticateResolvesPlayer(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	created, err := svc.Signup(ctx, "a@example.com", "Alice", "password123")
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}
	session, err := svc.Login(ctx, "a@example.com", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	player, err := svc.Authenticate(ctx, session.Token)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if player.ID != created.ID {
		t.Fatalf("Authenticate player.ID = %q, want %q", player.ID, created.ID)
	}
}

func TestServiceAuthenticateRejectsUnknownToken(t *testing.T) {
	svc := newTestService()

	_, err := svc.Authenticate(context.Background(), "not-a-real-token")
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("Authenticate error = %v, want ErrSessionNotFound", err)
	}
}
