package game

type Order interface {
	Unit() UnitID
}

type HoldOrder struct {
	UnitID UnitID
}

func (h HoldOrder) Unit() UnitID {
	return h.UnitID
}
