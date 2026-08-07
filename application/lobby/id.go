package lobby

import (
	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/internal/random"
)

// newGameID mints a new, random GameSetup/Game ID. A GameSetup has no
// natural key to derive an ID from, so it's drawn from a random source
// rather than being sequential — 16 bytes (128 bits) is far more than
// enough to avoid collisions for a map key/URL segment that's never
// hand-typed, without carrying the full 256-bit session-token budget.
func newGameID() (game.GameID, error) {
	token, err := random.Hex(16)
	if err != nil {
		return "", err
	}
	return game.GameID(token), nil
}

// newInviteCode mints a new, random invite code — the actual credential an
// invite/share link is built around. Unlike newGameID, this one is read
// and typed by a person, so it trades entropy for length: 8 characters
// from an unambiguous alphabet instead of a long hex string.
func newInviteCode() (string, error) {
	return random.Code(8)
}
