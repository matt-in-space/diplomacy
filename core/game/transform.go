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
)

// UnitTransform details the unit's final position and type.
type UnitTransform struct {
	UnitID UnitID
	Type   UnitTransformType
	From   gamemap.ProvinceID
	To     gamemap.ProvinceID
	Coast  gamemap.CoastID
}

func (g *Game) ApplyUnitTransforms(results []UnitTransform) error {
	if err := g.validateUnitTransforms(results); err != nil {
		return err
	}

	units := make(map[UnitID]Unit, len(results))
	positions := make(map[gamemap.ProvinceID]UnitID, len(results))
	fleetCoasts := make(map[UnitID]gamemap.CoastID)
	pendingRetreats := make(map[UnitID]Dislodgement)

	for _, result := range results {
		unit := g.Units[result.UnitID]

		switch result.Type {
		case UnitTransformMove, UnitTransformHold:
			unit.ProvinceID = result.To
			positions[result.To] = unit.ID
			if unit.Type == UnitTypeFleet {
				fleetCoasts[unit.ID] = result.Coast
			}
		case UnitTransformRetreat:
			unit.ProvinceID = ""
			pendingRetreats[unit.ID] = Dislodgement{
				From:  result.From,
				Coast: result.Coast,
			}
		}

		units[unit.ID] = unit
	}

	g.Units = units
	g.Positions = positions
	g.FleetCoasts = fleetCoasts
	g.PendingRetreats = pendingRetreats

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
		if occupant, ok := g.Positions[result.From]; !ok || occupant != result.UnitID {
			return fmt.Errorf("province %q is not occupied by unit %q", result.From, result.UnitID)
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
