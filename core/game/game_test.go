package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
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

func TestGameCloneCopiesReferenceState(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.PendingRetreats["fra-army-par-start"] = game.Dislodgement{From: "par"}

	clone := g.Clone()
	if clone == g {
		t.Fatal("Clone returned the original game pointer")
	}

	clone.Assignments["eng"] = "changed-player"
	unit := clone.Units["fra-army-par-start"]
	unit.ProvinceID = "bur"
	clone.Units[unit.ID] = unit
	delete(clone.Positions, "par")
	clone.FleetCoasts["fra-fleet-bre-start"] = "changed-coast"
	clone.Orders["fra-army-par-start"] = game.NewHoldOrder("fra-army-par-start", "fra")
	delete(clone.PendingRetreats, "fra-army-par-start")

	if got := g.Assignments["eng"]; got != "player-1" {
		t.Fatalf("original assignment = %q, want player-1", got)
	}
	if got := g.Units["fra-army-par-start"].ProvinceID; got != "par" {
		t.Fatalf("original unit province = %q, want par", got)
	}
	if got := g.Positions["par"]; got != "fra-army-par-start" {
		t.Fatalf("original position = %q, want fra-army-par-start", got)
	}
	if got := g.FleetCoasts["fra-fleet-bre-start"]; got != "bre" {
		t.Fatalf("original fleet coast = %q, want bre", got)
	}
	if _, ok := g.Orders["fra-army-par-start"]; ok {
		t.Fatal("clone order was added to original game")
	}
	if _, ok := g.PendingRetreats["fra-army-par-start"]; !ok {
		t.Fatal("clone retreat deletion affected original game")
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
