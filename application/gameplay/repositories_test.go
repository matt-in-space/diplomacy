package gameplay

import (
	"context"
	"errors"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestMemoryGameRepositoryCreateAndGet(t *testing.T) {
	repo := NewMemoryGameRepository()
	g := repositoryTestGame("test-game")

	if err := repo.Create(context.Background(), g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.Get(context.Background(), g.ID)
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

func TestMemoryGameRepositoryRejectsDuplicateGame(t *testing.T) {
	repo := NewMemoryGameRepository()
	g := repositoryTestGame("test-game")
	ctx := context.Background()

	if err := repo.Create(ctx, g); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if err := repo.Create(ctx, g); !errors.Is(err, ErrGameAlreadyExists) {
		t.Fatalf("second Create error = %v, want ErrGameAlreadyExists", err)
	}
}

func TestMemoryGameRepositoryGetRejectsUnknownGame(t *testing.T) {
	repo := NewMemoryGameRepository()

	_, err := repo.Get(context.Background(), "missing-game")
	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("Get error = %v, want ErrGameNotFound", err)
	}
}

func TestMemoryGameRepositorySaveUpdatesGameAndVersion(t *testing.T) {
	repo := NewMemoryGameRepository()
	ctx := context.Background()
	g := repositoryTestGame("test-game")
	if err := repo.Create(ctx, g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.Get(ctx, g.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	stored.Game.Turn.Year = 2

	version, err := repo.Save(ctx, stored.Game, stored.Version)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if version != 1 {
		t.Fatalf("Save version = %d, want 1", version)
	}

	updated, err := repo.Get(ctx, g.ID)
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

func TestMemoryGameRepositoryRejectsStaleSave(t *testing.T) {
	repo := NewMemoryGameRepository()
	ctx := context.Background()
	g := repositoryTestGame("test-game")
	if err := repo.Create(ctx, g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	first, err := repo.Get(ctx, g.ID)
	if err != nil {
		t.Fatalf("first Get failed: %v", err)
	}
	second, err := repo.Get(ctx, g.ID)
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}

	first.Game.Turn.Year = 2
	if _, err := repo.Save(ctx, first.Game, first.Version); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	second.Game.Turn.Year = 3
	if _, err := repo.Save(ctx, second.Game, second.Version); !errors.Is(err, ErrConcurrentUpdate) {
		t.Fatalf("stale Save error = %v, want ErrConcurrentUpdate", err)
	}

	stored, err := repo.Get(ctx, g.ID)
	if err != nil {
		t.Fatalf("final Get failed: %v", err)
	}
	if stored.Game.Turn.Year != 2 {
		t.Fatalf("stored Turn.Year = %d, want 2", stored.Game.Turn.Year)
	}
}

func TestMemoryGameRepositorySaveRejectsUnknownGame(t *testing.T) {
	repo := NewMemoryGameRepository()
	g := repositoryTestGame("missing-game")

	_, err := repo.Save(context.Background(), g, 0)
	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("Save error = %v, want ErrGameNotFound", err)
	}
}

func TestMemoryGameRepositoryStoresDetachedSnapshots(t *testing.T) {
	repo := NewMemoryGameRepository()
	ctx := context.Background()
	g := repositoryTestGame("test-game")

	if err := repo.Create(ctx, g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	g.Turn.Year = 9
	g.Assignments["eng"] = "changed-player"
	g.Units["unit-a"] = game.Unit{ID: "unit-a", ProvinceID: "changed"}

	stored, err := repo.Get(ctx, g.ID)
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
	stored.Game.Positions["lon"] = "changed-unit"
	again, err := repo.Get(ctx, g.ID)
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if again.Game.Turn.Year != 1 {
		t.Fatalf("Turn.Year after fetched snapshot mutation = %d, want 1", again.Game.Turn.Year)
	}
	if got := again.Game.Positions["lon"]; got != "unit-a" {
		t.Fatalf("Positions[lon] after fetched snapshot mutation = %q, want unit-a", got)
	}
}

func TestMemoryGameRepositorySaveStoresDetachedSnapshot(t *testing.T) {
	repo := NewMemoryGameRepository()
	ctx := context.Background()
	g := repositoryTestGame("test-game")
	if err := repo.Create(ctx, g); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stored, err := repo.Get(ctx, g.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	stored.Game.Turn.Year = 2
	if _, err := repo.Save(ctx, stored.Game, stored.Version); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	stored.Game.Turn.Year = 3

	again, err := repo.Get(ctx, g.ID)
	if err != nil {
		t.Fatalf("Get after Save failed: %v", err)
	}
	if again.Game.Turn.Year != 2 {
		t.Fatalf("stored Turn.Year after source mutation = %d, want 2", again.Game.Turn.Year)
	}
}

func TestMemoryGameRepositoryHonorsCancelledContext(t *testing.T) {
	repo := NewMemoryGameRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.Create(ctx, repositoryTestGame("test-game")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
	if _, err := repo.Get(ctx, "test-game"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want context.Canceled", err)
	}
	if _, err := repo.Save(ctx, repositoryTestGame("test-game"), 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}
}

func repositoryTestGame(id game.GameID) *game.Game {
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
			},
		},
		Positions: map[gamemap.ProvinceID]game.UnitID{
			"lon": "unit-a",
		},
		FleetCoasts: map[game.UnitID]gamemap.CoastID{
			"unit-a": "lon",
		},
		Orders:          make(map[game.UnitID]game.Order),
		PendingRetreats: make(map[game.UnitID]game.Dislodgement),
	}
}
