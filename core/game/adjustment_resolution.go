package game

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// AdjustmentResolution describes the outcome of a resolved adjustments
// phase: which units to add to the board and which to remove.
type AdjustmentResolution struct {
	Builds   []UnitBuild
	Disbands []UnitID
}

// UnitBuild describes a new unit minted by a successful build.
type UnitBuild struct {
	UnitID     UnitID
	NationID   gamemap.NationID
	Type       UnitType
	ProvinceID gamemap.ProvinceID
	Coast      gamemap.CoastID
}

// NextBuildUnitID returns the UnitID a build for this nation/type/province
// would receive this year. Builds happen at most once per year (Fall-only,
// gated by Turn.Next()'s season branching), and a province can only ever be
// a home center for one nation, so (nation, type, province, year) can't
// collide with a prior or concurrent build.
func (g *Game) NextBuildUnitID(nation gamemap.NationID, unitType UnitType, province gamemap.ProvinceID) UnitID {
	return UnitID(fmt.Sprintf("%s-%s-%s-build-%d", nation, unitType, province, g.Turn.Year))
}

// BeginAdjustmentResolution advances the game out of the accept adjustments
// phase once every nation with a nonzero AdjustmentBalance has committed. A
// nation with nothing to build or disband has no reason to be waited on.
func (g *Game) BeginAdjustmentResolution() (progressed bool, err error) {
	if g.Turn.Phase != AcceptAdjustments {
		return false, fmt.Errorf("adjustment resolution can only be started in the accept adjustments phase")
	}

	for nationID := range g.nationsAwaitingAdjustments() {
		if _, ok := g.CommittedOrders[nationID]; !ok {
			return false, nil
		}
	}

	g.Turn = g.Turn.Next()

	return true, nil
}

// nationsAwaitingAdjustments returns the set of nations with a nonzero
// AdjustmentBalance, which must commit before the adjustments phase can
// advance.
func (g *Game) nationsAwaitingAdjustments() map[gamemap.NationID]struct{} {
	awaiting := make(map[gamemap.NationID]struct{})
	for nationID := range g.Assignments {
		if g.AdjustmentBalance(nationID) != 0 {
			awaiting[nationID] = struct{}{}
		}
	}
	return awaiting
}

// CompleteAdjustmentResolution applies the outcome of adjustment resolution:
// a new Unit is added for each build, and each disbanded unit is removed.
func (g *Game) CompleteAdjustmentResolution(res AdjustmentResolution) error {
	if g.Turn.Phase != ResolveAdjustments {
		return fmt.Errorf("adjustment resolution can only be completed in the resolve adjustments phase")
	}

	if err := g.applyAdjustmentResolution(res); err != nil {
		return err
	}

	g.CommittedOrders = make(map[gamemap.NationID]struct{})
	g.Orders = nil
	g.Turn = g.Turn.Next()

	return nil
}

func (g *Game) applyAdjustmentResolution(res AdjustmentResolution) error {
	if err := g.validateAdjustmentResolution(res); err != nil {
		return err
	}

	for _, build := range res.Builds {
		g.Units[build.UnitID] = Unit{
			ID:         build.UnitID,
			NationID:   build.NationID,
			ProvinceID: build.ProvinceID,
			Type:       build.Type,
			Coast:      build.Coast,
		}
	}
	for _, id := range res.Disbands {
		delete(g.Units, id)
	}

	return nil
}

// validateAdjustmentResolution re-checks the invariants ResolveAdjustments
// should already guarantee before Units is mutated: no build collides with
// an existing UnitID or an occupied province, no two builds target the same
// province, and every disband names a unit that actually exists.
func (g *Game) validateAdjustmentResolution(res AdjustmentResolution) error {
	occupied := make(map[gamemap.ProvinceID]struct{}, len(g.Units))
	for _, unit := range g.Units {
		if unit.ProvinceID != "" {
			occupied[unit.ProvinceID] = struct{}{}
		}
	}

	seenProvinces := make(map[gamemap.ProvinceID]struct{}, len(res.Builds))
	for _, build := range res.Builds {
		if _, exists := g.Units[build.UnitID]; exists {
			return fmt.Errorf("unit %q already exists", build.UnitID)
		}
		if _, taken := occupied[build.ProvinceID]; taken {
			return fmt.Errorf("province %q is occupied and cannot be built on", build.ProvinceID)
		}
		if _, dup := seenProvinces[build.ProvinceID]; dup {
			return fmt.Errorf("duplicate build for province %q", build.ProvinceID)
		}
		seenProvinces[build.ProvinceID] = struct{}{}
	}

	seenDisbands := make(map[UnitID]struct{}, len(res.Disbands))
	for _, id := range res.Disbands {
		if _, ok := g.Units[id]; !ok {
			return fmt.Errorf("unit %q not found", id)
		}
		if _, dup := seenDisbands[id]; dup {
			return fmt.Errorf("duplicate disband for unit %q", id)
		}
		seenDisbands[id] = struct{}{}
	}

	return nil
}
