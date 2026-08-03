package adjudicator

import (
	"errors"
	"fmt"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// ResolveAdjustments determines the outcome of every nation's build and
// disband orders. Unlike Resolve/ResolveRetreats, orders from different
// nations never interact — each nation's balance and submitted orders are
// judged independently — so there is no conflict resolution, only per-nation
// bookkeeping: at most AdjustmentBalance builds (fewer is a legal waiver),
// or exactly -AdjustmentBalance disbands (fewer is not: real Diplomacy's
// "civil disorder" auto-selection is not yet implemented, so an
// under-ordering nation is a hard error rather than a guess).
func ResolveAdjustments(g *game.Game, gm *gamemap.GameMap) (game.AdjustmentResolution, error) {
	if g.MapID != gm.ID {
		return game.AdjustmentResolution{}, errors.New("unexpected game map provided")
	}

	buildsByNation, disbandsByNation := categorizeAdjustmentOrders(g)

	var res game.AdjustmentResolution
	for nation := range g.Assignments {
		balance := g.AdjustmentBalance(nation)
		builds := buildsByNation[nation]
		disbands := disbandsByNation[nation]

		switch {
		case balance > 0:
			if len(disbands) > 0 {
				return game.AdjustmentResolution{}, fmt.Errorf("nation %q submitted disbands but owes builds", nation)
			}
			if len(builds) > balance {
				return game.AdjustmentResolution{}, fmt.Errorf("nation %q submitted %d builds but is owed %d", nation, len(builds), balance)
			}
			for _, order := range builds {
				res.Builds = append(res.Builds, game.UnitBuild{
					UnitID:     g.NextBuildUnitID(nation, order.UnitType, order.ProvinceID),
					NationID:   nation,
					Type:       order.UnitType,
					ProvinceID: order.ProvinceID,
					Coast:      resolveBuildCoast(order, gm),
				})
			}
		case balance < 0:
			if len(builds) > 0 {
				return game.AdjustmentResolution{}, fmt.Errorf("nation %q submitted builds but owes disbands", nation)
			}
			owed := -balance
			if len(disbands) != owed {
				return game.AdjustmentResolution{}, fmt.Errorf("nation %q owes %d disbands but submitted %d", nation, owed, len(disbands))
			}
			for _, order := range disbands {
				res.Disbands = append(res.Disbands, order.Unit())
			}
		default:
			if len(builds) > 0 || len(disbands) > 0 {
				return game.AdjustmentResolution{}, fmt.Errorf("nation %q has no adjustment to make", nation)
			}
		}
	}

	return res, nil
}

// categorizeAdjustmentOrders partitions every submitted order by nation and
// type.
func categorizeAdjustmentOrders(g *game.Game) (map[gamemap.NationID][]game.BuildOrder, map[gamemap.NationID][]game.DisbandOrder) {
	builds := make(map[gamemap.NationID][]game.BuildOrder)
	disbands := make(map[gamemap.NationID][]game.DisbandOrder)
	for _, order := range g.Orders {
		switch order := order.(type) {
		case game.BuildOrder:
			builds[order.Nation()] = append(builds[order.Nation()], order)
		case game.DisbandOrder:
			disbands[order.Nation()] = append(disbands[order.Nation()], order)
		}
	}
	return builds, disbands
}

// resolveBuildCoast determines the coast a newly built fleet lands on,
// mirroring resolveRetreatCoast's shape (retreat.go).
func resolveBuildCoast(order game.BuildOrder, gm *gamemap.GameMap) gamemap.CoastID {
	if order.UnitType != game.UnitTypeFleet {
		return ""
	}
	if order.Coast != "" {
		return order.Coast
	}
	if coasts := gm.CoastsFor(order.ProvinceID); len(coasts) == 1 {
		return coasts[0]
	}
	return ""
}
