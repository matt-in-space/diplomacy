package game

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

// A SupportHoldOrder supports another unit holding its current province.
type SupportHoldOrder struct {
	BaseOrder
	SupportedUnit UnitID
}

func NewSupportHoldOrder(unit UnitID, nation gamemap.NationID, supportedUnit UnitID) SupportHoldOrder {
	return SupportHoldOrder{
		BaseOrder: BaseOrder{
			UnitID:   unit,
			NationID: nation,
		},
		SupportedUnit: supportedUnit,
	}
}

// A SupportMoveOrder supports another unit moving to a target province.
type SupportMoveOrder struct {
	BaseOrder
	SupportedUnit UnitID
	Target        gamemap.ProvinceID
	TargetCoast   gamemap.CoastID
}

func NewSupportMoveOrder(unit UnitID, nation gamemap.NationID, supportedUnit UnitID, target gamemap.ProvinceID, targetCoast gamemap.CoastID) SupportMoveOrder {
	return SupportMoveOrder{
		BaseOrder: BaseOrder{
			UnitID:   unit,
			NationID: nation,
		},
		SupportedUnit: supportedUnit,
		Target:        target,
		TargetCoast:   targetCoast,
	}
}

func (g *Game) validateSupportHoldOrder(order SupportHoldOrder, unit Unit, gm *gamemap.GameMap) error {
	supportedUnit, err := g.validateSupportOrderContext(order.UnitID, order.SupportedUnit)
	if err != nil {
		return err
	}

	targetCoast := gamemap.CoastID("")
	if unit.Type == UnitTypeFleet && supportedUnit.Type == UnitTypeFleet {
		coast, ok := g.FleetCoasts[supportedUnit.ID]
		if !ok {
			return fmt.Errorf("supported fleet unit %q has no coast", supportedUnit.ID)
		}
		targetCoast = coast
	}

	return g.validateUnitCanReach(unit, supportedUnit.ProvinceID, targetCoast, gm)
}

func (g *Game) validateSupportMoveOrder(order SupportMoveOrder, unit Unit, gm *gamemap.GameMap) error {
	supportedUnit, err := g.validateSupportOrderContext(order.UnitID, order.SupportedUnit)
	if err != nil {
		return err
	}
	if order.Target == supportedUnit.ProvinceID {
		return fmt.Errorf("support move target %q is supported unit's current province", order.Target)
	}

	if err := g.validateUnitCanReach(unit, order.Target, order.TargetCoast, gm); err != nil {
		return err
	}
	if err := g.validateUnitCanReach(supportedUnit, order.Target, order.TargetCoast, gm); err != nil {
		return fmt.Errorf("supported unit %q cannot move to %q: %w", supportedUnit.ID, order.Target, err)
	}

	return nil
}

func (g *Game) validateUnitCanReach(unit Unit, targetID gamemap.ProvinceID, targetCoastID gamemap.CoastID, gm *gamemap.GameMap) error {
	target, ok := gm.Province(targetID)
	if !ok {
		return fmt.Errorf("target province %q not found", targetID)
	}
	if targetID == unit.ProvinceID {
		return fmt.Errorf("unit %q cannot move to its current province", unit.ID)
	}

	switch unit.Type {
	case UnitTypeArmy:
		return g.validateArmyCanReach(unit, target, targetCoastID, gm)
	case UnitTypeFleet:
		return g.validateFleetCanReach(unit, target, targetCoastID, gm)
	default:
		return fmt.Errorf("unit %q has unknown type %q", unit.ID, unit.Type)
	}
}

func (g *Game) validateArmyCanReach(unit Unit, target gamemap.Province, targetCoastID gamemap.CoastID, gm *gamemap.GameMap) error {
	if targetCoastID != "" {
		return fmt.Errorf("army move cannot specify target coast")
	}
	if target.Type == gamemap.Water {
		return fmt.Errorf("army cannot move to water province %q", target.ID)
	}
	if !gm.CanArmyMove(unit.ProvinceID, target.ID) {
		return fmt.Errorf("army cannot move from %q to %q", unit.ProvinceID, target.ID)
	}

	return nil
}

func (g *Game) validateFleetCanReach(unit Unit, target gamemap.Province, targetCoastID gamemap.CoastID, gm *gamemap.GameMap) error {
	if target.Type == gamemap.Inland {
		return fmt.Errorf("fleet cannot move to inland province %q", target.ID)
	}

	sourceCoast, ok := g.FleetCoasts[unit.ID]
	if !ok {
		return fmt.Errorf("fleet unit %q has no source coast", unit.ID)
	}

	targetCoast, err := resolveFleetTargetCoast(targetCoastID, target)
	if err != nil {
		return err
	}
	if !gm.CanFleetMove(sourceCoast, targetCoast) {
		return fmt.Errorf("fleet cannot move from coast %q to coast %q", sourceCoast, targetCoast)
	}

	return nil
}

func (g *Game) validateSupportOrderContext(unitID UnitID, supportedUnitID UnitID) (Unit, error) {
	if supportedUnitID == "" {
		return Unit{}, fmt.Errorf("supported unit is required")
	}
	if unitID == supportedUnitID {
		return Unit{}, fmt.Errorf("unit %q cannot support itself", unitID)
	}

	supportedUnit, ok := g.Units[supportedUnitID]
	if !ok {
		return Unit{}, fmt.Errorf("supported unit %q not found", supportedUnitID)
	}
	if occupyingUnit, ok := g.Positions[supportedUnit.ProvinceID]; !ok || occupyingUnit != supportedUnitID {
		return Unit{}, fmt.Errorf("supported unit %q is not on the board", supportedUnitID)
	}

	return supportedUnit, nil
}
