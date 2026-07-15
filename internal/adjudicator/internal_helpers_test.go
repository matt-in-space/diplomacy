package adjudicator

import (
	"os"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
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
func newTestGame(gm *gamemap.GameMap, units []testUnit, orders ...game.Order) *game.Game {
	g := &game.Game{
		MapID:       gm.ID,
		Units:       make(map[game.UnitID]game.Unit, len(units)),
		Positions:   make(map[gamemap.ProvinceID]game.UnitID, len(units)),
		FleetCoasts: make(map[game.UnitID]gamemap.CoastID),
		Orders:      make(map[game.UnitID]game.Order, len(orders)),
	}

	for _, u := range units {
		g.Units[u.id] = game.Unit{ID: u.id, NationID: u.nation, ProvinceID: u.province, Type: u.kind}
		g.Positions[u.province] = u.id
		if u.kind == game.UnitTypeFleet {
			g.FleetCoasts[u.id] = u.coast
		}
	}

	for _, o := range orders {
		g.Orders[o.Unit()] = o
	}

	return g
}

// prunedContext builds a context and runs the pipeline through pruning.
func prunedContext(gm *gamemap.GameMap, units []testUnit, orders ...game.Order) resolutionContext {
	rc := newResolutionContext(newTestGame(gm, units, orders...), gm)
	rc.normalizeOrders()
	rc.categorizeOrders()
	rc.pruneMisalignedOrders()
	return rc
}

// graphContext builds a context and runs the pipeline through graph construction.
func graphContext(gm *gamemap.GameMap, units []testUnit, orders ...game.Order) resolutionContext {
	rc := prunedContext(gm, units, orders...)
	rc.buildDependencyGraph()
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
