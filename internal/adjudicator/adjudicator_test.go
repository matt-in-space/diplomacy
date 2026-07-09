package adjudicator_test

import (
	"os"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/adjudicator"
	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

// Units are
// map[eng-fleet-lon-start:{eng-fleet-lon-start eng lon fleet}
// fra-army-par-start:{fra-army-par-start fra par army}
// fra-fleet-bre-start:{fra-fleet-bre-start fra bre fleet}]

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

func TestResolve_UnitsWithoutOrdersDefaultToHold(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	cfg := game.NewGameConfig{
		ID: "test",
		Assignments: map[gamemap.NationID]game.PlayerID{
			"eng": "pe",
			"fra": "pf",
		},
	}
	g, _ := game.NewGame(cfg, gm)

	res, err := adjudicator.Resolve(g, gm)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	for unitID, _ := range g.Units {
		oo := res.OrderOutcomes[unitID]
		if oo.Reason != adjudicator.ReasonSuccess {
			t.Fatalf("expected success reason, got %s", oo.Reason)
		}

		uo := res.UnitOutcomes[unitID]
		if uo.Type != adjudicator.UnitOutcomeHold {
			t.Fatalf("expected hold outcome, got %s", uo.Type)
		}
	}
}

func TestResolve_AttackOfEqualStrength(t *testing.T) {
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

	u := game.Unit{
		ID:         "eng-1",
		NationID:   "eng",
		ProvinceID: "gas",
		Type:       game.UnitTypeArmy,
	}

	g.Units[u.ID] = u
	g.Positions["gas"] = u.ID

	ho := game.NewHoldOrder(u.ID, u.NationID, u.ProvinceID)
	mo := game.NewMoveOrder("fra-army-par-start", "fra", "gas", "")

	g.SubmitOrder(ho, gm)
	g.SubmitOrder(mo, gm)

	res, err := adjudicator.Resolve(g, gm)

	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	eoo := res.OrderOutcomes[u.ID]
	euo := res.UnitOutcomes[u.ID]

	if euo.Type != adjudicator.UnitOutcomeHold {
		t.Fatalf("expected hold outcome, got %s", euo.Type)
	}
	if eoo.Reason != adjudicator.ReasonSuccess {
		t.Fatalf("expected success reason, got %s", eoo.Reason)
	}

	foo := res.OrderOutcomes["fra-army-par-start"]
	fuo := res.UnitOutcomes["fra-army-par-start"]

	if foo.Reason != adjudicator.ReasonWeakAttack {
		t.Fatalf("expected weak attack reason, got %s", foo.Reason)
	}
	if fuo.Type != adjudicator.UnitOutcomeHold {
		t.Fatalf("expected hold outcome, got %s", fuo.Type)
	}
}

// func TestResolve_SupportedAttack(t *testing.T) {

// }

// func TestResolve_UnitsSwapPositions(t *testing.T) {

// }

// func TestResolve_UnitReplacesAnother(t *testing.T) {

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
