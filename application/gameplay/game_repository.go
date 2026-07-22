package gameplay

import (
	"context"
	"errors"

	"github.com/matt-in-space/diplomacy/core/game"
)

var (
	ErrGameNotFound      = errors.New("game not found")
	ErrGameAlreadyExists = errors.New("game already exists")
	ErrConcurrentUpdate  = errors.New("concurrent game update")
)

type StoredGame struct {
	Game    *game.Game
	Version uint64
}

type GameRepository interface {
	Create(ctx context.Context, g *game.Game) error
	Get(ctx context.Context, gameID game.GameID) (StoredGame, error)
	Save(ctx context.Context, g *game.Game, expectedVersion uint64) (uint64, error)
}
