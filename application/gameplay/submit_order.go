package gameplay

import (
	"context"
	"errors"
	"fmt"

	"github.com/matt-in-space/diplomacy/core/game"
)

type SubmitOrderCommand struct {
	GameID          game.GameID
	PlayerID        game.PlayerID
	ExpectedVersion uint64
	Order           game.Order
}

// NewSubmitOrderCommand validates and creates a new SubmitOrderCommand.
func NewSubmitOrderCommand(gameID game.GameID, playerID game.PlayerID, expectedVersion uint64, order game.Order) (*SubmitOrderCommand, error) {
	if gameID == "" {
		return nil, errors.New("game ID is required")
	}
	if playerID == "" {
		return nil, errors.New("player ID is required")
	}
	if order == nil {
		return nil, errors.New("order is required")
	}
	if expectedVersion == 0 { // Assuming 0 is an invalid version for expectedVersion
		return nil, errors.New("expected version is required")
	}

	return &SubmitOrderCommand{
		GameID:          gameID,
		PlayerID:        playerID,
		ExpectedVersion: expectedVersion,
		Order:           order,
	}, nil
}

func (s *GameplayService) SubmitOrder(ctx context.Context, cmd SubmitOrderCommand) error {
	stored, err := s.games.GetGame(ctx, cmd.GameID)
	if err != nil {
		// games.GetGame already returns specific errors like ErrGameNotFound
		return err
	}

	if !stored.Game.PlayerControlsNation(cmd.PlayerID, cmd.Order.Nation()) {
		return ErrUnauthorized
	}

	gameMap, err := s.maps.GetMap(stored.Game.MapID)
	if err != nil {
		// maps.GetMap should return a specific error if the map is not found
		return fmt.Errorf("failed to get game map %q: %w", stored.Game.MapID, err)
	}

	if err := stored.Game.SubmitOrder(cmd.Order, gameMap); err != nil {
		return fmt.Errorf("failed to submit order: %w", err)
	}

	_, err = s.games.SaveGame(ctx, stored.Game, cmd.ExpectedVersion)
	if err != nil {
		return fmt.Errorf("failed to save game: %w", err)
	}

	return nil
}
