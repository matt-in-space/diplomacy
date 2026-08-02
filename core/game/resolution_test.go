package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
)

func TestGameBeginOrderResolutionWaitsForCommittedOrders(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)

	progressed, err := g.BeginOrderResolution()
	if err != nil {
		t.Fatalf("BeginOrderResolution failed: %v", err)
	}
	if progressed {
		t.Fatal("BeginOrderResolution progressed before all nations committed")
	}
	if g.Turn.Phase != game.AcceptOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptOrders)
	}
}

func TestGameBeginOrderResolutionAdvancesWhenAllOrdersCommitted(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.CommittedOrders["eng"] = struct{}{}
	g.CommittedOrders["fra"] = struct{}{}

	progressed, err := g.BeginOrderResolution()
	if err != nil {
		t.Fatalf("BeginOrderResolution failed: %v", err)
	}
	if !progressed {
		t.Fatal("BeginOrderResolution did not progress after all nations committed")
	}
	if g.Turn.Phase != game.ResolveOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.ResolveOrders)
	}
}

func TestGameBeginOrderResolutionRejectsWrongPhase(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.ResolveOrders

	progressed, err := g.BeginOrderResolution()
	if err == nil {
		t.Fatal("expected BeginOrderResolution to reject wrong phase")
	}
	if progressed {
		t.Fatal("BeginOrderResolution progressed from wrong phase")
	}
}

func TestGameCompleteOrderResolution(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.ResolveOrders
	g.CommittedOrders["eng"] = struct{}{}
	g.CommittedOrders["fra"] = struct{}{}
	g.Orders = append(g.Orders, game.NewHoldOrder("eng-fleet-lon-start", "eng"))

	if err := g.CompleteOrderResolution(holdResolution(g)); err != nil {
		t.Fatalf("CompleteOrderResolution failed: %v", err)
	}
	if g.Turn.Phase != game.AcceptRetreats {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptRetreats)
	}
	if len(g.CommittedOrders) != 0 {
		t.Fatalf("CommittedOrders length = %d, want 0", len(g.CommittedOrders))
	}
	if len(g.Orders) != 0 {
		t.Fatalf("Orders length = %d, want 0", len(g.Orders))
	}
}

func TestGameCompleteOrderResolutionRejectsWrongPhase(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)

	if g.Turn.Phase != game.AcceptOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptOrders)
	}
}

func TestGameCompleteOrderResolutionPreservesLifecycleStateWhenTransformsFail(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.ResolveOrders
	g.CommittedOrders["eng"] = struct{}{}
	g.Orders = append(g.Orders, game.NewHoldOrder("eng-fleet-lon-start", "eng"))

	if g.Turn.Phase != game.ResolveOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.ResolveOrders)
	}
	if _, ok := g.CommittedOrders["eng"]; !ok {
		t.Fatal("CommittedOrders was cleared after failed transforms")
	}
	if _, ok := g.OrderFor("eng-fleet-lon-start"); !ok {
		t.Fatal("Orders was cleared after failed transforms")
	}
}

func TestGameCompleteRetreatResolution(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.ResolveRetreats
	dislodge(t, g, "fra-army-par-start")

	res := game.Resolution{
		"fra-army-par-start": game.Outcome{
			UnitID: "fra-army-par-start",
			Unit: game.UnitTransform{
				UnitID: "fra-army-par-start",
				Type:   game.UnitTransformMove,
				From:   "par",
				To:     "gas",
			},
			Order: game.CreateOrderSuccessOutcome(game.NewRetreatOrder("fra-army-par-start", "fra", "gas", "")),
		},
	}

	if err := g.CompleteRetreatResolution(res); err != nil {
		t.Fatalf("CompleteRetreatResolution failed: %v", err)
	}
	if g.Turn.Phase != game.AcceptOrders {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptOrders)
	}
	if g.Turn.Season != game.Fall {
		t.Fatalf("Turn.Season = %q, want %q", g.Turn.Season, game.Fall)
	}
	if got := g.Units["fra-army-par-start"].ProvinceID; got != "gas" {
		t.Fatalf("unit province = %q, want gas", got)
	}
	if len(g.CommittedOrders) != 0 {
		t.Fatalf("CommittedOrders length = %d, want 0", len(g.CommittedOrders))
	}
	if len(g.Orders) != 0 {
		t.Fatalf("Orders length = %d, want 0", len(g.Orders))
	}
	if len(g.LastRetreatResolution) != 1 {
		t.Fatalf("LastRetreatResolution length = %d, want 1", len(g.LastRetreatResolution))
	}
}

func TestGameCompleteRetreatResolutionDisbandsUnit(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.ResolveRetreats
	dislodge(t, g, "fra-army-par-start")

	res := game.Resolution{
		"fra-army-par-start": game.Outcome{
			UnitID: "fra-army-par-start",
			Unit: game.UnitTransform{
				UnitID: "fra-army-par-start",
				Type:   game.UnitTransformDisband,
				From:   "par",
			},
			Order: game.CreateOrderSuccessOutcome(game.NewDisbandOrder("fra-army-par-start", "fra")),
		},
	}

	if err := g.CompleteRetreatResolution(res); err != nil {
		t.Fatalf("CompleteRetreatResolution failed: %v", err)
	}
	if _, ok := g.Units["fra-army-par-start"]; ok {
		t.Fatal("unit should have been disbanded")
	}
}

func TestGameCompleteRetreatResolutionRejectsWrongPhase(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)

	err := g.CompleteRetreatResolution(game.Resolution{})
	if err == nil {
		t.Fatal("expected CompleteRetreatResolution to reject wrong phase")
	}
}

func TestGameCompleteRetreatResolutionDoesNotTouchLastOrderResolution(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.ResolveRetreats
	dislodge(t, g, "fra-army-par-start")
	g.LastOrderResolution = game.Resolution{"sentinel": game.Outcome{}}

	res := game.Resolution{
		"fra-army-par-start": game.Outcome{
			UnitID: "fra-army-par-start",
			Unit:   game.UnitTransform{UnitID: "fra-army-par-start", Type: game.UnitTransformDisband, From: "par"},
			Order:  game.CreateOrderSuccessOutcome(game.NewDisbandOrder("fra-army-par-start", "fra")),
		},
	}
	if err := g.CompleteRetreatResolution(res); err != nil {
		t.Fatalf("CompleteRetreatResolution failed: %v", err)
	}
	if len(g.LastOrderResolution) != 1 {
		t.Fatalf("LastOrderResolution was modified: %v", g.LastOrderResolution)
	}
}

func holdResolution(g *game.Game) game.Resolution {
	res := make(game.Resolution)

	for id, unit := range g.Units {
		transform := game.UnitTransform{
			UnitID: id,
			Type:   game.UnitTransformHold,
			From:   unit.ProvinceID,
			To:     unit.ProvinceID,
			Coast:  unit.Coast,
		}
		res[id] = game.Outcome{
			Unit: transform,
		}
	}
	return res
}
