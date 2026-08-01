package game

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// CompleteOwnershipUpdate assigns each occupied supply center to its
// occupant's nation. A vacant center keeps its previous owner. This is not
// adjudication — there is no order or conflict to resolve, just a read of
// already-settled board state — so unlike CompleteOrderResolution and
// CompleteRetreatResolution, it takes no Resolution.
//
// This phase is only ever reached via Turn.Next()'s Fall branch, so no
// season check is needed here: Spring's retreat resolution advances straight
// to Fall's accept orders phase and never produces UpdateOwnership at all.
func (g *Game) CompleteOwnershipUpdate(gm *gamemap.GameMap) error {
	if g.Turn.Phase != UpdateOwnership {
		return fmt.Errorf("supply center ownership can only be updated in the %q phase", UpdateOwnership)
	}

	occupied := make(map[gamemap.ProvinceID]gamemap.NationID, len(g.Units))
	for _, unit := range g.Units {
		if unit.ProvinceID != "" {
			occupied[unit.ProvinceID] = unit.NationID
		}
	}

	for province := range g.SupplyCenterOwners {
		if nation, ok := occupied[province]; ok {
			g.SupplyCenterOwners[province] = nation
		}
	}

	g.Turn = g.Turn.Next()

	return nil
}
