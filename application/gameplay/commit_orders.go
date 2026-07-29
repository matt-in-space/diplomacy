package gameplay

import (
	"context"
	"errors"
	"fmt"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

type CommitOrdersCommand struct {
	GameID          game.GameID
	PlayerID        game.PlayerID
	ExpectedVersion uint64
	NationID        gamemap.NationID
}

func NewCommitOrdersCommand(gameID game.GameID, playerID game.PlayerID, expectedVersion uint64, nationID gamemap.NationID) (CommitOrdersCommand, error) {
	if gameID == "" {
		return CommitOrdersCommand{}, errors.New("game ID is required")
	}
	if playerID == "" {
		return CommitOrdersCommand{}, errors.New("player ID is required")
	}
	if nationID == "" {
		return CommitOrdersCommand{}, errors.New("nation ID is required")
	}
	return CommitOrdersCommand{
		GameID:          gameID,
		PlayerID:        playerID,
		ExpectedVersion: expectedVersion,
		NationID:        nationID,
	}, nil
}

func (s *GameplayService) CommitOrders(ctx context.Context, cmd CommitOrdersCommand) error {
	stored, err := s.games.GetGame(ctx, cmd.GameID)
	if err != nil {
		return err
	}

	if !stored.Game.PlayerControlsNation(cmd.PlayerID, cmd.NationID) {
		return ErrUnauthorized
	}

	gameMap, err := s.maps.GetMap(stored.Game.MapID)
	if err != nil {
		return fmt.Errorf("failed to get game map %q: %w", stored.Game.MapID, err)
	}

	if err := stored.Game.CommitOrders(cmd.NationID, gameMap); err != nil {
		return fmt.Errorf("failed to commit orders: %w", err)
	}

	_, err = s.games.SaveGame(ctx, stored.Game, cmd.ExpectedVersion)
	if err != nil {
		return fmt.Errorf("failed to save game: %w", err)
	}

	return s.ProcessGame(ctx, stored.Game.ID)
}
