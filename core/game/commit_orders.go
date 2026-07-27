package game

import (
	"fmt"
	"slices"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func (g *Game) CommitOrders(nationID gamemap.NationID, gm *gamemap.GameMap) error {
	if !slices.Contains(gm.Nations, nationID) {
		return fmt.Errorf("Unknown nation ID: %s", nationID)
	}

	g.CommittedOrders[nationID] = struct{}{}
	return nil
}
