package auth

import (
	"time"

	"github.com/matt-in-space/diplomacy/core/game"
)

// A Session represents a logged-in player. Token is both the map key and
// the secret itself — it must come from a cryptographically random source,
// never anything sequential or derived from PlayerID, since guessing it is
// equivalent to hijacking the session. Generating it is the login handler's
// job, not the repository's.
type Session struct {
	Token     string
	PlayerID  game.PlayerID
	CreatedAt time.Time
	ExpiresAt time.Time
}
