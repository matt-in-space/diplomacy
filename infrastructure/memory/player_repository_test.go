package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/core/game"
)

func testPlayer(id game.PlayerID, email string) *auth.Player {
	return &auth.Player{
		ID:           id,
		Email:        email,
		DisplayName:  "Test Player",
		PasswordHash: []byte("hash"),
		CreatedAt:    time.Unix(0, 0).UTC(),
	}
}

func TestPlayerRepositoryCreateAndGetPlayer(t *testing.T) {
	repo := NewPlayerRepository()
	player := testPlayer("player-a", "a@example.com")

	if err := repo.CreatePlayer(context.Background(), player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetPlayer(context.Background(), player.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.ID != player.ID {
		t.Fatalf("Player.ID = %q, want %q", stored.ID, player.ID)
	}
	if stored.Email != player.Email {
		t.Fatalf("Player.Email = %q, want %q", stored.Email, player.Email)
	}
}

func TestPlayerRepositoryGetPlayerByEmail(t *testing.T) {
	repo := NewPlayerRepository()
	player := testPlayer("player-a", "a@example.com")
	ctx := context.Background()

	if err := repo.CreatePlayer(ctx, player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetPlayerByEmail(ctx, "a@example.com")
	if err != nil {
		t.Fatalf("GetPlayerByEmail failed: %v", err)
	}
	if stored.ID != player.ID {
		t.Fatalf("Player.ID = %q, want %q", stored.ID, player.ID)
	}
}

func TestPlayerRepositoryGetPlayerByEmailRejectsUnknownEmail(t *testing.T) {
	repo := NewPlayerRepository()

	_, err := repo.GetPlayerByEmail(context.Background(), "missing@example.com")
	if !errors.Is(err, auth.ErrPlayerNotFound) {
		t.Fatalf("GetPlayerByEmail error = %v, want ErrPlayerNotFound", err)
	}
}

func TestPlayerRepositoryRejectsDuplicatePlayerID(t *testing.T) {
	repo := NewPlayerRepository()
	ctx := context.Background()

	if err := repo.CreatePlayer(ctx, testPlayer("player-a", "a@example.com")); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	err := repo.CreatePlayer(ctx, testPlayer("player-a", "different@example.com"))
	if !errors.Is(err, auth.ErrPlayerAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrPlayerAlreadyExists", err)
	}
}

func TestPlayerRepositoryRejectsDuplicateEmail(t *testing.T) {
	repo := NewPlayerRepository()
	ctx := context.Background()

	if err := repo.CreatePlayer(ctx, testPlayer("player-a", "a@example.com")); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	err := repo.CreatePlayer(ctx, testPlayer("player-b", "a@example.com"))
	if !errors.Is(err, auth.ErrPlayerAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrPlayerAlreadyExists", err)
	}
}

func TestPlayerRepositoryGetRejectsUnknownPlayer(t *testing.T) {
	repo := NewPlayerRepository()

	_, err := repo.GetPlayer(context.Background(), "missing-player")
	if !errors.Is(err, auth.ErrPlayerNotFound) {
		t.Fatalf("Get error = %v, want ErrPlayerNotFound", err)
	}
}

func TestPlayerRepositorySaveExistingPlayer(t *testing.T) {
	repo := NewPlayerRepository()
	player := testPlayer("player-a", "a@example.com")
	ctx := context.Background()

	if err := repo.CreatePlayer(ctx, player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	player.DisplayName = "Changed Name"
	if err := repo.SavePlayer(ctx, player); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	stored, err := repo.GetPlayer(ctx, player.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.DisplayName != "Changed Name" {
		t.Fatalf("DisplayName = %q, want %q", stored.DisplayName, "Changed Name")
	}
}

func TestPlayerRepositorySaveUpdatesEmailIndex(t *testing.T) {
	repo := NewPlayerRepository()
	player := testPlayer("player-a", "old@example.com")
	ctx := context.Background()

	if err := repo.CreatePlayer(ctx, player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	player.Email = "new@example.com"
	if err := repo.SavePlayer(ctx, player); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := repo.GetPlayerByEmail(ctx, "old@example.com"); !errors.Is(err, auth.ErrPlayerNotFound) {
		t.Fatalf("GetPlayerByEmail(old) error = %v, want ErrPlayerNotFound", err)
	}
	stored, err := repo.GetPlayerByEmail(ctx, "new@example.com")
	if err != nil {
		t.Fatalf("GetPlayerByEmail(new) failed: %v", err)
	}
	if stored.ID != player.ID {
		t.Fatalf("Player.ID = %q, want %q", stored.ID, player.ID)
	}
}

func TestPlayerRepositorySaveRejectsUnknownPlayer(t *testing.T) {
	repo := NewPlayerRepository()

	err := repo.SavePlayer(context.Background(), testPlayer("missing-player", "missing@example.com"))
	if !errors.Is(err, auth.ErrPlayerNotFound) {
		t.Fatalf("Save error = %v, want ErrPlayerNotFound", err)
	}
}

func TestPlayerRepositoryStoresDetachedValues(t *testing.T) {
	repo := NewPlayerRepository()
	ctx := context.Background()
	player := testPlayer("player-a", "a@example.com")

	if err := repo.CreatePlayer(ctx, player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	player.DisplayName = "changed"
	player.PasswordHash[0] = 'X' // mutate the caller's backing array in place

	stored, err := repo.GetPlayer(ctx, "player-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.DisplayName != "Test Player" {
		t.Fatalf("stored DisplayName after source mutation = %q, want %q", stored.DisplayName, "Test Player")
	}
	if string(stored.PasswordHash) != "hash" {
		t.Fatalf("stored PasswordHash after source mutation = %q, want %q", stored.PasswordHash, "hash")
	}

	stored.PasswordHash[0] = 'Y' // mutate the fetched copy's backing array in place
	again, err := repo.GetPlayer(ctx, "player-a")
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if string(again.PasswordHash) != "hash" {
		t.Fatalf("stored PasswordHash after fetched copy mutation = %q, want %q", again.PasswordHash, "hash")
	}
}

func TestPlayerRepositoryRejectsNilPlayer(t *testing.T) {
	repo := NewPlayerRepository()
	ctx := context.Background()

	if err := repo.CreatePlayer(ctx, nil); err == nil {
		t.Fatal("expected Create to reject nil player")
	}
	if err := repo.SavePlayer(ctx, nil); err == nil {
		t.Fatal("expected Save to reject nil player")
	}
}

func TestPlayerRepositoryHonorsCancelledContext(t *testing.T) {
	repo := NewPlayerRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.CreatePlayer(ctx, testPlayer("player-a", "a@example.com")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetPlayer(ctx, "player-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetPlayerByEmail(ctx, "a@example.com"); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetPlayerByEmail error = %v, want context.Canceled", err)
	}
	if err := repo.SavePlayer(ctx, testPlayer("player-a", "a@example.com")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}
}
