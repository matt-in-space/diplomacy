package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/core/game"
)

func testSession(token string, playerID game.PlayerID) *auth.Session {
	now := time.Now()
	return &auth.Session{
		Token:     token,
		PlayerID:  playerID,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}
}

func TestSessionRepositoryCreateAndGetSession(t *testing.T) {
	repo := NewSessionRepository()
	session := testSession("token-a", "player-a")

	if err := repo.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetSession(context.Background(), session.Token)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.PlayerID != session.PlayerID {
		t.Fatalf("PlayerID = %q, want %q", stored.PlayerID, session.PlayerID)
	}
}

func TestSessionRepositoryRejectsDuplicateToken(t *testing.T) {
	repo := NewSessionRepository()
	ctx := context.Background()

	if err := repo.CreateSession(ctx, testSession("token-a", "player-a")); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	err := repo.CreateSession(ctx, testSession("token-a", "player-b"))
	if !errors.Is(err, auth.ErrSessionAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrSessionAlreadyExists", err)
	}
}

func TestSessionRepositoryRejectsNilSession(t *testing.T) {
	repo := NewSessionRepository()

	if err := repo.CreateSession(context.Background(), nil); err == nil {
		t.Fatal("expected Create to reject nil session")
	}
}

func TestSessionRepositoryGetRejectsUnknownToken(t *testing.T) {
	repo := NewSessionRepository()

	_, err := repo.GetSession(context.Background(), "missing-token")
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("Get error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionRepositoryGetRejectsExpiredSession(t *testing.T) {
	repo := NewSessionRepository()
	ctx := context.Background()
	session := &auth.Session{
		Token:     "token-a",
		PlayerID:  "player-a",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour), // already expired
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err := repo.GetSession(ctx, session.Token)
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("Get error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionRepositoryDeleteSessionIsIdempotent(t *testing.T) {
	repo := NewSessionRepository()
	ctx := context.Background()
	session := testSession("token-a", "player-a")

	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.DeleteSession(ctx, session.Token); err != nil {
		t.Fatalf("first Delete failed: %v", err)
	}
	if err := repo.DeleteSession(ctx, session.Token); err != nil {
		t.Fatalf("second Delete (already gone) failed: %v", err)
	}
	if err := repo.DeleteSession(ctx, "never-existed"); err != nil {
		t.Fatalf("Delete of unknown token failed: %v", err)
	}

	if _, err := repo.GetSession(ctx, session.Token); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("Get after Delete error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionRepositoryDeleteSessionsForPlayer(t *testing.T) {
	repo := NewSessionRepository()
	ctx := context.Background()

	if err := repo.CreateSession(ctx, testSession("token-a1", "player-a")); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.CreateSession(ctx, testSession("token-a2", "player-a")); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.CreateSession(ctx, testSession("token-b1", "player-b")); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.DeleteSessionsForPlayer(ctx, "player-a"); err != nil {
		t.Fatalf("DeleteSessionsForPlayer failed: %v", err)
	}

	if _, err := repo.GetSession(ctx, "token-a1"); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("token-a1 Get error = %v, want ErrSessionNotFound", err)
	}
	if _, err := repo.GetSession(ctx, "token-a2"); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("token-a2 Get error = %v, want ErrSessionNotFound", err)
	}
	if _, err := repo.GetSession(ctx, "token-b1"); err != nil {
		t.Fatalf("token-b1 Get failed: %v, want it to still exist", err)
	}
}

func TestSessionRepositoryDeleteSessionsForPlayerWithNoSessionsIsNotAnError(t *testing.T) {
	repo := NewSessionRepository()

	if err := repo.DeleteSessionsForPlayer(context.Background(), "no-such-player"); err != nil {
		t.Fatalf("DeleteSessionsForPlayer failed: %v", err)
	}
}

func TestSessionRepositoryHonorsCancelledContext(t *testing.T) {
	repo := NewSessionRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.CreateSession(ctx, testSession("token-a", "player-a")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetSession(ctx, "token-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if err := repo.DeleteSession(ctx, "token-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete error = %v, want context.Canceled", err)
	}
	if err := repo.DeleteSessionsForPlayer(ctx, "player-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("DeleteSessionsForPlayer error = %v, want context.Canceled", err)
	}
}
