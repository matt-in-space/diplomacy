package gameplay

import (
	"context"
	"errors"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
)

func TestMemoryPlayerRepositoryCreateAndGet(t *testing.T) {
	repo := NewMemoryPlayerRepository()
	player := &game.Player{ID: "player-a"}

	if err := repo.Create(context.Background(), player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.Get(context.Background(), player.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.ID != player.ID {
		t.Fatalf("Player.ID = %q, want %q", stored.ID, player.ID)
	}
}

func TestMemoryPlayerRepositoryRejectsDuplicatePlayer(t *testing.T) {
	repo := NewMemoryPlayerRepository()
	player := &game.Player{ID: "player-a"}
	ctx := context.Background()

	if err := repo.Create(ctx, player); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if err := repo.Create(ctx, player); !errors.Is(err, ErrPlayerAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrPlayerAlreadyExists", err)
	}
}

func TestMemoryPlayerRepositoryGetRejectsUnknownPlayer(t *testing.T) {
	repo := NewMemoryPlayerRepository()

	_, err := repo.Get(context.Background(), "missing-player")
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("Get error = %v, want ErrPlayerNotFound", err)
	}
}

func TestMemoryPlayerRepositorySaveExistingPlayer(t *testing.T) {
	repo := NewMemoryPlayerRepository()
	player := &game.Player{ID: "player-a"}
	ctx := context.Background()

	if err := repo.Create(ctx, player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.Save(ctx, player); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestMemoryPlayerRepositorySaveRejectsUnknownPlayer(t *testing.T) {
	repo := NewMemoryPlayerRepository()

	err := repo.Save(context.Background(), &game.Player{ID: "missing-player"})
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("Save error = %v, want ErrPlayerNotFound", err)
	}
}

func TestMemoryPlayerRepositoryStoresDetachedValues(t *testing.T) {
	repo := NewMemoryPlayerRepository()
	ctx := context.Background()
	player := &game.Player{ID: "player-a"}

	if err := repo.Create(ctx, player); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	player.ID = "changed-player"

	stored, err := repo.Get(ctx, "player-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.ID != "player-a" {
		t.Fatalf("stored ID after source mutation = %q, want player-a", stored.ID)
	}

	stored.ID = "another-player"
	again, err := repo.Get(ctx, "player-a")
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if again.ID != "player-a" {
		t.Fatalf("stored ID after fetched value mutation = %q, want player-a", again.ID)
	}
}

func TestMemoryPlayerRepositoryRejectsNilPlayer(t *testing.T) {
	repo := NewMemoryPlayerRepository()
	ctx := context.Background()

	if err := repo.Create(ctx, nil); err == nil {
		t.Fatal("expected Create to reject nil player")
	}
	if err := repo.Save(ctx, nil); err == nil {
		t.Fatal("expected Save to reject nil player")
	}
}

func TestMemoryPlayerRepositoryHonorsCancelledContext(t *testing.T) {
	repo := NewMemoryPlayerRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.Create(ctx, &game.Player{ID: "player-a"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
	if _, err := repo.Get(ctx, "player-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if err := repo.Save(ctx, &game.Player{ID: "player-a"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}
}
