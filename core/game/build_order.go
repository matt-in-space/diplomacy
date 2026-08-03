package game

import (
	"fmt"
	"slices"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// A BuildOrder creates a new unit at a home supply center the nation owns
// and controls. Unlike other orders it does not name an existing unit —
// resolution mints a UnitID if and when the build is applied.
type BuildOrder struct {
	BaseOrder
	ProvinceID gamemap.ProvinceID
	UnitType   UnitType
	Coast      gamemap.CoastID
}

func NewBuildOrder(nation gamemap.NationID, province gamemap.ProvinceID, unitType UnitType, coast gamemap.CoastID) BuildOrder {
	return BuildOrder{
		BaseOrder:  BaseOrder{NationID: nation},
		ProvinceID: province,
		UnitType:   unitType,
		Coast:      coast,
	}
}

// validateBuildOrder checks that a build order targets a province the nation
// may currently build at, and that the requested unit type (and, for a fleet
// in a bicoastal province, the requested coast) is valid there.
func (g *Game) validateBuildOrder(order BuildOrder, gm *gamemap.GameMap) error {
	if g.AdjustmentBalance(order.Nation()) <= 0 {
		return fmt.Errorf("nation %q has no builds available", order.Nation())
	}

	if !slices.Contains(g.LegalBuildProvinces(order.Nation(), gm), order.ProvinceID) {
		return fmt.Errorf("nation %q cannot build at province %q", order.Nation(), order.ProvinceID)
	}

	province, ok := gm.Province(order.ProvinceID)
	if !ok {
		return fmt.Errorf("province %q not found", order.ProvinceID)
	}

	switch order.UnitType {
	case UnitTypeArmy:
		if order.Coast != "" {
			return fmt.Errorf("army build cannot specify a coast")
		}
		if province.Type == gamemap.Water {
			return fmt.Errorf("army cannot build in water province %q", order.ProvinceID)
		}
	case UnitTypeFleet:
		if province.Type != gamemap.Coastal {
			return fmt.Errorf("fleet cannot build in province %q", order.ProvinceID)
		}
		if _, err := resolveFleetTargetCoast(order.Coast, province); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown unit type %q", order.UnitType)
	}

	return nil
}
