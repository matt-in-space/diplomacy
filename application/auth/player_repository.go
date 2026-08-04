package auth

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
	CreatePlayer(ctx context.Context, player *Player) error
	GetPlayer(ctx context.Context, id game.PlayerID) (*Player, error)
	GetPlayerByEmail(ctx context.Context, email string) (*Player, error)
	SavePlayer(ctx context.Context, player *Player) error
}
