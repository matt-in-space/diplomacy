package game

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

// A HoldOrder holds a unit in place, preventing it from moving.
type HoldOrder struct {
	BaseOrder
	Target gamemap.ProvinceID
}

func NewHoldOrder(unit UnitID, nation gamemap.NationID, target gamemap.ProvinceID) HoldOrder {
	return HoldOrder{
		BaseOrder: BaseOrder{
			UnitID:   unit,
			NationID: nation,
		},
		Target: target,
	}
}

func (g *Game) validateHoldOrder(order HoldOrder, unit Unit, gm *gamemap.GameMap) error {
	if unit.ProvinceID != order.Target {
		return fmt.Errorf("unit %q must be in province %q to hold", unit.ID, order.Target)
	}
	return nil
}
