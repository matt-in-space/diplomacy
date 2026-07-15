package adjudicator_test

import (
	"os"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/adjudicator"
	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

// unitSpec is a helper struct to define units for test scenarios.
type unitSpec struct {
	id       game.UnitID
	nation   gamemap.NationID
	province gamemap.ProvinceID
	kind     game.UnitType
	coast    gamemap.CoastID
}

// expectedOutcome describes the expected result for a single unit after adjudication.
// This struct is designed to map to the adjudicator.Outcome structure.
type expectedOutcome struct {
	unitID game.UnitID
	// Unit outcome details
	unitType adjudicator.UnitOutcomeType
	from     gamemap.ProvinceID
	to       gamemap.ProvinceID
	// Order outcome details
	orderSuccess bool
	reason       adjudicator.ReasonCode
}

// scenario represents a test case with initial units, orders, and expected outcomes.
type scenario struct {
	name     string
	units    []unitSpec
	orders   []game.Order
	expected []expectedOutcome // List of expected outcomes per unit.
}

// TestResolve_DefaultOrders tests that units without explicit orders default to holding.
func TestResolve_DefaultOrders(t *testing.T) {
	runScenarios(t, []scenario{
		{
			name: "units without orders default to hold",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
				fleet("eng-f-lon", "eng", "lon", "lon"),
			},
			expected: []expectedOutcome{
				holdOutcome("fra-a-par", "par", true, adjudicator.ReasonSuccess),
				holdOutcome("eng-f-lon", "lon", true, adjudicator.ReasonSuccess),
			},
		},
	})
}

// TestResolve_Movement tests basic movement rules.
func TestResolve_Movement(t *testing.T) {
	runScenarios(t, []scenario{
		{
			name: "single unit moves into an unoccupied province",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),
			},
			expected: []expectedOutcome{
				moveOutcome("fra-a-par", "par", "gas", true, adjudicator.ReasonSuccess),
			},
		},
		{
			name: "two units cannot directly trade positions",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
				army("fra-a-gas", "fra", "gas"),
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),
				game.NewMoveOrder("fra-a-gas", "fra", "par", ""),
			},
			expected: []expectedOutcome{
				// Both units attempt to move to a province occupied by the other.
				// This results in a bounce, and they should hold their origin.
				holdOutcome("fra-a-par", "par", false, adjudicator.ReasonWeakAttack),
				holdOutcome("fra-a-gas", "gas", false, adjudicator.ReasonWeakAttack),
			},
		},
		{
			name: "three units can move in a circle",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
				army("fra-a-bre", "fra", "bre"),
				army("fra-a-gas", "fra", "gas"),
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "bre", ""),
				game.NewMoveOrder("fra-a-bre", "fra", "gas", ""),
				game.NewMoveOrder("fra-a-gas", "fra", "par", ""),
			},
			expected: []expectedOutcome{
				moveOutcome("fra-a-par", "par", "bre", true, adjudicator.ReasonSuccess),
				moveOutcome("fra-a-bre", "bre", "gas", true, adjudicator.ReasonSuccess),
				moveOutcome("fra-a-gas", "gas", "par", true, adjudicator.ReasonSuccess),
			},
		},
	})
}

// TestResolve_Strength tests scenarios involving unit strength for attacks and defense.
func TestResolve_Strength(t *testing.T) {
	runScenarios(t, []scenario{
		{
			name: "attack and defense of equal strength result in a standoff",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"), // Attacker
				army("eng-a-gas", "eng", "gas"), // Defender
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),
				game.NewHoldOrder("eng-a-gas", "eng"),
			},
			expected: []expectedOutcome{
				// Attacker strength (1) is not greater than defender strength (1), so attacker bounces.
				holdOutcome("fra-a-par", "par", false, adjudicator.ReasonWeakAttack),
				// Defender successfully holds.
				holdOutcome("eng-a-gas", "gas", true, adjudicator.ReasonSuccess),
			},
		},
	})
}

