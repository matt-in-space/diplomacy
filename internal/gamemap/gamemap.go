package gamemap

type CoastID string
type ProvinceID string
type ProvinceType string

const (
	Inland  ProvinceType = "inland"
	Coastal ProvinceType = "coastal"
	Water   ProvinceType = "water"
)

type GameMap struct {
	Provinces      map[ProvinceID]Province
	ArmyAdjacency  map[ProvinceID][]ProvinceID
	FleetAdjacency map[CoastID][]CoastID
}

type Province struct {
	ID           ProvinceID
	Name         string
	Type         ProvinceType
	SupplyCenter bool
	HomeNation   string
	Coasts       []CoastID
}
