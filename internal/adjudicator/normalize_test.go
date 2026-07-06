package adjudicator

import (
	"os"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

func TestNormalizeOrders_AddsImplicitHoldOrders(t *testing.T) {
	g, gm := newTestResolutionGame(t)
	ctx := newContext(g, gm)

	effectiveOrders := normalizeOrders(ctx)

	if len(effectiveOrders) != len(g.Units) {
		t.Fatalf("effectiveOrders length = %d, want %d", len(effectiveOrders), len(g.Units))
	}

	for unitID, unit := range g.Units {
		order, ok := effectiveOrders[unitID]
		if !ok {
			t.Fatalf("missing effective order for unit %q", unitID)
		}

		holdOrder, ok := order.(game.HoldOrder)
		if !ok {
			t.Fatalf("effective order for unit %q = %T, want game.HoldOrder", unitID, order)
		}
		if holdOrder.Unit() != unitID {
			t.Fatalf("hold order unit = %q, want %q", holdOrder.Unit(), unitID)
		}
		if holdOrder.Nation() != unit.NationID {
			t.Fatalf("hold order nation = %q, want %q", holdOrder.Nation(), unit.NationID)
		}
	}
}

func TestNormalizeOrders_KeepsSubmittedOrders(t *testing.T) {
	g, gm := newTestResolutionGame(t)
	moveOrder := game.NewMoveOrder("fra-army-par-start", "fra", "gas", "")
	g.Orders[moveOrder.Unit()] = moveOrder
	ctx := newContext(g, gm)

	effectiveOrders := normalizeOrders(ctx)

	got := effectiveOrders[moveOrder.Unit()]
	if got != moveOrder {
		t.Fatalf("effective order = %+v, want %+v", got, moveOrder)
	}

	implicitOrder := effectiveOrders[game.UnitID("fra-fleet-bre-start")]
	if _, ok := implicitOrder.(game.HoldOrder); !ok {
		t.Fatalf("expected missing unit order to become HoldOrder, got %T", implicitOrder)
	}
}

func TestNormalizeOrders_DoesNotMutateGameOrders(t *testing.T) {
	g, gm := newTestResolutionGame(t)
	ctx := newContext(g, gm)

	_ = normalizeOrders(ctx)

	if len(g.Orders) != 0 {
		t.Fatalf("game orders length = %d, want 0", len(g.Orders))
	}
}

func newTestResolutionGame(t *testing.T) (*game.Game, *gamemap.GameMap) {
	t.Helper()

	gm := loadTestWesternEuropeMap(t)
	g, err := game.NewGame(game.NewGameConfig{
		ID: "game-1",
		Assignments: map[gamemap.NationID]game.PlayerID{
			"eng": "player-1",
			"fra": "player-2",
		},
	}, gm)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}
	g.Turn.Phase = game.ResolveOrders

	return g, gm
}

func loadTestWesternEuropeMap(t *testing.T) *gamemap.GameMap {
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
