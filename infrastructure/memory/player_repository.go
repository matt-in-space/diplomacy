package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/game"
)

type PlayerRepository struct {
	mu      sync.RWMutex
	players map[game.PlayerID]game.Player
}

func NewPlayerRepository() *PlayerRepository {
	return &PlayerRepository{
		players: make(map[game.PlayerID]game.Player),
	}
}

func (r *PlayerRepository) CreatePlayer(ctx context.Context, player *game.Player) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if player == nil {
		return errors.New("player is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[player.ID]; ok {
		return fmt.Errorf("%w: %q", gameplay.ErrPlayerAlreadyExists, player.ID)
	}

	r.players[player.ID] = *player
	return nil
}

func (r *PlayerRepository) GetPlayer(ctx context.Context, id game.PlayerID) (*game.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	player, ok := r.players[id]
	if !ok {
		return nil, fmt.Errorf("%w: %q", gameplay.ErrPlayerNotFound, id)
	}

	return &player, nil
}

func (r *PlayerRepository) SavePlayer(ctx context.Context, player *game.Player) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if player == nil {
		return errors.New("player is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[player.ID]; !ok {
		return fmt.Errorf("%w: %q", gameplay.ErrPlayerNotFound, player.ID)
	}

	r.players[player.ID] = *player
	return nil
}
