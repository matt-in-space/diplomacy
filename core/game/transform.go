package game

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

type UnitTransformType string

const (
	UnitTransformMove    UnitTransformType = "move"
	UnitTransformHold    UnitTransformType = "hold"
	UnitTransformRetreat UnitTransformType = "retreat"
	UnitTransformDisband UnitTransformType = "disband"
)

// UnitTransform details the unit's final position and type.
type UnitTransform struct {
	UnitID UnitID
	Type   UnitTransformType
	From   gamemap.ProvinceID
	To     gamemap.ProvinceID
	Coast  gamemap.CoastID
}

func (g *Game) applyUnitTransforms(results []UnitTransform) error {
	if err := g.validateUnitTransforms(results); err != nil {
		return err
	}

	units := make(map[UnitID]Unit, len(results))

	for _, result := range results {
		unit := g.Units[result.UnitID]

		switch result.Type {
		case UnitTransformMove, UnitTransformHold:
			unit.ProvinceID = result.To
			unit.DislodgedFrom = ""
			if unit.Type == UnitTypeFleet {
				unit.Coast = result.Coast
			}
		case UnitTransformRetreat:
			unit.ProvinceID = ""
			unit.DislodgedFrom = result.From
			unit.Coast = result.Coast
		}

		units[unit.ID] = unit
	}

	g.Units = units

	return nil
}

func (g *Game) validateUnitTransforms(results []UnitTransform) error {
	if len(results) != len(g.Units) {
		return fmt.Errorf("received %d unit transforms for %d units", len(results), len(g.Units))
	}

	units := make(map[UnitID]struct{}, len(results))
	destinations := make(map[gamemap.ProvinceID]UnitID, len(results))

	for _, result := range results {
		if _, ok := units[result.UnitID]; ok {
			return fmt.Errorf("duplicate transform for unit %q", result.UnitID)
		}
		units[result.UnitID] = struct{}{}

		unit, ok := g.Units[result.UnitID]
		if !ok {
			return fmt.Errorf("unit %q not found", result.UnitID)
		}
		if result.From != unit.ProvinceID {
			return fmt.Errorf("unit %q is in province %q, not %q", result.UnitID, unit.ProvinceID, result.From)
		}

		switch result.Type {
		case UnitTransformMove:
			if result.To == "" {
				return fmt.Errorf("move transform for unit %q has no destination", result.UnitID)
			}
			if result.To == result.From {
				return fmt.Errorf("move transform for unit %q does not change province", result.UnitID)
			}
		case UnitTransformHold:
			if result.To != result.From {
				return fmt.Errorf("hold transform for unit %q changes province", result.UnitID)
			}
		case UnitTransformRetreat:
			if result.To != "" {
				return fmt.Errorf("retreat transform for unit %q has destination %q", result.UnitID, result.To)
			}
			continue
		default:
			return fmt.Errorf("unknown transform type %q for unit %q", result.Type, result.UnitID)
		}

		if other, ok := destinations[result.To]; ok {
			return fmt.Errorf("units %q and %q both end in province %q", other, result.UnitID, result.To)
		}
		destinations[result.To] = result.UnitID
	}

	return nil
}

// applyRetreatTransforms applies the outcome of retreat resolution. Unlike
// applyUnitTransforms, it only concerns itself with the dislodged units the
// retreat phase actually decided the fate of: a Move lands a unit back on the
// board, a Disband removes it from the game entirely. Units that were never
// dislodged are untouched and not mentioned in results.
func (g *Game) applyRetreatTransforms(results []UnitTransform) error {
	if err := g.validateRetreatTransforms(results); err != nil {
		return err
	}

	for _, result := range results {
		switch result.Type {
		case UnitTransformMove:
			unit := g.Units[result.UnitID]
			unit.ProvinceID = result.To
			unit.DislodgedFrom = ""
			if unit.Type == UnitTypeFleet {
				unit.Coast = result.Coast
			}
			g.Units[result.UnitID] = unit
		case UnitTransformDisband:
			delete(g.Units, result.UnitID)
		}
	}

	return nil
}

// validateRetreatTransforms checks that results covers exactly the currently
// dislodged units, each transform matches the unit it dislodged from, and no
// two units retreat to the same destination.
func (g *Game) validateRetreatTransforms(results []UnitTransform) error {
	dislodged := make(map[UnitID]struct{})
	for id, unit := range g.Units {
		if unit.Dislodged() {
			dislodged[id] = struct{}{}
		}
	}
	if len(results) != len(dislodged) {
		return fmt.Errorf("received %d retreat transforms for %d dislodged units", len(results), len(dislodged))
	}

	seen := make(map[UnitID]struct{}, len(results))
	destinations := make(map[gamemap.ProvinceID]UnitID, len(results))

	for _, result := range results {
		if _, ok := seen[result.UnitID]; ok {
			return fmt.Errorf("duplicate retreat transform for unit %q", result.UnitID)
		}
		seen[result.UnitID] = struct{}{}

		unit, ok := g.Units[result.UnitID]
		if !ok || !unit.Dislodged() {
			return fmt.Errorf("unit %q is not dislodged", result.UnitID)
		}
		if result.From != unit.DislodgedFrom {
			return fmt.Errorf("unit %q was dislodged from %q, not %q", result.UnitID, unit.DislodgedFrom, result.From)
		}

		switch result.Type {
		case UnitTransformMove:
			if result.To == "" {
				return fmt.Errorf("retreat transform for unit %q has no destination", result.UnitID)
			}
			if other, ok := destinations[result.To]; ok {
				return fmt.Errorf("units %q and %q both retreat to %q", other, result.UnitID, result.To)
			}
			destinations[result.To] = result.UnitID
		case UnitTransformDisband:
			if result.To != "" {
				return fmt.Errorf("disband transform for unit %q has destination %q", result.UnitID, result.To)
			}
		default:
			return fmt.Errorf("unsupported retreat transform type %q for unit %q", result.Type, result.UnitID)
		}
	}

	return nil
}
