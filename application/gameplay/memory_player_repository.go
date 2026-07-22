package gameplay

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/matt-in-space/diplomacy/core/game"
)

type MemoryPlayerRepository struct {
	mu      sync.RWMutex
	players map[game.PlayerID]game.Player
}

func NewMemoryPlayerRepository() *MemoryPlayerRepository {
	return &MemoryPlayerRepository{
		players: make(map[game.PlayerID]game.Player),
	}
}

func (r *MemoryPlayerRepository) CreatePlayer(ctx context.Context, player *game.Player) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if player == nil {
		return errors.New("player is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[player.ID]; ok {
		return fmt.Errorf("%w: %q", ErrPlayerAlreadyExists, player.ID)
	}

	r.players[player.ID] = *player
	return nil
}

func (r *MemoryPlayerRepository) GetPlayer(ctx context.Context, id game.PlayerID) (*game.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	player, ok := r.players[id]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrPlayerNotFound, id)
	}

	return &player, nil
}

func (r *MemoryPlayerRepository) SavePlayer(ctx context.Context, player *game.Player) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if player == nil {
		return errors.New("player is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[player.ID]; !ok {
		return fmt.Errorf("%w: %q", ErrPlayerNotFound, player.ID)
	}

	r.players[player.ID] = *player
	return nil
}
