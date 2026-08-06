package lobby

import (
	"context"
	"errors"

	"github.com/matt-in-space/diplomacy/core/game"
)

var (
	ErrGameSetupNotFound      = errors.New("game setup not found")
	ErrGameSetupAlreadyExists = errors.New("game setup already exists")
	ErrGameSetupFull          = errors.New("game setup is full")
)

type GameSetupRepository interface {
	CreateGameSetup(ctx context.Context, setup *GameSetup) error
	GetGameSetup(ctx context.Context, id game.GameID) (*GameSetup, error)
	// GetGameSetupByInviteCode resolves the shared join code to its setup.
	// An unknown code returns ErrGameSetupNotFound, the same sentinel as an
	// unknown ID — a bad credential and a missing resource should be
	// indistinguishable to the caller.
	GetGameSetupByInviteCode(ctx context.Context, code string) (*GameSetup, error)
	SaveGameSetup(ctx context.Context, setup *GameSetup) error
	// AddPlayerToGameSetup atomically appends playerID to the setup's
	// PlayerIDs: a no-op if playerID is already present, ErrGameSetupFull
	// if PlayerIDs already holds capacity entries. It exists as its own
	// method rather than going through SaveGameSetup specifically so a join
	// is one atomic operation instead of a read-modify-write — two people
	// using the link within milliseconds of each other, or both racing a
	// capacity check, must not be able to lose or overrun a slot.
	AddPlayerToGameSetup(ctx context.Context, gameID game.GameID, playerID game.PlayerID, capacity int) error
	// ListGameSetupsForPlayer returns every setup playerID belongs to,
	// hosted or joined — the host is always present in PlayerIDs, so one
	// membership scan answers both.
	ListGameSetupsForPlayer(ctx context.Context, playerID game.PlayerID) ([]*GameSetup, error)
}
