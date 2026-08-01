package game

import (
	"fmt"
	"slices"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// SubmitOrder validates and stores an order for the active input phase. Both
// the eligibility of the ordering unit and the set of legal order types
// depend on the phase: movement orders require an on-board unit during
// AcceptOrders; retreat orders require a dislodged unit during AcceptRetreats.
func (g *Game) SubmitOrder(order Order, gm *gamemap.GameMap) error {
	unit, err := g.validateOrderSubmission(order, gm)
	if err != nil {
		return err
	}

	switch g.Turn.Phase {
	case AcceptOrders:
		return g.submitMovementOrder(order, unit, gm)
	case AcceptRetreats:
		return g.submitRetreatOrder(order, unit, gm)
	default:
		return fmt.Errorf("cannot submit order during phase %q", g.Turn.Phase)
	}
}

// validateOrderSubmission performs the checks shared by every input phase:
// the order and map are well-formed, the phase accepts orders, the nation is
// known and has not committed, and the ordering unit exists and belongs to
// that nation. Phase-specific eligibility (on-board vs. dislodged) and order
// type are checked by the caller.
func (g *Game) validateOrderSubmission(order Order, gm *gamemap.GameMap) (Unit, error) {
	if order == nil {
		return Unit{}, fmt.Errorf("order is required")
	}
	if gm == nil {
		return Unit{}, fmt.Errorf("game map is required")
	}
	if gm.ID != g.MapID {
		return Unit{}, fmt.Errorf("game map %q does not match game map %q", gm.ID, g.MapID)
	}
	if !g.Turn.AcceptsOrders() {
		return Unit{}, fmt.Errorf("cannot submit order during phase %q", g.Turn.Phase)
	}

	nation := order.Nation()
	if !slices.Contains(gm.Nations, nation) {
		return Unit{}, fmt.Errorf("order nation %q not found", nation)
	}

	if _, ok := g.CommittedOrders[nation]; ok {
		return Unit{}, fmt.Errorf("order from nation %q has already been committed", nation)
	}

	unitID := order.Unit()
	unit, ok := g.Units[unitID]
	if !ok {
		return Unit{}, fmt.Errorf("unit %q not found", unitID)
	}
	if unit.NationID != nation {
		return Unit{}, fmt.Errorf("unit %q belongs to nation %q, not %q", unitID, unit.NationID, nation)
	}

	return unit, nil
}

func (g *Game) submitMovementOrder(order Order, unit Unit, gm *gamemap.GameMap) error {
	if unit.Dislodged() {
		return fmt.Errorf("unit %q is not on the board", unit.ID)
	}

	switch order := order.(type) {
	case HoldOrder:
		// noop: hold orders have no additional validation
	case MoveOrder:
		if err := g.validateMoveOrder(order, unit, gm); err != nil {
			return err
		}
	case SupportHoldOrder:
		if err := g.validateSupportHoldOrder(order, unit, gm); err != nil {
			return err
		}
	case SupportMoveOrder:
		if err := g.validateSupportMoveOrder(order, unit, gm); err != nil {
			return err
		}
	case ConvoyOrder:
		if err := g.validateConvoyOrder(order, unit, gm); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported order type %T", order)
	}

	g.storeOrder(order)
	return nil
}

func (g *Game) submitRetreatOrder(order Order, unit Unit, gm *gamemap.GameMap) error {
	if !unit.Dislodged() {
		return fmt.Errorf("unit %q is not dislodged", unit.ID)
	}

	switch order := order.(type) {
	case DisbandOrder:
		// noop: disbanding has no additional validation
	case RetreatOrder:
		if err := g.validateRetreatOrder(order, unit, gm); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported order type %T", order)
	}

	g.storeOrder(order)
	return nil
}

func (g *Game) storeOrder(order Order) {
	if g.Orders == nil {
		g.Orders = make(map[UnitID]Order)
	}
	g.Orders[order.Unit()] = order
}
