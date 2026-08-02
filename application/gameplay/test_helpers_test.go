package gameplay

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

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
				Coast:      "lon",
			},
		},
		CommittedOrders: make(map[gamemap.NationID]struct{}),
	}
}

// dislodgeUnit marks an on-board unit as dislodged, clearing its province and
// recording where it was dislodged from.
func dislodgeUnit(t *testing.T, g *game.Game, id game.UnitID) {
	t.Helper()

	unit, ok := g.Units[id]
	if !ok {
		t.Fatalf("unit %q not found", id)
	}
	unit.DislodgedFrom = unit.ProvinceID
	unit.ProvinceID = ""
	g.Units[id] = unit
}