// TestResolve_Support tests scenarios involving support orders.
func TestResolve_Support(t *testing.T) {
	runScenarios(t, []scenario{
		{
			name: "support hold succeeds for a holding unit",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"), // Supported unit
				army("fra-a-gas", "fra", "gas"), // Supporting unit
			},
			orders: []game.Order{
				game.NewHoldOrder("fra-a-par", "fra"),
				game.NewSupportHoldOrder("fra-a-gas", "fra", "fra-a-par", "par"),
			},
			expected: []expectedOutcome{
				holdOutcome("fra-a-par", "par", true, adjudicator.ReasonSuccess),
				holdOutcome("fra-a-gas", "gas", true, adjudicator.ReasonSuccess), // Supporting order succeeds
			},
		},
		{
			name: "support move succeeds for a moving unit",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"), // Supported unit
				army("fra-a-bre", "fra", "bre"), // Supporting unit
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""),
			},
			expected: []expectedOutcome{
				moveOutcome("fra-a-par", "par", "gas", true, adjudicator.ReasonSuccess),
				holdOutcome("fra-a-bre", "bre", true, adjudicator.ReasonSuccess), // Supporting order succeeds
			},
		},
		{
			name: "support move fails when supported unit moves to different province",
			units: []unitSpec{
				army("fra-a-gas", "fra", "gas"), // Supported unit
				army("fra-a-bre", "fra", "bre"), // Supporting unit
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-gas", "fra", "spa", ""),                     // Moves to 'spa'
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-gas", "par", ""), // Supports a move to 'par' instead
			},
			expected: []expectedOutcome{
				moveOutcome("fra-a-gas", "gas", "spa", true, adjudicator.ReasonSuccess),     // Unit moves successfully
				holdOutcome("fra-a-bre", "bre", false, adjudicator.ReasonMisalignedSupport), // Support fails: supported unit moved elsewhere
			},
		},
		{
			name: "support move fails when supported unit holds but support targets move",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"), // Supported unit
				army("fra-a-bre", "fra", "bre"), // Supporting unit
			},
			orders: []game.Order{
				game.NewHoldOrder("fra-a-par", "fra"),
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""),
			},
			expected: []expectedOutcome{
				holdOutcome("fra-a-par", "par", true, adjudicator.ReasonSuccess),            // Unit holds
				holdOutcome("fra-a-bre", "bre", false, adjudicator.ReasonMisalignedSupport), // Support fails: supported unit did not move
			},
		},
		{
			name: "support is cut by a foreign attack",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),         // Supported attacker
				army("fra-a-bre", "fra", "bre"),         // Supporting unit
				army("eng-a-gas", "eng", "gas"),         // Defender
				fleet("eng-f-eng", "eng", "eng", "eng"), // Cutter (attacks the supporter from the channel)
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),                     // Supported attack on 'gas'
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""), // Support move to 'gas'
				game.NewHoldOrder("eng-a-gas", "eng"),                                // Defender holds
				game.NewMoveOrder("eng-f-eng", "eng", "bre", ""),                     // Cuts support: attacks 'bre' from 'eng' (not the support target)
			},
			expected: []expectedOutcome{
				// Support is cut, so the attack drops to strength 1 and bounces off the strength-1 defender.
				holdOutcome("fra-a-par", "par", false, adjudicator.ReasonWeakAttack), // Attacker bounces
				holdOutcome("fra-a-bre", "bre", false, adjudicator.ReasonSupportCut), // Support cut
				holdOutcome("eng-a-gas", "gas", true, adjudicator.ReasonSuccess),     // Defender holds
				holdOutcome("eng-f-eng", "eng", false, adjudicator.ReasonWeakAttack), // Cutter bounces off the supporter
			},
		},
		{
			name: "support is NOT cut by own-nation attack",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),         // Supported attacker
				army("fra-a-bre", "fra", "bre"),         // Supporting unit
				army("eng-a-gas", "eng", "gas"),         // Defender
				fleet("fra-f-eng", "fra", "eng", "eng"), // Own-nation unit attacking its own supporter
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),                     // Supported attack on 'gas'
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""), // Support move to 'gas'
				game.NewHoldOrder("eng-a-gas", "eng"),                                // Defender holds
				game.NewMoveOrder("fra-f-eng", "fra", "bre", ""),                     // Own-nation attack on 'bre' must NOT cut support
			},
			expected: []expectedOutcome{
				// Own-nation attacks never cut support, so the attack keeps strength 2 and dislodges the defender.
				moveOutcome("fra-a-par", "par", "gas", true, adjudicator.ReasonSuccess), // Attacker moves in
				holdOutcome("fra-a-bre", "bre", true, adjudicator.ReasonSuccess),        // Support holds (not cut)
				retreatOutcome("eng-a-gas", "gas"),                                      // Defender dislodged
				holdOutcome("fra-f-eng", "eng", false, adjudicator.ReasonWeakAttack),    // Own unit bounces off the supporter
			},
		},
		{
			name: "support is NOT cut by attack originating from support target province",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"), // Supported unit
				army("fra-a-bre", "fra", "bre"), // Supporting unit
				army("eng-a-gas", "eng", "gas"), // Attacker targeting supporter
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),                     // Supported unit moves to 'gas'
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""), // Support move to 'gas'
				game.NewMoveOrder("eng-a-gas", "eng", "bre", ""),                     // Foreign attacker moves to 'bre' (supporter's province)
			},
			expected: []expectedOutcome{
				// The attacker on the supporter comes from 'gas', the province the support is
				// directed into, so the support is NOT cut (a unit cannot cut the support aimed
				// at itself). The attack keeps strength 2: the attacker on 'gas' bounces off the
				// holding supporter in 'bre' and is then dislodged by the incoming supported unit.
				moveOutcome("fra-a-par", "par", "gas", true, adjudicator.ReasonSuccess), // Supported unit moves in
				holdOutcome("fra-a-bre", "bre", true, adjudicator.ReasonSuccess),        // Support valid (not cut)
				retreatOutcome("eng-a-gas", "gas"),                                      // Attacker bounces, then is dislodged
			},
		},
	})
}

