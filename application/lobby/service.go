package lobby

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

var (
	ErrNotHost          = errors.New("not the host")
	ErrGameSetupNotOpen = errors.New("game setup is not open")
)

// Service implements the game-setup lobby: creating a setup, joining it via
// its shared invite code, starting, and cancelling. It calls into
// gameplay.GameplayService for the one real use case it needs from that
// package — actually creating the core/game.Game at kickoff — rather than
// constructing games itself: lobby decides who/when, gameplay still owns
// how a game actually gets created and persisted.
type Service struct {
	setups   GameSetupRepository
	games    gameplay.GameRepository
	maps     gameplay.GameMapRepository
	gameplay *gameplay.GameplayService
}

func NewService(setups GameSetupRepository, games gameplay.GameRepository, maps gameplay.GameMapRepository, gp *gameplay.GameplayService) *Service {
	return &Service{
		setups:   setups,
		games:    games,
		maps:     maps,
		gameplay: gp,
	}
}

// StatusFor computes a GameSetup's status. It's never stored: Cancelled is
// read straight off CancelledAt, and Active vs. Pending is answered by
// checking whether a core/game.Game already exists for this setup's ID —
// the same "don't duplicate derivable state" reasoning already applied to
// AdjustmentBalance and LegalRetreats elsewhere in this codebase.
func (s *Service) StatusFor(ctx context.Context, setup *GameSetup) (Status, error) {
	if setup.CancelledAt != nil {
		return StatusCancelled, nil
	}
	if _, err := s.games.GetGame(ctx, setup.ID); err != nil {
		if errors.Is(err, gameplay.ErrGameNotFound) {
			return StatusPending, nil
		}
		return "", err
	}
	return StatusActive, nil
}

// CreateGameSetup starts a new lobby for the given map, hosted by hostID.
// The host is seeded into PlayerIDs immediately — they don't join via their
// own invite link, and count toward capacity and start-readiness like
// anyone else who joins.
func (s *Service) CreateGameSetup(ctx context.Context, hostID game.PlayerID, mapID gamemap.MapID) (*GameSetup, error) {
	if _, err := s.maps.GetMap(mapID); err != nil {
		return nil, fmt.Errorf("failed to get game map %q: %w", mapID, err)
	}

	id, err := newGameID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate game setup ID: %w", err)
	}
	code, err := newInviteCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", err)
	}

	setup := &GameSetup{
		ID:         id,
		MapID:      mapID,
		HostID:     hostID,
		InviteCode: code,
		PlayerIDs:  []game.PlayerID{hostID},
		CreatedAt:  time.Now(),
	}
	if err := s.setups.CreateGameSetup(ctx, setup); err != nil {
		return nil, err
	}
	return setup, nil
}

// JoinGameSetup adds playerID to the lobby behind code. It's idempotent —
// joining twice, or the host using their own link, succeeds and changes
// nothing — and capacity-checked against the map's nation count, so a
// forwarded link can't overfill a lobby past what StartGame could ever
// assign.
func (s *Service) JoinGameSetup(ctx context.Context, code string, playerID game.PlayerID) (*GameSetup, error) {
	setup, err := s.setups.GetGameSetupByInviteCode(ctx, code)
	if err != nil {
		return nil, err
	}

	status, err := s.StatusFor(ctx, setup)
	if err != nil {
		return nil, err
	}
	if status != StatusPending {
		return nil, fmt.Errorf("%w: %s", ErrGameSetupNotOpen, status)
	}

	gm, err := s.maps.GetMap(setup.MapID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game map %q: %w", setup.MapID, err)
	}

	if err := s.setups.AddPlayerToGameSetup(ctx, setup.ID, playerID, len(gm.Nations)); err != nil {
		return nil, err
	}
	return s.setups.GetGameSetup(ctx, setup.ID)
}

// StartGame kicks a setup off: only the host may start. Everyone currently
// in PlayerIDs (the host included, seeded at creation) is randomly shuffled
// onto the map's nations — capacity is already enforced at join time, so
// there are never more players than nations to assign. The real
// core/game.Game is created by delegating to
// gameplay.GameplayService.CreateGame.
func (s *Service) StartGame(ctx context.Context, gameID game.GameID, requesterID game.PlayerID) error {
	setup, status, err := s.loadHostedSetup(ctx, gameID, requesterID)
	if err != nil {
		return err
	}
	if status != StatusPending {
		return fmt.Errorf("%w: %s", ErrGameSetupNotOpen, status)
	}

	gm, err := s.maps.GetMap(setup.MapID)
	if err != nil {
		return fmt.Errorf("failed to get game map %q: %w", setup.MapID, err)
	}

	nations := slices.Clone(gm.Nations)
	players := slices.Clone(setup.PlayerIDs)
	rand.Shuffle(len(nations), func(i, j int) { nations[i], nations[j] = nations[j], nations[i] })
	rand.Shuffle(len(players), func(i, j int) { players[i], players[j] = players[j], players[i] })

	assignments := make(map[gamemap.NationID]game.PlayerID, min(len(nations), len(players)))
	for i, playerID := range players {
		if i >= len(nations) {
			break // unreachable in practice — capacity is enforced at join time
		}
		assignments[nations[i]] = playerID
	}

	_, err = s.gameplay.CreateGame(ctx, setup.ID, setup.MapID, assignments)
	return err
}

// CancelGameSetup ends a lobby before it starts, so a stalled setup with
// too few joiners isn't permanently stuck. Only the host may cancel.
// Cancelling an already-cancelled setup is a no-op, not an error; an
// already-active setup can't be cancelled this way (that's the separate,
// not-yet-built EndGame administrative action for a running game).
func (s *Service) CancelGameSetup(ctx context.Context, gameID game.GameID, requesterID game.PlayerID) error {
	setup, status, err := s.loadHostedSetup(ctx, gameID, requesterID)
	if err != nil {
		return err
	}
	switch status {
	case StatusCancelled:
		return nil
	case StatusActive:
		return fmt.Errorf("%w: game setup is already active", ErrGameSetupNotOpen)
	}

	now := time.Now()
	setup.CancelledAt = &now
	return s.setups.SaveGameSetup(ctx, setup)
}

// loadHostedSetup loads a setup, verifies requesterID is its host, and
// computes its current status in one step — the shared prelude for every
// host-only action.
func (s *Service) loadHostedSetup(ctx context.Context, gameID game.GameID, requesterID game.PlayerID) (*GameSetup, Status, error) {
	setup, err := s.setups.GetGameSetup(ctx, gameID)
	if err != nil {
		return nil, "", err
	}
	if setup.HostID != requesterID {
		return nil, "", ErrNotHost
	}
	status, err := s.StatusFor(ctx, setup)
	if err != nil {
		return nil, "", err
	}
	return setup, status, nil
}
