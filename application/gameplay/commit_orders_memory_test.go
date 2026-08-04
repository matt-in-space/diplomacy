package gameplay_test

import (
	"context"
	"testing"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
)

// TestGameplayServiceCommitOrdersProcessesGameAfterFinalCommitment exercises
// CommitOrders against the real in-memory repository rather than the
// call-counting fakes used elsewhere in this package's tests: ProcessGame
// loops through several Get/Save round trips as phases advance, and only a
// real, versioned store reflects each Save in the next Get the way a
// production repository would. This lives in the external gameplay_test
// package, rather than alongside CommitOrders' other tests, because
// gameplay's own internal tests can't import infrastructure/memory, which
// itself imports gameplay to implement its interfaces.
func TestGameplayServiceCommitOrdersProcessesGameAfterFinalCommitment(t *testing.T) {
	ctx := context.Background()
	games := memory.NewGameRepository()
	g := commitOrdersMemoryTestGame()
	if err := games.CreateGame(ctx, g); err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}
	maps := memory.NewGameMapRepository(&gamemap.GameMap{
		ID:      "test-map",
		Nations: []gamemap.NationID{"eng"},
	})
	service := gameplay.NewGameplayService(games, maps)

	cmd := gameplay.CommitOrdersCommand{
		GameID:          "test-game",
		PlayerID:        "player-a",
		ExpectedVersion: 0,
		NationID:        "eng",
	}
	if err := service.CommitOrders(ctx, cmd); err != nil {
		t.Fatalf("CommitOrders failed: %v", err)
	}

	stored, err := games.GetGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGame failed: %v", err)
	}
	// With no dislodged units, accept retreats and resolve retreats both run
	// without anyone needing to commit, carrying processing all the way into
	// fall's accept orders phase, where it stops waiting for a commitment
	// that hasn't happened yet.
	if stored.Version != 5 {
		t.Fatalf("stored version = %d, want 5", stored.Version)
	}
	if stored.Game.Turn.Phase != game.AcceptOrders {
		t.Fatalf("Turn.Phase = %q, want %q", stored.Game.Turn.Phase, game.AcceptOrders)
	}
	if stored.Game.Turn.Season != game.Fall {
		t.Fatalf("Turn.Season = %q, want %q", stored.Game.Turn.Season, game.Fall)
	}
	if len(stored.Game.CommittedOrders) != 0 {
		t.Fatalf("CommittedOrders length = %d, want 0", len(stored.Game.CommittedOrders))
	}
}

func commitOrdersMemoryTestGame() *game.Game {
	return &game.Game{
		ID:    "test-game",
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
