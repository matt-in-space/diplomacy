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
	CreatePlayer(ctx context.Context, player *game.Player) error
	GetPlayer(ctx context.Context, id game.PlayerID) (*game.Player, error)
	SavePlayer(ctx context.Context, player *game.Player) error
}
