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
		Coast:      "bre",
	})
	assertUnit(t, g, "eng-fleet-lon-start", game.Unit{
		ID:         "eng-fleet-lon-start",
		NationID:   "eng",
		ProvinceID: "lon",
		Type:       game.UnitTypeFleet,
		Coast:      "lon",
	})

	if got := g.Units["fra-army-par-start"].Coast; got != "" {
		t.Fatalf("army Coast = %q, want empty", got)
	}

	wantOwners := map[gamemap.ProvinceID]gamemap.NationID{
		"par": "fra",
		"bre": "fra",
		"lon": "eng",
		"spa": "",
		"por": "",
	}
	if len(g.SupplyCenterOwners) != len(wantOwners) {
		t.Fatalf("SupplyCenterOwners length = %d, want %d", len(g.SupplyCenterOwners), len(wantOwners))
	}
	for province, wantNation := range wantOwners {
		if got := g.SupplyCenterOwners[province]; got != wantNation {
			t.Fatalf("SupplyCenterOwners[%q] = %q, want %q", province, got, wantNation)
		}
	}
	if _, ok := g.SupplyCenterOwners["gas"]; ok {
		t.Fatal("SupplyCenterOwners contains gas, which is not a supply center")
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
	g.LastOrderResolution["sentinel"] = game.Outcome{}
	g.LastRetreatResolution["sentinel"] = game.Outcome{}
	g.SupplyCenterOwners["par"] = "fra"

	clone := g.Clone()
	if clone == g {
		t.Fatal("Clone returned the original game pointer")
	}

	clone.Assignments["eng"] = "changed-player"
	unit := clone.Units["fra-army-par-start"]
	unit.ProvinceID = "bur"
	clone.Units[unit.ID] = unit
	fleet := clone.Units["fra-fleet-bre-start"]
	fleet.Coast = "changed-coast"
	clone.Units[fleet.ID] = fleet
	clone.Orders = append(clone.Orders, game.NewHoldOrder("fra-army-par-start", "fra"))
	delete(clone.LastOrderResolution, "sentinel")
	delete(clone.LastRetreatResolution, "sentinel")
	clone.SupplyCenterOwners["par"] = "eng"

	if got := g.Assignments["eng"]; got != "player-1" {
		t.Fatalf("original assignment = %q, want player-1", got)
	}
	if got := g.Units["fra-army-par-start"].ProvinceID; got != "par" {
		t.Fatalf("original unit province = %q, want par", got)
	}
	if got := g.Units["fra-fleet-bre-start"].Coast; got != "bre" {
		t.Fatalf("original fleet coast = %q, want bre", got)
	}
	if _, ok := g.OrderFor("fra-army-par-start"); ok {
		t.Fatal("clone order was added to original game")
	}
	if _, ok := g.LastOrderResolution["sentinel"]; !ok {
		t.Fatal("clone deletion affected original LastOrderResolution")
	}
	if _, ok := g.LastRetreatResolution["sentinel"]; !ok {
		t.Fatal("clone deletion affected original LastRetreatResolution")
	}
	if got := g.SupplyCenterOwners["par"]; got != "fra" {
		t.Fatalf("original SupplyCenterOwners[par] = %q, want fra", got)
	}
}

func TestPlayerControlsNation(t *testing.T) {
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

	testCases := []struct {
		playerPlayerID game.PlayerID
		nation         gamemap.NationID
		want           bool
	}{
		{"player-1", "eng", true},
		{"player-1", "fra", false},
		{"player-2", "fra", true},
		{"player-2", "eng", false},
		{"player-3", "eng", false},  // Player not assigned to any nation
		{"player-1", "prus", false}, // Nation not in game
	}

	for _, tt := range testCases {
		if got := g.PlayerControlsNation(tt.playerPlayerID, tt.nation); got != tt.want {
			t.Errorf("PlayerControlsNation(%q, %q) = %v, want %v", tt.playerPlayerID, tt.nation, got, tt.want)
		}
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
