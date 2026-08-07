// Package random provides shared sources of cryptographically random
// values for anywhere an identifier or credential needs to be unguessable
// rather than sequential. It's internal/ rather than application/ because
// it's pure utility plumbing, not a bounded-context layer — it doesn't
// belong conceptually alongside auth/gameplay/lobby.
package random

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

// String returns a cryptographically random, hex-encoded string: 32 bytes
// (256 bits) of entropy. For anything never read or typed by a human that
// needs to be resistant to guessing — session tokens, player IDs — where
// length costs nothing and entropy is what matters.
func String() (string, error) {
	return Hex(32)
}

// Hex returns n bytes of cryptographically random data, hex-encoded (2n
// hex characters). The building block behind String, for callers that want
// a specific byte length instead of the 256-bit default — e.g. an
// identifier that's a map key or URL segment but never hand-typed, so it
// doesn't need session-token-grade entropy either.
func Hex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// codeAlphabet excludes visually ambiguous characters (0/O, 1/I/L) so a
// code read aloud or copied by hand isn't tripped up by characters that
// look alike in most fonts.
const codeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// Code returns an n-character random code drawn from codeAlphabet — for
// identifiers a human actually has to read, type, or share, like an invite
// code, where being short and unambiguous matters more than raw entropy.
func Code(n int) (string, error) {
	alphabetLen := big.NewInt(int64(len(codeAlphabet)))
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		b[i] = codeAlphabet[idx.Int64()]
	}
	return string(b), nil
}
