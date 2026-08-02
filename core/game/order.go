package game

import "github.com/matt-in-space/diplomacy/core/gamemap"

// An Order is submitted by a nation. Most orders act on an existing unit (see
// UnitOrder); a build order does not, since it creates the unit.
type Order interface {
	Nation() gamemap.NationID
}

// A UnitOrder is an Order that acts on a unit already on the board.
type UnitOrder interface {
	Order
	Unit() UnitID
}

type BaseOrder struct {
	NationID gamemap.NationID
}

func (o BaseOrder) Nation() gamemap.NationID {
	return o.NationID
}

// BaseUnitOrder is embedded by every order type that names an existing unit.
type BaseUnitOrder struct {
	BaseOrder
	UnitID UnitID
}

func (o BaseUnitOrder) Unit() UnitID {
	return o.UnitID
}

func newBaseUnitOrder(unit UnitID, nation gamemap.NationID) BaseUnitOrder {
	return BaseUnitOrder{
		BaseOrder: BaseOrder{NationID: nation},
		UnitID:    unit,
	}
}
