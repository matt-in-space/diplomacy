package adjudicator

import (
	"testing"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

func TestBuildAttacks(t *testing.T) {
	ctx := resolutionContext{
		unitPositions: map[game.UnitID]gamemap.ProvinceID{
			"u1": "p1",
			"u2": "p2",
			"u3": "p3",
		},
		fleetCoasts: map[game.UnitID]gamemap.CoastID{
			"u3": "c1",
		},
	}

	moves := make(map[game.UnitID]game.MoveOrder)
	moves["u1"] = game.MoveOrder{
		BaseOrder:   game.BaseOrder{UnitID: "u1", NationID: "n1"},
		Target:      "p2",
		TargetCoast: "",
		ViaConvoy:   false,
	}
	moves["u2"] = game.MoveOrder{
		BaseOrder:   game.BaseOrder{UnitID: "u2", NationID: "n2"},
		Target:      "p1",
		TargetCoast: "",
		ViaConvoy:   true,
	}
	moves["u3"] = game.MoveOrder{
		BaseOrder:   game.BaseOrder{UnitID: "u3", NationID: "n3"},
		Target:      "p1",
		TargetCoast: "c2",
		ViaConvoy:   false,
	}

	attacks := buildAttacks(ctx, moves)
	if len(attacks) != 2 {
		t.Errorf("expected 2 attacks, got %d", len(attacks))
	}
	if len(attacks["p1"]) != 2 {
		t.Errorf("expected 2 attacks from province 1, got %d", len(attacks["p1"]))
	}
	if len(attacks["p2"]) != 1 {
		t.Errorf("expected 1 attack from province 2, got %d", len(attacks["p2"]))
	}

	expectedAttacks := map[string][]Attack{
		"p1": {
			{UnitID: "u2", From: "p2", To: "p1", FromCoast: "", ToCoast: "", ViaConvoy: true},
			{UnitID: "u3", From: "p3", To: "p1", FromCoast: "c1", ToCoast: "c2", ViaConvoy: false},
		},
		"p2": {
			{UnitID: "u1", From: "p1", To: "p2", FromCoast: "", ToCoast: "", ViaConvoy: false},
		},
	}

	assertAttacks(t, "p1", attacks["p1"][0], expectedAttacks["p1"][0])
	assertAttacks(t, "p1", attacks["p1"][1], expectedAttacks["p1"][1])
	assertAttacks(t, "p2", attacks["p2"][0], expectedAttacks["p2"][0])
}

func assertAttacks(t *testing.T, provinceId string, attack, expected Attack) {
	t.Helper()

	if attack.UnitID != expected.UnitID {
		t.Errorf("expected attack from %s to %s, got %s", expected.UnitID, provinceId, attack.UnitID)
	}
	if attack.ViaConvoy != expected.ViaConvoy {
		t.Errorf("expected attack from %s to %s via convoy, got %t", expected.UnitID, provinceId, attack.ViaConvoy)
	}
	if attack.From != expected.From {
		t.Errorf("expected attack from %s to %s, got %s", expected.UnitID, provinceId, attack.From)
	}
	if attack.To != expected.To {
		t.Errorf("expected attack from %s to %s, got %s", expected.UnitID, provinceId, attack.To)
	}
	if attack.FromCoast != expected.FromCoast {
		t.Errorf("expected attack from %s to %s, got %s", expected.UnitID, provinceId, attack.FromCoast)
	}
	if attack.ToCoast != expected.ToCoast {
		t.Errorf("expected attack from %s to %s, got %s", expected.UnitID, provinceId, attack.ToCoast)
	}

}
