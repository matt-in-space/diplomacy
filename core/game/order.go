package game

import "github.com/matt-in-space/diplomacy/core/gamemap"

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
