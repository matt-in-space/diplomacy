package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestGameSubmitOrder_AcceptsBuildOrder(t *testing.T) {
	tests := []struct {
		name  string
		setup func(g *game.Game)
		order game.BuildOrder
	}{
		{
			name: "army build at vacated inland home center",
			setup: func(g *game.Game) {
				delete(g.Units, "fra-army-par-start")
			},
			order: game.NewBuildOrder("fra", "par", game.UnitTypeArmy, ""),
		},
		{
			name: "fleet build at vacated coastal home center with inferred coast",
			setup: func(g *game.Game) {
				delete(g.Units, "fra-fleet-bre-start")
			},
			order: game.NewBuildOrder("fra", "bre", game.UnitTypeFleet, ""),
		},
		{
			name: "army build at vacated coastal home center",
			setup: func(g *game.Game) {
				delete(g.Units, "fra-fleet-bre-start")
			},
			order: game.NewBuildOrder("fra", "bre", game.UnitTypeArmy, ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := newWesternEuropeGame(t, gm)
			g.Turn.Phase = game.AcceptAdjustments
			tt.setup(g)

			if err := g.SubmitOrder(tt.order, gm); err != nil {
				t.Fatalf("SubmitOrder failed: %v", err)
			}
			if len(g.Orders) != 1 {
				t.Fatalf("Orders length = %d, want 1", len(g.Orders))
			}
			got, ok := g.Orders[0].(game.BuildOrder)
			if !ok || got != tt.order {
				t.Fatalf("stored order = %+v, want %+v", got, tt.order)
			}
		})
	}
}

func TestGameSubmitOrder_RejectsInvalidBuildOrders(t *testing.T) {
	tests := []struct {
		name  string
		setup func(g *game.Game)
		order game.BuildOrder
		want  string
	}{
		{
			name:  "nation has no builds available",
			setup: func(g *game.Game) {},
			order: game.NewBuildOrder("fra", "par", game.UnitTypeArmy, ""),
			want:  "no builds available",
		},
		{
			name: "province still occupied",
			setup: func(g *game.Game) {
				g.SupplyCenterOwners["spa"] = "fra"
			},
			order: game.NewBuildOrder("fra", "par", game.UnitTypeArmy, ""),
			want:  "cannot build at province",
		},
		{
			name: "province is not a home center for this nation",
			setup: func(g *game.Game) {
				delete(g.Units, "fra-army-par-start")
			},
			order: game.NewBuildOrder("fra", "lon", game.UnitTypeArmy, ""),
			want:  "cannot build at province",
		},
		{
			name: "province is a neutral center, not a home center",
			setup: func(g *game.Game) {
				delete(g.Units, "fra-army-par-start")
				g.SupplyCenterOwners["spa"] = "fra"
			},
			order: game.NewBuildOrder("fra", "spa", game.UnitTypeArmy, ""),
			want:  "cannot build at province",
		},
		{
			name: "fleet cannot build inland",
			setup: func(g *game.Game) {
				delete(g.Units, "fra-army-par-start")
			},
			order: game.NewBuildOrder("fra", "par", game.UnitTypeFleet, ""),
			want:  "fleet cannot build in province",
		},
		{
			name: "army build cannot specify a coast",
			setup: func(g *game.Game) {
				delete(g.Units, "fra-fleet-bre-start")
			},
			order: game.NewBuildOrder("fra", "bre", game.UnitTypeArmy, "bre"),
			want:  "army build cannot specify a coast",
		},
		{
			name: "unknown unit type",
			setup: func(g *game.Game) {
				delete(g.Units, "fra-army-par-start")
			},
			order: game.NewBuildOrder("fra", "par", game.UnitType("dragoon"), ""),
			want:  "unknown unit type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := newWesternEuropeGame(t, gm)
			g.Turn.Phase = game.AcceptAdjustments
			tt.setup(g)

			assertSubmitOrderErrorContains(t, g, tt.order, gm, tt.want)
		})
	}
}

func TestGameSubmitOrder_RejectsBuildOrderDuringMovementPhase(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	delete(g.Units, "fra-army-par-start")

	order := game.NewBuildOrder("fra", "par", game.UnitTypeArmy, "")
	assertSubmitOrderErrorContains(t, g, order, gm, "unsupported order type")
}

