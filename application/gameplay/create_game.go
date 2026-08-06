package gameplay

import (
	"context"
	"fmt"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// CreateGame builds and persists a new Game for the given map and nation
// assignments. It's the first genuine "create a game" use case — until now
// only tests ever called GameRepository.CreateGame directly.
func (s *GameplayService) CreateGame(ctx context.Context, id game.GameID, mapID gamemap.MapID, assignments map[gamemap.NationID]game.PlayerID) (StoredGame, error) {
	gm, err := s.maps.GetMap(mapID)
	if err != nil {
		return StoredGame{}, fmt.Errorf("failed to get game map %q: %w", mapID, err)
	}

	g, err := game.NewGame(game.NewGameConfig{ID: id, Assignments: assignments}, gm)
	if err != nil {
		return StoredGame{}, err
	}

	if err := s.games.CreateGame(ctx, g); err != nil {
		return StoredGame{}, err
	}

	// Re-fetch rather than assume Version: 0 — don't bake in an assumption
	// about the repository's internal versioning scheme.
	return s.games.GetGame(ctx, id)
}
