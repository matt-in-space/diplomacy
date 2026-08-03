package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/game"
)

// GameRepository is an interim repository for development before games are
// stored in a database. It clones games at repository boundaries to mimic
// detached database records: callers can mutate a loaded game without
// changing stored state until Save succeeds.
type GameRepository struct {
	mu    sync.RWMutex
	games map[game.GameID]gameplay.StoredGame
}

func NewGameRepository() *GameRepository {
	return &GameRepository{
		games: make(map[game.GameID]gameplay.StoredGame),
	}
}

func (r *GameRepository) CreateGame(ctx context.Context, g *game.Game) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if g == nil {
		return errors.New("game is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.games[g.ID]; ok {
		return fmt.Errorf("%w: %q", gameplay.ErrGameAlreadyExists, g.ID)
	}

	r.games[g.ID] = gameplay.StoredGame{
		Game:    g.Clone(),
		Version: 0,
	}
	return nil
}

func (r *GameRepository) GetGame(ctx context.Context, gameID game.GameID) (gameplay.StoredGame, error) {
	if err := ctx.Err(); err != nil {
		return gameplay.StoredGame{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	stored, ok := r.games[gameID]
	if !ok {
		return gameplay.StoredGame{}, fmt.Errorf("%w: %q", gameplay.ErrGameNotFound, gameID)
	}
	// Return a detached snapshot so changes cannot bypass Save and its version check.
	stored.Game = stored.Game.Clone()
	return stored, nil
}

func (r *GameRepository) SaveGame(ctx context.Context, g *game.Game, expectedVersion uint64) (uint64, error) {
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
		return 0, fmt.Errorf("%w: %q", gameplay.ErrGameNotFound, g.ID)
	}
	if stored.Version != expectedVersion {
		return 0, fmt.Errorf("%w: game %q has version %d, expected %d", gameplay.ErrConcurrentUpdate, g.ID, stored.Version, expectedVersion)
	}

	version := expectedVersion + 1
	r.games[g.ID] = gameplay.StoredGame{
		Game:    g.Clone(),
		Version: version,
	}
	return version, nil
}
