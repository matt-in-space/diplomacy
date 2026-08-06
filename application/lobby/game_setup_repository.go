package lobby

import (
	"context"
	"errors"

	"github.com/matt-in-space/diplomacy/core/game"
)

var (
	ErrGameSetupNotFound      = errors.New("game setup not found")
	ErrGameSetupAlreadyExists = errors.New("game setup already exists")
)

type GameSetupRepository interface {
	CreateGameSetup(ctx context.Context, setup *GameSetup) error
	GetGameSetup(ctx context.Context, id game.GameID) (*GameSetup, error)
	SaveGameSetup(ctx context.Context, setup *GameSetup) error
	// ListGameSetupsForPlayer returns the setups hosted by the given
	// player. It does not also include setups the player has been
	// invited into and accepted — that's a presentation-layer
	// composition for the future /games list, not a repository concern.
	ListGameSetupsForPlayer(ctx context.Context, hostID game.PlayerID) ([]*GameSetup, error)
}
