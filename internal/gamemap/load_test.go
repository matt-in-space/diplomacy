package gamemap_test

import (
	"os"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

func TestLoad_CreatesHydratedGameMap(t *testing.T) {
	data, err := os.ReadFile("testdata/western_europe.json")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	gm, err := gamemap.Load(data)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if gm == nil {
		t.Fatalf("Load returned nil game map")
	}

	par, ok := gm.Provinces["par"]
	if !ok {
		t.Fatalf("Province 'par' not found")
	}

	if par.Name != "Paris" {
		t.Fatalf("Province 'par' has incorrect name: got %s, want Paris", par.Name)
	}
}
