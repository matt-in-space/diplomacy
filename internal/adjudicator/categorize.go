package adjudicator

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/internal/game"
)

type categorizedOrders struct {
	holds        map[game.UnitID]game.HoldOrder
	moves        map[game.UnitID]game.MoveOrder
	supportHolds map[game.UnitID]game.SupportHoldOrder
	supportMoves map[game.UnitID]game.SupportMoveOrder
	convoys      map[game.UnitID]game.ConvoyOrder
}

func categorizeOrders(orders map[game.UnitID]game.Order) (categorizedOrders, error) {
	categorized := categorizedOrders{
		holds:        make(map[game.UnitID]game.HoldOrder),
		moves:        make(map[game.UnitID]game.MoveOrder),
		supportHolds: make(map[game.UnitID]game.SupportHoldOrder),
		supportMoves: make(map[game.UnitID]game.SupportMoveOrder),
		convoys:      make(map[game.UnitID]game.ConvoyOrder),
	}

	for unitID, order := range orders {
		switch order := order.(type) {
		case game.HoldOrder:
			categorized.holds[unitID] = order
		case game.MoveOrder:
			categorized.moves[unitID] = order
		case game.SupportHoldOrder:
			categorized.supportHolds[unitID] = order
		case game.SupportMoveOrder:
			categorized.supportMoves[unitID] = order
		case game.ConvoyOrder:
			categorized.convoys[unitID] = order
		default:
			return categorizedOrders{}, fmt.Errorf("unsupported order type %T", order)
		}
	}

	return categorized, nil
}
