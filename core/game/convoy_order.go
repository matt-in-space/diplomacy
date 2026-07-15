package game

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// A ConvoyOrder orders a fleet in a water province to convoy an army from one coastal province to another.
type ConvoyOrder struct {
	BaseOrder
	ConvoyedUnit UnitID
	From         gamemap.ProvinceID
	To           gamemap.ProvinceID
}

func NewConvoyOrder(unit UnitID, nation gamemap.NationID, convoyedUnit UnitID, from gamemap.ProvinceID, to gamemap.ProvinceID) ConvoyOrder {
	return ConvoyOrder{
		BaseOrder: BaseOrder{
			UnitID:   unit,
			NationID: nation,
		},
		ConvoyedUnit: convoyedUnit,
		From:         from,
		To:           to,
	}
}

func (g *Game) validateConvoyOrder(order ConvoyOrder, unit Unit, gm *gamemap.GameMap) error {
	if unit.Type != UnitTypeFleet {
		return fmt.Errorf("unit %q must be a fleet to convoy", unit.ID)
	}

	fleetProvince, ok := gm.Province(unit.ProvinceID)
	if !ok {
		return fmt.Errorf("fleet province %q not found", unit.ProvinceID)
	}
	if fleetProvince.Type != gamemap.Water {
		return fmt.Errorf("fleet unit %q must be in a water province to convoy", unit.ID)
	}

	convoyedUnit, err := g.validateConvoyedUnit(order.ConvoyedUnit)
	if err != nil {
		return err
	}
	if order.From != convoyedUnit.ProvinceID {
		return fmt.Errorf("convoy origin %q does not match convoyed unit province %q", order.From, convoyedUnit.ProvinceID)
	}
	if order.To == order.From {
		return fmt.Errorf("convoy destination cannot be the origin province %q", order.From)
	}
	if err := validateConvoyEndpoint(order.From, "origin", gm); err != nil {
		return err
	}
	if err := validateConvoyEndpoint(order.To, "destination", gm); err != nil {
		return err
	}

	return nil
}

func (g *Game) validateConvoyedUnit(unitID UnitID) (Unit, error) {
	if unitID == "" {
		return Unit{}, fmt.Errorf("convoyed unit is required")
	}

	unit, ok := g.Units[unitID]
	if !ok {
		return Unit{}, fmt.Errorf("convoyed unit %q not found", unitID)
	}
	if unit.Type != UnitTypeArmy {
		return Unit{}, fmt.Errorf("convoyed unit %q must be an army", unitID)
	}
	if occupyingUnit, ok := g.Positions[unit.ProvinceID]; !ok || occupyingUnit != unitID {
		return Unit{}, fmt.Errorf("convoyed unit %q is not on the board", unitID)
	}

	return unit, nil
}

func validateConvoyEndpoint(provinceID gamemap.ProvinceID, label string, gm *gamemap.GameMap) error {
	province, ok := gm.Province(provinceID)
	if !ok {
		return fmt.Errorf("convoy %s province %q not found", label, provinceID)
	}
	if province.Type != gamemap.Coastal {
		return fmt.Errorf("convoy %s province %q must be coastal", label, provinceID)
	}

	return nil
}
