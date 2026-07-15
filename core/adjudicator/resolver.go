package adjudicator

import (
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// resolutionState tracks how far along a single order is in the recursive
// resolver.
type resolutionState int

const (
	stateUnresolved resolutionState = iota
	stateGuessing
	stateResolved
)

// resolveOrders resolves every unit's effective order using a recursive,
// backtracking resolver (Kruijswijk's "The Math of Adjudication"). It builds the
// lookup indices the strength calculations rely on, then resolves each unit.
//
// resolution[u] means, by effective order: a move succeeds; a support is given;
// a convoying fleet stands; or a holding unit is not dislodged.
func (rc *resolutionContext) resolveOrders() {
	rc.movesByTarget = make(map[gamemap.ProvinceID][]game.UnitID)
	for id, move := range rc.effectiveMoveOrders {
		rc.movesByTarget[move.Target] = append(rc.movesByTarget[move.Target], id)
	}

	rc.convoysByArmy = make(map[game.UnitID][]game.ConvoyOrder)
	for _, convoy := range rc.effectiveConvoyOrders {
		rc.convoysByArmy[convoy.ConvoyedUnit] = append(rc.convoysByArmy[convoy.ConvoyedUnit], convoy)
	}

	for id := range rc.units {
		rc.resolve(id)
	}
}

// resolve returns the resolution of unit u's effective order, running the
// backtracking guess logic when a cyclic dependency is encountered.
func (rc *resolutionContext) resolve(u game.UnitID) bool {
	switch rc.state[u] {
	case stateResolved:
		return rc.resolution[u]
	case stateGuessing:
		// We looped back to an order we are currently guessing: record the
		// cyclic dependency and return the current guess.
		if !containsUnit(rc.dependencyStack, u) {
			rc.dependencyStack = append(rc.dependencyStack, u)
		}
		return rc.resolution[u]
	}

	oldLen := len(rc.dependencyStack)

	// First hypothesis: the order fails.
	rc.resolution[u] = false
	rc.state[u] = stateGuessing
	firstResult := rc.adjudicate(u)

	if len(rc.dependencyStack) == oldLen {
		// No cyclic dependency was touched, so the result is reliable.
		rc.state[u] = stateResolved
		rc.resolution[u] = firstResult
		return firstResult
	}

	if rc.dependencyStack[oldLen] != u {
		// u is part of a cycle but not its entry point. Leave it guessing; the
		// entry point will resolve the cycle.
		if !containsUnit(rc.dependencyStack, u) {
			rc.dependencyStack = append(rc.dependencyStack, u)
		}
		rc.resolution[u] = firstResult
		return firstResult
	}

	// u is the entry point of a cycle. Try the second hypothesis.
	rc.resetCycle(oldLen)
	rc.resolution[u] = true
	rc.state[u] = stateGuessing
	secondResult := rc.adjudicate(u)

	if firstResult == secondResult {
		// The guess did not matter: the result is stable.
		rc.resetCycle(oldLen)
		rc.state[u] = stateResolved
		rc.resolution[u] = firstResult
		return firstResult
	}

	// A genuine paradox: collect the cycle and let the backup rule break it.
	cycle := make([]game.UnitID, len(rc.dependencyStack)-oldLen)
	copy(cycle, rc.dependencyStack[oldLen:])
	if !containsUnit(cycle, u) {
		cycle = append(cycle, u)
	}
	rc.resetCycle(oldLen)
	rc.state[u] = stateUnresolved
	rc.backupRule(cycle)

	return rc.resolve(u)
}

// resetCycle returns every order recorded since oldLen to the unresolved state
// so it can be re-evaluated cleanly, and truncates the dependency stack.
func (rc *resolutionContext) resetCycle(oldLen int) {
	for _, id := range rc.dependencyStack[oldLen:] {
		rc.state[id] = stateUnresolved
	}
	rc.dependencyStack = rc.dependencyStack[:oldLen]
}

// adjudicate computes the raw outcome of a single order, recursing into the
// orders it depends on.
func (rc *resolutionContext) adjudicate(u game.UnitID) bool {
	if move, ok := rc.effectiveMoveOrders[u]; ok {
		return rc.moveSucceeds(u, move)
	}
	if support, ok := rc.effectiveSupportHoldOrders[u]; ok {
		return rc.supportGiven(u, support.Target)
	}
	if support, ok := rc.effectiveSupportMoveOrders[u]; ok {
		return rc.supportGiven(u, support.Target)
	}
	// Hold or convoy: it stands unless a move into its province succeeds.
	return !rc.provinceAttackedSuccessfully(rc.units[u].ProvinceID, u)
}

// backupRule breaks a genuine cycle. A cycle made entirely of move orders is
// circular movement (everyone moves); otherwise it is a convoy paradox resolved
// by the Szykman rule (the paradoxical convoyed move fails).
func (rc *resolutionContext) backupRule(cycle []game.UnitID) {
	allMoves := true
	for _, id := range cycle {
		if _, ok := rc.effectiveMoveOrders[id]; !ok {
			allMoves = false
			break
		}
	}

	if allMoves {
		for _, id := range cycle {
			rc.state[id] = stateResolved
			rc.resolution[id] = true
		}
		return
	}

	broke := false
	for _, id := range cycle {
		if move, ok := rc.effectiveMoveOrders[id]; ok && move.ViaConvoy {
			rc.state[id] = stateResolved
			rc.resolution[id] = false
			broke = true
		}
	}
	if broke {
		return
	}

	// Fallback: fail every member so the cycle cannot recur.
	for _, id := range cycle {
		rc.state[id] = stateResolved
		rc.resolution[id] = false
	}
}

func containsUnit(units []game.UnitID, target game.UnitID) bool {
	for _, id := range units {
		if id == target {
			return true
		}
	}
	return false
}
