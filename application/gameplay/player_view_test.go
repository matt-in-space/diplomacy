package gameplay

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// playerViewTestGame builds a two-nation game (unlike repositoryTestGame's
// single nation) — needed to test that one nation's orders never leak into
// another's view, the whole point of PlayerView existing.
func playerViewTestGame() *game.Game {
	return &game.Game{
		ID:    "test-game",
		MapID: "test-map",
		Turn:  game.StartingTurn(),
		Assignments: map[gamemap.NationID]game.PlayerID{
			"eng": "player-eng",
			"fra": "player-fra",
		},
		Units: map[game.UnitID]game.Unit{
			"eng-fleet-lon": {ID: "eng-fleet-lon", NationID: "eng", ProvinceID: "lon", Type: game.UnitTypeFleet, Coast: "lon"},
			"fra-army-par":  {ID: "fra-army-par", NationID: "fra", ProvinceID: "par", Type: game.UnitTypeArmy},
		},
		CommittedOrders:    make(map[gamemap.NationID]struct{}),
		SupplyCenterOwners: map[gamemap.ProvinceID]gamemap.NationID{"lon": "eng", "par": "fra"},
	}
}

func playerViewTestMap() *gamemap.GameMap {
	return &gamemap.GameMap{
		ID:      "test-map",
		Nations: []gamemap.NationID{"eng", "fra"},
		Provinces: map[gamemap.ProvinceID]gamemap.Province{
			"lon": {ID: "lon", Name: "London", Type: gamemap.Coastal},
			"par": {ID: "par", Name: "Paris", Type: gamemap.Inland},
			"gas": {ID: "gas", Name: "Gascony", Type: gamemap.Coastal},
		},
	}
}

func newPlayerViewTestService(g *game.Game) *GameplayService {
	games := &submitOrderGameRepository{stored: StoredGame{Game: g, Version: 0}}
	maps := &submitOrderMapRepository{gameMap: playerViewTestMap()}
	return NewGameplayService(games, maps)
}

func TestGetPlayerViewNeverLeaksAnotherNationsOrders(t *testing.T) {
	g := playerViewTestGame()
	g.Orders = []game.Order{
		game.NewMoveOrder("fra-army-par", "fra", "gas", ""),
	}
	service := newPlayerViewTestService(g)

	// The ordering nation's own player sees it.
	fraView, err := service.GetPlayerView(context.Background(), g.ID, "player-fra")
	if err != nil {
		t.Fatalf("GetPlayerView (fra) failed: %v", err)
	}
	if len(fraView.YourOrders) != 1 || fraView.YourOrders[0].UnitID != "fra-army-par" {
		t.Fatalf("fra view YourOrders = %+v, want the fra-army-par move order", fraView.YourOrders)
	}
	if !strings.Contains(fraView.YourOrders[0].Description, "Gascony") {
		t.Fatalf("fra order description = %q, want it to mention Gascony", fraView.YourOrders[0].Description)
	}

	// The anchor assertion: England's own view has no orders at all (they
	// haven't submitted any) — and, more importantly, the raw JSON of
	// England's view never contains France's order data anywhere, not just
	// "the field I expected is empty."
	engView, err := service.GetPlayerView(context.Background(), g.ID, "player-eng")
	if err != nil {
		t.Fatalf("GetPlayerView (eng) failed: %v", err)
	}
	if engView.YourNation != "eng" {
		t.Fatalf("eng view YourNation = %q, want %q", engView.YourNation, "eng")
	}
	if len(engView.YourOrders) != 0 {
		t.Fatalf("eng view YourOrders = %+v, want none", engView.YourOrders)
	}

	// "fra-army-par" itself is expected to appear (via Units — positions
	// are legitimately public); what must never appear is anything that
	// only exists because of the order's *content* — its target province
	// or its description text.
	engJSON, err := json.Marshal(engView)
	if err != nil {
		t.Fatalf("failed to marshal eng view: %v", err)
	}
	for _, leak := range []string{"Gascony", "Move"} {
		if strings.Contains(string(engJSON), leak) {
			t.Fatalf("eng view JSON leaked France's order data (%q found): %s", leak, engJSON)
		}
	}
}

