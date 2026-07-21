package game

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

func TestApplyUnitTransformsRejectsDuplicateDestination(t *testing.T) {
	g := newTransformTestGame(
		testUnit("unit-a", "a"),
		testUnit("unit-b", "b"),
	)

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-a", Type: UnitTransformMove, From: "a", To: "c"},
		{UnitID: "unit-b", Type: UnitTransformMove, From: "b", To: "c"},
	})

	if err == nil {
		t.Fatal("expected ApplyUnitTransforms to reject duplicate destination")
	}
}

func TestApplyUnitTransformsMovesUnit(t *testing.T) {
	g := newTransformTestGame(testUnit("unit-a", "a"))

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-a", Type: UnitTransformMove, From: "a", To: "b"},
	})
	if err != nil {
		t.Fatalf("ApplyUnitTransforms failed: %v", err)
	}

	if got := g.Positions["b"]; got != "unit-a" {
		t.Fatalf("Positions[b] = %q, want unit-a", got)
	}
	if got := g.Units["unit-a"].ProvinceID; got != "b" {
		t.Fatalf("unit-a ProvinceID = %q, want b", got)
	}
}

func TestApplyUnitTransformsHoldsUnitInPlace(t *testing.T) {
	g := newTransformTestGame(testUnit("unit-a", "a"))

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-a", Type: UnitTransformHold, From: "a", To: "a"},
	})
	if err != nil {
		t.Fatalf("ApplyUnitTransforms failed: %v", err)
	}

	if got := g.Positions["a"]; got != "unit-a" {
		t.Fatalf("Positions[a] = %q, want unit-a", got)
	}
	if got := g.Units["unit-a"].ProvinceID; got != "a" {
		t.Fatalf("unit-a ProvinceID = %q, want a", got)
	}
}

func TestApplyUnitTransformsRemovesPreviousPosition(t *testing.T) {
	g := newTransformTestGame(testUnit("unit-a", "a"))

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-a", Type: UnitTransformMove, From: "a", To: "b"},
	})
	if err != nil {
		t.Fatalf("ApplyUnitTransforms failed: %v", err)
	}

	if unitID, ok := g.Positions["a"]; ok {
		t.Fatalf("Positions[a] = %q, want province to be unoccupied", unitID)
	}
}

func TestApplyUnitTransformsDoesNotRemoveUnitThatMovedIntoPreviousPosition(t *testing.T) {
	g := newTransformTestGame(
		testUnit("unit-a", "a"),
		testUnit("unit-b", "b"),
	)

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-b", Type: UnitTransformMove, From: "b", To: "a"},
		{UnitID: "unit-a", Type: UnitTransformMove, From: "a", To: "c"},
	})
	if err != nil {
		t.Fatalf("ApplyUnitTransforms failed: %v", err)
	}

	if got := g.Positions["a"]; got != "unit-b" {
		t.Fatalf("Positions[a] = %q, want unit-b", got)
	}
	if got := g.Positions["c"]; got != "unit-a" {
		t.Fatalf("Positions[c] = %q, want unit-a", got)
	}
}

func TestApplyUnitTransformsAddsRetreat(t *testing.T) {
	g := newTransformTestGame(testUnit("unit-a", "a"))

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-a", Type: UnitTransformRetreat, From: "a"},
	})
	if err != nil {
		t.Fatalf("ApplyUnitTransforms failed: %v", err)
	}

	if unitID, ok := g.Positions["a"]; ok {
		t.Fatalf("Positions[a] = %q, want province to be unoccupied", unitID)
	}
	got, ok := g.PendingRetreats["unit-a"]
	if !ok {
		t.Fatal("PendingRetreats does not contain unit-a")
	}
	if got.From != "a" {
		t.Fatalf("PendingRetreats[unit-a].From = %q, want a", got.From)
	}
}

func newTransformTestGame(units ...Unit) *Game {
	g := &Game{
		Units:           make(map[UnitID]Unit, len(units)),
		Positions:       make(map[gamemap.ProvinceID]UnitID, len(units)),
		FleetCoasts:     make(map[UnitID]gamemap.CoastID),
		PendingRetreats: make(map[UnitID]Dislodgement),
	}
	for _, unit := range units {
		g.Units[unit.ID] = unit
		g.Positions[unit.ProvinceID] = unit.ID
	}
	return g
}

func testUnit(id UnitID, province gamemap.ProvinceID) Unit {
	return Unit{
		ID:         id,
		NationID:   "nation-a",
		ProvinceID: province,
		Type:       UnitTypeArmy,
	}
}
