package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
)

func TestGameSubmitOrder_AcceptsSupportOrders(t *testing.T) {
	tests := []struct {
		name  string
		order game.Order
	}{
		{
			name:  "support hold",
			order: game.NewSupportHoldOrder("fra-army-par-start", "fra", "fra-fleet-bre-start", "bre"),
		},
		{
			name:  "support move",
			order: game.NewSupportMoveOrder("fra-army-par-start", "fra", "fra-fleet-bre-start", "gas", ""),
		},
		{
			name:  "support another nation",
			order: game.NewSupportMoveOrder("eng-fleet-lon-start", "eng", "fra-fleet-bre-start", "eng", ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := newWesternEuropeGame(t, gm)

			if err := g.SubmitOrder(tt.order, gm); err != nil {
				t.Fatalf("SubmitOrder failed: %v", err)
			}
			if got := g.Orders[tt.order.Unit()]; got != tt.order {
				t.Fatalf("stored order = %+v, want %+v", got, tt.order)
			}
		})
	}
}

func TestGameSubmitOrder_RejectsInvalidSupportOrders(t *testing.T) {
	tests := []struct {
		name  string
		setup func(g *game.Game)
		order game.Order
		want  string
	}{
		{
			name:  "support missing unit",
			setup: func(g *game.Game) {},
			order: game.NewSupportHoldOrder("fra-army-par-start", "fra", "missing", "par"),
			want:  "supported unit \"missing\" not found",
		},
		{
			name:  "support self",
			setup: func(g *game.Game) {},
			order: game.NewSupportHoldOrder("fra-army-par-start", "fra", "fra-army-par-start", "par"),
			want:  "cannot support itself",
		},
		{
			name: "supported unit not on board",
			setup: func(g *game.Game) {
				delete(g.Positions, "bre")
			},
			order: game.NewSupportHoldOrder("fra-army-par-start", "fra", "fra-fleet-bre-start", "bre"),
			want:  "supported unit \"fra-fleet-bre-start\" is not on the board",
		},
		{
			name:  "support hold cannot reach province",
			setup: func(g *game.Game) {},
			order: game.NewSupportHoldOrder("eng-fleet-lon-start", "eng", "fra-army-par-start", "par"),
			want:  "fleet cannot move to inland province",
		},
		{
			name:  "support move to current province",
			setup: func(g *game.Game) {},
			order: game.NewSupportMoveOrder("fra-army-par-start", "fra", "fra-fleet-bre-start", "bre", ""),
			want:  "is supported unit's current province",
		},
		{
			name:  "support move supporter cannot reach target",
			setup: func(g *game.Game) {},
			order: game.NewSupportMoveOrder("fra-army-par-start", "fra", "fra-fleet-bre-start", "mao", ""),
			want:  "army cannot move to water province",
		},
		{
			name:  "support move supported unit cannot move to target",
			setup: func(g *game.Game) {},
			order: game.NewSupportMoveOrder("fra-fleet-bre-start", "fra", "fra-army-par-start", "eng", ""),
			want:  "supported unit \"fra-army-par-start\" cannot move",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := newWesternEuropeGame(t, gm)
			tt.setup(g)
			assertSubmitOrderErrorContains(t, g, tt.order, gm, tt.want)
		})
	}
}
