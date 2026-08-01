package game

import (
	"fmt"
	"maps"
	"slices"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

type GameID string

type NewGameConfig struct {
	ID          GameID
	Assignments map[gamemap.NationID]PlayerID
}

type Game struct {
	ID                  GameID
	MapID               gamemap.MapID
	Assignments         map[gamemap.NationID]PlayerID
	Turn                Turn
	Units               map[UnitID]Unit
	Orders              map[UnitID]Order
	CommittedOrders     map[gamemap.NationID]struct{}
	LastOrderResolution Resolution
}

func NewGame(cfg NewGameConfig, gm *gamemap.GameMap) (*Game, error) {
	if gm == nil {
		return nil, fmt.Errorf("game map is required")
	}

	g := &Game{
		ID:                  cfg.ID,
		MapID:               gm.ID,
		Assignments:         make(map[gamemap.NationID]PlayerID, len(cfg.Assignments)),
		Turn:                StartingTurn(),
		Units:               make(map[UnitID]Unit, len(gm.StartingUnits)),
		Orders:              make(map[UnitID]Order),
		CommittedOrders:     make(map[gamemap.NationID]struct{}),
		LastOrderResolution: Resolution{},
	}

	for nation, player := range cfg.Assignments {
		if !slices.Contains(gm.Nations, nation) {
			return nil, fmt.Errorf("assignment nation %q not found", nation)
		}
		g.Assignments[nation] = player
	}

	seen := make(map[gamemap.ProvinceID]struct{}, len(gm.StartingUnits))
	for _, startingUnit := range gm.StartingUnits {
		unitType, err := unitTypeFromStartingUnit(startingUnit.Type)
		if err != nil {
			return nil, err
		}

		unitID := startingUnitID(startingUnit)
		if _, ok := g.Units[unitID]; ok {
			return nil, fmt.Errorf("duplicate unit %q", unitID)
		}
		if _, ok := seen[startingUnit.Province]; ok {
			return nil, fmt.Errorf("province %q already occupied", startingUnit.Province)
		}
		seen[startingUnit.Province] = struct{}{}

		unit := Unit{
			ID:         unitID,
			NationID:   startingUnit.Nation,
			ProvinceID: startingUnit.Province,
			Type:       unitType,
		}
		if unitType == UnitTypeFleet {
			unit.Coast = startingUnit.Coast
		}
		g.Units[unitID] = unit
	}

	return g, nil
}

func (g *Game) PlayerControlsNation(player PlayerID, nation gamemap.NationID) bool {
	if pid, ok := g.Assignments[nation]; ok && pid == player {
		return true
	}
	return false
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

func (g *Game) Clone() *Game {
	clone := *g
	clone.Assignments = maps.Clone(g.Assignments)
	clone.Units = maps.Clone(g.Units)
	clone.Orders = maps.Clone(g.Orders)
	clone.CommittedOrders = maps.Clone(g.CommittedOrders)
	clone.LastOrderResolution = maps.Clone(g.LastOrderResolution)

	return &clone
}
