package game_test

import (
	"os"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

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

func addArmy(t *testing.T, g *game.Game, id game.UnitID, nation gamemap.NationID, province gamemap.ProvinceID) {
	t.Helper()

	g.Units[id] = game.Unit{
		ID:         id,
		NationID:   nation,
		ProvinceID: province,
		Type:       game.UnitTypeArmy,
	}
	g.Positions[province] = id
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
