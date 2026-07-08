package adjudicator_test

import (
	"os"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/adjudicator"
	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

func TestResolve_UnhinderedMovement(t *testing.T) {
	t.Parallel()
	gm := loadWesternEuropeMap(t)
	cfg := game.NewGameConfig{
		ID: "test",
		Assignments: map[gamemap.NationID]game.PlayerID{
			"eng": "pe",
			"fra": "pf",
		},
	}
	g, _ := game.NewGame(cfg, gm)
	o := game.NewMoveOrder("fra-army-par-start", "fra", "gas", "")

	err := g.SubmitOrder(o, gm)
	if err != nil {
		t.Fatalf("submit order: %v", err)
	}

	res, err := adjudicator.Resolve(g, gm)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	totalUnits := len(g.Units)

	if len(res.OrderOutcomes) != totalUnits {
		t.Fatalf("expected %d order outcomes, got %d", totalUnits, len(res.OrderOutcomes))
	}

	if len(res.UnitOutcomes) != totalUnits {
		t.Fatalf("expected %d unit outcomes, got %d", totalUnits, len(res.UnitOutcomes))
	}

	oo := res.OrderOutcomes["fra-army-par-start"]

	if oo.Success != true {
		t.Fatalf("expected success, got %v", oo.Success)
	}

	if oo.Reason != adjudicator.ReasonSuccess {
		t.Fatalf("expected no reason, got %s", oo.Reason)
	}

	uo := res.UnitOutcomes["fra-army-par-start"]

	if uo.Type != adjudicator.UnitOutcomeMove {
		t.Fatalf("expected move outcome, got %s", uo.Type)
	}

	if uo.From != gamemap.ProvinceID("par") {
		t.Fatalf("expected from province to be par-start, got %s", uo.From)
	}

	if uo.To != gamemap.ProvinceID("gas") {
		t.Fatalf("expected to province to be gas, got %s", uo.To)
	}
}

// func TestResolve_AttackOfEqualStrength(t *testing.T) {

// }

// func TestResolve_SupportedAttack(t *testing.T) {

// }

// func TestResolve_UnitsSwapPositions(t *testing.T) {

// }

// func TestResolve_UnitsMoveInACircle(t *testing.T) {

// }

func loadWesternEuropeMap(t *testing.T) *gamemap.GameMap {
	t.Helper()

	data, err := os.ReadFile("../gamemap/testdata/western_europe.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	gm, err := gamemap.Load(data)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	return gm
}
