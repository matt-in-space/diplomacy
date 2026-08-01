package game

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// A RetreatOrder moves a dislodged unit to a province it was not forced out
// of. It is only valid during the accept retreats phase, for units that were
// dislodged during the preceding order resolution.
type RetreatOrder struct {
	BaseOrder
	Target      gamemap.ProvinceID
	TargetCoast gamemap.CoastID
}

func NewRetreatOrder(unit UnitID, nation gamemap.NationID, target gamemap.ProvinceID, targetCoast gamemap.CoastID) RetreatOrder {
	return RetreatOrder{
		BaseOrder: BaseOrder{
			UnitID:   unit,
			NationID: nation,
		},
		Target:      target,
		TargetCoast: targetCoast,
	}
}

// A DisbandOrder removes a dislodged unit from the game instead of retreating
// it, either by player choice or because no legal retreat exists.
type DisbandOrder struct {
	BaseOrder
}

func NewDisbandOrder(unit UnitID, nation gamemap.NationID) DisbandOrder {
	return DisbandOrder{
		BaseOrder: BaseOrder{
			UnitID:   unit,
			NationID: nation,
		},
	}
}

// validateRetreatOrder checks that a retreat order's destination is both
// reachable (the same adjacency and coast rules a move order uses) and
// permitted by the retreat-specific restrictions: not occupied, not a
// standoff province, and not the attacker's origin unless the attacker
// arrived by convoy.
func (g *Game) validateRetreatOrder(order RetreatOrder, unit Unit, gm *gamemap.GameMap) error {
	// A dislodged unit has no current province; validate reachability as if
	// it were still standing where it was dislodged from.
	asIfOnBoard := unit
	asIfOnBoard.ProvinceID = unit.DislodgedFrom

	if err := g.validateUnitCanReach(asIfOnBoard, order.Target, order.TargetCoast, gm); err != nil {
		return err
	}

	constraints := g.retreatConstraintsFor(unit.DislodgedFrom)
	if _, occupied := constraints.occupied[order.Target]; occupied {
		return fmt.Errorf("province %q is occupied and cannot be retreated to", order.Target)
	}
	if _, standoff := constraints.standoffs[order.Target]; standoff {
		return fmt.Errorf("province %q was contested this turn and cannot be retreated to", order.Target)
	}
	if order.Target == constraints.attackerOrigin && !constraints.attackerConvoyed {
		return fmt.Errorf("unit %q cannot retreat to %q, the attacking unit's origin", unit.ID, order.Target)
	}

	return nil
}
