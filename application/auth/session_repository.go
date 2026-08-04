package auth

import (
	"context"
	"errors"

	"github.com/matt-in-space/diplomacy/core/game"
)

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionAlreadyExists = errors.New("session already exists")
)

type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session) error
	// GetSession returns ErrSessionNotFound for both an unknown token and an
	// expired one — callers never need to remember to check ExpiresAt
	// themselves.
	GetSession(ctx context.Context, token string) (*Session, error)
	// DeleteSession is idempotent: deleting an unknown or already-deleted
	// token is not an error. A logout call for a session that's already
	// gone is a normal case, not a bug.
	DeleteSession(ctx context.Context, token string) error
	// DeleteSessionsForPlayer powers "log out everywhere" and invalidating
	// existing sessions on password change. A player with no sessions is
	// not an error.
	DeleteSessionsForPlayer(ctx context.Context, id game.PlayerID) error
}
