package gamemap

import (
	"encoding/json"
	"fmt"
	"slices"
)

type gameMapData struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Nations        []string            `json:"nations"`
	StartingUnits  []startingUnitData  `json:"starting_units"`
	Provinces      []provinceData      `json:"provinces"`
	ArmyAdjacency  map[string][]string `json:"army_adjacency"`
	FleetAdjacency map[string][]string `json:"fleet_adjacency"`
}

type provinceData struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	SupplyCenter bool     `json:"supply_center"`
	HomeNation   string   `json:"home_nation"`
	Coasts       []string `json:"coasts"`
}

type startingUnitData struct {
	Nation   string `json:"nation"`
	Type     string `json:"type"`
	Province string `json:"province"`
	Coast    string `json:"coast"`
}

// Load parses the given JSON data into a GameMap.
// It validates the data and returns an error if it is invalid.
func Load(data []byte) (*GameMap, error) {
	var g gameMapData

	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("Unable to load game map from JSON: %w", err)
	}

	return hydrateGameMap(g)
}

func hydrateGameMap(g gameMapData) (*GameMap, error) {
	m := &GameMap{
		ID:             MapID(g.ID),
		Name:           g.Name,
		Nations:        make([]NationID, 0, len(g.Nations)),
		Provinces:      make(map[ProvinceID]Province, len(g.Provinces)),
		ArmyAdjacency:  make(map[ProvinceID][]ProvinceID, len(g.ArmyAdjacency)),
		FleetAdjacency: make(map[CoastID][]CoastID, len(g.FleetAdjacency)),
	}
	coastToProvince := make(map[CoastID]ProvinceID)
	nations := make(map[NationID]struct{}, len(g.Nations))

	if err := hydrateNations(g.Nations, m, nations); err != nil {
		return nil, err
	}
	if err := hydrateProvinces(g.Provinces, m, coastToProvince, nations); err != nil {
		return nil, err
	}
	if err := hydrateStartingUnits(g.StartingUnits, m, nations, coastToProvince); err != nil {
		return nil, err
	}
	if err := hydrateArmyAdjacency(g.ArmyAdjacency, m); err != nil {
		return nil, err
	}
	if err := hydrateFleetAdjacency(g.FleetAdjacency, m, coastToProvince); err != nil {
		return nil, err
	}
	if err := validateArmyAdjacencySymmetry(m); err != nil {
		return nil, err
	}
	if err := validateFleetAdjacencySymmetry(m); err != nil {
		return nil, err
	}

	return m, nil
}

func hydrateNations(ids []string, m *GameMap, nations map[NationID]struct{}) error {
	for _, id := range ids {
		nid := NationID(id)
		if nid == "" {
			return fmt.Errorf("nation id is required")
		}
		if _, ok := nations[nid]; ok {
			return fmt.Errorf("duplicate nation %q", nid)
		}

		nations[nid] = struct{}{}
		m.Nations = append(m.Nations, nid)
	}

	return nil
}

func hydrateProvinces(provinces []provinceData, m *GameMap, coastToProvince map[CoastID]ProvinceID, nations map[NationID]struct{}) error {
	for _, p := range provinces {
		pid := ProvinceID(p.ID)
		if pid == "" {
			return fmt.Errorf("province id is required")
		}
		if _, ok := m.Provinces[pid]; ok {
			return fmt.Errorf("duplicate province %q", pid)
		}

		pt, err := parseProvinceType(p.Type)
		if err != nil {
			return fmt.Errorf("province %q: %w", pid, err)
		}
		if err := validateProvinceCoasts(pid, pt, p.Coasts); err != nil {
			return err
		}
		homeNation := NationID(p.HomeNation)
		if err := validateHomeNation(pid, homeNation, nations); err != nil {
			return err
		}

		province := Province{
			ID:           pid,
			Name:         p.Name,
			Type:         pt,
			SupplyCenter: p.SupplyCenter,
			HomeNation:   homeNation,
			Coasts:       make([]CoastID, len(p.Coasts)),
		}

		for i, c := range p.Coasts {
			cid := CoastID(c)
			if _, ok := coastToProvince[cid]; ok {
				return fmt.Errorf("duplicate coast %q", cid)
			}
			province.Coasts[i] = cid
			coastToProvince[cid] = pid
		}

		m.Provinces[pid] = province
	}

	return nil
}

func hydrateStartingUnits(units []startingUnitData, m *GameMap, nations map[NationID]struct{}, coastToProvince map[CoastID]ProvinceID) error {
	occupiedProvinces := make(map[ProvinceID]struct{}, len(units))

	for _, unit := range units {
		nation := NationID(unit.Nation)
		unitType, err := parseStartingUnitType(unit.Type)
		if err != nil {
			return err
		}

		provinceID := ProvinceID(unit.Province)
		province, ok := m.Provinces[provinceID]
		if !ok {
			return fmt.Errorf("starting unit province %q not found", provinceID)
		}
		if _, ok := occupiedProvinces[provinceID]; ok {
			return fmt.Errorf("multiple starting units in province %q", provinceID)
		}
		if err := validateStartingUnitNation(nation, nations); err != nil {
			return err
		}
		if err := validateStartingUnitPosition(unitType, province, CoastID(unit.Coast), coastToProvince); err != nil {
			return err
		}

		m.StartingUnits = append(m.StartingUnits, StartingUnit{
			Nation:   nation,
			Type:     unitType,
			Province: provinceID,
			Coast:    CoastID(unit.Coast),
		})
		occupiedProvinces[provinceID] = struct{}{}
	}

	return nil
}

