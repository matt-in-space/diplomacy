package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

func TestGameSubmitOrder_ReplacesExistingOrder(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	unitID := game.UnitID("fra-army-par-start")
	g.Orders[unitID] = testOrder{unitID: unitID, nationID: "fra"}

	order := game.NewHoldOrder(unitID, "fra")
	if err := g.SubmitOrder(order, gm); err != nil {
		t.Fatalf("SubmitOrder failed: %v", err)
	}

	got, ok := g.Orders[unitID].(game.HoldOrder)
	if !ok {
		t.Fatalf("expected replacement order to be HoldOrder, got %T", g.Orders[unitID])
	}
	if got != order {
		t.Fatalf("stored order = %+v, want %+v", got, order)
	}
}

func TestGameSubmitOrder_RejectsInvalidOrders(t *testing.T) {
	tests := []struct {
		name string
		edit func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap)
		want string
	}{
		{
			name: "nil order",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				return nil, gm
			},
			want: "order is required",
		},
		{
			name: "nil map",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				return game.NewHoldOrder("fra-army-par-start", "fra"), nil
			},
			want: "game map is required",
		},
		{
			name: "map mismatch",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				return game.NewHoldOrder("fra-army-par-start", "fra"), &gamemap.GameMap{ID: "other-map"}
			},
			want: "does not match",
		},
		{
			name: "wrong phase",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				g.Turn.Phase = game.ResolveOrders
				return game.NewHoldOrder("fra-army-par-start", "fra"), gm
			},
			want: "cannot submit order during phase",
		},
		{
			name: "unknown nation",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				return game.NewHoldOrder("fra-army-par-start", "ita"), gm
			},
			want: "order nation \"ita\" not found",
		},
		{
			name: "unknown unit",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				return game.NewHoldOrder("missing", "fra"), gm
			},
			want: "unit \"missing\" not found",
		},
		{
			name: "wrong nation for unit",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				return game.NewHoldOrder("eng-fleet-lon-start", "fra"), gm
			},
			want: "belongs to nation \"eng\", not \"fra\"",
		},
		{
			name: "unit not on board",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				delete(g.Positions, "par")
				return game.NewHoldOrder("fra-army-par-start", "fra"), gm
			},
			want: "is not on the board",
		},
		{
			name: "unsupported order type",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				return testOrder{unitID: "fra-army-par-start", nationID: "fra"}, gm
			},
			want: "unsupported order type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := newWesternEuropeGame(t, gm)
			order, orderMap := tt.edit(g, gm)
			assertSubmitOrderErrorContains(t, g, order, orderMap, tt.want)
		})
	}
}
