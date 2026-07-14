package game

import (
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

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
