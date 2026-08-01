package game_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
)

func TestGameCompleteOwnershipUpdate(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)
	g.Turn.Phase = game.UpdateOwnership
	g.Turn.Season = game.Fall

	// fra-army-par-start still occupies "par", which it already owns —
	// exercises the "stays the same" path, not just a change.
	// Move the english fleet off "lon" (still owned by eng, now unoccupied)
	// and onto neutral "spa" to prove capture works.
	unit := g.Units["eng-fleet-lon-start"]
	unit.ProvinceID = "spa"
	unit.Coast = "spa-nc"
	g.Units["eng-fleet-lon-start"] = unit

	if err := g.CompleteOwnershipUpdate(gm); err != nil {
		t.Fatalf("CompleteOwnershipUpdate failed: %v", err)
	}

	if got := g.SupplyCenterOwners["spa"]; got != "eng" {
		t.Fatalf("SupplyCenterOwners[spa] = %q, want eng (captured)", got)
	}
	if got := g.SupplyCenterOwners["lon"]; got != "eng" {
		t.Fatalf("SupplyCenterOwners[lon] = %q, want eng (unoccupied, keeps previous owner)", got)
	}
	if got := g.SupplyCenterOwners["par"]; got != "fra" {
		t.Fatalf("SupplyCenterOwners[par] = %q, want fra (unchanged)", got)
	}
	if got := g.SupplyCenterOwners["por"]; got != "" {
		t.Fatalf("SupplyCenterOwners[por] = %q, want unowned (never occupied)", got)
	}
	if g.Turn.Phase != game.AcceptAdjustments {
		t.Fatalf("Turn.Phase = %q, want %q", g.Turn.Phase, game.AcceptAdjustments)
	}
}

func TestGameCompleteOwnershipUpdateRejectsWrongPhase(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)

	if err := g.CompleteOwnershipUpdate(gm); err == nil {
		t.Fatal("expected CompleteOwnershipUpdate to reject wrong phase")
	}
}
