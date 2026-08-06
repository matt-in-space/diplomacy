package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
)

func testInvite(code string, gameID game.GameID, email string) *lobby.Invite {
	return &lobby.Invite{
		Code:      code,
		GameID:    gameID,
		Email:     email,
		Status:    lobby.InvitePending,
		CreatedAt: time.Unix(0, 0).UTC(),
	}
}

func TestInviteRepositoryCreateAndGetByCode(t *testing.T) {
	repo := NewInviteRepository()
	invite := testInvite("code-a", "game-a", "a@example.com")

	if err := repo.CreateInvite(context.Background(), invite); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetInviteByCode(context.Background(), "code-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.Email != invite.Email {
		t.Fatalf("Email = %q, want %q", stored.Email, invite.Email)
	}
}

func TestInviteRepositoryGetRejectsUnknownCode(t *testing.T) {
	repo := NewInviteRepository()

	_, err := repo.GetInviteByCode(context.Background(), "missing-code")
	if !errors.Is(err, lobby.ErrInviteNotFound) {
		t.Fatalf("Get error = %v, want ErrInviteNotFound", err)
	}
}

func TestInviteRepositoryRejectsDuplicateCode(t *testing.T) {
	repo := NewInviteRepository()
	ctx := context.Background()

	if err := repo.CreateInvite(ctx, testInvite("code-a", "game-a", "a@example.com")); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	err := repo.CreateInvite(ctx, testInvite("code-a", "game-b", "b@example.com"))
	if !errors.Is(err, lobby.ErrInviteAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrInviteAlreadyExists", err)
	}
}

func TestInviteRepositorySaveExistingInvite(t *testing.T) {
	repo := NewInviteRepository()
	invite := testInvite("code-a", "game-a", "a@example.com")
	ctx := context.Background()

	if err := repo.CreateInvite(ctx, invite); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	invite.Status = lobby.InviteAccepted
	invite.PlayerID = "player-a"
	if err := repo.SaveInvite(ctx, invite); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	stored, err := repo.GetInviteByCode(ctx, "code-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.Status != lobby.InviteAccepted {
		t.Fatalf("Status = %q, want %q", stored.Status, lobby.InviteAccepted)
	}
	if stored.PlayerID != "player-a" {
		t.Fatalf("PlayerID = %q, want %q", stored.PlayerID, "player-a")
	}
}

func TestInviteRepositorySaveRejectsUnknownCode(t *testing.T) {
	repo := NewInviteRepository()

	err := repo.SaveInvite(context.Background(), testInvite("missing-code", "game-a", "a@example.com"))
	if !errors.Is(err, lobby.ErrInviteNotFound) {
		t.Fatalf("Save error = %v, want ErrInviteNotFound", err)
	}
}

func TestInviteRepositoryListInvitesForGame(t *testing.T) {
	repo := NewInviteRepository()
	ctx := context.Background()

	if err := repo.CreateInvite(ctx, testInvite("code-a", "game-a", "a@example.com")); err != nil {
		t.Fatalf("Create code-a failed: %v", err)
	}
	if err := repo.CreateInvite(ctx, testInvite("code-b", "game-a", "b@example.com")); err != nil {
		t.Fatalf("Create code-b failed: %v", err)
	}
	if err := repo.CreateInvite(ctx, testInvite("code-c", "game-b", "c@example.com")); err != nil {
		t.Fatalf("Create code-c failed: %v", err)
	}

	invites, err := repo.ListInvitesForGame(ctx, "game-a")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(invites) != 2 {
		t.Fatalf("len(invites) = %d, want 2", len(invites))
	}
}

func TestInviteRepositoryListInvitesForEmail(t *testing.T) {
	repo := NewInviteRepository()
	ctx := context.Background()

	if err := repo.CreateInvite(ctx, testInvite("code-a", "game-a", "a@example.com")); err != nil {
		t.Fatalf("Create code-a failed: %v", err)
	}
	if err := repo.CreateInvite(ctx, testInvite("code-b", "game-b", "a@example.com")); err != nil {
		t.Fatalf("Create code-b failed: %v", err)
	}
	if err := repo.CreateInvite(ctx, testInvite("code-c", "game-c", "c@example.com")); err != nil {
		t.Fatalf("Create code-c failed: %v", err)
	}

	invites, err := repo.ListInvitesForEmail(ctx, "a@example.com")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(invites) != 2 {
		t.Fatalf("len(invites) = %d, want 2", len(invites))
	}
}

func TestInviteRepositoryRejectsNilInvite(t *testing.T) {
	repo := NewInviteRepository()
	ctx := context.Background()

	if err := repo.CreateInvite(ctx, nil); err == nil {
		t.Fatal("expected Create to reject nil invite")
	}
	if err := repo.SaveInvite(ctx, nil); err == nil {
		t.Fatal("expected Save to reject nil invite")
	}
}

func TestInviteRepositoryHonorsCancelledContext(t *testing.T) {
	repo := NewInviteRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.CreateInvite(ctx, testInvite("code-a", "game-a", "a@example.com")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetInviteByCode(ctx, "code-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if err := repo.SaveInvite(ctx, testInvite("code-a", "game-a", "a@example.com")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}
	if _, err := repo.ListInvitesForGame(ctx, "game-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListInvitesForGame error = %v, want context.Canceled", err)
	}
	if _, err := repo.ListInvitesForEmail(ctx, "a@example.com"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListInvitesForEmail error = %v, want context.Canceled", err)
	}
}