// adjustmentDisbandScenarioGame builds a minimal game in the accept
// adjustments phase with fra owing one disband: three units (par, bre, gas)
// but only two centers (par, bre), for a balance of -1. eng has one center
// (lon) and no units, for a balance of +1, so it owes a build instead.
func adjustmentDisbandScenarioGame(gm *gamemap.GameMap) *game.Game {
	return &game.Game{
		MapID: gm.ID,
		Turn:  game.Turn{Season: game.Fall, Phase: game.AcceptAdjustments, Year: 1},
		Units: map[game.UnitID]game.Unit{
			"fra-army-par":  {ID: "fra-army-par", NationID: "fra", Type: game.UnitTypeArmy, ProvinceID: "par"},
			"fra-fleet-bre": {ID: "fra-fleet-bre", NationID: "fra", Type: game.UnitTypeFleet, ProvinceID: "bre", Coast: "bre"},
			"fra-army-gas":  {ID: "fra-army-gas", NationID: "fra", Type: game.UnitTypeArmy, ProvinceID: "gas"},
		},
		SupplyCenterOwners: map[gamemap.ProvinceID]gamemap.NationID{
			"par": "fra",
			"bre": "fra",
			"lon": "eng",
		},
		CommittedOrders: make(map[gamemap.NationID]struct{}),
	}
}

func TestGameSubmitOrder_AcceptsAdjustmentDisbandOrder(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := adjustmentDisbandScenarioGame(gm)

	order := game.NewDisbandOrder("fra-army-gas", "fra")
	if err := g.SubmitOrder(order, gm); err != nil {
		t.Fatalf("SubmitOrder failed: %v", err)
	}
	got, ok := g.OrderFor(order.Unit())
	if !ok || got != order {
		t.Fatalf("stored order = %+v, want %+v", got, order)
	}
}

func TestGameSubmitOrder_RejectsAdjustmentBuildAndDisbandTogether(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	t.Run("build rejected when nation owes a disband", func(t *testing.T) {
		g := adjustmentDisbandScenarioGame(gm)
		assertSubmitOrderErrorContains(t, g, game.NewBuildOrder("fra", "par", game.UnitTypeArmy, ""), gm, "no builds available")
	})

	t.Run("disband rejected when nation does not owe one", func(t *testing.T) {
		g := adjustmentDisbandScenarioGame(gm)
		// Adding this unit brings eng's balance from +1 (a center, no
		// units) to 0 (a center, one unit) — still not negative, so the
		// disband is rejected either way.
		addArmy(t, g, "eng-army-lon", "eng", "lon")
		assertSubmitOrderErrorContains(t, g, game.NewDisbandOrder("eng-army-lon", "eng"), gm, "no disbands owed")
	})
}

func TestGameSubmitOrder_ReplacesAdjustmentBuildOrderForSameProvince(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.AcceptAdjustments
	delete(g.Units, "fra-army-par-start")

	first := game.NewBuildOrder("fra", "par", game.UnitTypeArmy, "")
	if err := g.SubmitOrder(first, gm); err != nil {
		t.Fatalf("first SubmitOrder failed: %v", err)
	}
	if err := g.SubmitOrder(first, gm); err != nil {
		t.Fatalf("second SubmitOrder failed: %v", err)
	}
	if len(g.Orders) != 1 {
		t.Fatalf("Orders length = %d, want 1 (replaced, not appended)", len(g.Orders))
	}
}

func TestGameSubmitOrder_AcceptsMultipleAdjustmentBuildOrdersForDifferentProvinces(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.AcceptAdjustments
	delete(g.Units, "fra-army-par-start")
	delete(g.Units, "fra-fleet-bre-start")

	if err := g.SubmitOrder(game.NewBuildOrder("fra", "par", game.UnitTypeArmy, ""), gm); err != nil {
		t.Fatalf("SubmitOrder for par failed: %v", err)
	}
	if err := g.SubmitOrder(game.NewBuildOrder("fra", "bre", game.UnitTypeFleet, ""), gm); err != nil {
		t.Fatalf("SubmitOrder for bre failed: %v", err)
	}
	if len(g.Orders) != 2 {
		t.Fatalf("Orders length = %d, want 2", len(g.Orders))
	}
}
