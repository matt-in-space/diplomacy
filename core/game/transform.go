package game

import "github.com/matt-in-space/diplomacy/core/gamemap"

type MovementResultType string

const (
	MovementResultMove    MovementResultType = "move"
	MovementResultHold    MovementResultType = "hold"
	MovementResultRetreat MovementResultType = "retreat"
)

// MovementResult details the unit's final position and type.
type MovementResult struct {
	UnitID UnitID
	Type   MovementResultType
	From   gamemap.ProvinceID
	To     gamemap.ProvinceID
	Coast  gamemap.CoastID
}
