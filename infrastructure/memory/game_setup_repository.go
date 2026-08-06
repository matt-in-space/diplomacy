package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
)

// GameSetupRepository is an interim repository for development before game
// setups are stored in a database. Like PlayerRepository it clones on the
// way in and out so callers can't bypass Save/mutate stored state by
// holding onto a pointer — the one reference-type field here is
// CancelledAt *time.Time, whose pointee needs copying too, not just the
// pointer itself.
type GameSetupRepository struct {
	mu     sync.RWMutex
	setups map[game.GameID]lobby.GameSetup
}

func NewGameSetupRepository() *GameSetupRepository {
	return &GameSetupRepository{
		setups: make(map[game.GameID]lobby.GameSetup),
	}
}

func (r *GameSetupRepository) CreateGameSetup(ctx context.Context, setup *lobby.GameSetup) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if setup == nil {
		return errors.New("game setup is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.setups[setup.ID]; ok {
		return fmt.Errorf("%w: %q", lobby.ErrGameSetupAlreadyExists, setup.ID)
	}

	r.setups[setup.ID] = detachedGameSetup(*setup)
	return nil
}

func (r *GameSetupRepository) GetGameSetup(ctx context.Context, id game.GameID) (*lobby.GameSetup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	setup, ok := r.setups[id]
	if !ok {
		return nil, fmt.Errorf("%w: %q", lobby.ErrGameSetupNotFound, id)
	}
	setup = detachedGameSetup(setup)
	return &setup, nil
}

func (r *GameSetupRepository) SaveGameSetup(ctx context.Context, setup *lobby.GameSetup) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if setup == nil {
		return errors.New("game setup is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.setups[setup.ID]; !ok {
		return fmt.Errorf("%w: %q", lobby.ErrGameSetupNotFound, setup.ID)
	}

	r.setups[setup.ID] = detachedGameSetup(*setup)
	return nil
}

func (r *GameSetupRepository) ListGameSetupsForPlayer(ctx context.Context, hostID game.PlayerID) ([]*lobby.GameSetup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var setups []*lobby.GameSetup
	for _, setup := range r.setups {
		if setup.HostID != hostID {
			continue
		}
		setup = detachedGameSetup(setup)
		setups = append(setups, &setup)
	}
	return setups, nil
}

// detachedGameSetup returns a copy of setup safe to store or return without
// aliasing the caller's CancelledAt backing value.
func detachedGameSetup(setup lobby.GameSetup) lobby.GameSetup {
	if setup.CancelledAt != nil {
		cancelledAt := *setup.CancelledAt
		setup.CancelledAt = &cancelledAt
	}
	return setup
}
