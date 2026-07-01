package gamemap

import (
	"encoding/json"
	"fmt"
)

type gameMapData struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
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

func Load(data []byte) (*GameMap, error) {
	var g gameMapData

	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("Unable to load game map from JSON: %w", err)
	}

	return hydrateGameMap(g)
}

func hydrateGameMap(g gameMapData) (*GameMap, error) {
	return nil, nil
}
