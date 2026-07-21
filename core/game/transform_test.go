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

func TestApplyUnitTransformsPreservesRetreatingFleetCoast(t *testing.T) {
	fleet := testUnit("fleet-a", "spa")
	fleet.Type = UnitTypeFleet
	g := newTransformTestGame(fleet)
	g.FleetCoasts[fleet.ID] = "spa-nc"

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: fleet.ID, Type: UnitTransformRetreat, From: "spa", Coast: "spa-nc"},
	})
	if err != nil {
		t.Fatalf("ApplyUnitTransforms failed: %v", err)
	}

	if got := g.PendingRetreats[fleet.ID].Coast; got != "spa-nc" {
		t.Fatalf("PendingRetreats[fleet-a].Coast = %q, want spa-nc", got)
	}
	if _, ok := g.FleetCoasts[fleet.ID]; ok {
		t.Fatal("FleetCoasts still contains retreating fleet")
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

	if got := g.Positions["a"]; got != "unit-a" {
		t.Fatalf("Positions[a] = %q, want unit-a", got)
	}
	if got := g.Units["unit-a"].ProvinceID; got != "a" {
		t.Fatalf("unit-a ProvinceID = %q, want a", got)
	}
	if _, ok := g.Positions["c"]; ok {
		t.Fatal("Positions contains destination from partially applied transform")
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

func TestApplyUnitTransformsRebuildsPositionAndCoastMaps(t *testing.T) {
	army := testUnit("army-a", "a")
	fleet := testUnit("fleet-a", "b")
	fleet.Type = UnitTypeFleet
	g := newTransformTestGame(army, fleet)
	g.Positions["stale"] = "missing-unit"
	g.FleetCoasts[fleet.ID] = "old-coast"
	g.FleetCoasts["missing-unit"] = "stale-coast"
	g.PendingRetreats["missing-unit"] = Dislodgement{From: "stale"}

	err := g.ApplyUnitTransforms([]UnitTransform{
		{UnitID: army.ID, Type: UnitTransformMove, From: "a", To: "c"},
		{UnitID: fleet.ID, Type: UnitTransformHold, From: "b", To: "b", Coast: "new-coast"},
	})
	if err != nil {
		t.Fatalf("ApplyUnitTransforms failed: %v", err)
	}

	if len(g.Positions) != 2 {
		t.Fatalf("Positions length = %d, want 2", len(g.Positions))
	}
	if got := g.Positions["c"]; got != army.ID {
		t.Fatalf("Positions[c] = %q, want %q", got, army.ID)
	}
	if got := g.Positions["b"]; got != fleet.ID {
		t.Fatalf("Positions[b] = %q, want %q", got, fleet.ID)
	}
	if len(g.FleetCoasts) != 1 {
		t.Fatalf("FleetCoasts length = %d, want 1", len(g.FleetCoasts))
	}
	if got := g.FleetCoasts[fleet.ID]; got != "new-coast" {
		t.Fatalf("FleetCoasts[fleet-a] = %q, want new-coast", got)
	}
	if len(g.PendingRetreats) != 0 {
		t.Fatalf("PendingRetreats length = %d, want 0", len(g.PendingRetreats))
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
