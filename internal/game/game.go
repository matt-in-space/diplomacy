package game

import "github.com/matt-in-space/diplomacy/internal/gamemap"

type GameID string
type PlayerID string

type Game struct {
	ID          GameID
	Assignments map[gamemap.NationID]PlayerID
	Turn        Turn
	Units       map[UnitID]Unit
	Positions   map[gamemap.ProvinceID]UnitID
	FleetCoasts map[UnitID]gamemap.CoastID
	Orders      map[UnitID]Order
}
