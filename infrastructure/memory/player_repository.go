package memory

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/core/game"
)

// PlayerRepository is an interim repository for development before players
// are stored in a database. Like GameRepository it clones on the way in and
// out so callers can't bypass Save/mutate stored state by holding onto a
// pointer or slice header — the one new wrinkle here is PasswordHash, a
// []byte field, whose slice header would otherwise be shared between the
// caller and the stored copy.
type PlayerRepository struct {
	mu      sync.RWMutex
	players map[game.PlayerID]auth.Player
	byEmail map[string]game.PlayerID
}

func NewPlayerRepository() *PlayerRepository {
	return &PlayerRepository{
		players: make(map[game.PlayerID]auth.Player),
		byEmail: make(map[string]game.PlayerID),
	}
}

func (r *PlayerRepository) CreatePlayer(ctx context.Context, player *auth.Player) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if player == nil {
		return errors.New("player is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[player.ID]; ok {
		return fmt.Errorf("%w: %q", auth.ErrPlayerAlreadyExists, player.ID)
	}
	if _, ok := r.byEmail[player.Email]; ok {
		return fmt.Errorf("%w: email %q already registered", auth.ErrPlayerAlreadyExists, player.Email)
	}

	r.players[player.ID] = detachedPlayer(*player)
	r.byEmail[player.Email] = player.ID
	return nil
}

func (r *PlayerRepository) GetPlayer(ctx context.Context, id game.PlayerID) (*auth.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	player, ok := r.players[id]
	if !ok {
		return nil, fmt.Errorf("%w: %q", auth.ErrPlayerNotFound, id)
	}
	player = detachedPlayer(player)
	return &player, nil
}

func (r *PlayerRepository) GetPlayerByEmail(ctx context.Context, email string) (*auth.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("%w: email %q", auth.ErrPlayerNotFound, email)
	}
	player := detachedPlayer(r.players[id])
	return &player, nil
}

func (r *PlayerRepository) SavePlayer(ctx context.Context, player *auth.Player) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if player == nil {
		return errors.New("player is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.players[player.ID]
	if !ok {
		return fmt.Errorf("%w: %q", auth.ErrPlayerNotFound, player.ID)
	}
	if existing.Email != player.Email {
		delete(r.byEmail, existing.Email)
		r.byEmail[player.Email] = player.ID
	}

	r.players[player.ID] = detachedPlayer(*player)
	return nil
}

// detachedPlayer returns a copy of player safe to store or return without
// aliasing the caller's PasswordHash backing array.
func detachedPlayer(player auth.Player) auth.Player {
	player.PasswordHash = bytes.Clone(player.PasswordHash)
	return player
}
