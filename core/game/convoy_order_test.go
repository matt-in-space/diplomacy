package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
)

func TestGameSubmitOrder_AcceptsConvoyOrder(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	addArmy(t, g, "eng-army-gas-test", "eng", "gas")
	addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")

	order := game.NewConvoyOrder("fra-fleet-mao-test", "fra", "eng-army-gas-test", "gas", "lon")
	if err := g.SubmitOrder(order, gm); err != nil {
		t.Fatalf("SubmitOrder failed: %v", err)
	}
	if got := g.Orders[order.Unit()]; got != order {
		t.Fatalf("stored order = %+v, want %+v", got, order)
	}
}

func TestGameSubmitOrder_RejectsInvalidConvoyOrders(t *testing.T) {
	tests := []struct {
		name  string
		setup func(g *game.Game)
		order game.ConvoyOrder
		want  string
	}{
		{
			name:  "convoying unit must be fleet",
			setup: func(g *game.Game) {},
			order: game.NewConvoyOrder("fra-army-par-start", "fra", "eng-army-gas-test", "gas", "lon"),
			want:  "must be a fleet to convoy",
		},
		{
			name:  "convoying fleet must be in water",
			setup: func(g *game.Game) {},
			order: game.NewConvoyOrder("fra-fleet-bre-start", "fra", "eng-army-gas-test", "gas", "lon"),
			want:  "must be in a water province to convoy",
		},
		{
			name: "missing convoyed unit",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
			},
			order: game.NewConvoyOrder("fra-fleet-mao-test", "fra", "missing", "gas", "lon"),
			want:  "convoyed unit \"missing\" not found",
		},
		{
			name: "convoyed unit must be army",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
			},
			order: game.NewConvoyOrder("fra-fleet-mao-test", "fra", "fra-fleet-bre-start", "bre", "lon"),
			want:  "must be an army",
		},
		{
			name: "convoyed unit not on board",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
				addArmy(t, g, "eng-army-gas-test", "eng", "gas")
				delete(g.Positions, "gas")
			},
			order: game.NewConvoyOrder("fra-fleet-mao-test", "fra", "eng-army-gas-test", "gas", "lon"),
			want:  "is not on the board",
		},
		{
			name: "from must match army province",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
				addArmy(t, g, "eng-army-gas-test", "eng", "gas")
			},
			order: game.NewConvoyOrder("fra-fleet-mao-test", "fra", "eng-army-gas-test", "bre", "lon"),
			want:  "does not match convoyed unit province",
		},
		{
			name: "destination cannot equal origin",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
				addArmy(t, g, "eng-army-gas-test", "eng", "gas")
			},
			order: game.NewConvoyOrder("fra-fleet-mao-test", "fra", "eng-army-gas-test", "gas", "gas"),
			want:  "destination cannot be the origin",
		},
		{
			name: "origin must be coastal",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
				addArmy(t, g, "eng-army-par-test", "eng", "par")
			},
			order: game.NewConvoyOrder("fra-fleet-mao-test", "fra", "eng-army-par-test", "par", "lon"),
			want:  "convoy origin province \"par\" must be coastal",
		},
		{
			name: "destination must be coastal",
			setup: func(g *game.Game) {
				addFleet(t, g, "fra-fleet-mao-test", "fra", "mao", "mao")
				addArmy(t, g, "eng-army-gas-test", "eng", "gas")
			},
			order: game.NewConvoyOrder("fra-fleet-mao-test", "fra", "eng-army-gas-test", "gas", "par"),
			want:  "convoy destination province \"par\" must be coastal",
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
