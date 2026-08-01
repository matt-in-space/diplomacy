package game

import (
	"slices"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// retreatConstraints bundles the facts derived from the last order resolution
// that determine where a dislodged unit may retreat: which provinces are
// occupied, which were left vacant by a standoff, and where the attacker that
// caused this dislodgement came from.
type retreatConstraints struct {
	occupied         map[gamemap.ProvinceID]struct{}
	standoffs        map[gamemap.ProvinceID]struct{}
	attackerOrigin   gamemap.ProvinceID
	attackerConvoyed bool
}

// retreatConstraintsFor derives retreat constraints for a unit dislodged from
// dislodgedFrom, reading g.Units (for current occupancy) and
// g.LastOrderResolution (for the attack that caused the dislodgement and any
// standoffs).
func (g *Game) retreatConstraintsFor(dislodgedFrom gamemap.ProvinceID) retreatConstraints {
	occupied := make(map[gamemap.ProvinceID]struct{}, len(g.Units))
	for _, unit := range g.Units {
		if unit.ProvinceID != "" {
			occupied[unit.ProvinceID] = struct{}{}
		}
	}

	rc := retreatConstraints{
		occupied:  occupied,
		standoffs: make(map[gamemap.ProvinceID]struct{}),
	}

	for _, outcome := range g.LastOrderResolution {
		move, ok := outcome.Order.Order.(MoveOrder)
		if !ok {
			continue
		}

		if outcome.Order.Success {
			if move.Target == dislodgedFrom {
				rc.attackerOrigin = outcome.Unit.From
				rc.attackerConvoyed = move.ViaConvoy
			}
			continue
		}

		// A bounce only creates a standoff if it left the province vacant. A
		// failed attack against a province that held its occupant is simply
		// occupied, not a standoff, and the occupied check above already
		// excludes it.
		if outcome.Order.Reason != ReasonWeakAttack {
			continue
		}
		if _, stillOccupied := occupied[move.Target]; stillOccupied {
			continue
		}
		rc.standoffs[move.Target] = struct{}{}
	}

	return rc
}

// allows reports whether a dislodged unit may retreat into province, ignoring
// adjacency (that is a separate, unit-type-specific check).
func (rc retreatConstraints) allows(province gamemap.ProvinceID) bool {
	if _, occupied := rc.occupied[province]; occupied {
		return false
	}
	if _, standoff := rc.standoffs[province]; standoff {
		return false
	}
	if province == rc.attackerOrigin && !rc.attackerConvoyed {
		return false
	}
	return true
}

// candidateRetreatProvinces returns the provinces adjacent to a dislodged
// unit's former position, ignoring occupancy and standoffs. For a fleet, a
// province is a candidate if any of its coasts are reachable; SubmitOrder
// validates the specific requested coast at submission time.
func candidateRetreatProvinces(unit Unit, gm *gamemap.GameMap) []gamemap.ProvinceID {
	switch unit.Type {
	case UnitTypeArmy:
		return slices.Clone(gm.ArmyNeighbors(unit.DislodgedFrom))
	case UnitTypeFleet:
		seen := make(map[gamemap.ProvinceID]struct{})
		var provinces []gamemap.ProvinceID
		for _, coast := range gm.FleetNeighbors(unit.Coast) {
			province, ok := gm.ProvinceForCoast(coast)
			if !ok {
				continue
			}
			if _, dup := seen[province]; dup {
				continue
			}
			seen[province] = struct{}{}
			provinces = append(provinces, province)
		}
		return provinces
	default:
		return nil
	}
}

// LegalRetreats returns the provinces a dislodged unit may retreat to. It
// returns nil for a unit that is not dislodged, and an empty slice for a
// dislodged unit with nowhere to go (which must disband).
func (g *Game) LegalRetreats(id UnitID, gm *gamemap.GameMap) []gamemap.ProvinceID {
	unit, ok := g.Units[id]
	if !ok || !unit.Dislodged() {
		return nil
	}

	constraints := g.retreatConstraintsFor(unit.DislodgedFrom)
	candidates := candidateRetreatProvinces(unit, gm)

	legal := make([]gamemap.ProvinceID, 0, len(candidates))
	for _, province := range candidates {
		if constraints.allows(province) {
			legal = append(legal, province)
		}
	}
	slices.Sort(legal)

	return legal
}
