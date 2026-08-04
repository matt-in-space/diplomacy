package auth

import (
	"time"

	"github.com/matt-in-space/diplomacy/core/game"
)

// A Player is a registered account: the credential-bearing identity behind
// a game.PlayerID. It's kept out of core/game deliberately — the domain
// layer only ever needs the ID to assign nations to, never email addresses
// or password hashes.
type Player struct {
	ID                game.PlayerID
	Email             string
	DisplayName       string
	PasswordHash      []byte
	EmailVerified     bool
	CreatedAt         time.Time
	PasswordChangedAt time.Time
}
