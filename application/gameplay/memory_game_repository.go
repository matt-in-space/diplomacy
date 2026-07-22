package gameplay

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/matt-in-space/diplomacy/core/game"
)

// MemoryGameRepository is an interim repository for development before games
// are stored in a database. It clones games at repository boundaries to mimic
// detached database records: callers can mutate a loaded game without changing
// stored state until Save succeeds.
type MemoryGameRepository struct {
	mu    sync.RWMutex
	games map[game.GameID]StoredGame
}

func NewMemoryGameRepository() *MemoryGameRepository {
	return &MemoryGameRepository{
		games: make(map[game.GameID]StoredGame),
	}
}

func (r *MemoryGameRepository) Create(ctx context.Context, g *game.Game) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if g == nil {
		return errors.New("game is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.games[g.ID]; ok {
		return fmt.Errorf("%w: %q", ErrGameAlreadyExists, g.ID)
	}

	r.games[g.ID] = StoredGame{
		Game:    g.Clone(),
		Version: 0,
	}
	return nil
}

func (r *MemoryGameRepository) Get(ctx context.Context, gameID game.GameID) (StoredGame, error) {
	if err := ctx.Err(); err != nil {
		return StoredGame{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	stored, ok := r.games[gameID]
	if !ok {
		return StoredGame{}, fmt.Errorf("%w: %q", ErrGameNotFound, gameID)
	}
	// Return a detached snapshot so changes cannot bypass Save and its version check.
	stored.Game = stored.Game.Clone()
	return stored, nil
}

func (r *MemoryGameRepository) Save(ctx context.Context, g *game.Game, expectedVersion uint64) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if g == nil {
		return 0, errors.New("game is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.games[g.ID]
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrGameNotFound, g.ID)
	}
	if stored.Version != expectedVersion {
		return 0, fmt.Errorf("%w: game %q has version %d, expected %d", ErrConcurrentUpdate, g.ID, stored.Version, expectedVersion)
	}

	version := expectedVersion + 1
	r.games[g.ID] = StoredGame{
		Game:    g.Clone(),
		Version: version,
	}
	return version, nil
}
