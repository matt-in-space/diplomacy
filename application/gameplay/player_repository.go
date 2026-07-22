package gameplay

import (
	"context"
	"errors"

	"github.com/matt-in-space/diplomacy/core/game"
)

var (
	ErrPlayerNotFound      = errors.New("player not found")
	ErrPlayerAlreadyExists = errors.New("player already exists")
)

type PlayerRepository interface {
	Create(ctx context.Context, player *game.Player) error
	Get(ctx context.Context, id game.PlayerID) (*game.Player, error)
	Save(ctx context.Context, player *game.Player) error
}
