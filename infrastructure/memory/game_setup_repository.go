package memory

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
)

// GameSetupRepository is an interim repository for development before game
// setups are stored in a database. Like PlayerRepository it clones on the
// way in and out so callers can't bypass Save/mutate stored state by
// holding onto a pointer — CancelledAt *time.Time and PlayerIDs
// []game.PlayerID are the two reference-type fields whose contents need
// copying too, not just the field itself.
type GameSetupRepository struct {
	mu           sync.RWMutex
	setups       map[game.GameID]lobby.GameSetup
	byInviteCode map[string]game.GameID
}

func NewGameSetupRepository() *GameSetupRepository {
	return &GameSetupRepository{
		setups:       make(map[game.GameID]lobby.GameSetup),
		byInviteCode: make(map[string]game.GameID),
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
	r.byInviteCode[setup.InviteCode] = setup.ID
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

func (r *GameSetupRepository) GetGameSetupByInviteCode(ctx context.Context, code string) (*lobby.GameSetup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byInviteCode[code]
	if !ok {
		return nil, fmt.Errorf("%w: invite code %q", lobby.ErrGameSetupNotFound, code)
	}
	setup := detachedGameSetup(r.setups[id])
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

	existing, ok := r.setups[setup.ID]
	if !ok {
		return fmt.Errorf("%w: %q", lobby.ErrGameSetupNotFound, setup.ID)
	}
	if existing.InviteCode != setup.InviteCode {
		delete(r.byInviteCode, existing.InviteCode)
		r.byInviteCode[setup.InviteCode] = setup.ID
	}

	r.setups[setup.ID] = detachedGameSetup(*setup)
	return nil
}

// AddPlayerToGameSetup is the one write path that isn't a plain
// Get-then-Save: it appends under a single lock acquisition so a join is
// atomic. Two callers racing to join the same setup can't both observe
// room for one more and both get in, and neither can silently overwrite
// the other's join the way a read-modify-write through SaveGameSetup could.
func (r *GameSetupRepository) AddPlayerToGameSetup(ctx context.Context, gameID game.GameID, playerID game.PlayerID, capacity int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	setup, ok := r.setups[gameID]
	if !ok {
		return fmt.Errorf("%w: %q", lobby.ErrGameSetupNotFound, gameID)
	}
	if slices.Contains(setup.PlayerIDs, playerID) {
		return nil
	}
	if len(setup.PlayerIDs) >= capacity {
		return fmt.Errorf("%w: %q", lobby.ErrGameSetupFull, gameID)
	}

	setup.PlayerIDs = append(slices.Clone(setup.PlayerIDs), playerID)
	r.setups[gameID] = setup
	return nil
}

func (r *GameSetupRepository) ListGameSetupsForPlayer(ctx context.Context, playerID game.PlayerID) ([]*lobby.GameSetup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var setups []*lobby.GameSetup
	for _, setup := range r.setups {
		if !slices.Contains(setup.PlayerIDs, playerID) {
			continue
		}
		setup = detachedGameSetup(setup)
		setups = append(setups, &setup)
	}

	// Map iteration order is randomized — without an explicit sort, callers
	// would see this list reshuffle on every call. Newest first.
	slices.SortFunc(setups, func(a, b *lobby.GameSetup) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return setups, nil
}

// detachedGameSetup returns a copy of setup safe to store or return without
// aliasing the caller's CancelledAt or PlayerIDs backing values.
func detachedGameSetup(setup lobby.GameSetup) lobby.GameSetup {
	if setup.CancelledAt != nil {
		cancelledAt := *setup.CancelledAt
		setup.CancelledAt = &cancelledAt
	}
	setup.PlayerIDs = slices.Clone(setup.PlayerIDs)
	return setup
}
