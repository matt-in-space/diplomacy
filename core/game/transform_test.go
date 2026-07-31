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

	if got := g.Units["unit-a"].ProvinceID; got == "a" {
		t.Fatal("unit-a still occupies its previous province")
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

	if got := g.Units["unit-b"].ProvinceID; got != "a" {
		t.Fatalf("unit-b ProvinceID = %q, want a", got)
	}
	if got := g.Units["unit-a"].ProvinceID; got != "c" {
		t.Fatalf("unit-a ProvinceID = %q, want c", got)
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

	got := g.Units["unit-a"]
	if got.ProvinceID != "" {
		t.Fatalf("unit-a ProvinceID = %q, want empty", got.ProvinceID)
	}
	if !got.Dislodged() {
		t.Fatal("unit-a is not marked dislodged")
	}
	if got.DislodgedFrom != "a" {
		t.Fatalf("unit-a DislodgedFrom = %q, want a", got.DislodgedFrom)
	}
}

func TestApplyUnitTransformsPreservesRetreatingFleetCoast(t *testing.T) {
	fleet := testUnit("fleet-a", "spa")
	fleet.Type = UnitTypeFleet
	fleet.Coast = "spa-nc"
	g := newTransformTestGame(fleet)

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: fleet.ID, Type: UnitTransformRetreat, From: "spa", Coast: "spa-nc"},
	})
	if err != nil {
		t.Fatalf("ApplyUnitTransforms failed: %v", err)
	}

	got := g.Units[fleet.ID]
	if got.Coast != "spa-nc" {
		t.Fatalf("fleet-a Coast = %q, want spa-nc", got.Coast)
	}
	if got.DislodgedFrom != "spa" {
		t.Fatalf("fleet-a DislodgedFrom = %q, want spa", got.DislodgedFrom)
	}
}

func TestApplyUnitTransformsValidatesBeforeChangingGame(t *testing.T) {
	g := newTransformTestGame(
		testUnit("unit-a", "a"),
		testUnit("unit-b", "b"),
	)

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-a", Type: UnitTransformMove, From: "a", To: "c"},
		{UnitID: "unit-b", Type: UnitTransformMove, From: "wrong", To: "d"},
	})
	if err == nil {
		t.Fatal("expected ApplyUnitTransforms to reject incorrect origin")
	}

	if got := g.Units["unit-a"].ProvinceID; got != "a" {
		t.Fatalf("unit-a ProvinceID = %q, want a", got)
	}
}

func TestApplyUnitTransformsRejectsDuplicateUnit(t *testing.T) {
	g := newTransformTestGame(
		testUnit("unit-a", "a"),
		testUnit("unit-b", "b"),
	)

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-a", Type: UnitTransformMove, From: "a", To: "b"},
		{UnitID: "unit-a", Type: UnitTransformMove, From: "a", To: "c"},
	})
	if err == nil {
		t.Fatal("expected ApplyUnitTransforms to reject duplicate unit")
	}
}

func TestApplyUnitTransformsRequiresResultForEveryUnit(t *testing.T) {
	g := newTransformTestGame(
		testUnit("unit-a", "a"),
		testUnit("unit-b", "b"),
	)

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-a", Type: UnitTransformHold, From: "a", To: "a"},
	})
	if err == nil {
		t.Fatal("expected ApplyUnitTransforms to require a result for every unit")
	}
}

func TestApplyUnitTransformsRejectsUnknownType(t *testing.T) {
	g := newTransformTestGame(testUnit("unit-a", "a"))

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: "unit-a", Type: "unknown", From: "a", To: "a"},
	})
	if err == nil {
		t.Fatal("expected ApplyUnitTransforms to reject unknown transform type")
	}
}

func newTransformTestGame(units ...Unit) *Game {
	g := &Game{
		Units: make(map[UnitID]Unit, len(units)),
	}
	for _, unit := range units {
		g.Units[unit.ID] = unit
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
