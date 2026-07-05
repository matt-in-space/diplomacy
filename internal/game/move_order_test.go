package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
)

func TestGameSubmitOrder_AcceptsMoveOrder(t *testing.T) {
	tests := []struct {
		name  string
		setup func(g *game.Game)
		order game.MoveOrder
	}{
		{
			name:  "army move",
			setup: func(g *game.Game) {},
			order: game.NewMoveOrder("fra-army-par-start", "fra", "gas", ""),
		},
		{
			name:  "fleet move with inferred coast",
			setup: func(g *game.Game) {},
			order: game.NewMoveOrder("fra-fleet-bre-start", "fra", "mao", ""),
		},
		{
			name: "fleet move with explicit bicoastal target",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
			},
			order: game.NewMoveOrder("fra-fleet-mao-test", "fra", "spa", "spa-nc"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := newWesternEuropeGame(t, gm)
			tt.setup(g)

			if err := g.SubmitOrder(tt.order, gm); err != nil {
				t.Fatalf("SubmitOrder failed: %v", err)
			}
			if got := g.Orders[tt.order.Unit()]; got != tt.order {
				t.Fatalf("stored order = %+v, want %+v", got, tt.order)
			}
		})
	}
}

func TestGameSubmitOrder_RejectsInvalidMoveOrders(t *testing.T) {
	tests := []struct {
		name  string
		setup func(g *game.Game)
		order game.MoveOrder
		want  string
	}{
		{
			name:  "unknown target province",
			setup: func(g *game.Game) {},
			order: game.NewMoveOrder("fra-army-par-start", "fra", "missing", ""),
			want:  "target province \"missing\" not found",
		},
		{
			name:  "move to current province",
			setup: func(g *game.Game) {},
			order: game.NewMoveOrder("fra-army-par-start", "fra", "par", ""),
			want:  "cannot move to its current province",
		},
		{
			name:  "army with target coast",
			setup: func(g *game.Game) {},
			order: game.NewMoveOrder("fra-army-par-start", "fra", "gas", "gas"),
			want:  "army move cannot specify target coast",
		},
		{
			name:  "army to water",
			setup: func(g *game.Game) {},
			order: game.NewMoveOrder("fra-army-par-start", "fra", "mao", ""),
			want:  "army cannot move to water province",
		},
		{
			name:  "army non-adjacent move",
			setup: func(g *game.Game) {},
			order: game.NewMoveOrder("fra-army-par-start", "fra", "spa", ""),
			want:  "army cannot move from",
		},
		{
			name:  "fleet to inland",
			setup: func(g *game.Game) {},
			order: game.NewMoveOrder("fra-fleet-bre-start", "fra", "par", ""),
			want:  "fleet cannot move to inland province",
		},
		{
			name: "fleet missing source coast",
			setup: func(g *game.Game) {
				delete(g.FleetCoasts, "fra-fleet-bre-start")
			},
			order: game.NewMoveOrder("fra-fleet-bre-start", "fra", "mao", ""),
			want:  "has no source coast",
		},
		{
			name: "fleet missing bicoastal target coast",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
			},
			order: game.NewMoveOrder("fra-fleet-mao-test", "fra", "spa", ""),
			want:  "requires target coast",
		},
		{
			name: "fleet target coast from another province",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
			},
			order: game.NewMoveOrder("fra-fleet-mao-test", "fra", "spa", "por"),
			want:  "does not belong to province",
		},
		{
			name:  "fleet non-adjacent move",
			setup: func(g *game.Game) {},
			order: game.NewMoveOrder("eng-fleet-lon-start", "eng", "mao", ""),
			want:  "fleet cannot move from coast",
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
