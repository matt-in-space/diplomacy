package adjudicator

import "github.com/matt-in-space/diplomacy/internal/game"

func normalizeOrders(ctx resolutionContext) map[game.UnitID]game.Order {
	effectiveOrders := make(map[game.UnitID]game.Order, len(ctx.units))

	for unitID, unit := range ctx.units {
		order, ok := ctx.orders[unitID]
		if ok {
			effectiveOrders[unitID] = order
			continue
		}

		effectiveOrders[unitID] = game.NewHoldOrder(unitID, unit.NationID)
	}

	return effectiveOrders
}