func hydrateArmyAdjacency(adjacency map[string][]string, m *GameMap) error {
	for from, tos := range adjacency {
		pid := ProvinceID(from)
		if err := validateArmyProvince(pid, m); err != nil {
			return err
		}

		m.ArmyAdjacency[pid] = make([]ProvinceID, len(tos))
		for i, to := range tos {
			adjacentProvince := ProvinceID(to)
			if err := validateArmyProvince(adjacentProvince, m); err != nil {
				return err
			}
			m.ArmyAdjacency[pid][i] = adjacentProvince
		}
	}

	return nil
}

func hydrateFleetAdjacency(adjacency map[string][]string, m *GameMap, coastToProvince map[CoastID]ProvinceID) error {
	for from, tos := range adjacency {
		cid := CoastID(from)
		if err := validateCoast(cid, coastToProvince); err != nil {
			return err
		}

		m.FleetAdjacency[cid] = make([]CoastID, len(tos))
		for i, to := range tos {
			adjacentCoast := CoastID(to)
			if err := validateCoast(adjacentCoast, coastToProvince); err != nil {
				return err
			}
			m.FleetAdjacency[cid][i] = adjacentCoast
		}
	}

	return nil
}

func parseProvinceType(t string) (ProvinceType, error) {
	switch ProvinceType(t) {
	case Inland, Coastal, Water:
		return ProvinceType(t), nil
	default:
		return "", fmt.Errorf("unknown province type %q", t)
	}
}

func parseStartingUnitType(t string) (StartingUnitType, error) {
	switch StartingUnitType(t) {
	case StartingUnitTypeArmy, StartingUnitTypeFleet:
		return StartingUnitType(t), nil
	default:
		return "", fmt.Errorf("unknown starting unit type %q", t)
	}
}

func validateProvinceCoasts(pid ProvinceID, pt ProvinceType, coasts []string) error {
	if pt == Inland && len(coasts) > 0 {
		return fmt.Errorf("province %q: inland provinces cannot have coasts", pid)
	}
	if pt != Inland && len(coasts) == 0 {
		return fmt.Errorf("province %q: %s provinces must have at least one coast", pid, pt)
	}

	if slices.Contains(coasts, "") {
		return fmt.Errorf("province %q: coast id is required", pid)
	}

	return nil
}

func validateHomeNation(pid ProvinceID, homeNation NationID, nations map[NationID]struct{}) error {
	if homeNation == "" {
		return nil
	}
	if _, ok := nations[homeNation]; !ok {
		return fmt.Errorf("province %q: home nation %q not found", pid, homeNation)
	}

	return nil
}

func validateStartingUnitNation(nation NationID, nations map[NationID]struct{}) error {
	if nation == "" {
		return fmt.Errorf("starting unit nation is required")
	}
	if _, ok := nations[nation]; !ok {
		return fmt.Errorf("starting unit nation %q not found", nation)
	}

	return nil
}

func validateStartingUnitPosition(unitType StartingUnitType, province Province, coast CoastID, coastToProvince map[CoastID]ProvinceID) error {
	if unitType == StartingUnitTypeArmy {
		if province.Type == Water {
			return fmt.Errorf("army cannot start in water province %q", province.ID)
		}
		if coast != "" {
			return fmt.Errorf("army starting in province %q cannot have coast", province.ID)
		}
		return nil
	}

	if coast == "" {
		return fmt.Errorf("fleet starting in province %q must have coast", province.ID)
	}
	coastProvince, ok := coastToProvince[coast]
	if !ok {
		return fmt.Errorf("fleet starting coast %q not found", coast)
	}
	if coastProvince != province.ID {
		return fmt.Errorf("fleet starting coast %q does not belong to province %q", coast, province.ID)
	}

	return nil
}

func validateArmyProvince(pid ProvinceID, m *GameMap) error {
	province, ok := m.Provinces[pid]
	if !ok {
		return fmt.Errorf("province %q not found", pid)
	}
	if province.Type == Water {
		return fmt.Errorf("province %q: water provinces cannot have army adjacency", pid)
	}

	return nil
}

func validateCoast(cid CoastID, coastToProvince map[CoastID]ProvinceID) error {
	if _, ok := coastToProvince[cid]; !ok {
		return fmt.Errorf("coast %q not found", cid)
	}

	return nil
}

func validateArmyAdjacencySymmetry(m *GameMap) error {
	for from, tos := range m.ArmyAdjacency {
		for _, to := range tos {
			if !slices.Contains(m.ArmyAdjacency[to], from) {
				return fmt.Errorf("army adjacency %q -> %q is not bidirectional", from, to)
			}
		}
	}

	return nil
}

func validateFleetAdjacencySymmetry(m *GameMap) error {
	for from, tos := range m.FleetAdjacency {
		for _, to := range tos {
			if !slices.Contains(m.FleetAdjacency[to], from) {
				return fmt.Errorf("fleet adjacency %q -> %q is not bidirectional", from, to)
			}
		}
	}

	return nil
}
