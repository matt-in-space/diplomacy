package gameplay

import (
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
		Orders:          make(map[game.UnitID]game.Order),
		CommittedOrders: make(map[gamemap.NationID]struct{}),
	}
}
