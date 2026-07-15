package gamemap_test

import (
	"os"
	"slices"
	"testing"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestGameMap_Province(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	province, ok := gm.Province("par")
	if !ok {
		t.Fatalf("expected province par to exist")
	}
	if province.Name != "Paris" {
		t.Fatalf("province name = %q, want %q", province.Name, "Paris")
	}

	_, ok = gm.Province("missing")
	if ok {
		t.Fatalf("expected missing province to return false")
	}
}

func TestGameMap_CoastsFor(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	tests := []struct {
		name string
		id   gamemap.ProvinceID
		want []gamemap.CoastID
	}{
		{name: "inland", id: "par", want: nil},
		{name: "water", id: "mao", want: []gamemap.CoastID{"mao"}},
		{name: "bicoastal", id: "spa", want: []gamemap.CoastID{"spa-nc", "spa-sc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gm.CoastsFor(tt.id)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("CoastsFor(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestGameMap_ProvinceTypeHelpers(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	if !gm.IsInland("par") {
		t.Fatalf("expected par to be inland")
	}
	if !gm.IsWater("mao") {
		t.Fatalf("expected mao to be water")
	}
	if !gm.IsCoastal("bre") {
		t.Fatalf("expected bre to be coastal")
	}
	if gm.IsCoastal("mao") {
		t.Fatalf("expected water province mao not to be coastal")
	}
}

func TestGameMap_ArmyAdjacent(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	if !gm.ArmyAdjacent("par", "gas") {
		t.Fatalf("expected par and gas to be army-adjacent")
	}
	if gm.ArmyAdjacent("par", "mao") {
		t.Fatalf("expected par and mao not to be army-adjacent")
	}
}

func TestGameMap_ArmyNeighbors(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	neighbors := gm.ArmyNeighbors("par")
	if !slices.Contains(neighbors, gamemap.ProvinceID("bre")) {
		t.Fatalf("expected par army neighbors to contain bre")
	}
	if !slices.Contains(neighbors, gamemap.ProvinceID("gas")) {
		t.Fatalf("expected par army neighbors to contain gas")
	}
}

func TestGameMap_FleetAdjacent(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	if !gm.FleetAdjacent("mao", "spa-nc") {
		t.Fatalf("expected mao and spa-nc to be fleet-adjacent")
	}
	if gm.FleetAdjacent("eng", "spa-sc") {
		t.Fatalf("expected eng and spa-sc not to be fleet-adjacent")
	}
}

func TestGameMap_FleetNeighbors(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	neighbors := gm.FleetNeighbors("spa-nc")
	for _, want := range []gamemap.CoastID{"gas", "mao", "por"} {
		if !slices.Contains(neighbors, want) {
			t.Fatalf("expected spa-nc fleet neighbors to contain %s", want)
		}
	}
}

func TestGameMap_ProvinceForCoast(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	tests := []struct {
		name string
		id   gamemap.CoastID
		want gamemap.ProvinceID
	}{
		{name: "bicoastal land coast", id: "spa-nc", want: "spa"},
		{name: "water coast", id: "mao", want: "mao"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := gm.ProvinceForCoast(tt.id)
			if !ok {
				t.Fatalf("expected coast %q to exist", tt.id)
			}
			if got != tt.want {
				t.Fatalf("ProvinceForCoast(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}

	_, ok := gm.ProvinceForCoast("missing")
	if ok {
		t.Fatalf("expected missing coast to return false")
	}
}

func TestGameMap_CanMove(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	if !gm.CanArmyMove("par", "bre") {
		t.Fatalf("expected army to move from par to bre")
	}
	if gm.CanArmyMove("par", "mao") {
		t.Fatalf("expected army not to move from par to mao")
	}
	if !gm.CanFleetMove("mao", "bre") {
		t.Fatalf("expected fleet to move from mao to bre")
	}
	if gm.CanFleetMove("mao", "par") {
		t.Fatalf("expected fleet not to move from mao to par")
	}
}

func TestGameMap_ConvoyPathExists(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	tests := []struct {
		name string
		from gamemap.ProvinceID
		to   gamemap.ProvinceID
		via  []gamemap.CoastID
		want bool
	}{
		{name: "single fleet bridge", from: "bre", to: "lon", via: []gamemap.CoastID{"eng"}, want: true},
		{name: "single fleet bridge via ocean", from: "bre", to: "por", via: []gamemap.CoastID{"mao"}, want: true},
		{name: "multi-hop chain", from: "lon", to: "por", via: []gamemap.CoastID{"eng", "mao"}, want: true},
		{name: "destination with multiple coasts", from: "bre", to: "spa", via: []gamemap.CoastID{"mao"}, want: true},
		{name: "origin with multiple coasts", from: "spa", to: "por", via: []gamemap.CoastID{"mao"}, want: true},
		{name: "irrelevant water present", from: "bre", to: "lon", via: []gamemap.CoastID{"eng", "mao"}, want: true},
		{name: "empty via", from: "bre", to: "lon", via: nil, want: false},
		{name: "none adjacent to origin", from: "lon", to: "por", via: []gamemap.CoastID{"mao"}, want: false},
		{name: "none adjacent to destination", from: "bre", to: "lon", via: []gamemap.CoastID{"mao"}, want: false},
		{name: "incomplete chain", from: "lon", to: "por", via: []gamemap.CoastID{"eng"}, want: false},
		{name: "disconnected water", from: "gas", to: "por", via: []gamemap.CoastID{"eng"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gm.ConvoyPathExists(tt.from, tt.to, tt.via); got != tt.want {
				t.Fatalf("ConvoyPathExists(%q, %q, %v) = %v, want %v", tt.from, tt.to, tt.via, got, tt.want)
			}
		})
	}
}

func loadWesternEuropeMap(t *testing.T) *gamemap.GameMap {
	t.Helper()

	data, err := os.ReadFile("testdata/western_europe.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	gm, err := gamemap.Load(data)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	return gm
}
