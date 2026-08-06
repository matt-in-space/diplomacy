// Package random provides a single shared source of cryptographically
// random, hex-encoded strings for anywhere an identifier or credential
// needs to be unguessable rather than sequential. It's internal/ rather
// than application/ because it's pure utility plumbing, not a
// bounded-context layer — it doesn't belong conceptually alongside
// auth/gameplay/lobby.
package random

import (
	"crypto/rand"
	"encoding/hex"
)

// String returns a cryptographically random, hex-encoded string: 32 bytes
// (256 bits) of entropy. For anywhere an identifier or credential needs to
// be unguessable rather than sequential — session tokens, invite codes,
// and entity IDs with no natural key to derive from.
func String() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
