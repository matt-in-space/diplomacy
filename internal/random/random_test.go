package random

import (
	"encoding/hex"
	"strings"
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

func TestHexReturnsRequestedByteLength(t *testing.T) {
	s, err := Hex(16)
	if err != nil {
		t.Fatalf("Hex failed: %v", err)
	}
	decoded, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("Hex(16) = %q, not valid hex: %v", s, err)
	}
	if len(decoded) != 16 {
		t.Fatalf("decoded length = %d, want 16", len(decoded))
	}
}

func TestCodeReturnsRequestedLengthFromAlphabet(t *testing.T) {
	c, err := Code(8)
	if err != nil {
		t.Fatalf("Code failed: %v", err)
	}
	if len(c) != 8 {
		t.Fatalf("len(Code(8)) = %d, want 8", len(c))
	}
	for _, r := range c {
		if !strings.ContainsRune(codeAlphabet, r) {
			t.Fatalf("Code() = %q contains %q, not in codeAlphabet %q", c, r, codeAlphabet)
		}
	}
}

func TestCodeReturnsDistinctValues(t *testing.T) {
	a, err := Code(8)
	if err != nil {
		t.Fatalf("Code failed: %v", err)
	}
	b, err := Code(8)
	if err != nil {
		t.Fatalf("Code failed: %v", err)
	}
	if a == b {
		t.Fatalf("two calls to Code returned the same value: %q", a)
	}
}
