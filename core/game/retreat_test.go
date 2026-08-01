package game_test

import (
	"slices"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestLegalRetreats(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	tests := []struct {
		name  string
		units map[game.UnitID]game.Unit
		res   game.Resolution
		unit  game.UnitID
		want  []gamemap.ProvinceID
	}{
		{
			name: "excludes occupied and attacker origin provinces",
			units: map[game.UnitID]game.Unit{
				"fra-army-gas": {
					ID: "fra-army-gas", NationID: "fra", Type: game.UnitTypeArmy,
					DislodgedFrom: "gas",
				},
				"eng-army-attacker": {
					ID: "eng-army-attacker", NationID: "eng", Type: game.UnitTypeArmy,
					ProvinceID: "gas",
				},
			},
			res: game.Resolution{
				"eng-army-attacker": game.Outcome{
					UnitID: "eng-army-attacker",
					Unit:   game.UnitTransform{UnitID: "eng-army-attacker", Type: game.UnitTransformMove, From: "par", To: "gas"},
					Order:  game.CreateOrderSuccessOutcome(game.NewMoveOrder("eng-army-attacker", "eng", "gas", "")),
				},
			},
			unit: "fra-army-gas",
			// gas's army neighbors are par, bre, spa; par is excluded as the
			// non-convoyed attacker's origin.
			want: []gamemap.ProvinceID{"bre", "spa"},
		},
		{
			name: "includes attacker origin when the attacker was convoyed",
			units: map[game.UnitID]game.Unit{
				"fra-army-gas": {
					ID: "fra-army-gas", NationID: "fra", Type: game.UnitTypeArmy,
					DislodgedFrom: "gas",
				},
				"eng-army-attacker": {
					ID: "eng-army-attacker", NationID: "eng", Type: game.UnitTypeArmy,
					ProvinceID: "gas",
				},
			},
			res: game.Resolution{
				"eng-army-attacker": game.Outcome{
					UnitID: "eng-army-attacker",
					Unit:   game.UnitTransform{UnitID: "eng-army-attacker", Type: game.UnitTransformMove, From: "par", To: "gas"},
					Order:  game.CreateOrderSuccessOutcome(game.NewConvoyedMoveOrder("eng-army-attacker", "eng", "gas")),
				},
			},
			unit: "fra-army-gas",
			want: []gamemap.ProvinceID{"bre", "par", "spa"},
		},
		{
			name: "excludes standoff provinces",
			units: map[game.UnitID]game.Unit{
				"fra-army-gas": {
					ID: "fra-army-gas", NationID: "fra", Type: game.UnitTypeArmy,
					DislodgedFrom: "gas",
				},
				"eng-army-attacker": {
					ID: "eng-army-attacker", NationID: "eng", Type: game.UnitTypeArmy,
					ProvinceID: "gas",
				},
				"eng-fleet-bouncer": {
					ID: "eng-fleet-bouncer", NationID: "eng", Type: game.UnitTypeFleet,
					ProvinceID: "mao", Coast: "mao",
				},
			},
			res: game.Resolution{
				"eng-army-attacker": game.Outcome{
					UnitID: "eng-army-attacker",
					Unit:   game.UnitTransform{UnitID: "eng-army-attacker", Type: game.UnitTransformMove, From: "par", To: "gas"},
					Order:  game.CreateOrderSuccessOutcome(game.NewMoveOrder("eng-army-attacker", "eng", "gas", "")),
				},
				// A bounced move into an empty bre registers as a standoff
				// (the province is left vacant, not occupied).
				"eng-fleet-bouncer": game.Outcome{
					UnitID: "eng-fleet-bouncer",
					Unit:   game.UnitTransform{UnitID: "eng-fleet-bouncer", Type: game.UnitTransformHold, From: "mao", To: "mao"},
					Order:  game.CreateOrderFailOutcome(game.NewMoveOrder("eng-fleet-bouncer", "eng", "bre", ""), game.ReasonWeakAttack),
				},
			},
			unit: "fra-army-gas",
			want: []gamemap.ProvinceID{"spa"},
		},
		{
			name: "fleet candidates come from coast adjacency, deduplicated by province",
			units: map[game.UnitID]game.Unit{
				"fra-fleet-gas": {
					ID: "fra-fleet-gas", NationID: "fra", Type: game.UnitTypeFleet,
					Coast: "gas", DislodgedFrom: "gas",
				},
			},
			res: game.Resolution{},
			// No recorded attacker for this fixture: this case isolates
			// fleet coast-adjacency enumeration from the attacker-origin
			// exclusion, which is covered above.
			unit: "fra-fleet-gas",
			want: []gamemap.ProvinceID{"bre", "mao", "spa"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &game.Game{Units: tt.units, LastOrderResolution: tt.res}

			got := g.LegalRetreats(tt.unit, gm)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("LegalRetreats(%q) = %v, want %v", tt.unit, got, tt.want)
			}
		})
	}
}

func TestLegalRetreats_BoxedInReturnsEmptyNotNil(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	// London's only fleet neighbor is the English Channel, and the attacker
	// that dislodged this fleet came from there without a convoy, so no
	// retreat is legal.
	g := &game.Game{
		Units: map[game.UnitID]game.Unit{
			"eng-fleet-lon": {
				ID: "eng-fleet-lon", NationID: "eng", Type: game.UnitTypeFleet,
				Coast: "lon", DislodgedFrom: "lon",
			},
			"fra-fleet-attacker": {
				ID: "fra-fleet-attacker", NationID: "fra", Type: game.UnitTypeFleet,
				ProvinceID: "lon", Coast: "lon",
			},
		},
		LastOrderResolution: game.Resolution{
			"fra-fleet-attacker": game.Outcome{
				UnitID: "fra-fleet-attacker",
				Unit:   game.UnitTransform{UnitID: "fra-fleet-attacker", Type: game.UnitTransformMove, From: "eng", To: "lon", Coast: "lon"},
				Order:  game.CreateOrderSuccessOutcome(game.NewMoveOrder("fra-fleet-attacker", "fra", "lon", "lon")),
			},
		},
	}

	got := g.LegalRetreats("eng-fleet-lon", gm)
	if got == nil {
		t.Fatal("LegalRetreats returned nil, want a non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("LegalRetreats = %v, want empty", got)
	}
}

func TestLegalRetreats_NotDislodgedReturnsNil(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := &game.Game{
		Units: map[game.UnitID]game.Unit{
			"fra-army-par": {ID: "fra-army-par", NationID: "fra", Type: game.UnitTypeArmy, ProvinceID: "par"},
		},
	}

	if got := g.LegalRetreats("fra-army-par", gm); got != nil {
		t.Fatalf("LegalRetreats for on-board unit = %v, want nil", got)
	}
	if got := g.LegalRetreats("missing", gm); got != nil {
		t.Fatalf("LegalRetreats for unknown unit = %v, want nil", got)
	}
}
