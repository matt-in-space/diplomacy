package game

import (
	"slices"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// AdjustmentBalance returns the number of supply centers a nation owns minus
// the number of units it has. A positive balance means the nation may build
// that many units, a negative balance means it must disband that many, and
// zero means it has nothing to do this adjustment phase.
func (g *Game) AdjustmentBalance(nation gamemap.NationID) int {
	centers := 0
	for _, owner := range g.SupplyCenterOwners {
		if owner == nation {
			centers++
		}
	}

	units := 0
	for _, unit := range g.Units {
		if unit.NationID == nation {
			units++
		}
	}

	return centers - units
}

// LegalBuildProvinces returns the provinces where a nation may build: its
// home supply centers that it still owns and that no unit occupies. The
// result is sorted and may be empty, which means no build is possible even
// when AdjustmentBalance is positive — the surplus builds are simply waived.
//
// It does not consult AdjustmentBalance; callers combine the two.
func (g *Game) LegalBuildProvinces(nation gamemap.NationID, gm *gamemap.GameMap) []gamemap.ProvinceID {
	occupied := make(map[gamemap.ProvinceID]struct{}, len(g.Units))
	for _, unit := range g.Units {
		if unit.ProvinceID != "" {
			occupied[unit.ProvinceID] = struct{}{}
		}
	}

	provinces := make([]gamemap.ProvinceID, 0)
	for id, province := range gm.Provinces {
		if province.HomeNation != nation {
			continue
		}
		if g.SupplyCenterOwners[id] != nation {
			continue
		}
		if _, ok := occupied[id]; ok {
			continue
		}
		provinces = append(provinces, id)
	}
	slices.Sort(provinces)

	return provinces
}
