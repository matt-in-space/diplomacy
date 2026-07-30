package gameplay

import (
	"context"
	"fmt"

	"github.com/matt-in-space/diplomacy/core/adjudicator"
	"github.com/matt-in-space/diplomacy/core/game"
)

func (s *GameplayService) ProcessGame(ctx context.Context, gameID game.GameID) error {
	stored, err := s.games.GetGame(ctx, gameID)
	if err != nil {
		return err
	}

	version := stored.Version

	for {
		progressed, err := s.processGameStep(stored.Game)
		if err != nil {
			return err
		}
		if !progressed {
			break
		}

		next, err := s.games.SaveGame(ctx, stored.Game, version)
		if err != nil {
			return fmt.Errorf("failed to save game: %w", err)
		}
		version = next
	}

	return nil
}

func (s *GameplayService) processGameStep(g *game.Game) (progressed bool, err error) {
	switch g.Turn.Phase {
	case game.AcceptOrders:
		return g.BeginOrderResolution()

	case game.ResolveOrders:
		gm, err := s.maps.GetMap(g.MapID)
		if err != nil {
			return false, fmt.Errorf("failed to get game map %q: %w", g.MapID, err)
		}

		res, err := adjudicator.Resolve(g, gm)
		if err != nil {
			return false, fmt.Errorf("failed to resolve game: %w", err)
		}

		if err := g.CompleteOrderResolution(res); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}
