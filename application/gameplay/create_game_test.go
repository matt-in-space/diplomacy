package gameplay_test

import (
	"context"
	"errors"
	"testing"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
)

func createGameTestMap() *gamemap.GameMap {
	return &gamemap.GameMap{
		ID:      "test-map",
		Nations: []gamemap.NationID{"eng"},
	}
}

func TestCreateGameCreatesAndPersistsGame(t *testing.T) {
	games := memory.NewGameRepository()
	maps := memory.NewGameMapRepository(createGameTestMap())
	service := gameplay.NewGameplayService(games, maps)
	ctx := context.Background()

	assignments := map[gamemap.NationID]game.PlayerID{"eng": "player-a"}
	stored, err := service.CreateGame(ctx, "game-a", "test-map", assignments)
	if err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}
	if stored.Game.ID != "game-a" {
		t.Fatalf("Game.ID = %q, want %q", stored.Game.ID, "game-a")
	}
	if stored.Game.Assignments["eng"] != "player-a" {
		t.Fatalf("Assignments[eng] = %q, want %q", stored.Game.Assignments["eng"], "player-a")
	}
	if stored.Version != 0 {
		t.Fatalf("Version = %d, want 0", stored.Version)
	}

	fetched, err := games.GetGame(ctx, "game-a")
	if err != nil {
		t.Fatalf("GetGame failed: %v", err)
	}
	if fetched.Game.ID != "game-a" {
		t.Fatalf("persisted Game.ID = %q, want %q", fetched.Game.ID, "game-a")
	}
}

func TestCreateGamePropagatesMapNotFound(t *testing.T) {
	games := memory.NewGameRepository()
	maps := memory.NewGameMapRepository()
	service := gameplay.NewGameplayService(games, maps)

	_, err := service.CreateGame(context.Background(), "game-a", "missing-map", nil)
	if !errors.Is(err, gameplay.ErrMapNotFound) {
		t.Fatalf("CreateGame error = %v, want ErrMapNotFound", err)
	}
}

func TestCreateGameRejectsDuplicateID(t *testing.T) {
	games := memory.NewGameRepository()
	maps := memory.NewGameMapRepository(createGameTestMap())
	service := gameplay.NewGameplayService(games, maps)
	ctx := context.Background()

	assignments := map[gamemap.NationID]game.PlayerID{"eng": "player-a"}
	if _, err := service.CreateGame(ctx, "game-a", "test-map", assignments); err != nil {
		t.Fatalf("first CreateGame failed: %v", err)
	}

	_, err := service.CreateGame(ctx, "game-a", "test-map", assignments)
	if !errors.Is(err, gameplay.ErrGameAlreadyExists) {
		t.Fatalf("second CreateGame error = %v, want ErrGameAlreadyExists", err)
	}
}
