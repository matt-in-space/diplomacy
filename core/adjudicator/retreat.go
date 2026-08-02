package adjudicator

import (
	"errors"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// ResolveRetreats determines the outcome of retreat orders for every
// dislodged unit. Unlike Resolve, there are no supports, no convoys, and no
// cyclic dependencies to unwind, so a single pass is enough: a unit with no
// submitted order (or an explicit DisbandOrder) is removed from the game, and
// units that retreat to the same province all disband — a retreat conflict
// has no fallback the way a movement bounce does, since the unit has nowhere
// left to hold.
func ResolveRetreats(g *game.Game, gm *gamemap.GameMap) (game.Resolution, error) {
	if g.MapID != gm.ID {
		return game.Resolution{}, errors.New("unexpected game map provided")
	}

	dislodged := dislodgedUnits(g)
	retreatOrders, disbandOrders := categorizeRetreatOrders(g, dislodged)

	byTarget := make(map[gamemap.ProvinceID][]game.UnitID, len(retreatOrders))
	for id, order := range retreatOrders {
		byTarget[order.Target] = append(byTarget[order.Target], id)
	}

	res := make(game.Resolution, len(dislodged))

	for id, order := range disbandOrders {
		res[id] = disbandOutcome(id, dislodged[id], game.CreateOrderSuccessOutcome(order))
	}

	for target, ids := range byTarget {
		if len(ids) > 1 {
			for _, id := range ids {
				order := retreatOrders[id]
				res[id] = disbandOutcome(id, dislodged[id], game.CreateOrderFailOutcome(order, game.ReasonRetreatConflict))
			}
			continue
		}

		id := ids[0]
		order := retreatOrders[id]
		unit := dislodged[id]
		res[id] = game.Outcome{
			UnitID: id,
			Unit: game.UnitTransform{
				UnitID: id,
				Type:   game.UnitTransformMove,
				From:   unit.DislodgedFrom,
				To:     target,
				Coast:  resolveRetreatCoast(unit, order, gm),
			},
			Order: game.CreateOrderSuccessOutcome(order),
		}
	}

	return res, nil
}

// dislodgedUnits returns every unit awaiting a retreat order, keyed by ID.
func dislodgedUnits(g *game.Game) map[game.UnitID]game.Unit {
	dislodged := make(map[game.UnitID]game.Unit)
	for id, unit := range g.Units {
		if unit.Dislodged() {
			dislodged[id] = unit
		}
	}
	return dislodged
}

// categorizeRetreatOrders splits every dislodged unit's order into an
// attempted retreat or a disband, defaulting any unit with no submitted
// order to a disband — mirroring how movement resolution defaults an
// unordered unit to a hold.
func categorizeRetreatOrders(g *game.Game, dislodged map[game.UnitID]game.Unit) (map[game.UnitID]game.RetreatOrder, map[game.UnitID]game.Order) {
	retreats := make(map[game.UnitID]game.RetreatOrder)
	disbands := make(map[game.UnitID]game.Order)
	unitOrders := g.UnitOrders()

	for id, unit := range dislodged {
		order, ok := unitOrders[id]
		if !ok {
			order = game.NewDisbandOrder(id, unit.NationID)
		}

		switch order := order.(type) {
		case game.RetreatOrder:
			retreats[id] = order
		default:
			disbands[id] = order
		}
	}

	return retreats, disbands
}

func disbandOutcome(id game.UnitID, unit game.Unit, outcome game.OrderOutcome) game.Outcome {
	return game.Outcome{
		UnitID: id,
		Unit: game.UnitTransform{
			UnitID: id,
			Type:   game.UnitTransformDisband,
			From:   unit.DislodgedFrom,
		},
		Order: outcome,
	}
}

// resolveRetreatCoast determines the coast a retreating fleet lands on,
// mirroring resolutionContext.resolveMoveCoast's shape for the movement phase.
func resolveRetreatCoast(unit game.Unit, order game.RetreatOrder, gm *gamemap.GameMap) gamemap.CoastID {
	if unit.Type != game.UnitTypeFleet {
		return ""
	}
	if order.TargetCoast != "" {
		return order.TargetCoast
	}
	if coasts := gm.CoastsFor(order.Target); len(coasts) == 1 {
		return coasts[0]
	}
	return ""
}
