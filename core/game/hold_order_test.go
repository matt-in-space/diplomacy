package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
)

func TestGameSubmitOrder_AcceptsHoldOrder(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	order := game.NewHoldOrder("fra-army-par-start", "fra")

	if err := g.SubmitOrder(order, gm); err != nil {
		t.Fatalf("SubmitOrder failed: %v", err)
	}

	got, ok := g.Orders["fra-army-par-start"]
	if !ok {
		t.Fatalf("expected order to be stored")
	}
	if got != order {
		t.Fatalf("stored order = %+v, want %+v", got, order)
	}
}
