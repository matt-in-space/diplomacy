package gamemap

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
