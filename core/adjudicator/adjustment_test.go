package adjudicator

import (
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// newAdjustmentTestGame builds a game for ResolveAdjustments tests directly,
// bypassing SubmitOrder validation the way newRetreatTestGame does, since a
// nation submitting several orders in one phase isn't expressible through
// the single-scenario helpers adjudicator_test.go uses.
func newAdjustmentTestGame(gm *gamemap.GameMap, assignments map[gamemap.NationID]game.PlayerID, units []testUnit, owners map[gamemap.ProvinceID]gamemap.NationID, orders []game.Order, year int) *game.Game {
	g := &game.Game{
		MapID:              gm.ID,
		Assignments:        assignments,
		Turn:               game.Turn{Season: game.Fall, Phase: game.ResolveAdjustments, Year: year},
		Units:              make(map[game.UnitID]game.Unit, len(units)),
		SupplyCenterOwners: owners,
		Orders:             orders,
	}
	for _, u := range units {
		unit := game.Unit{ID: u.id, NationID: u.nation, ProvinceID: u.province, Type: u.kind}
		if u.kind == game.UnitTypeFleet {
			unit.Coast = u.coast
		}
		g.Units[u.id] = unit
	}
	return g
}

func TestResolveAdjustments_BuildsFewerThanBalanceIsWaived(t *testing.T) {
	gm := loadTestMap(t)
	g := newAdjustmentTestGame(gm,
		map[gamemap.NationID]game.PlayerID{"fra": "pf"},
		[]testUnit{tArmy("fra-army-gas", "fra", "gas")},
		map[gamemap.ProvinceID]gamemap.NationID{"par": "fra", "bre": "fra", "spa": "fra"}, // balance +2
		[]game.Order{game.NewBuildOrder("fra", "par", game.UnitTypeArmy, "")},
		1,
	)

	res, err := ResolveAdjustments(g, gm)
	if err != nil {
		t.Fatalf("ResolveAdjustments failed: %v", err)
	}
	if len(res.Builds) != 1 {
		t.Fatalf("Builds length = %d, want 1", len(res.Builds))
	}
	if len(res.Disbands) != 0 {
		t.Fatalf("Disbands length = %d, want 0", len(res.Disbands))
	}
}

func TestResolveAdjustments_BuildsExactlyBalance(t *testing.T) {
	gm := loadTestMap(t)
	g := newAdjustmentTestGame(gm,
		map[gamemap.NationID]game.PlayerID{"fra": "pf"},
		[]testUnit{tArmy("fra-army-gas", "fra", "gas")},
		map[gamemap.ProvinceID]gamemap.NationID{"par": "fra", "bre": "fra", "spa": "fra"}, // balance +2
		[]game.Order{
			game.NewBuildOrder("fra", "par", game.UnitTypeArmy, ""),
			game.NewBuildOrder("fra", "bre", game.UnitTypeFleet, ""),
		},
		1,
	)

	res, err := ResolveAdjustments(g, gm)
	if err != nil {
		t.Fatalf("ResolveAdjustments failed: %v", err)
	}
	if len(res.Builds) != 2 {
		t.Fatalf("Builds length = %d, want 2", len(res.Builds))
	}

	wantParID := g.NextBuildUnitID("fra", game.UnitTypeArmy, "par")
	wantBreID := g.NextBuildUnitID("fra", game.UnitTypeFleet, "bre")
	seen := make(map[game.UnitID]game.UnitBuild, len(res.Builds))
	for _, build := range res.Builds {
		seen[build.UnitID] = build
	}

	parBuild, ok := seen[wantParID]
	if !ok {
		t.Fatalf("no build for minted ID %q; got %+v", wantParID, res.Builds)
	}
	if parBuild.ProvinceID != "par" || parBuild.Type != game.UnitTypeArmy || parBuild.NationID != "fra" || parBuild.Coast != "" {
		t.Fatalf("par build = %+v, want army at par for fra with no coast", parBuild)
	}

	breBuild, ok := seen[wantBreID]
	if !ok {
		t.Fatalf("no build for minted ID %q; got %+v", wantBreID, res.Builds)
	}
	if breBuild.ProvinceID != "bre" || breBuild.Type != game.UnitTypeFleet || breBuild.Coast != "bre" {
		t.Fatalf("bre build = %+v, want fleet at bre with inferred coast bre", breBuild)
	}
}

func TestResolveAdjustments_DisbandsExactlyBalance(t *testing.T) {
	gm := loadTestMap(t)
	g := newAdjustmentTestGame(gm,
		map[gamemap.NationID]game.PlayerID{"fra": "pf"},
		[]testUnit{
			tArmy("fra-army-par", "fra", "par"),
			tFleet("fra-fleet-bre", "fra", "bre", "bre"),
			tArmy("fra-army-gas", "fra", "gas"),
		},
		map[gamemap.ProvinceID]gamemap.NationID{"par": "fra", "bre": "fra"}, // balance -1
		[]game.Order{game.NewDisbandOrder("fra-army-gas", "fra")},
		1,
	)

	res, err := ResolveAdjustments(g, gm)
	if err != nil {
		t.Fatalf("ResolveAdjustments failed: %v", err)
	}
	if len(res.Builds) != 0 {
		t.Fatalf("Builds length = %d, want 0", len(res.Builds))
	}
	if len(res.Disbands) != 1 || res.Disbands[0] != "fra-army-gas" {
		t.Fatalf("Disbands = %v, want [fra-army-gas]", res.Disbands)
	}
}

func TestResolveAdjustments_RejectsOverBuilding(t *testing.T) {
	gm := loadTestMap(t)
	g := newAdjustmentTestGame(gm,
		map[gamemap.NationID]game.PlayerID{"fra": "pf"},
		[]testUnit{tArmy("fra-army-gas", "fra", "gas")},
		map[gamemap.ProvinceID]gamemap.NationID{"par": "fra", "bre": "fra"}, // balance +1
		[]game.Order{
			game.NewBuildOrder("fra", "par", game.UnitTypeArmy, ""),
			game.NewBuildOrder("fra", "bre", game.UnitTypeFleet, ""),
		},
		1,
	)

	assertResolveAdjustmentsErrorContains(t, g, gm, "submitted 2 builds but is owed 1")
}

func TestResolveAdjustments_RejectsUnderDisbanding(t *testing.T) {
	gm := loadTestMap(t)
	g := newAdjustmentTestGame(gm,
		map[gamemap.NationID]game.PlayerID{"fra": "pf"},
		[]testUnit{
			tArmy("fra-army-par", "fra", "par"),
			tFleet("fra-fleet-bre", "fra", "bre", "bre"),
			tArmy("fra-army-gas", "fra", "gas"),
		},
		map[gamemap.ProvinceID]gamemap.NationID{"par": "fra", "bre": "fra"}, // balance -1
		nil, // no disband submitted
		1,
	)

	assertResolveAdjustmentsErrorContains(t, g, gm, "owes 1 disbands but submitted 0")
}

func TestResolveAdjustments_RejectsOverDisbanding(t *testing.T) {
	gm := loadTestMap(t)
	g := newAdjustmentTestGame(gm,
		map[gamemap.NationID]game.PlayerID{"fra": "pf"},
		[]testUnit{
			tArmy("fra-army-par", "fra", "par"),
			tFleet("fra-fleet-bre", "fra", "bre", "bre"),
			tArmy("fra-army-gas", "fra", "gas"),
		},
		map[gamemap.ProvinceID]gamemap.NationID{"par": "fra", "bre": "fra"}, // balance -1
		[]game.Order{
			game.NewDisbandOrder("fra-army-gas", "fra"),
			game.NewDisbandOrder("fra-army-par", "fra"),
		},
		1,
	)

	assertResolveAdjustmentsErrorContains(t, g, gm, "owes 1 disbands but submitted 2")
}

func TestResolveAdjustments_RejectsMapMismatch(t *testing.T) {
	gm := loadTestMap(t)
	g := newAdjustmentTestGame(gm, nil, nil, nil, nil, 1)
	g.MapID = "other-map"

	if _, err := ResolveAdjustments(g, gm); err == nil {
		t.Fatal("expected ResolveAdjustments to reject a mismatched map")
	}
}

func assertResolveAdjustmentsErrorContains(t *testing.T, g *game.Game, gm *gamemap.GameMap, want string) {
	t.Helper()

	_, err := ResolveAdjustments(g, gm)
	if err == nil {
		t.Fatal("expected ResolveAdjustments to fail")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("ResolveAdjustments error = %q, want substring %q", err.Error(), want)
	}
}
