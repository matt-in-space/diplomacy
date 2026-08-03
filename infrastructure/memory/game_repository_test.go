package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func testGame(id game.GameID) *game.Game {
	return &game.Game{
		ID:    id,
		MapID: "test-map",
		Turn:  game.StartingTurn(),
		Assignments: map[gamemap.NationID]game.PlayerID{
			"eng": "player-a",
		},
		Units: map[game.UnitID]game.Unit{
			"unit-a": {
				ID:         "unit-a",
				NationID:   "eng",
				ProvinceID: "lon",
				Type:       game.UnitTypeFleet,
				Coast:      "lon",
			},
		},
		CommittedOrders: make(map[gamemap.NationID]struct{}),
	}
}

func TestGameRepositoryCreateAndGetGame(t *testing.T) {
	repo := NewGameRepository()
	g := testGame("test-game")

	if err := repo.CreateGame(context.Background(), g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetGame(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.Version != 0 {
		t.Fatalf("Version = %d, want 0", stored.Version)
	}
	if stored.Game.ID != g.ID {
		t.Fatalf("Game.ID = %q, want %q", stored.Game.ID, g.ID)
	}
}

func TestGameRepositoryRejectsDuplicateGame(t *testing.T) {
	repo := NewGameRepository()
	g := testGame("test-game")
	ctx := context.Background()

	if err := repo.CreateGame(ctx, g); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if err := repo.CreateGame(ctx, g); !errors.Is(err, gameplay.ErrGameAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrGameAlreadyExists", err)
	}
}

func TestGameRepositoryGetRejectsUnknownGame(t *testing.T) {
	repo := NewGameRepository()

	_, err := repo.GetGame(context.Background(), "missing-game")
	if !errors.Is(err, gameplay.ErrGameNotFound) {
		t.Fatalf("Get error = %v, want ErrGameNotFound", err)
	}
}

func TestGameRepositorySaveUpdatesGameAndVersion(t *testing.T) {
	repo := NewGameRepository()
	ctx := context.Background()
	g := testGame("test-game")
	if err := repo.CreateGame(ctx, g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	stored.Game.Turn.Year = 2

	version, err := repo.SaveGame(ctx, stored.Game, stored.Version)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if version != 1 {
		t.Fatalf("Save version = %d, want 1", version)
	}

	updated, err := repo.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("Get after Save failed: %v", err)
	}
	if updated.Version != 1 {
		t.Fatalf("stored Version = %d, want 1", updated.Version)
	}
	if updated.Game.Turn.Year != 2 {
		t.Fatalf("stored Turn.Year = %d, want 2", updated.Game.Turn.Year)
	}
}

func TestGameRepositoryRejectsStaleSaveGame(t *testing.T) {
	repo := NewGameRepository()
	ctx := context.Background()
	g := testGame("test-game")
	if err := repo.CreateGame(ctx, g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	first, err := repo.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("first Get failed: %v", err)
	}
	second, err := repo.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}

	first.Game.Turn.Year = 2
	if _, err := repo.SaveGame(ctx, first.Game, first.Version); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	second.Game.Turn.Year = 3
	if _, err := repo.SaveGame(ctx, second.Game, second.Version); !errors.Is(err, gameplay.ErrConcurrentUpdate) {
		t.Fatalf("stale Save error = %v, want ErrConcurrentUpdate", err)
	}

	stored, err := repo.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("final Get failed: %v", err)
	}
	if stored.Game.Turn.Year != 2 {
		t.Fatalf("stored Turn.Year = %d, want 2", stored.Game.Turn.Year)
	}
}

func TestGameRepositorySaveRejectsUnknownGame(t *testing.T) {
	repo := NewGameRepository()
	g := testGame("missing-game")

	_, err := repo.SaveGame(context.Background(), g, 0)
	if !errors.Is(err, gameplay.ErrGameNotFound) {
		t.Fatalf("Save error = %v, want ErrGameNotFound", err)
	}
}

func TestGameRepositoryStoresDetachedSnapshots(t *testing.T) {
	repo := NewGameRepository()
	ctx := context.Background()
	g := testGame("test-game")

	if err := repo.CreateGame(ctx, g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	g.Turn.Year = 9
	g.Assignments["eng"] = "changed-player"
	g.Units["unit-a"] = game.Unit{ID: "unit-a", ProvinceID: "changed"}

	stored, err := repo.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored.Game.Turn.Year != 1 {
		t.Fatalf("Turn.Year after source mutation = %d, want 1", stored.Game.Turn.Year)
	}
	if got := stored.Game.Assignments["eng"]; got != "player-a" {
		t.Fatalf("Assignments[eng] after source mutation = %q, want player-a", got)
	}
	if got := stored.Game.Units["unit-a"].ProvinceID; got != "lon" {
		t.Fatalf("unit province after source mutation = %q, want lon", got)
	}

	stored.Game.Turn.Year = 8
	stored.Game.Units["unit-a"] = game.Unit{ID: "unit-a", ProvinceID: "changed-again"}
	again, err := repo.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if again.Game.Turn.Year != 1 {
		t.Fatalf("Turn.Year after fetched snapshot mutation = %d, want 1", again.Game.Turn.Year)
	}
	if got := again.Game.Units["unit-a"].ProvinceID; got != "lon" {
		t.Fatalf("unit province after fetched snapshot mutation = %q, want lon", got)
	}
}

func TestGameRepositorySaveStoresDetachedSnapshot(t *testing.T) {
	repo := NewGameRepository()
	ctx := context.Background()
	g := testGame("test-game")
	if err := repo.CreateGame(ctx, g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	stored.Game.Turn.Year = 2
	if _, err := repo.SaveGame(ctx, stored.Game, stored.Version); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	stored.Game.Turn.Year = 3

	again, err := repo.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("Get after Save failed: %v", err)
	}
	if again.Game.Turn.Year != 2 {
		t.Fatalf("stored Turn.Year after source mutation = %d, want 2", again.Game.Turn.Year)
	}
}

func TestGameRepositoryHonorsCancelledContext(t *testing.T) {
	repo := NewGameRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.CreateGame(ctx, testGame("test-game")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetGame(ctx, "test-game"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if _, err := repo.SaveGame(ctx, testGame("test-game"), 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}
}