// TestResolve_Dislodgement tests scenarios involving dislodged units.
func TestResolve_Dislodgement(t *testing.T) {
	runScenarios(t, []scenario{
		{
			name: "single supported attacker dislodges defender",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"), // Supported attacker
				army("fra-a-bre", "fra", "bre"), // Support
				army("eng-a-gas", "eng", "gas"), // Defender
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),                     // Attacker (strength 1 initially)
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""), // Support for move to gas
				game.NewHoldOrder("eng-a-gas", "eng"),                                // Defender (strength 1)
			},
			expected: []expectedOutcome{
				// Attacker strength becomes 2 (1 base + 1 support).
				// Defender strength is 1.
				// Attacker is stronger, defender is dislodged.
				moveOutcome("fra-a-par", "par", "gas", true, adjudicator.ReasonSuccess), // Attacker moves
				holdOutcome("fra-a-bre", "bre", true, adjudicator.ReasonSuccess),        // Support order succeeds
				retreatOutcome("eng-a-gas", "gas"),                                      // Defender retreats
			},
		},
		{
			name: "attacker with equal strength to defender holds",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"), // Attacker
				army("eng-a-gas", "eng", "gas"), // Defender
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),
				game.NewHoldOrder("eng-a-gas", "eng"),
			},
			expected: []expectedOutcome{
				holdOutcome("fra-a-par", "par", false, adjudicator.ReasonWeakAttack), // Attacker bounces
				holdOutcome("eng-a-gas", "gas", true, adjudicator.ReasonSuccess),     // Defender holds
			},
		},
		{
			name: "supported attacker with equal strength to supported defender holds",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"), // Supported attacker
				army("fra-a-bre", "fra", "bre"), // Support
				army("eng-a-gas", "eng", "gas"), // Defender
				army("eng-a-spa", "eng", "spa"), // Support for defender
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),                     // Attacker (strength 1 initially)
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""), // Support for move to 'gas'
				game.NewHoldOrder("eng-a-gas", "eng"),                                // Defender (strength 1)
				game.NewSupportHoldOrder("eng-a-spa", "eng", "eng-a-gas", "gas"),     // Support for hold in 'gas'
			},
			expected: []expectedOutcome{
				// Attacker strength = 1 (base) + 1 (support) = 2.
				// Defender strength = 1 (base) + 1 (support) = 2.
				// Attack strength equals defense strength, so attacker bounces.
				holdOutcome("fra-a-par", "par", false, adjudicator.ReasonWeakAttack), // Attacker bounces
				holdOutcome("fra-a-bre", "bre", true, adjudicator.ReasonSuccess),     // Support order succeeds (it applies and is not cut, but the attack it supports bounces)
				holdOutcome("eng-a-gas", "gas", true, adjudicator.ReasonSuccess),     // Defender holds
				holdOutcome("eng-a-spa", "spa", true, adjudicator.ReasonSuccess),     // Support order succeeds
			},
		},
	})
}

