package game

import (
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// A HoldOrder holds a unit in place, preventing it from moving.
type HoldOrder struct {
	BaseUnitOrder
}

func NewHoldOrder(unit UnitID, nation gamemap.NationID) HoldOrder {
	return HoldOrder{
		BaseUnitOrder: newBaseUnitOrder(unit, nation),
	}
}
