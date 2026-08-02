package adjudicator

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// retreatUnit describes a dislodged unit to place in a retreat test's game.
type retreatUnit struct {
	id            game.UnitID
	nation        gamemap.NationID
	kind          game.UnitType
	dislodgedFrom gamemap.ProvinceID
	coast         gamemap.CoastID
}

func rArmy(id game.UnitID, nation gamemap.NationID, from gamemap.ProvinceID) retreatUnit {
	return retreatUnit{id: id, nation: nation, kind: game.UnitTypeArmy, dislodgedFrom: from}
}

func rFleet(id game.UnitID, nation gamemap.NationID, from gamemap.ProvinceID, coast gamemap.CoastID) retreatUnit {
	return retreatUnit{id: id, nation: nation, kind: game.UnitTypeFleet, dislodgedFrom: from, coast: coast}
}

// newRetreatTestGame builds a game with the given dislodged units (plus any
// on-board units, e.g. an occupant blocking a target) and orders.
func newRetreatTestGame(gm *gamemap.GameMap, dislodged []retreatUnit, onBoard []testUnit, orders ...game.UnitOrder) *game.Game {
	g := newTestGame(gm, onBoard, orders...)
	for _, u := range dislodged {
		unit := game.Unit{ID: u.id, NationID: u.nation, Type: u.kind, DislodgedFrom: u.dislodgedFrom}
		if u.kind == game.UnitTypeFleet {
			unit.Coast = u.coast
		}
		g.Units[u.id] = unit
	}
	return g
}

func TestResolveRetreats_SoloRetreatSucceeds(t *testing.T) {
	gm := loadTestMap(t)
	g := newRetreatTestGame(gm,
		[]retreatUnit{rArmy("fra-army-a", "fra", "par")},
		nil,
		game.NewRetreatOrder("fra-army-a", "fra", "gas", ""),
	)

	res, err := ResolveRetreats(g, gm)
	if err != nil {
		t.Fatalf("ResolveRetreats failed: %v", err)
	}

	outcome, ok := res["fra-army-a"]
	if !ok {
		t.Fatal("no outcome for fra-army-a")
	}
	if !outcome.Order.Success {
		t.Fatalf("Order.Success = false, want true (reason %q)", outcome.Order.Reason)
	}
	if outcome.Unit.Type != game.UnitTransformMove {
		t.Fatalf("Unit.Type = %q, want move", outcome.Unit.Type)
	}
	if outcome.Unit.To != "gas" {
		t.Fatalf("Unit.To = %q, want gas", outcome.Unit.To)
	}
	if outcome.Unit.From != "par" {
		t.Fatalf("Unit.From = %q, want par", outcome.Unit.From)
	}
}

func TestResolveRetreats_ConflictingRetreatsAllDisband(t *testing.T) {
	gm := loadTestMap(t)
	g := newRetreatTestGame(gm,
		[]retreatUnit{
			rArmy("fra-army-a", "fra", "par"),
			rArmy("eng-army-a", "eng", "spa"),
		},
		nil,
		game.NewRetreatOrder("fra-army-a", "fra", "gas", ""),
		game.NewRetreatOrder("eng-army-a", "eng", "gas", ""),
	)

	res, err := ResolveRetreats(g, gm)
	if err != nil {
		t.Fatalf("ResolveRetreats failed: %v", err)
	}

	for _, id := range []game.UnitID{"fra-army-a", "eng-army-a"} {
		outcome, ok := res[id]
		if !ok {
			t.Fatalf("no outcome for %q", id)
		}
		if outcome.Order.Success {
			t.Fatalf("%q: Order.Success = true, want false", id)
		}
		if outcome.Order.Reason != game.ReasonRetreatConflict {
			t.Fatalf("%q: Order.Reason = %q, want %q", id, outcome.Order.Reason, game.ReasonRetreatConflict)
		}
		if outcome.Unit.Type != game.UnitTransformDisband {
			t.Fatalf("%q: Unit.Type = %q, want disband", id, outcome.Unit.Type)
		}
	}
}

func TestResolveRetreats_ThreeWayConflictAllDisband(t *testing.T) {
	gm := loadTestMap(t)
	g := newRetreatTestGame(gm,
		[]retreatUnit{
			rArmy("fra-army-a", "fra", "par"),
			rArmy("eng-army-a", "eng", "spa"),
			rArmy("eng-army-b", "eng", "por"),
		},
		nil,
		game.NewRetreatOrder("fra-army-a", "fra", "gas", ""),
		game.NewRetreatOrder("eng-army-a", "eng", "gas", ""),
		game.NewRetreatOrder("eng-army-b", "eng", "gas", ""),
	)

	res, err := ResolveRetreats(g, gm)
	if err != nil {
		t.Fatalf("ResolveRetreats failed: %v", err)
	}

	for _, id := range []game.UnitID{"fra-army-a", "eng-army-a", "eng-army-b"} {
		outcome, ok := res[id]
		if !ok {
			t.Fatalf("no outcome for %q", id)
		}
		if outcome.Unit.Type != game.UnitTransformDisband {
			t.Fatalf("%q: Unit.Type = %q, want disband", id, outcome.Unit.Type)
		}
	}
}

