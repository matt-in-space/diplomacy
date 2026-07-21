package adjudicator

import (
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// buildResolution turns the resolved order states into the final Resolution,
// with one outcome per unit.
func (rc *resolutionContext) buildResolution(turn game.Turn) Resolution {
	outcomes := make(map[game.UnitID]Outcome, len(rc.units))
	for id, unit := range rc.units {
		unitOutcome, orderOutcome := rc.outcomeFor(id, unit)
		outcomes[id] = Outcome{
			UnitID: id,
			Unit:   unitOutcome,
			Order:  orderOutcome,
		}
	}
	return Resolution{Turn: turn, Outcomes: outcomes}
}

func (rc *resolutionContext) outcomeFor(id game.UnitID, unit game.Unit) (game.UnitTransform, OrderOutcome) {
	order := rc.allOrders[id]

	if move, ok := rc.effectiveMoveOrders[id]; ok {
		if rc.resolution[id] {
			unitOutcome := game.UnitTransform{
				UnitID: id,
				Type:   game.UnitTransformMove,
				From:   unit.ProvinceID,
				To:     move.Target,
				Coast:  rc.resolveMoveCoast(unit, move),
			}
			return unitOutcome, createOrderSuccessOutcome(order)
		}
		if rc.isDislodged(id, unit) {
			return rc.retreatOutcome(id, unit), createOrderFailOutcome(order, ReasonDislodged)
		}
		// A convoyed move whose path is no longer intact failed because its convoy
		// broke, not because it bounced.
		if move.ViaConvoy && !rc.path(id, move) {
			return rc.holdOutcome(id, unit), createOrderFailOutcome(order, ReasonConvoyFailure)
		}
		return rc.holdOutcome(id, unit), createOrderFailOutcome(order, ReasonWeakAttack)
	}

	// Non-move orders (hold, support, convoy, or a pruned order now holding).
	if rc.isDislodged(id, unit) {
		return rc.retreatOutcome(id, unit), createOrderFailOutcome(order, ReasonDislodged)
	}

	// Orders demoted during pruning keep the reason recorded there.
	if pruned, ok := rc.orderOutcomes[id]; ok {
		return rc.holdOutcome(id, unit), pruned
	}

	return rc.holdOutcome(id, unit), rc.nonMoveOrderOutcome(id, order)
}

// nonMoveOrderOutcome reports success for a hold or convoy, and for a support
// whether it was given.
func (rc *resolutionContext) nonMoveOrderOutcome(id game.UnitID, order game.Order) OrderOutcome {
	_, isSupportHold := rc.effectiveSupportHoldOrders[id]
	_, isSupportMove := rc.effectiveSupportMoveOrders[id]
	if isSupportHold || isSupportMove {
		if rc.resolution[id] {
			return createOrderSuccessOutcome(order)
		}
		return createOrderFailOutcome(order, ReasonSupportCut)
	}
	return createOrderSuccessOutcome(order)
}

// isDislodged reports whether a unit is forced to retreat: it did not move away
// and a foreign move succeeded into its province.
func (rc *resolutionContext) isDislodged(id game.UnitID, unit game.Unit) bool {
	if _, ok := rc.effectiveMoveOrders[id]; ok && rc.resolution[id] {
		return false
	}
	return rc.provinceAttackedSuccessfully(unit.ProvinceID, id)
}

func (rc *resolutionContext) holdOutcome(id game.UnitID, unit game.Unit) game.UnitTransform {
	return game.UnitTransform{
		UnitID: id,
		Type:   game.UnitTransformHold,
		From:   unit.ProvinceID,
		To:     unit.ProvinceID,
		Coast:  rc.fleetCoasts[id],
	}
}

func (rc *resolutionContext) retreatOutcome(id game.UnitID, unit game.Unit) game.UnitTransform {
	return game.UnitTransform{
		UnitID: id,
		Type:   game.UnitTransformRetreat,
		From:   unit.ProvinceID,
		To:     "",
		Coast:  rc.fleetCoasts[id],
	}
}

// resolveMoveCoast determines the coast a fleet ends on after a successful move.
func (rc *resolutionContext) resolveMoveCoast(unit game.Unit, move game.MoveOrder) gamemap.CoastID {
	if unit.Type != game.UnitTypeFleet {
		return ""
	}
	if move.TargetCoast != "" {
		return move.TargetCoast
	}
	if coasts := rc.gm.CoastsFor(move.Target); len(coasts) == 1 {
		return coasts[0]
	}
	return ""
}
