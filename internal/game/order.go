package game

import "github.com/matt-in-space/diplomacy/internal/gamemap"

type Order interface {
	Unit() UnitID
	Nation() gamemap.NationID
}

type HoldOrder struct {
	UnitID   UnitID
	NationID gamemap.NationID
}

func (h HoldOrder) Unit() UnitID {
	return h.UnitID
}

func (h HoldOrder) Nation() gamemap.NationID {
	return h.NationID
}
