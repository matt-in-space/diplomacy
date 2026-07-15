package adjudicator

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
)

func TestPruneMisalignedOrders_HoldsAndMoves(t *testing.T) {
	gm := loadTestMap(t)

	t.Run("hold stays effective", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "par")},
			game.NewHoldOrder("a", "fra"),
		)
		if _, ok := rc.effectiveHoldOrders["a"]; !ok {
			t.Fatalf("expected hold order to be effective")
		}
	})

	t.Run("direct move stays effective", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "par")},
			game.NewMoveOrder("a", "fra", "gas", ""),
		)
		if _, ok := rc.effectiveMoveOrders["a"]; !ok {
			t.Fatalf("expected move order to be effective")
		}
	})
}

func TestPruneMisalignedOrders_SupportHold(t *testing.T) {
	gm := loadTestMap(t)

	t.Run("aligned when supported unit holds", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("b", "fra", "gas")},
			game.NewHoldOrder("a", "fra"),
			game.NewSupportHoldOrder("b", "fra", "a", "par"),
		)
		if _, ok := rc.effectiveSupportHoldOrders["b"]; !ok {
			t.Fatalf("expected support-hold to be effective")
		}
		assertNoFailure(t, rc, "b")
	})

	t.Run("misaligned when supported unit moves", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("b", "fra", "gas")},
			game.NewMoveOrder("a", "fra", "bre", ""),
			game.NewSupportHoldOrder("b", "fra", "a", "par"),
		)
		assertDemoted(t, rc, "b", ReasonMisalignedSupport)
	})
}

func TestPruneMisalignedOrders_SupportMove(t *testing.T) {
	gm := loadTestMap(t)

	t.Run("aligned when supported unit moves to target", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("b", "fra", "bre")},
			game.NewMoveOrder("a", "fra", "gas", ""),
			game.NewSupportMoveOrder("b", "fra", "a", "gas", ""),
		)
		if _, ok := rc.effectiveSupportMoveOrders["b"]; !ok {
			t.Fatalf("expected support-move to be effective")
		}
		assertNoFailure(t, rc, "b")
	})

	t.Run("misaligned when supported unit holds", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("b", "fra", "bre")},
			game.NewHoldOrder("a", "fra"),
			game.NewSupportMoveOrder("b", "fra", "a", "gas", ""),
		)
		assertDemoted(t, rc, "b", ReasonMisalignedSupport)
	})

	t.Run("misaligned when supported unit moves elsewhere", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "par"), tArmy("b", "fra", "bre")},
			game.NewMoveOrder("a", "fra", "bre", ""),
			game.NewSupportMoveOrder("b", "fra", "a", "gas", ""),
		)
		assertDemoted(t, rc, "b", ReasonMisalignedSupport)
	})
}