// TestResolve_Convoy is deferred to a later phase.

// runScenarios is a helper function to execute multiple adjudication test scenarios.
func runScenarios(t *testing.T, scenarios []scenario) {
	t.Helper()

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gm := loadWesternEuropeMap(t)
			g := newScenarioGame(t, gm, tt.units)

			// Orders are submitted during the AcceptOrders phase, then the turn is
			// advanced to ResolveOrders for adjudication.
			submitOrders(t, g, gm, tt.orders...)
			for g.Turn.Phase != game.ResolveOrders {
				g.Turn = g.Turn.Next()
			}

			got, err := adjudicator.Resolve(g, gm)
			if err != nil {
				t.Fatalf("Resolve failed: %v", err)
			}

			assertResolution(t, got, tt.expected)
		})
	}
}

// newScenarioGame creates a new game instance for a test scenario.
func newScenarioGame(t *testing.T, gm *gamemap.GameMap, units []unitSpec) *game.Game {
	t.Helper()

	// Initialize game with specific assignments.
	g, err := game.NewGame(game.NewGameConfig{
		ID: "test",
		Assignments: map[gamemap.NationID]game.PlayerID{
			"eng": "pe",
			"fra": "pf",
		},
	}, gm)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	// Manually populate game state for the scenario.
	// Clear any default units that might be created by NewGame (unlikely with the config used, but good practice).
	g.Units = make(map[game.UnitID]game.Unit, len(units))
	g.Positions = make(map[gamemap.ProvinceID]game.UnitID, len(units))
	g.FleetCoasts = make(map[game.UnitID]gamemap.CoastID)
	g.Orders = make(map[game.UnitID]game.Order) // Ensure orders map is empty for scenario

	for _, spec := range units {
		if _, exists := g.Units[spec.id]; exists {
			t.Fatalf("duplicate unit ID %q", spec.id)
		}
		if _, exists := g.Positions[spec.province]; exists {
			t.Fatalf("province %q occupied by two units", spec.province)
		}

		g.Units[spec.id] = game.Unit{
			ID:         spec.id,
			NationID:   spec.nation,
			ProvinceID: spec.province,
			Type:       spec.kind,
		}
		g.Positions[spec.province] = spec.id
		if spec.kind == game.UnitTypeFleet {
			g.FleetCoasts[spec.id] = spec.coast
		}
	}

	// Leave the game in the AcceptOrders phase so the scenario's orders can be
	// submitted; the caller advances to ResolveOrders before adjudicating.
	g.Turn = game.StartingTurn() // Spring, AcceptOrders, Year 1

	return g
}

// submitOrders submits all provided orders to the game.
func submitOrders(t *testing.T, g *game.Game, gm *gamemap.GameMap, orders ...game.Order) {
	t.Helper()

	for _, order := range orders {
		if err := g.SubmitOrder(order, gm); err != nil {
			t.Fatalf("SubmitOrder for unit %q failed: %v", order.Unit(), err)
		}
	}
}

