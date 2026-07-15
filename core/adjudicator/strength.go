package adjudicator

import (
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// moveSucceeds reports whether a move order takes its destination. It must beat
// the prevent strength of every competing move and either the defend strength of
// a head-to-head opponent or the hold strength of the destination.
func (rc *resolutionContext) moveSucceeds(u game.UnitID, move game.MoveOrder) bool {
	if !rc.path(u, move) {
		return false
	}

	attack := rc.attackStrength(u, move)
	if attack <= 0 {
		return false
	}

	for _, competitor := range rc.movesByTarget[move.Target] {
		if competitor == u {
			continue
		}
		if rc.preventStrength(competitor) >= attack {
			return false
		}
	}

	if opponent, ok := rc.opponentMove(u, move); ok {
		return rc.defendStrength(opponent) < attack
	}
	return rc.holdStrength(move.Target) < attack
}

// attackStrength is the force a move brings to its destination: base 1 plus valid
// supports, with the rules that a unit cannot dislodge its own unit and that a
// support given by the occupant's own nation cannot help dislodge it.
func (rc *resolutionContext) attackStrength(u game.UnitID, move game.MoveOrder) int {
	if !rc.path(u, move) {
		return 0
	}

	occupant, occupied := rc.currentPositions[move.Target]
	_, headToHead := rc.opponentMove(u, move)

	occupantLeaves := false
	if occupied && !headToHead {
		if _, moving := rc.effectiveMoveOrders[occupant]; moving {
			occupantLeaves = rc.resolve(occupant)
		}
	}

	if !occupied || occupantLeaves {
		return 1 + rc.countMoveSupports(u, move.Target, "")
	}

	if rc.units[occupant].NationID == rc.units[u].NationID {
		return 0
	}
	return 1 + rc.countMoveSupports(u, move.Target, rc.units[occupant].NationID)
}

// defendStrength is the strength a move brings to a head-to-head battle.
func (rc *resolutionContext) defendStrength(move game.MoveOrder) int {
	return 1 + rc.countMoveSupports(move.Unit(), move.Target, "")
}

// preventStrength is the force a move exerts to keep any other unit out of its
// destination.
func (rc *resolutionContext) preventStrength(u game.UnitID) int {
	move := rc.effectiveMoveOrders[u]
	if !rc.path(u, move) {
		return 0
	}
	if opponent, ok := rc.opponentMove(u, move); ok && rc.resolve(opponent.Unit()) {
		return 0
	}
	return 1 + rc.countMoveSupports(u, move.Target, "")
}

// holdStrength is the force resisting a move into a province.
func (rc *resolutionContext) holdStrength(province gamemap.ProvinceID) int {
	occupant, ok := rc.currentPositions[province]
	if !ok {
		return 0
	}
	if _, moving := rc.effectiveMoveOrders[occupant]; moving {
		if rc.resolve(occupant) {
			return 0
		}
		return 1
	}
	return 1 + rc.countHoldSupports(occupant)
}

// path reports whether the move can physically reach its destination. Direct
// moves are guaranteed by order validation; convoyed moves need an unbroken
// chain of non-dislodged convoying fleets.
func (rc *resolutionContext) path(u game.UnitID, move game.MoveOrder) bool {
	if !move.ViaConvoy {
		return true
	}

	var via []gamemap.CoastID
	for _, convoy := range rc.convoysByArmy[u] {
		if rc.resolve(convoy.UnitID) {
			via = append(via, rc.fleetCoasts[convoy.UnitID])
		}
	}
	return rc.gm.ConvoyPathExists(rc.units[u].ProvinceID, move.Target, via)
}

// opponentMove returns the direct (non-convoyed) move of the unit occupying this
// move's destination when it is heading back into this move's origin, i.e. a
// head-to-head battle.
func (rc *resolutionContext) opponentMove(u game.UnitID, move game.MoveOrder) (game.MoveOrder, bool) {
	if move.ViaConvoy {
		return game.MoveOrder{}, false
	}
	occupant, ok := rc.currentPositions[move.Target]
	if !ok {
		return game.MoveOrder{}, false
	}
	opponent, ok := rc.effectiveMoveOrders[occupant]
	if !ok || opponent.ViaConvoy {
		return game.MoveOrder{}, false
	}
	if opponent.Target != rc.units[u].ProvinceID {
		return game.MoveOrder{}, false
	}
	return opponent, true
}

// supportGiven reports whether a support order provides its strength: it must not
// be cut and its unit must not be dislodged. exempt is the province the support
// is directed into, an attack from which does not cut.
func (rc *resolutionContext) supportGiven(u game.UnitID, exempt gamemap.ProvinceID) bool {
	province := rc.units[u].ProvinceID
	if rc.supportCut(u, province, exempt) {
		return false
	}
	return !rc.provinceAttackedSuccessfully(province, u)
}

// supportCut reports whether a foreign unit attacks the supporter from a province
// other than the one the support is directed into.
func (rc *resolutionContext) supportCut(u game.UnitID, province, exempt gamemap.ProvinceID) bool {
	nation := rc.units[u].NationID
	for _, attacker := range rc.movesByTarget[province] {
		if attacker == u {
			continue
		}
		if rc.units[attacker].NationID == nation {
			continue
		}
		if rc.units[attacker].ProvinceID == exempt {
			continue
		}
		if rc.path(attacker, rc.effectiveMoveOrders[attacker]) {
			return true
		}
	}
	return false
}

// provinceAttackedSuccessfully reports whether any move (other than except)
// succeeds into the province.
func (rc *resolutionContext) provinceAttackedSuccessfully(province gamemap.ProvinceID, except game.UnitID) bool {
	for _, mover := range rc.movesByTarget[province] {
		if mover == except {
			continue
		}
		if rc.resolve(mover) {
			return true
		}
	}
	return false
}

// countMoveSupports counts the valid supports for a move into target. Supports
// given by units of excludeNation are ignored (they cannot help dislodge their
// own countryman); pass "" to count every nation.
func (rc *resolutionContext) countMoveSupports(supported game.UnitID, target gamemap.ProvinceID, excludeNation gamemap.NationID) int {
	count := 0
	for id, support := range rc.effectiveSupportMoveOrders {
		if support.SupportedUnit != supported || support.Target != target {
			continue
		}
		if excludeNation != "" && rc.units[id].NationID == excludeNation {
			continue
		}
		if rc.resolve(id) {
			count++
		}
	}
	return count
}

// countHoldSupports counts the valid support-hold orders for a unit.
func (rc *resolutionContext) countHoldSupports(supported game.UnitID) int {
	count := 0
	for id, support := range rc.effectiveSupportHoldOrders {
		if support.SupportedUnit != supported {
			continue
		}
		if rc.resolve(id) {
			count++
		}
	}
	return count
}
