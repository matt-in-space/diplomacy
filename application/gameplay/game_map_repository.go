package gameplay

import (
	"errors"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

var ErrMapNotFound = errors.New("map not found")

type GameMapRepository interface {
	GetMap(mapID gamemap.MapID) (*gamemap.GameMap, error)
}
