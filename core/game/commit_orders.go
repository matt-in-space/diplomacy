package game

import (
	"fmt"
	"slices"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func (g *Game) CommitOrders(nationID gamemap.NationID, gm *gamemap.GameMap) error {
	if gm == nil {
		return fmt.Errorf("game map is required")
	}
	if gm.ID != g.MapID {
		return fmt.Errorf("game map %q does not match game map %q", gm.ID, g.MapID)
	}
	if g.Turn.Phase != AcceptOrders {
		return fmt.Errorf("cannot commit orders during phase %q", g.Turn.Phase)
	}
	if !slices.Contains(gm.Nations, nationID) {
		return fmt.Errorf("nation %q not found", nationID)
	}
	if _, ok := g.Assignments[nationID]; !ok {
		return fmt.Errorf("nation %q is not assigned", nationID)
	}
	if g.CommittedOrders == nil {
		g.CommittedOrders = make(map[gamemap.NationID]struct{})
	}

	g.CommittedOrders[nationID] = struct{}{}
	return nil
}
