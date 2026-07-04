package game

import "github.com/matt-in-space/diplomacy/internal/gamemap"

type UnitID string
type UnitType string

const (
	UnitTypeArmy  UnitType = "army"
	UnitTypeFleet UnitType = "fleet"
)

type Unit struct {
	ID         UnitID
	NationID   gamemap.NationID
	ProvinceID gamemap.ProvinceID
	Type       UnitType
}
