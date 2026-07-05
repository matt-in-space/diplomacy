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
	target, ok := gm.Province(order.Target)
	if !ok {
		return fmt.Errorf("target province %q not found", order.Target)
	}
	if order.Target == unit.ProvinceID {
		return fmt.Errorf("unit %q cannot move to its current province", unit.ID)
	}

	switch unit.Type {
	case UnitTypeArmy:
		return g.validateArmyMoveOrder(order, unit, target, gm)
	case UnitTypeFleet:
		return g.validateFleetMoveOrder(order, unit, target, gm)
	default:
		return fmt.Errorf("unit %q has unknown type %q", unit.ID, unit.Type)
	}
}

func (g *Game) validateArmyMoveOrder(order MoveOrder, unit Unit, target gamemap.Province, gm *gamemap.GameMap) error {
	if order.TargetCoast != "" {
		return fmt.Errorf("army move cannot specify target coast")
	}
	if target.Type == gamemap.Water {
		return fmt.Errorf("army cannot move to water province %q", target.ID)
	}
	if !gm.CanArmyMove(unit.ProvinceID, order.Target) {
		return fmt.Errorf("army cannot move from %q to %q", unit.ProvinceID, order.Target)
	}

	return nil
}

func (g *Game) validateFleetMoveOrder(order MoveOrder, unit Unit, target gamemap.Province, gm *gamemap.GameMap) error {
	if target.Type == gamemap.Inland {
		return fmt.Errorf("fleet cannot move to inland province %q", target.ID)
	}

	sourceCoast, ok := g.FleetCoasts[unit.ID]
	if !ok {
		return fmt.Errorf("fleet unit %q has no source coast", unit.ID)
	}

	targetCoast, err := resolveFleetTargetCoast(order, target)
	if err != nil {
		return err
	}
	if !gm.CanFleetMove(sourceCoast, targetCoast) {
		return fmt.Errorf("fleet cannot move from coast %q to coast %q", sourceCoast, targetCoast)
	}

	return nil
}

func resolveFleetTargetCoast(order MoveOrder, target gamemap.Province) (gamemap.CoastID, error) {
	if order.TargetCoast != "" {
		if !slices.Contains(target.Coasts, order.TargetCoast) {
			return "", fmt.Errorf("target coast %q does not belong to province %q", order.TargetCoast, target.ID)
		}
		return order.TargetCoast, nil
	}

	if len(target.Coasts) != 1 {
		return "", fmt.Errorf("target province %q requires target coast", target.ID)
	}

	return target.Coasts[0], nil
}
