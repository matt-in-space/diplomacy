package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
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
			name: "committed nation",
			edit: func(g *game.Game, gm *gamemap.GameMap) (game.Order, *gamemap.GameMap) {
				g.CommittedOrders["fra"] = struct{}{}
				return game.NewHoldOrder("fra-army-par-start", "fra"), gm
			},
			want: "nation \"fra\" has already been committed",
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
				dislodge(t, g, "fra-army-par-start")
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

// retreatScenarioGame builds a minimal game in the accept retreats phase: an
// army dislodged from "par" by an attacker that came from "gas" (so "gas" is
// excluded from legal retreats and "bre" is the only legal destination).
func retreatScenarioGame(gm *gamemap.GameMap) *game.Game {
	return &game.Game{
		MapID: gm.ID,
		Turn:  game.Turn{Season: game.Spring, Phase: game.AcceptRetreats, Year: 1},
		Units: map[game.UnitID]game.Unit{
			"fra-army-par": {
				ID: "fra-army-par", NationID: "fra", Type: game.UnitTypeArmy,
				DislodgedFrom: "par",
			},
			"eng-army-attacker": {
				ID: "eng-army-attacker", NationID: "eng", Type: game.UnitTypeArmy,
				ProvinceID: "par",
			},
		},
		LastOrderResolution: game.Resolution{
			"eng-army-attacker": game.Outcome{
				UnitID: "eng-army-attacker",
				Unit:   game.UnitTransform{UnitID: "eng-army-attacker", Type: game.UnitTransformMove, From: "gas", To: "par"},
				Order:  game.CreateOrderSuccessOutcome(game.NewMoveOrder("eng-army-attacker", "eng", "par", "")),
			},
		},
		Orders:          make(map[game.UnitID]game.Order),
		CommittedOrders: make(map[gamemap.NationID]struct{}),
	}
}

func TestGameSubmitOrder_AcceptsRetreatOrder(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := retreatScenarioGame(gm)

	order := game.NewRetreatOrder("fra-army-par", "fra", "bre", "")
	if err := g.SubmitOrder(order, gm); err != nil {
		t.Fatalf("SubmitOrder failed: %v", err)
	}
	got, ok := g.Orders[order.Unit()].(game.RetreatOrder)
	if !ok {
		t.Fatalf("expected stored order to be RetreatOrder, got %T", g.Orders[order.Unit()])
	}
	if got != order {
		t.Fatalf("stored order = %+v, want %+v", got, order)
	}
}

func TestGameSubmitOrder_AcceptsDisbandOrder(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := retreatScenarioGame(gm)

	order := game.NewDisbandOrder("fra-army-par", "fra")
	if err := g.SubmitOrder(order, gm); err != nil {
		t.Fatalf("SubmitOrder failed: %v", err)
	}
	if _, ok := g.Orders[order.Unit()].(game.DisbandOrder); !ok {
		t.Fatalf("expected stored order to be DisbandOrder, got %T", g.Orders[order.Unit()])
	}
}

func TestGameSubmitOrder_RejectsInvalidRetreatOrders(t *testing.T) {
	tests := []struct {
		name  string
		order func() game.Order
		want  string
	}{
		{
			name:  "destination is the attacker's origin",
			order: func() game.Order { return game.NewRetreatOrder("fra-army-par", "fra", "gas", "") },
			want:  "attacking unit's origin",
		},
		{
			name:  "unit is not dislodged",
			order: func() game.Order { return game.NewRetreatOrder("eng-army-attacker", "eng", "bre", "") },
			want:  "is not dislodged",
		},
		{
			name:  "movement order during retreats",
			order: func() game.Order { return game.NewMoveOrder("fra-army-par", "fra", "bre", "") },
			want:  "unsupported order type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := loadWesternEuropeMap(t)
			g := retreatScenarioGame(gm)
			assertSubmitOrderErrorContains(t, g, tt.order(), gm, tt.want)
		})
	}
}

func TestGameSubmitOrder_RejectsRetreatOrderDuringMovementPhase(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)

	order := game.NewRetreatOrder("fra-army-par-start", "fra", "gas", "")
	assertSubmitOrderErrorContains(t, g, order, gm, "unsupported order type")
}
