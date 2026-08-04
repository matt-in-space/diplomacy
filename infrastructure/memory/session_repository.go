package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/core/game"
)

// SessionRepository is an interim repository for development before
// sessions are stored in a database. Session has no reference-type fields
// (Token and PlayerID are strings, CreatedAt/ExpiresAt are time.Time), so
// unlike GameRepository/PlayerRepository a plain struct copy is already a
// safe, detached snapshot — no cloning needed at the repository boundary.
type SessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]auth.Session
}

func NewSessionRepository() *SessionRepository {
	return &SessionRepository{
		sessions: make(map[string]auth.Session),
	}
}

func (r *SessionRepository) CreateSession(ctx context.Context, session *auth.Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if session == nil {
		return errors.New("session is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.sessions[session.Token]; ok {
		return auth.ErrSessionAlreadyExists
	}

	r.sessions[session.Token] = *session
	return nil
}

func (r *SessionRepository) GetSession(ctx context.Context, token string) (*auth.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	session, ok := r.sessions[token]
	if !ok || session.ExpiresAt.Before(time.Now()) {
		return nil, auth.ErrSessionNotFound
	}
	return &session, nil
}

func (r *SessionRepository) DeleteSession(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessions, token)
	return nil
}

func (r *SessionRepository) DeleteSessionsForPlayer(ctx context.Context, id game.PlayerID) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for token, session := range r.sessions {
		if session.PlayerID == id {
			delete(r.sessions, token)
		}
	}
	return nil
}
