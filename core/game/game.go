package game

import (
	"fmt"
	"maps"
	"slices"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

type GameID string
type PlayerID string

type NewGameConfig struct {
	ID          GameID
	Assignments map[gamemap.NationID]PlayerID
}

type Dislodgement struct {
	From  gamemap.ProvinceID
	Coast gamemap.CoastID
}

type Game struct {
	ID              GameID
	MapID           gamemap.MapID
	Assignments     map[gamemap.NationID]PlayerID
	Turn            Turn
	Units           map[UnitID]Unit
	Positions       map[gamemap.ProvinceID]UnitID
	FleetCoasts     map[UnitID]gamemap.CoastID
	Orders          map[UnitID]Order
	PendingRetreats map[UnitID]Dislodgement
}

func NewGame(cfg NewGameConfig, gm *gamemap.GameMap) (*Game, error) {
	if gm == nil {
		return nil, fmt.Errorf("game map is required")
	}

	g := &Game{
		ID:              cfg.ID,
		MapID:           gm.ID,
		Assignments:     make(map[gamemap.NationID]PlayerID, len(cfg.Assignments)),
		Turn:            StartingTurn(),
		Units:           make(map[UnitID]Unit, len(gm.StartingUnits)),
		Positions:       make(map[gamemap.ProvinceID]UnitID, len(gm.StartingUnits)),
		FleetCoasts:     make(map[UnitID]gamemap.CoastID),
		Orders:          make(map[UnitID]Order),
		PendingRetreats: make(map[UnitID]Dislodgement),
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
	clone.Positions = maps.Clone(g.Positions)
	clone.FleetCoasts = maps.Clone(g.FleetCoasts)
	clone.Orders = maps.Clone(g.Orders)
	clone.PendingRetreats = maps.Clone(g.PendingRetreats)
	return &clone
}
