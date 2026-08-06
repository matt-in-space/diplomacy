package lobby

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/matt-in-space/diplomacy/application/gameplay"
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

var (
	ErrNotHost               = errors.New("not the host")
	ErrGameSetupNotOpen      = errors.New("game setup is not open")
	ErrInvitesPending        = errors.New("invites still pending")
	ErrInviteAlreadyResolved = errors.New("invite already responded to")
)

// Service implements the game-setup lobby: creating a setup, inviting
// players, accepting/declining, starting, and cancelling. It calls into
// gameplay.GameplayService for the one real use case it needs from that
// package — actually creating the core/game.Game at kickoff — rather than
// constructing games itself: lobby decides who/when, gameplay still owns
// how a game actually gets created and persisted.
type Service struct {
	setups   GameSetupRepository
	invites  InviteRepository
	games    gameplay.GameRepository
	maps     gameplay.GameMapRepository
	gameplay *gameplay.GameplayService
}

func NewService(setups GameSetupRepository, invites InviteRepository, games gameplay.GameRepository, maps gameplay.GameMapRepository, gp *gameplay.GameplayService) *Service {
	return &Service{
		setups:   setups,
		invites:  invites,
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
// The host is a participant by construction — they don't invite
// themselves, and count as accepted for start-readiness.
func (s *Service) CreateGameSetup(ctx context.Context, hostID game.PlayerID, mapID gamemap.MapID) (*GameSetup, error) {
	if _, err := s.maps.GetMap(mapID); err != nil {
		return nil, fmt.Errorf("failed to get game map %q: %w", mapID, err)
	}

	id, err := newGameID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate game setup ID: %w", err)
	}

	setup := &GameSetup{
		ID:        id,
		MapID:     mapID,
		HostID:    hostID,
		CreatedAt: time.Now(),
	}
	if err := s.setups.CreateGameSetup(ctx, setup); err != nil {
		return nil, err
	}
	return setup, nil
}

// InvitePlayer adds an invite for email to gameID's lobby. Only the host
// may invite. It's idempotent: a Pending or Accepted invite already on file
// for that email is returned as-is rather than duplicated. A previously
// Declined invite for the same email gets a fresh one — the host
// re-inviting someone who said no is a reasonable action, not an error.
func (s *Service) InvitePlayer(ctx context.Context, gameID game.GameID, requesterID game.PlayerID, email string) (*Invite, error) {
	setup, status, err := s.loadHostedSetup(ctx, gameID, requesterID)
	if err != nil {
		return nil, err
	}
	if status != StatusPending {
		return nil, fmt.Errorf("%w: %s", ErrGameSetupNotOpen, status)
	}

	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, errors.New("a valid email is required")
	}

	existing, err := s.invites.ListInvitesForGame(ctx, setup.ID)
	if err != nil {
		return nil, err
	}
	for _, invite := range existing {
		if invite.Email == email && invite.Status != InviteDeclined {
			return invite, nil
		}
	}

	code, err := newInviteCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", err)
	}

	invite := &Invite{
		Code:      code,
		GameID:    setup.ID,
		Email:     email,
		Status:    InvitePending,
		CreatedAt: time.Now(),
	}
	if err := s.invites.CreateInvite(ctx, invite); err != nil {
		return nil, err
	}
	return invite, nil
}

// AcceptInvite records playerID as accepting the invite behind code.
func (s *Service) AcceptInvite(ctx context.Context, code string, playerID game.PlayerID) error {
	return s.respondToInvite(ctx, code, playerID, InviteAccepted)
}

// DeclineInvite records playerID as declining the invite behind code. The
// host can always send a fresh invite to the same email afterward — see
// InvitePlayer.
func (s *Service) DeclineInvite(ctx context.Context, code string, playerID game.PlayerID) error {
	return s.respondToInvite(ctx, code, playerID, InviteDeclined)
}

func (s *Service) respondToInvite(ctx context.Context, code string, playerID game.PlayerID, status InviteStatus) error {
	invite, err := s.invites.GetInviteByCode(ctx, code)
	if err != nil {
		return err
	}
	if invite.Status != InvitePending {
		return fmt.Errorf("%w: %s", ErrInviteAlreadyResolved, invite.Status)
	}

	invite.Status = status
	invite.PlayerID = playerID
	invite.RespondedAt = time.Now()
	return s.invites.SaveInvite(ctx, invite)
}

// StartGame kicks a setup off: only the host may start, and only once zero
// invites remain Pending (a decline just sits there — the host is expected
// to send a replacement invite rather than the system tracking a minimum
// headcount). Accepted players plus the host are randomly shuffled onto the
// map's nations; more accepted players than nations simply leaves the
// extras unassigned rather than erroring — the host over-invited, that's
// their call to make. The real core/game.Game is created by delegating to
// gameplay.GameplayService.CreateGame.
func (s *Service) StartGame(ctx context.Context, gameID game.GameID, requesterID game.PlayerID) error {
	setup, status, err := s.loadHostedSetup(ctx, gameID, requesterID)
	if err != nil {
		return err
	}
	if status != StatusPending {
		return fmt.Errorf("%w: %s", ErrGameSetupNotOpen, status)
	}

	invites, err := s.invites.ListInvitesForGame(ctx, setup.ID)
	if err != nil {
		return err
	}

	players := []game.PlayerID{setup.HostID}
	for _, invite := range invites {
		switch invite.Status {
		case InvitePending:
			return fmt.Errorf("%w: %s", ErrInvitesPending, invite.Email)
		case InviteAccepted:
			players = append(players, invite.PlayerID)
		}
	}

	gm, err := s.maps.GetMap(setup.MapID)
	if err != nil {
		return fmt.Errorf("failed to get game map %q: %w", setup.MapID, err)
	}

	nations := slices.Clone(gm.Nations)
	rand.Shuffle(len(nations), func(i, j int) { nations[i], nations[j] = nations[j], nations[i] })
	rand.Shuffle(len(players), func(i, j int) { players[i], players[j] = players[j], players[i] })

	assignments := make(map[gamemap.NationID]game.PlayerID, min(len(nations), len(players)))
	for i, playerID := range players {
		if i >= len(nations) {
			break // more accepted players than nations — host over-invited, extras go unassigned
		}
		assignments[nations[i]] = playerID
	}

	_, err = s.gameplay.CreateGame(ctx, setup.ID, setup.MapID, assignments)
	return err
}

// CancelGameSetup ends a lobby before it starts, so a stalled setup with
// unresponsive invitees isn't permanently stuck. Only the host may cancel.
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