func TestResolveRetreats_ExplicitDisbandRespected(t *testing.T) {
	gm := loadTestMap(t)
	g := newRetreatTestGame(gm,
		[]retreatUnit{rArmy("fra-army-a", "fra", "par")},
		nil,
		game.NewDisbandOrder("fra-army-a", "fra"),
	)

	res, err := ResolveRetreats(g, gm)
	if err != nil {
		t.Fatalf("ResolveRetreats failed: %v", err)
	}

	outcome, ok := res["fra-army-a"]
	if !ok {
		t.Fatal("no outcome for fra-army-a")
	}
	if !outcome.Order.Success {
		t.Fatal("Order.Success = false, want true")
	}
	if outcome.Unit.Type != game.UnitTransformDisband {
		t.Fatalf("Unit.Type = %q, want disband", outcome.Unit.Type)
	}
	if _, ok := outcome.Order.Order.(game.DisbandOrder); !ok {
		t.Fatalf("Order.Order = %T, want game.DisbandOrder", outcome.Order.Order)
	}
}

func TestResolveRetreats_MissingOrderDefaultsToDisband(t *testing.T) {
	gm := loadTestMap(t)
	g := newRetreatTestGame(gm,
		[]retreatUnit{rArmy("fra-army-a", "fra", "par")},
		nil,
		// no order submitted for fra-army-a
	)

	res, err := ResolveRetreats(g, gm)
	if err != nil {
		t.Fatalf("ResolveRetreats failed: %v", err)
	}

	outcome, ok := res["fra-army-a"]
	if !ok {
		t.Fatal("no outcome for fra-army-a")
	}
	if outcome.Unit.Type != game.UnitTransformDisband {
		t.Fatalf("Unit.Type = %q, want disband", outcome.Unit.Type)
	}
	if !outcome.Order.Success {
		t.Fatal("Order.Success = false, want true")
	}
	if _, ok := outcome.Order.Order.(game.DisbandOrder); !ok {
		t.Fatalf("Order.Order = %T, want a defaulted game.DisbandOrder", outcome.Order.Order)
	}
}

func TestResolveRetreats_FleetLandsOnSoleCoast(t *testing.T) {
	gm := loadTestMap(t)
	g := newRetreatTestGame(gm,
		[]retreatUnit{rFleet("fra-fleet-a", "fra", "mao", "mao")},
		nil,
		game.NewRetreatOrder("fra-fleet-a", "fra", "bre", ""),
	)

	res, err := ResolveRetreats(g, gm)
	if err != nil {
		t.Fatalf("ResolveRetreats failed: %v", err)
	}

	outcome, ok := res["fra-fleet-a"]
	if !ok {
		t.Fatal("no outcome for fra-fleet-a")
	}
	if outcome.Unit.Coast != "bre" {
		t.Fatalf("Unit.Coast = %q, want bre", outcome.Unit.Coast)
	}
}

func TestResolveRetreats_FleetLandsOnRequestedCoast(t *testing.T) {
	gm := loadTestMap(t)
	g := newRetreatTestGame(gm,
		[]retreatUnit{rFleet("fra-fleet-a", "fra", "mao", "mao")},
		nil,
		game.NewRetreatOrder("fra-fleet-a", "fra", "spa", "spa-nc"),
	)

	res, err := ResolveRetreats(g, gm)
	if err != nil {
		t.Fatalf("ResolveRetreats failed: %v", err)
	}

	outcome, ok := res["fra-fleet-a"]
	if !ok {
		t.Fatal("no outcome for fra-fleet-a")
	}
	if outcome.Unit.Coast != "spa-nc" {
		t.Fatalf("Unit.Coast = %q, want spa-nc", outcome.Unit.Coast)
	}
}

func TestResolveRetreats_RejectsMapMismatch(t *testing.T) {
	gm := loadTestMap(t)
	g := newRetreatTestGame(gm, nil, nil)
	g.MapID = "other-map"

	if _, err := ResolveRetreats(g, gm); err == nil {
		t.Fatal("expected ResolveRetreats to reject a mismatched map")
	}
}
