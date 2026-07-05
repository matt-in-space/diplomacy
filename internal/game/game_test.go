package game_test

import (
	"os"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

func TestNewGame_CreatesGameFromMapSetup(t *testing.T) {
	gm := loadWesternEuropeMap(t)

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

	if g.ID != "game-1" {
		t.Fatalf("ID = %q, want game-1", g.ID)
	}
	if g.MapID != gm.ID {
		t.Fatalf("MapID = %q, want %q", g.MapID, gm.ID)
	}
	if g.Turn != game.StartingTurn() {
		t.Fatalf("Turn = %+v, want %+v", g.Turn, game.StartingTurn())
	}
	if len(g.Orders) != 0 {
		t.Fatalf("Orders length = %d, want 0", len(g.Orders))
	}
	if len(g.Units) != 3 {
		t.Fatalf("Units length = %d, want 3", len(g.Units))
	}

	assertUnit(t, g, "fra-army-par-start", game.Unit{
		ID:         "fra-army-par-start",
		NationID:   "fra",
		ProvinceID: "par",
		Type:       game.UnitTypeArmy,
	})
	assertUnit(t, g, "fra-fleet-bre-start", game.Unit{
		ID:         "fra-fleet-bre-start",
		NationID:   "fra",
		ProvinceID: "bre",
		Type:       game.UnitTypeFleet,
	})
	assertUnit(t, g, "eng-fleet-lon-start", game.Unit{
		ID:         "eng-fleet-lon-start",
		NationID:   "eng",
		ProvinceID: "lon",
		Type:       game.UnitTypeFleet,
	})

	if got := g.Positions["par"]; got != "fra-army-par-start" {
		t.Fatalf("Positions[par] = %q, want fra-army-par-start", got)
	}
	if got := g.Positions["bre"]; got != "fra-fleet-bre-start" {
		t.Fatalf("Positions[bre] = %q, want fra-fleet-bre-start", got)
	}
	if got := g.FleetCoasts["fra-fleet-bre-start"]; got != "bre" {
		t.Fatalf("FleetCoasts[fra-fleet-bre-start] = %q, want bre", got)
	}
	if _, ok := g.FleetCoasts["fra-army-par-start"]; ok {
		t.Fatalf("army should not have fleet coast")
	}
}

func TestNewGame_CopiesAssignments(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	assignments := map[gamemap.NationID]game.PlayerID{
		"eng": "player-1",
	}

	g, err := game.NewGame(game.NewGameConfig{
		ID:          "game-1",
		Assignments: assignments,
	}, gm)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	assignments["eng"] = "changed"
	if g.Assignments["eng"] != "player-1" {
		t.Fatalf("assignment was not copied")
	}
}

func TestNewGame_RejectsUnknownAssignmentNation(t *testing.T) {
	gm := loadWesternEuropeMap(t)

	_, err := game.NewGame(game.NewGameConfig{
		ID: "game-1",
		Assignments: map[gamemap.NationID]game.PlayerID{
			"ita": "player-1",
		},
	}, gm)
	if err == nil {
		t.Fatalf("expected NewGame to fail")
	}
}

func TestNewGame_RejectsNilMap(t *testing.T) {
	_, err := game.NewGame(game.NewGameConfig{ID: "game-1"}, nil)
	if err == nil {
		t.Fatalf("expected NewGame to fail")
	}
}

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

func assertSubmitOrderErrorContains(t *testing.T, g *game.Game, order game.Order, gm *gamemap.GameMap, want string) {
	t.Helper()

	err := g.SubmitOrder(order, gm)
	if err == nil {
		t.Fatalf("expected SubmitOrder to fail")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("SubmitOrder error = %q, want substring %q", err.Error(), want)
	}
}

func addFleet(t *testing.T, g *game.Game, id game.UnitID, nation gamemap.NationID, province gamemap.ProvinceID, coast gamemap.CoastID) {
	t.Helper()

	g.Units[id] = game.Unit{
		ID:         id,
		NationID:   nation,
		ProvinceID: province,
		Type:       game.UnitTypeFleet,
	}
	g.Positions[province] = id
	g.FleetCoasts[id] = coast
}

func assertUnit(t *testing.T, g *game.Game, id game.UnitID, want game.Unit) {
	t.Helper()

	got, ok := g.Units[id]
	if !ok {
		t.Fatalf("unit %q not found", id)
	}
	if got != want {
		t.Fatalf("unit %q = %+v, want %+v", id, got, want)
	}
}

func newWesternEuropeGame(t *testing.T, gm *gamemap.GameMap) *game.Game {
	t.Helper()

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

	return g
}

type testOrder struct {
	unitID   game.UnitID
	nationID gamemap.NationID
}

func (o testOrder) Unit() game.UnitID {
	return o.unitID
}

func (o testOrder) Nation() gamemap.NationID {
	return o.nationID
}

func loadWesternEuropeMap(t *testing.T) *gamemap.GameMap {
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
