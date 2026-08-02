package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestGameAdjustmentBalance(t *testing.T) {
	tests := []struct {
		name   string
		nation gamemap.NationID
		setup  func(t *testing.T, g *game.Game)
		want   int
	}{
		{
			name:   "baseline fra has two centers and two units",
			nation: "fra",
			setup:  func(t *testing.T, g *game.Game) {},
			want:   0,
		},
		{
			name:   "fra gains a center it does not have a unit for",
			nation: "fra",
			setup: func(t *testing.T, g *game.Game) {
				g.SupplyCenterOwners["spa"] = "fra"
			},
			want: 1,
		},
		{
			name:   "eng loses its only unit",
			nation: "eng",
			setup: func(t *testing.T, g *game.Game) {
				delete(g.Units, "eng-fleet-lon-start")
			},
			want: 1,
		},
		{
			name:   "fra loses par to eng",
			nation: "fra",
			setup: func(t *testing.T, g *game.Game) {
				g.SupplyCenterOwners["par"] = "eng"
			},
			want: -1,
		},
		{
			name:   "unknown nation has no centers and no units",
			nation: "ita",
			setup:  func(t *testing.T, g *game.Game) {},
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := newWesternEuropeGame(t, gm)
			tt.setup(t, g)

			if got := g.AdjustmentBalance(tt.nation); got != tt.want {
				t.Fatalf("AdjustmentBalance(%q) = %d, want %d", tt.nation, got, tt.want)
			}
		})
	}
}

func TestGameLegalBuildProvinces(t *testing.T) {
	t.Run("baseline fra has both home centers occupied", func(t *testing.T) {
		gm := loadWesternEuropeMap(t)
		g := newWesternEuropeGame(t, gm)

		got := g.LegalBuildProvinces("fra", gm)
		if got == nil {
			t.Fatal("LegalBuildProvinces returned nil, want non-nil empty slice")
		}
		if len(got) != 0 {
			t.Fatalf("LegalBuildProvinces(fra) = %v, want empty", got)
		}
	})

	t.Run("fra can build at a vacated home center", func(t *testing.T) {
		gm := loadWesternEuropeMap(t)
		g := newWesternEuropeGame(t, gm)
		delete(g.Units, "fra-army-par-start")

		got := g.LegalBuildProvinces("fra", gm)
		want := []gamemap.ProvinceID{"par"}
		assertProvinceSlice(t, got, want)
	})

	t.Run("a captured neutral center is never buildable", func(t *testing.T) {
		gm := loadWesternEuropeMap(t)
		g := newWesternEuropeGame(t, gm)
		delete(g.Units, "fra-army-par-start")
		g.SupplyCenterOwners["spa"] = "fra"

		got := g.LegalBuildProvinces("fra", gm)
		want := []gamemap.ProvinceID{"par"}
		assertProvinceSlice(t, got, want)
	})

	t.Run("a vacant home center owned by another nation is buildable by neither", func(t *testing.T) {
		gm := loadWesternEuropeMap(t)
		g := newWesternEuropeGame(t, gm)
		delete(g.Units, "fra-army-par-start")
		g.SupplyCenterOwners["par"] = "eng"

		if got := g.LegalBuildProvinces("fra", gm); len(got) != 0 {
			t.Fatalf("LegalBuildProvinces(fra) = %v, want empty (owned by eng)", got)
		}
		if got := g.LegalBuildProvinces("eng", gm); len(got) != 0 {
			t.Fatalf("LegalBuildProvinces(eng) = %v, want empty (not eng's home center)", got)
		}
	})

	t.Run("a home center occupied by a foreign unit is not buildable", func(t *testing.T) {
		gm := loadWesternEuropeMap(t)
		g := newWesternEuropeGame(t, gm)
		unit := g.Units["fra-army-par-start"]
		unit.ProvinceID = "" // vacate par...
		g.Units[unit.ID] = unit
		addArmy(t, g, "eng-army-par", "eng", "par") // ...then an eng army occupies it

		if got := g.LegalBuildProvinces("fra", gm); len(got) != 0 {
			t.Fatalf("LegalBuildProvinces(fra) = %v, want empty (occupied by eng)", got)
		}
	})

	t.Run("unknown nation has no buildable provinces", func(t *testing.T) {
		gm := loadWesternEuropeMap(t)
		g := newWesternEuropeGame(t, gm)

		if got := g.LegalBuildProvinces("ita", gm); len(got) != 0 {
			t.Fatalf("LegalBuildProvinces(ita) = %v, want empty", got)
		}
	})
}

func assertProvinceSlice(t *testing.T, got, want []gamemap.ProvinceID) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
