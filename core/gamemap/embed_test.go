package gamemap_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestWesternEurope(t *testing.T) {
	gm, err := gamemap.WesternEurope()
	if err != nil {
		t.Fatalf("WesternEurope failed: %v", err)
	}
	if gm == nil {
		t.Fatal("WesternEurope returned nil game map")
	}
	if gm.ID != gamemap.WesternEuropeMapID {
		t.Fatalf("ID = %q, want %q", gm.ID, gamemap.WesternEuropeMapID)
	}
}
