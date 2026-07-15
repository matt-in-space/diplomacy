package gamemap_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/core/gamemap"
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

	if !slices.Contains(gm.Nations, gamemap.NationID("eng")) {
		t.Fatalf("expected nations to contain eng")
	}
	if !slices.Contains(gm.Nations, gamemap.NationID("fra")) {
		t.Fatalf("expected nations to contain fra")
	}
	if par.HomeNation != "fra" {
		t.Fatalf("Province 'par' has incorrect home nation: got %s, want fra", par.HomeNation)
	}
	if len(gm.StartingUnits) != 3 {
		t.Fatalf("StartingUnits length = %d, want 3", len(gm.StartingUnits))
	}

	assertStartingUnit(t, gm.StartingUnits, gamemap.StartingUnit{
		Nation:   "fra",
		Type:     gamemap.StartingUnitTypeArmy,
		Province: "par",
		Coast:    "",
	})
	assertStartingUnit(t, gm.StartingUnits, gamemap.StartingUnit{
		Nation:   "fra",
		Type:     gamemap.StartingUnitTypeFleet,
		Province: "bre",
		Coast:    "bre",
	})
	assertStartingUnit(t, gm.StartingUnits, gamemap.StartingUnit{
		Nation:   "eng",
		Type:     gamemap.StartingUnitTypeFleet,
		Province: "lon",
		Coast:    "lon",
	})
}

func TestLoad_RejectsInvalidMaps(t *testing.T) {
	for _, tc := range loadErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			assertLoadErrorContains(t, []byte(tc.data), tc.want)
		})
	}
}

func assertStartingUnit(t *testing.T, units []gamemap.StartingUnit, want gamemap.StartingUnit) {
	t.Helper()

	if !slices.Contains(units, want) {
		t.Fatalf("expected starting units to contain %+v", want)
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
