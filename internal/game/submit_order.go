package game

import (
	"fmt"
	"slices"

	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

func (g *Game) SubmitOrder(order Order, gm *gamemap.GameMap) error {
	if order == nil {
		return fmt.Errorf("order is required")
	}
	if gm == nil {
		return fmt.Errorf("game map is required")
	}
	if gm.ID != g.MapID {
		return fmt.Errorf("game map %q does not match game map %q", gm.ID, g.MapID)
	}
	if g.Turn.Phase != AcceptOrders {
		return fmt.Errorf("cannot submit order during phase %q", g.Turn.Phase)
	}

	nation := order.Nation()
	if !slices.Contains(gm.Nations, nation) {
		return fmt.Errorf("order nation %q not found", nation)
	}

	unitID := order.Unit()
	unit, ok := g.Units[unitID]
	if !ok {
		return fmt.Errorf("unit %q not found", unitID)
	}
	if unit.NationID != nation {
		return fmt.Errorf("unit %q belongs to nation %q, not %q", unitID, unit.NationID, nation)
	}
	if occupyingUnit, ok := g.Positions[unit.ProvinceID]; !ok || occupyingUnit != unitID {
		return fmt.Errorf("unit %q is not on the board", unitID)
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

	if g.Orders == nil {
		g.Orders = make(map[UnitID]Order)
	}
	g.Orders[unitID] = order

	return nil
}
