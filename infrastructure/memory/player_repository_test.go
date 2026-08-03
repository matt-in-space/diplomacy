package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/game"
)

func TestPlayerRepositoryCreateAndGetPlayer(t *testing.T) {
	repo := NewPlayerRepository()
	player := &game.Player{ID: "player-a"}

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
}

func TestPlayerRepositoryRejectsDuplicatePlayer(t *testing.T) {
	repo := NewPlayerRepository()
	player := &game.Player{ID: "player-a"}
	ctx := context.Background()

	if err := repo.CreatePlayer(ctx, player); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if err := repo.CreatePlayer(ctx, player); !errors.Is(err, gameplay.ErrPlayerAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrPlayerAlreadyExists", err)
	}
}

func TestPlayerRepositoryGetRejectsUnknownPlayer(t *testing.T) {
	repo := NewPlayerRepository()

	_, err := repo.GetPlayer(context.Background(), "missing-player")
	if !errors.Is(err, gameplay.ErrPlayerNotFound) {
		t.Fatalf("Get error = %v, want ErrPlayerNotFound", err)
	}
}

func TestPlayerRepositorySaveExistingPlayer(t *testing.T) {
	repo := NewPlayerRepository()
	player := &game.Player{ID: "player-a"}
	ctx := context.Background()

	if err := repo.CreatePlayer(ctx, player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.SavePlayer(ctx, player); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestPlayerRepositorySaveRejectsUnknownPlayer(t *testing.T) {
	repo := NewPlayerRepository()

	err := repo.SavePlayer(context.Background(), &game.Player{ID: "missing-player"})
	if !errors.Is(err, gameplay.ErrPlayerNotFound) {
		t.Fatalf("Save error = %v, want ErrPlayerNotFound", err)
	}
}

func TestPlayerRepositoryStoresDetachedValues(t *testing.T) {
	repo := NewPlayerRepository()
	ctx := context.Background()
	player := &game.Player{ID: "player-a"}

	if err := repo.CreatePlayer(ctx, player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	player.ID = "changed-player"

	stored, err := repo.GetPlayer(ctx, "player-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.ID != "player-a" {
		t.Fatalf("stored ID after source mutation = %q, want player-a", stored.ID)
	}

	stored.ID = "another-player"
	again, err := repo.GetPlayer(ctx, "player-a")
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if again.ID != "player-a" {
		t.Fatalf("stored ID after fetched value mutation = %q, want player-a", again.ID)
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

	if err := repo.CreatePlayer(ctx, &game.Player{ID: "player-a"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetPlayer(ctx, "player-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if err := repo.SavePlayer(ctx, &game.Player{ID: "player-a"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}
}
