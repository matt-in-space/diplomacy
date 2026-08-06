package random

import (
	"encoding/hex"
	"testing"
)

func TestStringReturnsDistinctValidHex(t *testing.T) {
	a, err := String()
	if err != nil {
		t.Fatalf("String failed: %v", err)
	}
	b, err := String()
	if err != nil {
		t.Fatalf("String failed: %v", err)
	}

	if a == b {
		t.Fatalf("two calls to String returned the same value: %q", a)
	}

	for _, s := range []string{a, b} {
		decoded, err := hex.DecodeString(s)
		if err != nil {
			t.Fatalf("String() = %q, not valid hex: %v", s, err)
		}
		if len(decoded) != 32 {
			t.Fatalf("decoded length = %d, want 32", len(decoded))
		}
	}
}
