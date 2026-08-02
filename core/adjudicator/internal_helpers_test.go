package adjudicator

import (
	"os"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// testUnit describes a unit to place on the board for an internal stage test.
type testUnit struct {
	id       game.UnitID
	nation   gamemap.NationID
	province gamemap.ProvinceID
	kind     game.UnitType
	coast    gamemap.CoastID
}

func tArmy(id game.UnitID, nation gamemap.NationID, province gamemap.ProvinceID) testUnit {
	return testUnit{id: id, nation: nation, province: province, kind: game.UnitTypeArmy}
}

func tFleet(id game.UnitID, nation gamemap.NationID, province gamemap.ProvinceID, coast gamemap.CoastID) testUnit {
	return testUnit{id: id, nation: nation, province: province, kind: game.UnitTypeFleet, coast: coast}
}

// newTestGame builds a game directly from unit specs and orders. It bypasses
// SubmitOrder validation so stage tests can construct arbitrary board states.
func newTestGame(gm *gamemap.GameMap, units []testUnit, orders ...game.UnitOrder) *game.Game {
	g := &game.Game{
		MapID: gm.ID,
		Units: make(map[game.UnitID]game.Unit, len(units)),
	}

	for _, u := range units {
		unit := game.Unit{ID: u.id, NationID: u.nation, ProvinceID: u.province, Type: u.kind}
		if u.kind == game.UnitTypeFleet {
			unit.Coast = u.coast
		}
		g.Units[u.id] = unit
	}

	for _, o := range orders {
		g.Orders = append(g.Orders, o)
	}

	return g
}

// prunedContext builds a context and runs the pipeline through pruning.
func prunedContext(gm *gamemap.GameMap, units []testUnit, orders ...game.UnitOrder) resolutionContext {
	rc := newResolutionContext(newTestGame(gm, units, orders...), gm)
	rc.normalizeOrders()
	rc.categorizeOrders()
	rc.pruneMisalignedOrders()
	return rc
}

func loadTestMap(t *testing.T) *gamemap.GameMap {
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
