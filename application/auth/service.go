package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/matt-in-space/diplomacy/core/game"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

const (
	minPasswordLength = 8
	sessionDuration   = 7 * 24 * time.Hour
)

// dummyHash is compared against on a login attempt for an email that isn't
// registered, so the response time doesn't reveal whether the email exists.
var dummyHash = mustHash("this-is-not-a-real-password")

func mustHash(password string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err) // only possible if bcrypt.DefaultCost itself is invalid
	}
	return hash
}

// Service implements signup, login, logout, and session lookup. It hashes
// and verifies passwords and mints session tokens; it never returns or logs
// a plaintext password or a PasswordHash to a caller that didn't already
// have it.
type Service struct {
	players  PlayerRepository
	sessions SessionRepository
}

func NewService(players PlayerRepository, sessions SessionRepository) *Service {
	return &Service{players: players, sessions: sessions}
}

// Signup validates and creates a new Player. Validation here is
// deliberately minimal — email is checked for shape, not deliverability;
// real verification is a future feature (a clicked link in a sent email),
// not something worth approximating with a stricter regex now.
func (s *Service) Signup(ctx context.Context, email, displayName, password string) (*Player, error) {
	email = strings.TrimSpace(email)
	displayName = strings.TrimSpace(displayName)

	if email == "" || !strings.Contains(email, "@") {
		return nil, errors.New("a valid email is required")
	}
	if displayName == "" {
		return nil, errors.New("display name is required")
	}
	if len(password) < minPasswordLength {
		return nil, fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	id, err := newPlayerID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate player ID: %w", err)
	}

	now := time.Now()
	player := &Player{
		ID:                id,
		Email:             email,
		DisplayName:       displayName,
		PasswordHash:      hash,
		CreatedAt:         now,
		PasswordChangedAt: now,
	}
	if err := s.players.CreatePlayer(ctx, player); err != nil {
		return nil, err
	}
	return player, nil
}

// Login verifies credentials and, on success, creates a new Session. An
// unknown email and a wrong password are indistinguishable to the caller —
// both return ErrInvalidCredentials — and take comparable time, so neither
// the error nor a timing side-channel reveals whether an email is
// registered.
func (s *Service) Login(ctx context.Context, email, password string) (*Session, error) {
	player, err := s.players.GetPlayerByEmail(ctx, strings.TrimSpace(email))
	if err != nil {
		if !errors.Is(err, ErrPlayerNotFound) {
			return nil, err
		}
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword(player.PasswordHash, []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := newToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	now := time.Now()
	session := &Session{
		Token:     token,
		PlayerID:  player.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionDuration),
	}
	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// Logout deletes the session. Deleting an unknown or already-deleted token
// is not an error — SessionRepository.DeleteSession is idempotent by
// design.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.DeleteSession(ctx, token)
}

// GetPlayer returns the player by ID, for read-only display — e.g.
// resolving a PlayerID to a DisplayName when rendering a list of
// participants elsewhere in the app.
func (s *Service) GetPlayer(ctx context.Context, id game.PlayerID) (*Player, error) {
	return s.players.GetPlayer(ctx, id)
}

// Authenticate resolves a session token to the Player it belongs to. It
// returns ErrSessionNotFound for a missing, invalid, or expired token
// (SessionRepository.GetSession already collapses "expired" into
// "not found"), unchanged rather than wrapped, so callers can check it with
// errors.Is against the same sentinel the repository defines.
func (s *Service) Authenticate(ctx context.Context, token string) (*Player, error) {
	session, err := s.sessions.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return s.players.GetPlayer(ctx, session.PlayerID)
}
