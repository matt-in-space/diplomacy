package gamemap

import "slices"

type MapID string
type CoastID string
type ProvinceID string
type ProvinceType string

const (
	Inland  ProvinceType = "inland"
	Coastal ProvinceType = "coastal"
	Water   ProvinceType = "water"
)

// A GameMap represents the static map of the game. In addition to Provinces
// it also contains adjacency information for armies and fleets.
type GameMap struct {
	ID             MapID
	Name           string
	Provinces      map[ProvinceID]Province
	ArmyAdjacency  map[ProvinceID][]ProvinceID
	FleetAdjacency map[CoastID][]CoastID
}

// A Province represents a single province on the game map.
type Province struct {
	ID           ProvinceID
	Name         string
	Type         ProvinceType
	SupplyCenter bool
	HomeNation   string
	Coasts       []CoastID
}

// Province returns the province for the given ID.
func (g *GameMap) Province(id ProvinceID) (Province, bool) {
	province, ok := g.Provinces[id]
	return province, ok
}

// CoastsFor returns the coasts attached to the given province.
func (g *GameMap) CoastsFor(id ProvinceID) []CoastID {
	province, ok := g.Provinces[id]
	if !ok {
		return nil
	}

	return province.Coasts
}

// IsInland reports whether the province is an inland province.
func (g *GameMap) IsInland(id ProvinceID) bool {
	province, ok := g.Provinces[id]
	return ok && province.Type == Inland
}

// IsWater reports whether the province is a water province.
func (g *GameMap) IsWater(id ProvinceID) bool {
	province, ok := g.Provinces[id]
	return ok && province.Type == Water
}

// IsCoastal reports whether the province is a coastal land province.
func (g *GameMap) IsCoastal(id ProvinceID) bool {
	province, ok := g.Provinces[id]
	return ok && province.Type == Coastal
}

// ArmyAdjacent reports whether two provinces are adjacent for army movement.
func (g *GameMap) ArmyAdjacent(from ProvinceID, to ProvinceID) bool {
	return slices.Contains(g.ArmyAdjacency[from], to)
}

// ArmyNeighbors returns the provinces adjacent for army movement.
func (g *GameMap) ArmyNeighbors(id ProvinceID) []ProvinceID {
	return g.ArmyAdjacency[id]
}

// FleetAdjacent reports whether two coasts are adjacent for fleet movement.
func (g *GameMap) FleetAdjacent(from CoastID, to CoastID) bool {
	return slices.Contains(g.FleetAdjacency[from], to)
}

// FleetNeighbors returns the coasts adjacent for fleet movement.
func (g *GameMap) FleetNeighbors(id CoastID) []CoastID {
	return g.FleetAdjacency[id]
}

// ProvinceForCoast returns the province attached to the given coast.
func (g *GameMap) ProvinceForCoast(id CoastID) (ProvinceID, bool) {
	for _, province := range g.Provinces {
		if slices.Contains(province.Coasts, id) {
			return province.ID, true
		}
	}

	return "", false
}

// CanArmyMove reports whether an army can move between two provinces.
func (g *GameMap) CanArmyMove(from ProvinceID, to ProvinceID) bool {
	return g.ArmyAdjacent(from, to)
}

// CanFleetMove reports whether a fleet can move between two coasts.
func (g *GameMap) CanFleetMove(from CoastID, to CoastID) bool {
	return g.FleetAdjacent(from, to)
}
