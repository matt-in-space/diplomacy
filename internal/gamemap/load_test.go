package gamemap_test

import (
	"os"
	"strings"
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

func TestLoad_RejectsInvalidMaps(t *testing.T) {
	for _, tc := range loadErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			assertLoadErrorContains(t, []byte(tc.data), tc.want)
		})
	}
}

func assertLoadErrorContains(t *testing.T, data []byte, want string) {
	t.Helper()

	_, err := gamemap.Load(data)
	if err == nil {
		t.Fatalf("expected Load to fail")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Load error = %q, want substring %q", err.Error(), want)
	}
}
