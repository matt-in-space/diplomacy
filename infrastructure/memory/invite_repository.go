package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/game"
)

// InviteRepository is an interim repository for development before invites
// are stored in a database. Invite has no reference-type fields (Code,
// GameID, Email, PlayerID are strings/typed strings, CreatedAt/RespondedAt
// are time.Time), so — like SessionRepository — a plain struct copy is
// already a safe, detached snapshot; no cloning helper needed.
type InviteRepository struct {
	mu      sync.RWMutex
	invites map[string]lobby.Invite
}

func NewInviteRepository() *InviteRepository {
	return &InviteRepository{
		invites: make(map[string]lobby.Invite),
	}
}

func (r *InviteRepository) CreateInvite(ctx context.Context, invite *lobby.Invite) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if invite == nil {
		return errors.New("invite is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.invites[invite.Code]; ok {
		return fmt.Errorf("%w: %q", lobby.ErrInviteAlreadyExists, invite.Code)
	}

	r.invites[invite.Code] = *invite
	return nil
}

func (r *InviteRepository) GetInviteByCode(ctx context.Context, code string) (*lobby.Invite, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	invite, ok := r.invites[code]
	if !ok {
		return nil, fmt.Errorf("%w: %q", lobby.ErrInviteNotFound, code)
	}
	return &invite, nil
}

func (r *InviteRepository) SaveInvite(ctx context.Context, invite *lobby.Invite) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if invite == nil {
		return errors.New("invite is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.invites[invite.Code]; !ok {
		return fmt.Errorf("%w: %q", lobby.ErrInviteNotFound, invite.Code)
	}

	r.invites[invite.Code] = *invite
	return nil
}

func (r *InviteRepository) ListInvitesForGame(ctx context.Context, gameID game.GameID) ([]*lobby.Invite, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var invites []*lobby.Invite
	for _, invite := range r.invites {
		if invite.GameID != gameID {
			continue
		}
		invite := invite
		invites = append(invites, &invite)
	}
	return invites, nil
}

func (r *InviteRepository) ListInvitesForEmail(ctx context.Context, email string) ([]*lobby.Invite, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var invites []*lobby.Invite
	for _, invite := range r.invites {
		if invite.Email != email {
			continue
		}
		invite := invite
		invites = append(invites, &invite)
	}
	return invites, nil
}
