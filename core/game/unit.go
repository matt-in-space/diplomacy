package game

import "github.com/matt-in-space/diplomacy/core/gamemap"

type UnitID string
type UnitType string

const (
	UnitTypeArmy  UnitType = "army"
	UnitTypeFleet UnitType = "fleet"
)

// A Unit is the single source of truth for a unit's position, coast, and
// dislodged status. Exactly one of ProvinceID and DislodgedFrom is set: a unit
// is either on the board at ProvinceID, or dislodged and awaiting a retreat
// order from DislodgedFrom.
type Unit struct {
	ID            UnitID
	NationID      gamemap.NationID
	ProvinceID    gamemap.ProvinceID // "" while dislodged
	Type          UnitType
	Coast         gamemap.CoastID // "" for armies
	DislodgedFrom gamemap.ProvinceID
}

// Dislodged reports whether the unit was forced out of its province and owes
// a retreat order.
func (u Unit) Dislodged() bool {
	return u.DislodgedFrom != ""
}
