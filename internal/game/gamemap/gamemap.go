package gamemap

type ProvinceID string
type CoastID string
type ProvinceType int

const (
	Inland ProvinceType = iota
	Coastal
	Water
)

type Province struct {
	ID           ProvinceID
	Name         string
	Type         ProvinceType
	SupplyCenter bool
	HomeNation   string
	Coasts       []CoastID
}

type GameMap struct {
	Provinces      map[ProvinceID]Province
	ArmyAdjacency  map[ProvinceID][]ProvinceID
	FleetAdjacency map[CoastID][]CoastID
}
