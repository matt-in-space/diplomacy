package gameplay

import (
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

type MemoryGameMapRepository struct {
	maps map[gamemap.MapID]*gamemap.GameMap
}

func NewMemoryGameMapRepository(maps ...*gamemap.GameMap) *MemoryGameMapRepository {
	r := &MemoryGameMapRepository{
		maps: make(map[gamemap.MapID]*gamemap.GameMap),
	}

	for _, m := range maps {
		r.maps[m.ID] = m
	}

	return r
}

func (r *MemoryGameMapRepository) GetMap(mapID gamemap.MapID) (*gamemap.GameMap, error) {
	m, ok := r.maps[mapID]
	if !ok {
		return nil, ErrMapNotFound
	}
	return m, nil
}
