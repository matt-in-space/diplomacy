package adjudicator

import (
	"slices"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
)

func TestBuildDependencyGraph_SupportEdges(t *testing.T) {
	gm := loadTestMap(t)

	t.Run("support-move depends on the supported move", func(t *testing.T) {
		rc := graphContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("b", "fra", "bre")},
			game.NewMoveOrder("a", "fra", "gas", ""),
			game.NewSupportMoveOrder("b", "fra", "a", "gas", ""),
		)
		assertEdge(t, rc, "a", "b")
		assertIndegree(t, rc, "b", 1)
	})

	t.Run("support-hold depends on the supported hold", func(t *testing.T) {
		rc := graphContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("b", "fra", "gas")},
			game.NewHoldOrder("a", "fra"),
			game.NewSupportHoldOrder("b", "fra", "a", "par"),
		)
		assertEdge(t, rc, "a", "b")
		assertIndegree(t, rc, "b", 1)
	})

	t.Run("demoted support creates no edge", func(t *testing.T) {
		rc := graphContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("b", "fra", "bre")},
			game.NewHoldOrder("a", "fra"),
			game.NewSupportMoveOrder("b", "fra", "a", "gas", ""), // misaligned: a holds
		)
		assertNoEdge(t, rc, "a", "b")
		assertIndegree(t, rc, "b", 0)
	})
}

func TestBuildDependencyGraph_ConvoyEdges(t *testing.T) {
	gm := loadTestMap(t)

	t.Run("convoy enables the convoyed move", func(t *testing.T) {
		rc := graphContext(gm,
			[]testUnit{tArmy("a", "fra", "bre"), tFleet("f", "fra", "eng", "eng")},
			game.NewConvoyedMoveOrder("a", "fra", "lon"),
			game.NewConvoyOrder("f", "fra", "a", "bre", "lon"),
		)
		assertEdge(t, rc, "f", "a")
	})

	t.Run("attack on a convoying fleet influences the convoy", func(t *testing.T) {
		rc := graphContext(gm,
			[]testUnit{
				tArmy("a", "fra", "bre"),
				tFleet("f", "fra", "eng", "eng"),
				tFleet("c", "eng", "lon", "lon"),
			},
			game.NewConvoyedMoveOrder("a", "fra", "lon"),
			game.NewConvoyOrder("f", "fra", "a", "bre", "lon"),
			game.NewMoveOrder("c", "eng", "eng", ""), // attacks the fleet in the channel
		)
		assertEdge(t, rc, "c", "f")
	})
}

func TestBuildDependencyGraph_SupportCutEdges(t *testing.T) {
	gm := loadTestMap(t)

	t.Run("foreign attack on a supporter cuts support and accumulates indegree", func(t *testing.T) {
		rc := graphContext(gm,
			[]testUnit{
				tArmy("a", "fra", "par"),
				tArmy("b", "fra", "gas"),
				tArmy("c", "eng", "spa"),
			},
			game.NewHoldOrder("a", "fra"),
			game.NewSupportHoldOrder("b", "fra", "a", "par"),
			game.NewMoveOrder("c", "eng", "gas", ""), // attacks the supporter
		)
		assertEdge(t, rc, "a", "b") // M -> S
		assertEdge(t, rc, "c", "b") // A -> S
		assertIndegree(t, rc, "b", 2)
	})

	t.Run("own-nation attack on a supporter creates no edge", func(t *testing.T) {
		rc := graphContext(gm,
			[]testUnit{
				tArmy("a", "fra", "par"),
				tArmy("b", "fra", "gas"),
				tArmy("c", "fra", "spa"),
			},
			game.NewHoldOrder("a", "fra"),
			game.NewSupportHoldOrder("b", "fra", "a", "par"),
			game.NewMoveOrder("c", "fra", "gas", ""),
		)
		assertNoEdge(t, rc, "c", "b")
		assertIndegree(t, rc, "b", 1) // only the M -> S edge
	})
}

func TestBuildDependencyGraph_NoConflictEdges(t *testing.T) {
	gm := loadTestMap(t)

	t.Run("two moves into the same empty province are independent", func(t *testing.T) {
		rc := graphContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("c", "eng", "spa")},
			game.NewMoveOrder("a", "fra", "gas", ""),
			game.NewMoveOrder("c", "eng", "gas", ""),
		)
		assertNoEdge(t, rc, "a", "c")
		assertNoEdge(t, rc, "c", "a")
		assertIndegree(t, rc, "a", 0)
		assertIndegree(t, rc, "c", 0)
	})

	t.Run("attack on a plain holder creates no edge", func(t *testing.T) {
		rc := graphContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("h", "eng", "gas")},
			game.NewMoveOrder("a", "fra", "gas", ""),
			game.NewHoldOrder("h", "eng"),
		)
		assertNoEdge(t, rc, "a", "h")
		assertIndegree(t, rc, "h", 0)
	})
}

// TestBuildDependencyGraph_AllUnitsAreNodes verifies every unit is present in
// the graph with an initialized indegree.
func TestBuildDependencyGraph_AllUnitsAreNodes(t *testing.T) {
	gm := loadTestMap(t)

	rc := graphContext(gm,
		[]testUnit{tArmy("a", "fra", "par"), tArmy("b", "fra", "gas"), tArmy("c", "eng", "spa")},
		game.NewHoldOrder("a", "fra"),
		game.NewSupportHoldOrder("b", "fra", "a", "par"),
		game.NewMoveOrder("c", "eng", "gas", ""),
	)

	if len(rc.indegree) != len(rc.units) {
		t.Errorf("indegree has %d nodes, want %d", len(rc.indegree), len(rc.units))
	}
	for id := range rc.units {
		if _, ok := rc.indegree[id]; !ok {
			t.Errorf("unit %q missing from indegree map", id)
		}
	}
}

func assertEdge(t *testing.T, rc resolutionContext, from, to game.UnitID) {
	t.Helper()
	if !slices.Contains(rc.dependents[from], to) {
		t.Errorf("expected dependency edge %q -> %q", from, to)
	}
}

func assertNoEdge(t *testing.T, rc resolutionContext, from, to game.UnitID) {
	t.Helper()
	if slices.Contains(rc.dependents[from], to) {
		t.Errorf("did not expect dependency edge %q -> %q", from, to)
	}
}

func assertIndegree(t *testing.T, rc resolutionContext, id game.UnitID, want int) {
	t.Helper()
	if got := rc.indegree[id]; got != want {
		t.Errorf("indegree[%q] = %d, want %d", id, got, want)
	}
}
