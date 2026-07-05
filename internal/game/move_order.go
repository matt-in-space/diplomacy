package game

import (
	"fmt"
	"slices"

	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

// A MoveOrder moves a unit from one location to another. If the target province is occupied by a
// unit of a different nation it is considered an attack order.
type MoveOrder struct {
	BaseOrder
	Target      gamemap.ProvinceID
	TargetCoast gamemap.CoastID
	ViaConvoy   bool
}

func NewMoveOrder(unit UnitID, nation gamemap.NationID, target gamemap.ProvinceID, targetCoast gamemap.CoastID) MoveOrder {
	return MoveOrder{
		BaseOrder: BaseOrder{
			UnitID:   unit,
			NationID: nation,
		},
		Target:      target,
		TargetCoast: targetCoast,
	}
}

func NewConvoyedMoveOrder(unit UnitID, nation gamemap.NationID, target gamemap.ProvinceID) MoveOrder {
	return MoveOrder{
		BaseOrder: BaseOrder{
			UnitID:   unit,
			NationID: nation,
		},
		Target:    target,
		ViaConvoy: true,
	}
}

func (g *Game) validateMoveOrder(order MoveOrder, unit Unit, gm *gamemap.GameMap) error {
	if order.ViaConvoy {
		return g.validateConvoyedMoveOrder(order, unit, gm)
	}

	return g.validateUnitCanReach(unit, order.Target, order.TargetCoast, gm)
}

func (g *Game) validateConvoyedMoveOrder(order MoveOrder, unit Unit, gm *gamemap.GameMap) error {
	if unit.Type != UnitTypeArmy {
		return fmt.Errorf("unit %q must be an army to move via convoy", unit.ID)
	}
	if order.TargetCoast != "" {
		return fmt.Errorf("convoyed army move cannot specify target coast")
	}
	if order.Target == unit.ProvinceID {
		return fmt.Errorf("unit %q cannot move to its current province", unit.ID)
	}

	origin, ok := gm.Province(unit.ProvinceID)
	if !ok {
		return fmt.Errorf("origin province %q not found", unit.ProvinceID)
	}
	if origin.Type != gamemap.Coastal {
		return fmt.Errorf("convoy origin province %q must be coastal", origin.ID)
	}

	target, ok := gm.Province(order.Target)
	if !ok {
		return fmt.Errorf("target province %q not found", order.Target)
	}
	if target.Type != gamemap.Coastal {
		return fmt.Errorf("convoy destination province %q must be coastal", target.ID)
	}

	return nil
}

func resolveFleetTargetCoast(targetCoast gamemap.CoastID, target gamemap.Province) (gamemap.CoastID, error) {
	if targetCoast != "" {
		if !slices.Contains(target.Coasts, targetCoast) {
			return "", fmt.Errorf("target coast %q does not belong to province %q", targetCoast, target.ID)
		}
		return targetCoast, nil
	}

	if len(target.Coasts) != 1 {
		return "", fmt.Errorf("target province %q requires target coast", target.ID)
	}

	return target.Coasts[0], nil
}
