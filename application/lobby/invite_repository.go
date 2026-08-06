package lobby

import (
	"context"
	"errors"

	"github.com/matt-in-space/diplomacy/core/game"
)

var (
	ErrInviteNotFound      = errors.New("invite not found")
	ErrInviteAlreadyExists = errors.New("invite already exists")
)

type InviteRepository interface {
	CreateInvite(ctx context.Context, invite *Invite) error
	GetInviteByCode(ctx context.Context, code string) (*Invite, error)
	SaveInvite(ctx context.Context, invite *Invite) error
	ListInvitesForGame(ctx context.Context, gameID game.GameID) ([]*Invite, error)
	// ListInvitesForEmail returns invites addressed to an email, regardless
	// of who's logged in — powers a future "pending invites for you" view
	// for a player who never received (or lost) the link.
	ListInvitesForEmail(ctx context.Context, email string) ([]*Invite, error)
}
