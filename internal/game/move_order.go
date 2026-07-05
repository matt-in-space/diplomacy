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

func (g *Game) validateMoveOrder(order MoveOrder, unit Unit, gm *gamemap.GameMap) error {
	return g.validateUnitCanReach(unit, order.Target, order.TargetCoast, gm)
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