func TestPruneMisalignedOrders_Convoy(t *testing.T) {
	gm := loadTestMap(t)

	t.Run("aligned convoy and convoyed move survive", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "bre"), tFleet("f", "fra", "eng", "eng")},
			game.NewConvoyedMoveOrder("a", "fra", "lon"),
			game.NewConvoyOrder("f", "fra", "a", "bre", "lon"),
		)
		if _, ok := rc.effectiveConvoyOrders["f"]; !ok {
			t.Fatalf("expected convoy order to be effective")
		}
		if _, ok := rc.effectiveMoveOrders["a"]; !ok {
			t.Fatalf("expected convoyed move to be effective")
		}
	})

	t.Run("multi-fleet convoy chain survives", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{
				tArmy("a", "fra", "lon"),
				tFleet("f1", "fra", "eng", "eng"),
				tFleet("f2", "fra", "mao", "mao"),
			},
			game.NewConvoyedMoveOrder("a", "fra", "por"),
			game.NewConvoyOrder("f1", "fra", "a", "lon", "por"),
			game.NewConvoyOrder("f2", "fra", "a", "lon", "por"),
		)
		for _, id := range []game.UnitID{"f1", "f2"} {
			if _, ok := rc.effectiveConvoyOrders[id]; !ok {
				t.Fatalf("expected convoy %q to be effective", id)
			}
		}
		if _, ok := rc.effectiveMoveOrders["a"]; !ok {
			t.Fatalf("expected convoyed move to be effective")
		}
	})

	t.Run("convoy misaligned when convoyed unit is not moving", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "bre"), tFleet("f", "fra", "eng", "eng")},
			game.NewHoldOrder("a", "fra"),
			game.NewConvoyOrder("f", "fra", "a", "bre", "lon"),
		)
		assertDemoted(t, rc, "f", ReasonMisalignedConvoy)
	})

	t.Run("convoy misaligned when move is not via convoy", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "bre"), tFleet("f", "fra", "eng", "eng")},
			game.NewMoveOrder("a", "fra", "gas", ""),
			game.NewConvoyOrder("f", "fra", "a", "bre", "lon"),
		)
		assertDemoted(t, rc, "f", ReasonMisalignedConvoy)
	})

	t.Run("convoy misaligned when destinations differ", func(t *testing.T) {
		// One aligned carrier (bre->lon via eng) keeps the army moving, while a
		// second carrier convoying to a different destination is misaligned.
		rc := prunedContext(gm,
			[]testUnit{
				tArmy("a", "fra", "bre"),
				tFleet("f1", "fra", "eng", "eng"),
				tFleet("f2", "fra", "mao", "mao"),
			},
			game.NewConvoyedMoveOrder("a", "fra", "lon"),
			game.NewConvoyOrder("f1", "fra", "a", "bre", "lon"),
			game.NewConvoyOrder("f2", "fra", "a", "bre", "por"),
		)
		assertDemoted(t, rc, "f2", ReasonMisalignedConvoy)
		if _, ok := rc.effectiveConvoyOrders["f1"]; !ok {
			t.Fatalf("expected aligned convoy f1 to be effective")
		}
		if _, ok := rc.effectiveMoveOrders["a"]; !ok {
			t.Fatalf("expected convoyed move to be effective")
		}
	})

	t.Run("convoyed move fails with no carriers", func(t *testing.T) {
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "bre")},
			game.NewConvoyedMoveOrder("a", "fra", "lon"),
		)
		assertDemoted(t, rc, "a", ReasonConvoyFailure)
		if _, ok := rc.effectiveMoveOrders["a"]; ok {
			t.Fatalf("did not expect failed convoyed move to be effective")
		}
	})

	t.Run("convoyed move and carriers fail with no complete path", func(t *testing.T) {
		// eng alone cannot reach por (needs mao), so the whole convoy is doomed.
		rc := prunedContext(gm,
			[]testUnit{tArmy("a", "fra", "lon"), tFleet("f", "fra", "eng", "eng")},
			game.NewConvoyedMoveOrder("a", "fra", "por"),
			game.NewConvoyOrder("f", "fra", "a", "lon", "por"),
		)
		assertDemoted(t, rc, "a", ReasonConvoyFailure)
		assertDemoted(t, rc, "f", ReasonConvoyFailure)
		if _, ok := rc.effectiveConvoyOrders["f"]; ok {
			t.Fatalf("expected doomed carrier to be removed from effective convoys")
		}
	})
}

// TestPruneMisalignedOrders_Partition verifies every unit ends up in exactly
// one effective order map after pruning.
func TestPruneMisalignedOrders_Partition(t *testing.T) {
	gm := loadTestMap(t)

	rc := prunedContext(gm,
		[]testUnit{
			tArmy("hold", "fra", "par"),
			tArmy("mover", "fra", "gas"),
			tArmy("supH", "fra", "spa"),
			tArmy("supM", "fra", "bre"),
			tFleet("conv", "fra", "eng", "eng"),
			tArmy("convoyed", "fra", "lon"),
		},
		game.NewHoldOrder("hold", "fra"),
		game.NewMoveOrder("mover", "fra", "por", ""),
		game.NewSupportHoldOrder("supH", "fra", "hold", "par"),
		game.NewSupportMoveOrder("supM", "fra", "mover", "por", ""),
		game.NewConvoyedMoveOrder("convoyed", "fra", "bre"),
		game.NewConvoyOrder("conv", "fra", "convoyed", "lon", "bre"),
	)

	for id := range rc.units {
		count := 0
		if _, ok := rc.effectiveHoldOrders[id]; ok {
			count++
		}
		if _, ok := rc.effectiveMoveOrders[id]; ok {
			count++
		}
		if _, ok := rc.effectiveSupportHoldOrders[id]; ok {
			count++
		}
		if _, ok := rc.effectiveSupportMoveOrders[id]; ok {
			count++
		}
		if _, ok := rc.effectiveConvoyOrders[id]; ok {
			count++
		}
		if count != 1 {
			t.Errorf("unit %q appears in %d effective maps, want exactly 1", id, count)
		}
	}
}

func assertDemoted(t *testing.T, rc resolutionContext, id game.UnitID, reason ReasonCode) {
	t.Helper()

	if _, ok := rc.effectiveHoldOrders[id]; !ok {
		t.Errorf("expected unit %q to be demoted to an effective hold", id)
	}
	outcome, ok := rc.orderOutcomes[id]
	if !ok {
		t.Fatalf("expected a failed outcome for unit %q", id)
	}
	if outcome.Success {
		t.Errorf("expected outcome for %q to be unsuccessful", id)
	}
	if outcome.Reason != reason {
		t.Errorf("outcome reason for %q = %q, want %q", id, outcome.Reason, reason)
	}
}

func assertNoFailure(t *testing.T, rc resolutionContext, id game.UnitID) {
	t.Helper()

	if _, ok := rc.orderOutcomes[id]; ok {
		t.Errorf("did not expect a failure outcome for unit %q", id)
	}
}
