package memory

import (
	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

type GameMapRepository struct {
	maps map[gamemap.MapID]*gamemap.GameMap
}

func NewGameMapRepository(maps ...*gamemap.GameMap) *GameMapRepository {
	r := &GameMapRepository{
		maps: make(map[gamemap.MapID]*gamemap.GameMap),
	}

	for _, m := range maps {
		r.maps[m.ID] = m
	}

	return r
}

func (r *GameMapRepository) GetMap(mapID gamemap.MapID) (*gamemap.GameMap, error) {
	m, ok := r.maps[mapID]
	if !ok {
		return nil, gameplay.ErrMapNotFound
	}
	return m, nil
}
