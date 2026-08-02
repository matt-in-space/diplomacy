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
	if err := g.validateOrderSubmission(order, gm); err != nil {
		return err
	}

	switch g.Turn.Phase {
	case AcceptOrders:
		return g.submitMovementOrder(order, gm)
	case AcceptRetreats:
		return g.submitRetreatOrder(order, gm)
	default:
		return fmt.Errorf("cannot submit order during phase %q", g.Turn.Phase)
	}
}

// validateOrderSubmission performs the checks shared by every input phase:
// the order and map are well-formed, the phase accepts orders, and the
// nation is known and has not committed. Order-type legality, the ordering
// unit's existence and eligibility, and phase-specific validation are all
// checked by the caller once it knows which order type it's dealing with.
func (g *Game) validateOrderSubmission(order Order, gm *gamemap.GameMap) error {
	if order == nil {
		return fmt.Errorf("order is required")
	}
	if gm == nil {
		return fmt.Errorf("game map is required")
	}
	if gm.ID != g.MapID {
		return fmt.Errorf("game map %q does not match game map %q", gm.ID, g.MapID)
	}
	if !g.Turn.AcceptsOrders() {
		return fmt.Errorf("cannot submit order during phase %q", g.Turn.Phase)
	}

	nation := order.Nation()
	if !slices.Contains(gm.Nations, nation) {
		return fmt.Errorf("order nation %q not found", nation)
	}

	if _, ok := g.CommittedOrders[nation]; ok {
		return fmt.Errorf("order from nation %q has already been committed", nation)
	}

	return nil
}

// unitForOrder resolves the unit a UnitOrder names and checks that it belongs
// to the order's nation.
func (g *Game) unitForOrder(order UnitOrder) (Unit, error) {
	unitID := order.Unit()
	unit, ok := g.Units[unitID]
	if !ok {
		return Unit{}, fmt.Errorf("unit %q not found", unitID)
	}
	if unit.NationID != order.Nation() {
		return Unit{}, fmt.Errorf("unit %q belongs to nation %q, not %q", unitID, unit.NationID, order.Nation())
	}
	return unit, nil
}

func (g *Game) submitMovementOrder(order Order, gm *gamemap.GameMap) error {
	unitOrder, ok := order.(UnitOrder)
	if !ok {
		return fmt.Errorf("unsupported order type %T", order)
	}
	unit, err := g.unitForOrder(unitOrder)
	if err != nil {
		return err
	}
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

	g.storeOrder(unitOrder)
	return nil
}

func (g *Game) submitRetreatOrder(order Order, gm *gamemap.GameMap) error {
	unitOrder, ok := order.(UnitOrder)
	if !ok {
		return fmt.Errorf("unsupported order type %T", order)
	}
	unit, err := g.unitForOrder(unitOrder)
	if err != nil {
		return err
	}
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

	g.storeOrder(unitOrder)
	return nil
}

// storeOrder records order, replacing any previously submitted order for the
// same unit.
func (g *Game) storeOrder(order UnitOrder) {
	for i, existing := range g.Orders {
		unitOrder, ok := existing.(UnitOrder)
		if ok && unitOrder.Unit() == order.Unit() {
			g.Orders[i] = order
			return
		}
	}
	g.Orders = append(g.Orders, order)
}
