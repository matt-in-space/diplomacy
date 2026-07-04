package game

import "github.com/matt-in-space/diplomacy/internal/gamemap"

type Order interface {
	Unit() UnitID
	Nation() gamemap.NationID
}

type BaseOrder struct {
	UnitID   UnitID
	NationID gamemap.NationID
}

func (o BaseOrder) Unit() UnitID {
	return o.UnitID
}

func (o BaseOrder) Nation() gamemap.NationID {
	return o.NationID
}

// A HoldOrder holds a unit in place, preventing it from moving.
type HoldOrder struct {
	BaseOrder
}

func NewHoldOrder(unit UnitID, nation gamemap.NationID) HoldOrder {
	return HoldOrder{
		BaseOrder: BaseOrder{
			UnitID:   unit,
			NationID: nation,
		},
	}
}

// A MoveOrder moves a unit from one location to another. If the target province is occupied by a
// unit of a different nation it is considered an attack order.
type MoveOrder struct {
	BaseOrder
	Target gamemap.ProvinceID
}