func TestGetPlayerViewIncludesPublicDataRegardlessOfRequester(t *testing.T) {
	g := playerViewTestGame()
	g.CommittedOrders["fra"] = struct{}{}
	service := newPlayerViewTestService(g)

	for _, playerID := range []game.PlayerID{"player-eng", "player-fra"} {
		view, err := service.GetPlayerView(context.Background(), g.ID, playerID)
		if err != nil {
			t.Fatalf("GetPlayerView(%s) failed: %v", playerID, err)
		}
		if view.Turn != g.Turn {
			t.Fatalf("GetPlayerView(%s) Turn = %+v, want %+v", playerID, view.Turn, g.Turn)
		}
		if len(view.Units) != 2 {
			t.Fatalf("GetPlayerView(%s) Units = %+v, want both units (positions are public)", playerID, view.Units)
		}
		if view.SupplyCenterOwners["lon"] != "eng" || view.SupplyCenterOwners["par"] != "fra" {
			t.Fatalf("GetPlayerView(%s) SupplyCenterOwners = %+v, want lon=eng par=fra", playerID, view.SupplyCenterOwners)
		}
		if len(view.CommittedNations) != 1 || view.CommittedNations[0] != "fra" {
			t.Fatalf("GetPlayerView(%s) CommittedNations = %+v, want [fra]", playerID, view.CommittedNations)
		}
	}
}

func TestGetPlayerViewForNonParticipantHasNoNationOrOrders(t *testing.T) {
	g := playerViewTestGame()
	g.Orders = []game.Order{game.NewHoldOrder("fra-army-par", "fra")}
	service := newPlayerViewTestService(g)

	view, err := service.GetPlayerView(context.Background(), g.ID, "someone-else")
	if err != nil {
		t.Fatalf("GetPlayerView failed: %v", err)
	}
	if view.YourNation != "" {
		t.Fatalf("YourNation = %q, want empty for a non-participant", view.YourNation)
	}
	if len(view.YourOrders) != 0 {
		t.Fatalf("YourOrders = %+v, want none for a non-participant", view.YourOrders)
	}
	// Public data is still assembled — GetPlayerView only assembles the
	// view; deciding whether the requester should be allowed to ask at all
	// is the HTTP layer's job, not this one's.
	if len(view.Units) != 2 {
		t.Fatalf("Units = %+v, want both units even for a non-participant view", view.Units)
	}
}

func TestGetPlayerViewDescribesEachOrderType(t *testing.T) {
	g := playerViewTestGame()
	g.Units["fra-fleet-gas"] = game.Unit{ID: "fra-fleet-gas", NationID: "fra", ProvinceID: "gas", Type: game.UnitTypeFleet}
	g.Orders = []game.Order{
		game.NewHoldOrder("fra-army-par", "fra"),
		game.NewSupportHoldOrder("fra-fleet-gas", "fra", "fra-army-par", "par"),
	}
	service := newPlayerViewTestService(g)

	view, err := service.GetPlayerView(context.Background(), g.ID, "player-fra")
	if err != nil {
		t.Fatalf("GetPlayerView failed: %v", err)
	}

	descriptions := make(map[game.UnitID]string, len(view.YourOrders))
	for _, o := range view.YourOrders {
		descriptions[o.UnitID] = o.Description
	}
	if descriptions["fra-army-par"] != "Hold" {
		t.Fatalf("fra-army-par description = %q, want %q", descriptions["fra-army-par"], "Hold")
	}
	if want := "Support A Paris to hold"; descriptions["fra-fleet-gas"] != want {
		t.Fatalf("fra-fleet-gas description = %q, want %q", descriptions["fra-fleet-gas"], want)
	}
}

func TestGetPlayerViewReturnsGameLookupError(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	games := &submitOrderGameRepository{getErr: lookupErr}
	maps := &submitOrderMapRepository{gameMap: playerViewTestMap()}
	service := NewGameplayService(games, maps)

	_, err := service.GetPlayerView(context.Background(), "missing-game", "player-eng")
	if !errors.Is(err, lookupErr) {
		t.Fatalf("GetPlayerView error = %v, want the lookup error", err)
	}
}
