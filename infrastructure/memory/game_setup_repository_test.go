package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
)

func testGameSetup(id game.GameID, hostID game.PlayerID) *lobby.GameSetup {
	return &lobby.GameSetup{
		ID:        id,
		MapID:     "test-map",
		HostID:    hostID,
		CreatedAt: time.Unix(0, 0).UTC(),
	}
}

func TestGameSetupRepositoryCreateAndGet(t *testing.T) {
	repo := NewGameSetupRepository()
	setup := testGameSetup("game-a", "player-a")

	if err := repo.CreateGameSetup(context.Background(), setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetGameSetup(context.Background(), setup.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.HostID != setup.HostID {
		t.Fatalf("HostID = %q, want %q", stored.HostID, setup.HostID)
	}
}

func TestGameSetupRepositoryGetRejectsUnknownSetup(t *testing.T) {
	repo := NewGameSetupRepository()

	_, err := repo.GetGameSetup(context.Background(), "missing-game")
	if !errors.Is(err, lobby.ErrGameSetupNotFound) {
		t.Fatalf("Get error = %v, want ErrGameSetupNotFound", err)
	}
}

func TestGameSetupRepositoryRejectsDuplicateID(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()

	if err := repo.CreateGameSetup(ctx, testGameSetup("game-a", "player-a")); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	err := repo.CreateGameSetup(ctx, testGameSetup("game-a", "player-b"))
	if !errors.Is(err, lobby.ErrGameSetupAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrGameSetupAlreadyExists", err)
	}
}

func TestGameSetupRepositorySaveExistingSetup(t *testing.T) {
	repo := NewGameSetupRepository()
	setup := testGameSetup("game-a", "player-a")
	ctx := context.Background()

	if err := repo.CreateGameSetup(ctx, setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	cancelledAt := time.Unix(100, 0).UTC()
	setup.CancelledAt = &cancelledAt
	if err := repo.SaveGameSetup(ctx, setup); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	stored, err := repo.GetGameSetup(ctx, setup.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.CancelledAt == nil || !stored.CancelledAt.Equal(cancelledAt) {
		t.Fatalf("CancelledAt = %v, want %v", stored.CancelledAt, cancelledAt)
	}
}

func TestGameSetupRepositorySaveRejectsUnknownSetup(t *testing.T) {
	repo := NewGameSetupRepository()

	err := repo.SaveGameSetup(context.Background(), testGameSetup("missing-game", "player-a"))
	if !errors.Is(err, lobby.ErrGameSetupNotFound) {
		t.Fatalf("Save error = %v, want ErrGameSetupNotFound", err)
	}
}

func TestGameSetupRepositoryListGameSetupsForPlayer(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()

	if err := repo.CreateGameSetup(ctx, testGameSetup("game-a", "player-a")); err != nil {
		t.Fatalf("Create game-a failed: %v", err)
	}
	if err := repo.CreateGameSetup(ctx, testGameSetup("game-b", "player-a")); err != nil {
		t.Fatalf("Create game-b failed: %v", err)
	}
	if err := repo.CreateGameSetup(ctx, testGameSetup("game-c", "player-b")); err != nil {
		t.Fatalf("Create game-c failed: %v", err)
	}

	setups, err := repo.ListGameSetupsForPlayer(ctx, "player-a")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(setups) != 2 {
		t.Fatalf("len(setups) = %d, want 2", len(setups))
	}
}

func TestGameSetupRepositoryStoresDetachedValues(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()
	setup := testGameSetup("game-a", "player-a")
	cancelledAt := time.Unix(100, 0).UTC()
	want := cancelledAt // a value copy, so mutating *setup.CancelledAt below can't move this
	setup.CancelledAt = &cancelledAt

	if err := repo.CreateGameSetup(ctx, setup); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	*setup.CancelledAt = time.Unix(200, 0).UTC() // mutate the caller's pointee in place

	stored, err := repo.GetGameSetup(ctx, "game-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !stored.CancelledAt.Equal(want) {
		t.Fatalf("stored CancelledAt after source mutation = %v, want %v", stored.CancelledAt, want)
	}

	*stored.CancelledAt = time.Unix(300, 0).UTC() // mutate the fetched copy's pointee in place
	again, err := repo.GetGameSetup(ctx, "game-a")
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if !again.CancelledAt.Equal(want) {
		t.Fatalf("stored CancelledAt after fetched copy mutation = %v, want %v", again.CancelledAt, want)
	}
}

func TestGameSetupRepositoryRejectsNilSetup(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx := context.Background()

	if err := repo.CreateGameSetup(ctx, nil); err == nil {
		t.Fatal("expected Create to reject nil setup")
	}
	if err := repo.SaveGameSetup(ctx, nil); err == nil {
		t.Fatal("expected Save to reject nil setup")
	}
}

func TestGameSetupRepositoryHonorsCancelledContext(t *testing.T) {
	repo := NewGameSetupRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.CreateGameSetup(ctx, testGameSetup("game-a", "player-a")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetGameSetup(ctx, "game-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if err := repo.SaveGameSetup(ctx, testGameSetup("game-a", "player-a")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}
	if _, err := repo.ListGameSetupsForPlayer(ctx, "player-a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("List error = %v, want context.Canceled", err)
	}
}
