package game

import (
	"fmt"
	"slices"

	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

type GameID string
type PlayerID string

type NewGameConfig struct {
	ID          GameID
	Assignments map[gamemap.NationID]PlayerID
}

type Game struct {
	ID          GameID
	MapID       gamemap.MapID
	Assignments map[gamemap.NationID]PlayerID
	Turn        Turn
	Units       map[UnitID]Unit
	Positions   map[gamemap.ProvinceID]UnitID
	FleetCoasts map[UnitID]gamemap.CoastID
	Orders      map[UnitID]Order
}

func NewGame(cfg NewGameConfig, gm *gamemap.GameMap) (*Game, error) {
	if gm == nil {
		return nil, fmt.Errorf("game map is required")
	}

	g := &Game{
		ID:          cfg.ID,
		MapID:       gm.ID,
		Assignments: make(map[gamemap.NationID]PlayerID, len(cfg.Assignments)),
		Turn:        StartingTurn(),
		Units:       make(map[UnitID]Unit, len(gm.StartingUnits)),
		Positions:   make(map[gamemap.ProvinceID]UnitID, len(gm.StartingUnits)),
		FleetCoasts: make(map[UnitID]gamemap.CoastID),
		Orders:      make(map[UnitID]Order),
	}

	for nation, player := range cfg.Assignments {
		if !slices.Contains(gm.Nations, nation) {
			return nil, fmt.Errorf("assignment nation %q not found", nation)
		}
		g.Assignments[nation] = player
	}

	for _, startingUnit := range gm.StartingUnits {
		unitType, err := unitTypeFromStartingUnit(startingUnit.Type)
		if err != nil {
			return nil, err
		}

		unitID := startingUnitID(startingUnit)
		if _, ok := g.Units[unitID]; ok {
			return nil, fmt.Errorf("duplicate unit %q", unitID)
		}
		if _, ok := g.Positions[startingUnit.Province]; ok {
			return nil, fmt.Errorf("province %q already occupied", startingUnit.Province)
		}

		g.Units[unitID] = Unit{
			ID:         unitID,
			NationID:   startingUnit.Nation,
			ProvinceID: startingUnit.Province,
			Type:       unitType,
		}
		g.Positions[startingUnit.Province] = unitID
		if unitType == UnitTypeFleet {
			g.FleetCoasts[unitID] = startingUnit.Coast
		}
	}

	return g, nil
}

func (g *Game) SubmitOrder(order Order, gm *gamemap.GameMap) error {
	if order == nil {
		return fmt.Errorf("order is required")
	}
	if gm == nil {
		return fmt.Errorf("game map is required")
	}
	if gm.ID != g.MapID {
		return fmt.Errorf("game map %q does not match game map %q", gm.ID, g.MapID)
	}
	if g.Turn.Phase != AcceptOrders {
		return fmt.Errorf("cannot submit order during phase %q", g.Turn.Phase)
	}

	nation := order.Nation()
	if !slices.Contains(gm.Nations, nation) {
		return fmt.Errorf("order nation %q not found", nation)
	}

	unitID := order.Unit()
	unit, ok := g.Units[unitID]
	if !ok {
		return fmt.Errorf("unit %q not found", unitID)
	}
	if unit.NationID != nation {
		return fmt.Errorf("unit %q belongs to nation %q, not %q", unitID, unit.NationID, nation)
	}
	if occupyingUnit, ok := g.Positions[unit.ProvinceID]; !ok || occupyingUnit != unitID {
		return fmt.Errorf("unit %q is not on the board", unitID)
	}

	switch order := order.(type) {
	case HoldOrder:
	case MoveOrder:
		if err := g.validateMoveOrder(order, unit, gm); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported order type %T", order)
	}

	if g.Orders == nil {
		g.Orders = make(map[UnitID]Order)
	}
	g.Orders[unitID] = order

	return nil
}

func unitTypeFromStartingUnit(unitType gamemap.StartingUnitType) (UnitType, error) {
	switch unitType {
	case gamemap.StartingUnitTypeArmy:
		return UnitTypeArmy, nil
	case gamemap.StartingUnitTypeFleet:
		return UnitTypeFleet, nil
	default:
		return "", fmt.Errorf("unknown starting unit type %q", unitType)
	}
}

func startingUnitID(unit gamemap.StartingUnit) UnitID {
	return UnitID(fmt.Sprintf("%s-%s-%s-start", unit.Nation, unit.Type, unit.Province))
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
