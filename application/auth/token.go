package auth

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/matt-in-space/diplomacy/core/game"
)

// newToken returns a cryptographically random, hex-encoded string suitable
// for a session token: 32 bytes (256 bits) of entropy, never anything
// sequential or predictable.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// newPlayerID mints a new, random PlayerID. Players have no natural key to
// derive an ID from the way starting units do (nation+type+province), so —
// like session tokens — it's drawn from a random source rather than being
// sequential.
func newPlayerID() (game.PlayerID, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	return game.PlayerID(token), nil
}
