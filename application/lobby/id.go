package lobby

import (
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/internal/random"
)

// newGameID mints a new, random GameSetup/Game ID. A GameSetup has no
// natural key to derive an ID from, so — like session tokens and player
// IDs — it's drawn from a random source rather than being sequential.
func newGameID() (game.GameID, error) {
	token, err := random.String()
	if err != nil {
		return "", err
	}
	return game.GameID(token), nil
}

// newInviteCode mints a new, random invite code. It's the actual credential
// an invite link is built around, so — like a session token — it must be
// unguessable.
func newInviteCode() (string, error) {
	return random.String()
}
