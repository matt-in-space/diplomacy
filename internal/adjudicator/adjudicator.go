package adjudicator

import (
	"maps"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

type Resolution struct {
	Turn          game.Turn
	UnitOutcomes  map[game.UnitID]UnitOutcome
	OrderOutcomes map[game.UnitID]OrderOutcome
}

type UnitOutcomeType string

const (
	UnitOutcomeMove    UnitOutcomeType = "move"
	UnitOutcomeHold    UnitOutcomeType = "hold"
	UnitOutcomeRetreat UnitOutcomeType = "retreat"
)

type UnitOutcome struct {
	UnitID game.UnitID
	Type   UnitOutcomeType
	From   gamemap.ProvinceID
	To     gamemap.ProvinceID
	Coast  gamemap.CoastID
}

type ReasonCode string

const (
	ReasonSuccess ReasonCode = "success"
)

type OrderOutcome struct {
	UnitID  game.UnitID
	Order   game.Order
	Success bool
	Reason  ReasonCode
}

func Resolve(g *game.Game, gm *gamemap.GameMap) (Resolution, error) {
	// Main concepts
	// - All units start on a province
	// - There are 3 outcomes: move, hold, retreat regardless of the order
	// - We figure all of the intents first: intends to move, intends to support, and intends to convoy
	// - Then we cut support and convoy orders since those don't move and anyone moving into their
	// provinces will take precedence over them
	// - Then we resolve the movement and attack orders after those are all that are left
	//
	// Order of operations
	// - Get the effective orders, which creates default Hold orders for units that have no orders
	// - Categorize the intents of the orders into groups of move, support, and convoy. Each of these
	// should be accessed by the *target* province
	// - Cancel any supports or convoys that don't match the move order
	// - Then cancel any supports or convoys that are disrupted by an enemy move
	// - Finally attempt all the moves that can still occur, using support to determine if a move
	// is successful or note

	effectiveOrders := normalizeOrders(g)
	// intendedActions := categorizeIntents(effectiveOrders)

	res := Resolution{
		Turn:          g.Turn,
		UnitOutcomes:  make(map[game.UnitID]UnitOutcome),
		OrderOutcomes: make(map[game.UnitID]OrderOutcome),
	}

	for unitID, order := range effectiveOrders {
		pos := g.Units[unitID].ProvinceID
		uo := UnitOutcome{
			UnitID: unitID,
			From:   pos,
		}
		oo := OrderOutcome{
			UnitID: unitID,
			Order:  order,
		}

		switch order := order.(type) {
		case game.MoveOrder:
			uo.Type = UnitOutcomeMove
			uo.To = order.Target
			uo.Coast = order.TargetCoast
			oo.Success = true
			oo.Reason = ReasonSuccess

		}

		res.UnitOutcomes[unitID] = uo
		res.OrderOutcomes[unitID] = oo
	}

	return res, nil
}

func normalizeOrders(g *game.Game) map[game.UnitID]game.Order {
	orders := make(map[game.UnitID]game.Order)
	maps.Copy(orders, g.Orders)

	for unitID, unit := range g.Units {
		if _, ok := orders[unitID]; !ok {
			orders[unitID] = game.NewHoldOrder(unitID, unit.NationID)
		}
	}

	return orders
}

type intents struct {
	move    map[gamemap.ProvinceID][]game.Order
	support map[gamemap.ProvinceID][]game.Order
	convoy  map[gamemap.ProvinceID][]game.Order
}

func categorizeIntents(orders map[game.UnitID]game.Order) intents {
	i := intents{
		move:    make(map[gamemap.ProvinceID][]game.Order),
		support: make(map[gamemap.ProvinceID][]game.Order),
		convoy:  make(map[gamemap.ProvinceID][]game.Order),
	}

	for _, order := range orders {
		switch o := order.(type) {
		case game.MoveOrder:
			i.move[o.Target] = append(i.move[o.Target], o)
		case game.SupportHoldOrder:
			i.support[o.Target] = append(i.support[o.Target], o)
		case game.SupportMoveOrder:
			i.support[o.Target] = append(i.support[o.Target], o)
		case game.ConvoyOrder:
			i.convoy[o.From] = append(i.convoy[o.From], o)
		default:
			panic("unhandled order type")
		}

	}

	return i
}
