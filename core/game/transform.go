package game

import (
	"errors"

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
	err := validateResults(results)
	if err != nil {
		return err
	}
	for _, result := range results {
		unit, ok := g.Units[result.UnitID]
		if !ok {
			return errors.New("unit not found")
		}

		if result.From != unit.ProvinceID {
			return errors.New("from position does not match")
		}

		pid, ok := g.Positions[result.From]
		if !ok {
			return errors.New("previous position for unit not found")
		}

		switch result.Type {
		case UnitTransformMove:
			unit.ProvinceID = result.To
			g.Positions[result.To] = unit.ID
			if pid == unit.ID {
				delete(g.Positions, result.From)
			}
			if unit.Type == UnitTypeFleet {
				g.FleetCoasts[unit.ID] = result.Coast
			}
			g.Units[unit.ID] = unit

		case UnitTransformRetreat:
			unit.ProvinceID = ""
			if pid == unit.ID {
				delete(g.Positions, result.From)
			}
			if unit.Type == UnitTypeFleet {
				delete(g.FleetCoasts, unit.ID)
			}
			g.Units[unit.ID] = unit
			g.PendingRetreats[unit.ID] = Dislodgement{
				From: result.From,
			}

		case UnitTransformHold:
			// No-op
		}

	}
	return nil
}

func validateResults(results []UnitTransform) error {
	seen := make(map[gamemap.ProvinceID]struct{})
	for _, result := range results {
		if result.Type == UnitTransformRetreat {
			continue
		}
		if _, ok := seen[result.To]; ok {
			return errors.New("duplicate province in results")
		}
		seen[result.To] = struct{}{}
	}
	return nil
}
