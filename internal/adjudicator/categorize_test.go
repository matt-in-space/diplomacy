package adjudicator

import (
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

func TestCategorizeOrders(t *testing.T) {
	orders := map[game.UnitID]game.Order{
		"hold":         game.NewHoldOrder("hold", "fra"),
		"move":         game.NewMoveOrder("move", "fra", "gas", ""),
		"support-hold": game.NewSupportHoldOrder("support-hold", "fra", "hold"),
		"support-move": game.NewSupportMoveOrder("support-move", "fra", "move", "gas", ""),
		"convoy":       game.NewConvoyOrder("convoy", "fra", "move", "gas", "lon"),
	}

	categorized, err := categorizeOrders(orders)
	if err != nil {
		t.Fatalf("categorizeOrders failed: %v", err)
	}

	assertCategorizedOrder(t, categorized.holds, game.UnitID("hold"))
	assertCategorizedOrder(t, categorized.moves, game.UnitID("move"))
	assertCategorizedOrder(t, categorized.supportHolds, game.UnitID("support-hold"))
	assertCategorizedOrder(t, categorized.supportMoves, game.UnitID("support-move"))
	assertCategorizedOrder(t, categorized.convoys, game.UnitID("convoy"))
}

func TestCategorizeOrders_RejectsUnsupportedOrder(t *testing.T) {
	orders := map[game.UnitID]game.Order{
		"unsupported": unsupportedOrder{unitID: "unsupported", nationID: "fra"},
	}

	_, err := categorizeOrders(orders)
	if err == nil {
		t.Fatalf("expected categorizeOrders to fail")
	}
	if !strings.Contains(err.Error(), "unsupported order type") {
		t.Fatalf("categorizeOrders error = %q, want unsupported order type", err.Error())
	}
}

func assertCategorizedOrder[T any](t *testing.T, orders map[game.UnitID]T, unitID game.UnitID) {
	t.Helper()

	if _, ok := orders[unitID]; !ok {
		t.Fatalf("expected categorized orders to contain unit %q", unitID)
	}
}

type unsupportedOrder struct {
	unitID   game.UnitID
	nationID gamemap.NationID
}

func (o unsupportedOrder) Unit() game.UnitID {
	return o.unitID
}

func (o unsupportedOrder) Nation() gamemap.NationID {
	return o.nationID
}