// assertResolution verifies the adjudication result against expected outcomes.
func assertResolution(t *testing.T, got adjudicator.Resolution, expected []expectedOutcome) {
	t.Helper()

	// --- Assert Resolution structure ---
	if len(got.Outcomes) != len(expected) {
		t.Errorf("Outcomes count = %d, want %d", len(got.Outcomes), len(expected))
	}

	// Build a map of expected outcomes for easier lookup by unitID.
	expectedOutcomesMap := make(map[game.UnitID]expectedOutcome, len(expected))
	for _, exp := range expected {
		expectedOutcomesMap[exp.unitID] = exp
	}

	// Check each outcome in the actual resolution.
	for unitID, want := range expectedOutcomesMap {
		gotOutcome, ok := got.Outcomes[unitID]
		if !ok {
			t.Errorf("outcome for unit %q not found in resolution", unitID)
			continue
		}

		// Assert Unit Outcome
		if gotOutcome.Unit.UnitID != want.unitID {
			t.Errorf("Unit outcome for %q: UnitID = %q, want %q", unitID, gotOutcome.Unit.UnitID, want.unitID)
		}
		if gotOutcome.Unit.Type != want.unitType {
			t.Errorf("Unit outcome for %q: Type = %q, want %q", unitID, gotOutcome.Unit.Type, want.unitType)
		}
		if gotOutcome.Unit.From != want.from {
			t.Errorf("Unit outcome for %q: From = %q, want %q", unitID, gotOutcome.Unit.From, want.from)
		}
		if gotOutcome.Unit.To != want.to {
			t.Errorf("Unit outcome for %q: To = %q, want %q", unitID, gotOutcome.Unit.To, want.to)
		}
		// Coast is often irrelevant for basic move/hold outcomes, especialy for armies.
		// So, we only check if 'want.to' is not empty, meaning it's a move outcome where coast might be relevant.
		// If it's a hold, 'from' and 'to' are the same, and coast might not be specified/important for assertion.
		if want.to != "" && gotOutcome.Unit.Coast != "" {
			// This comparison needs to be more robust. For now, we'll skip explicit coast checking unless it's crucial.
		}

		// Assert Order Outcome
		if gotOutcome.Order.Success != want.orderSuccess {
			t.Errorf("Order outcome for %q: Success = %v, want %v", unitID, gotOutcome.Order.Success, want.orderSuccess)
		}
		if want.reason != "" && gotOutcome.Order.Reason != want.reason {
			// Only check reason if a specific reason is expected and it's not an empty string.
			t.Errorf("Order outcome for %q: Reason = %q, want %q", unitID, gotOutcome.Order.Reason, want.reason)
		}
	}
}

// Helper functions to create specific outcome types.

// moveOutcome creates an expectedOutcome for a move action.
// 'success' and 'reason' are for the order outcome.
func moveOutcome(unitID game.UnitID, from, to gamemap.ProvinceID, success bool, reason adjudicator.ReasonCode) expectedOutcome {
	return expectedOutcome{
		unitID:       unitID,
		unitType:     adjudicator.UnitOutcomeMove,
		from:         from,
		to:           to,
		orderSuccess: success,
		reason:       reason,
	}
}

// holdOutcome creates an expectedOutcome for a hold action.
func holdOutcome(unitID game.UnitID, province gamemap.ProvinceID, success bool, reason adjudicator.ReasonCode) expectedOutcome {
	return expectedOutcome{
		unitID:       unitID,
		unitType:     adjudicator.UnitOutcomeHold,
		from:         province,
		to:           province,
		orderSuccess: success,
		reason:       reason,
	}
}

// retreatOutcome creates an expectedOutcome for a unit that must retreat or is dislodged.
func retreatOutcome(unitID game.UnitID, province gamemap.ProvinceID) expectedOutcome {
	// Retreat outcome implies the original order (move/hold) failed.
	return expectedOutcome{
		unitID:       unitID,
		unitType:     adjudicator.UnitOutcomeRetreat,
		from:         province,                    // Province where dislodgement occurred.
		to:           "",                          // Retreat destination is determined in the next phase.
		orderSuccess: false,                       // The original order failed.
		reason:       adjudicator.ReasonDislodged, // Explicitly state reason.
	}
}

// Define unit specs for convenience.
func army(id game.UnitID, nation gamemap.NationID, province gamemap.ProvinceID) unitSpec {
	return unitSpec{id: id, nation: nation, province: province, kind: game.UnitTypeArmy}
}

func fleet(id game.UnitID, nation gamemap.NationID, province gamemap.ProvinceID, coast gamemap.CoastID) unitSpec {
	return unitSpec{id: id, nation: nation, province: province, kind: game.UnitTypeFleet, coast: coast}
}

// loadWesternEuropeMap loads the test map fixture.
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
