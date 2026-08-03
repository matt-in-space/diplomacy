package game_test

import (
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
)

func TestGameBeginAdjustmentResolutionWaitsForNationsWithBalance(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.AcceptAdjustments
	delete(g.Units, "fra-army-par-start") // fra: 2 centers, 1 unit -> balance +1

	progressed, err := g.BeginAdjustmentResolution()
	if err != nil {
		t.Fatalf("BeginAdjustmentResolution failed: %v", err)
	}
	if progressed {
		t.Fatal("BeginAdjustmentResolution progressed before fra committed")
	}

	if err := g.CommitOrders("fra", gm); err != nil {
		t.Fatalf("CommitOrders failed: %v", err)
	}

	progressed, err = g.BeginAdjustmentResolution()
	if err != nil {
		t.Fatalf("BeginAdjustmentResolution failed: %v", err)
	}
	if !progressed {
		t.Fatal("BeginAdjustmentResolution did not progress after fra committed")
	}
	if g.Turn.Phase != game.ResolveAdjustments {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.ResolveAdjustments)
	}
}

func TestGameBeginAdjustmentResolutionIgnoresNationsWithZeroBalance(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.AcceptAdjustments
	// Both nations start with centers == units, so neither owes anything and
	// neither needs to commit for the phase to advance.

	progressed, err := g.BeginAdjustmentResolution()
	if err != nil {
		t.Fatalf("BeginAdjustmentResolution failed: %v", err)
	}
	if !progressed {
		t.Fatal("BeginAdjustmentResolution did not progress with no nation owing an adjustment")
	}
}

func TestGameBeginAdjustmentResolutionRejectsWrongPhase(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)

	if _, err := g.BeginAdjustmentResolution(); err == nil {
		t.Fatal("expected BeginAdjustmentResolution to reject wrong phase")
	}
}

func TestGameCompleteAdjustmentResolutionAppliesBuild(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	delete(g.Units, "fra-army-par-start") // fra: 2 centers, 1 unit -> balance +1
	g.Turn = game.Turn{Season: game.Fall, Phase: game.ResolveAdjustments, Year: 1}

	unitID := g.NextBuildUnitID("fra", game.UnitTypeArmy, "par")
	res := game.AdjustmentResolution{
		Builds: []game.UnitBuild{
			{UnitID: unitID, NationID: "fra", Type: game.UnitTypeArmy, ProvinceID: "par"},
		},
	}

	if err := g.CompleteAdjustmentResolution(res); err != nil {
		t.Fatalf("CompleteAdjustmentResolution failed: %v", err)
	}

	assertUnit(t, g, unitID, game.Unit{ID: unitID, NationID: "fra", Type: game.UnitTypeArmy, ProvinceID: "par"})
	if g.Turn.Phase != game.AcceptOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptOrders)
	}
	if g.Turn.Season != game.Spring {
		t.Fatalf("Turn.Season = %q, want %q", g.Turn.Season, game.Spring)
	}
	if g.Turn.Year != 2 {
		t.Fatalf("Turn.Year = %d, want 2", g.Turn.Year)
	}
	if len(g.CommittedOrders) != 0 {
		t.Fatalf("CommittedOrders length = %d, want 0", len(g.CommittedOrders))
	}
	if len(g.Orders) != 0 {
		t.Fatalf("Orders length = %d, want 0", len(g.Orders))
	}
}

func TestGameCompleteAdjustmentResolutionAppliesDisband(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := adjustmentDisbandScenarioGame(gm)
	g.Turn.Phase = game.ResolveAdjustments

	res := game.AdjustmentResolution{Disbands: []game.UnitID{"fra-army-gas"}}
	if err := g.CompleteAdjustmentResolution(res); err != nil {
		t.Fatalf("CompleteAdjustmentResolution failed: %v", err)
	}

	if _, ok := g.Units["fra-army-gas"]; ok {
		t.Fatal("unit should have been disbanded")
	}
}

func TestGameCompleteAdjustmentResolutionRejectsWrongPhase(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)

	if err := g.CompleteAdjustmentResolution(game.AdjustmentResolution{}); err == nil {
		t.Fatal("expected CompleteAdjustmentResolution to reject wrong phase")
	}
}

func TestGameCompleteAdjustmentResolutionRejectsInvalidResolutions(t *testing.T) {
	tests := []struct {
		name string
		res  func(g *game.Game) game.AdjustmentResolution
		want string
	}{
		{
			name: "build targets an occupied province",
			res: func(g *game.Game) game.AdjustmentResolution {
				return game.AdjustmentResolution{
					Builds: []game.UnitBuild{
						{UnitID: g.NextBuildUnitID("fra", game.UnitTypeFleet, "bre"), NationID: "fra", Type: game.UnitTypeFleet, ProvinceID: "bre", Coast: "bre"},
					},
				}
			},
			want: "occupied",
		},
		{
			name: "build reuses an existing UnitID",
			res: func(g *game.Game) game.AdjustmentResolution {
				return game.AdjustmentResolution{
					Builds: []game.UnitBuild{
						{UnitID: "fra-fleet-bre-start", NationID: "fra", Type: game.UnitTypeArmy, ProvinceID: "par"},
					},
				}
			},
			want: "already exists",
		},
		{
			name: "disband names a unit that does not exist",
			res: func(g *game.Game) game.AdjustmentResolution {
				return game.AdjustmentResolution{Disbands: []game.UnitID{"missing-unit"}}
			},
			want: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := newWesternEuropeGame(t, gm)
			delete(g.Units, "fra-army-par-start")
			g.Turn = game.Turn{Season: game.Fall, Phase: game.ResolveAdjustments, Year: 1}

			err := g.CompleteAdjustmentResolution(tt.res(g))
			if err == nil {
				t.Fatalf("expected CompleteAdjustmentResolution to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("CompleteAdjustmentResolution error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}
