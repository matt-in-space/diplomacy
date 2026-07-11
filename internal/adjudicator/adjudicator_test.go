package adjudicator_test

import (
	"os"
	"testing"

	"github.com/matt-in-space/diplomacy/internal/adjudicator"
	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

type unitSpec struct {
	id       game.UnitID
	nation   gamemap.NationID
	province gamemap.ProvinceID
	kind     game.UnitType
	coast    gamemap.CoastID
}

type expectedOutcome struct {
	unitID       game.UnitID
	unitType     adjudicator.UnitOutcomeType
	from         gamemap.ProvinceID
	to           gamemap.ProvinceID
	orderSuccess bool
	reason       adjudicator.ReasonCode
}

type scenario struct {
	name     string
	units    []unitSpec
	orders   []game.Order
	expected []expectedOutcome
}

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
				moveOutcome("fra-a-par", "par", "gas"),
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
				moveOutcome("fra-a-par", "par", "bre"),
				moveOutcome("fra-a-bre", "bre", "gas"),
				moveOutcome("fra-a-gas", "gas", "par"),
			},
		},
	})
}

func TestResolve_Strength(t *testing.T) {
	runScenarios(t, []scenario{
		{
			name: "attack and defense of equal strength result in a standoff",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
				army("eng-a-gas", "eng", "gas"),
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),
				game.NewHoldOrder("eng-a-gas", "eng", "gas"),
			},
			expected: []expectedOutcome{
				holdOutcome("fra-a-par", "par", false, adjudicator.ReasonWeakAttack),
				holdOutcome("eng-a-gas", "gas", true, adjudicator.ReasonSuccess),
			},
		},
		{
			name: "supported attack dislodges an unsupported defender",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
				army("fra-a-bre", "fra", "bre"),
				army("eng-a-gas", "eng", "gas"),
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""),
				game.NewHoldOrder("eng-a-gas", "eng", "gas"),
			},
			expected: []expectedOutcome{
				moveOutcome("fra-a-par", "par", "gas"),
				holdOutcome("fra-a-bre", "bre", true, adjudicator.ReasonSuccess),
				retreatOutcome("eng-a-gas", "gas"),
			},
		},
		{
			name: "equally supported attack and defense result in a standoff",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
				army("fra-a-bre", "fra", "bre"),
				army("eng-a-gas", "eng", "gas"),
				army("eng-a-spa", "eng", "spa"),
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""),
				game.NewHoldOrder("eng-a-gas", "eng", "gas"),
				game.NewSupportHoldOrder("eng-a-spa", "eng", "eng-a-gas", "gas"),
			},
			expected: []expectedOutcome{
				holdOutcome("fra-a-par", "par", false, adjudicator.ReasonWeakAttack),
				holdOutcome("fra-a-bre", "bre", true, adjudicator.ReasonSuccess),
				holdOutcome("eng-a-gas", "gas", true, adjudicator.ReasonSuccess),
				holdOutcome("eng-a-spa", "spa", true, adjudicator.ReasonSuccess),
			},
		},
	})
}

func TestResolve_Support(t *testing.T) {
	runScenarios(t, []scenario{
		{
			name: "an attack cuts support and leaves the primary attack tied",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
				army("fra-a-bre", "fra", "bre"),
				army("eng-a-gas", "eng", "gas"),
				fleet("eng-f-eng", "eng", "eng", "eng"),
			},
			orders: []game.Order{
				game.NewMoveOrder("fra-a-par", "fra", "gas", ""),
				game.NewSupportMoveOrder("fra-a-bre", "fra", "fra-a-par", "gas", ""),
				game.NewHoldOrder("eng-a-gas", "eng", "gas"),
				game.NewMoveOrder("eng-f-eng", "eng", "bre", "bre"),
			},
			expected: []expectedOutcome{
				holdOutcome("fra-a-par", "par", false, adjudicator.ReasonWeakAttack),
				holdOutcome("fra-a-bre", "bre", false, ""),
				holdOutcome("eng-a-gas", "gas", true, adjudicator.ReasonSuccess),
				holdOutcome("eng-f-eng", "eng", false, adjudicator.ReasonWeakAttack),
			},
		},
		{
			name: "support hold fails when the supported unit moves",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
				army("fra-a-bre", "fra", "bre"),
			},
			orders: []game.Order{
				game.NewSupportHoldOrder("fra-a-par", "fra", "fra-a-bre", "bre"),
				game.NewMoveOrder("fra-a-bre", "fra", "gas", ""),
			},
			expected: []expectedOutcome{
				holdOutcome("fra-a-par", "par", false, ""),
				moveOutcome("fra-a-bre", "bre", "gas"),
			},
		},
		{
			name: "support move fails when the supported unit holds",
			units: []unitSpec{
				army("fra-a-par", "fra", "par"),
				army("fra-a-bre", "fra", "bre"),
			},
			orders: []game.Order{
				game.NewSupportMoveOrder("fra-a-par", "fra", "fra-a-bre", "gas", ""),
				game.NewHoldOrder("fra-a-bre", "fra", "bre"),
			},
			expected: []expectedOutcome{
				holdOutcome("fra-a-par", "par", false, ""),
				holdOutcome("fra-a-bre", "bre", true, adjudicator.ReasonSuccess),
			},
		},
	})
}

func TestResolve_Convoy(t *testing.T) {
	runScenarios(t, []scenario{
		{
			name: "army moves across a complete convoy route",
			units: []unitSpec{
				army("eng-a-lon", "eng", "lon"),
				fleet("eng-f-eng", "eng", "eng", "eng"),
			},
			orders: []game.Order{
				game.NewConvoyedMoveOrder("eng-a-lon", "eng", "bre"),
				game.NewConvoyOrder("eng-f-eng", "eng", "eng-a-lon", "lon", "bre"),
			},
			expected: []expectedOutcome{
				moveOutcome("eng-a-lon", "lon", "bre"),
				holdOutcome("eng-f-eng", "eng", true, adjudicator.ReasonSuccess),
			},
		},
		{
			name: "dislodging a convoying fleet disrupts the convoy",
			units: []unitSpec{
				army("eng-a-gas", "eng", "gas"),
				fleet("eng-f-eng", "eng", "eng", "eng"),
				fleet("eng-f-mao", "eng", "mao", "mao"),
				fleet("fra-f-lon", "fra", "lon", "lon"),
				fleet("fra-f-bre", "fra", "bre", "bre"),
			},
			orders: []game.Order{
				game.NewConvoyedMoveOrder("eng-a-gas", "eng", "lon"),
				game.NewConvoyOrder("eng-f-eng", "eng", "eng-a-gas", "gas", "lon"),
				game.NewConvoyOrder("eng-f-mao", "eng", "eng-a-gas", "gas", "lon"),
				game.NewMoveOrder("fra-f-lon", "fra", "eng", "eng"),
				game.NewSupportMoveOrder("fra-f-bre", "fra", "fra-f-lon", "eng", "eng"),
			},
			expected: []expectedOutcome{
				holdOutcome("eng-a-gas", "gas", false, ""),
				retreatOutcome("eng-f-eng", "eng"),
				holdOutcome("eng-f-mao", "mao", false, ""),
				moveOutcome("fra-f-lon", "lon", "eng"),
				holdOutcome("fra-f-bre", "bre", true, adjudicator.ReasonSuccess),
			},
		},
		{
			name: "convoy fails when the fleet orders a different route",
			units: []unitSpec{
				army("eng-a-lon", "eng", "lon"),
				fleet("eng-f-eng", "eng", "eng", "eng"),
			},
			orders: []game.Order{
				game.NewConvoyedMoveOrder("eng-a-lon", "eng", "bre"),
				game.NewConvoyOrder("eng-f-eng", "eng", "eng-a-lon", "lon", "gas"),
			},
			expected: []expectedOutcome{
				holdOutcome("eng-a-lon", "lon", false, ""),
				holdOutcome("eng-f-eng", "eng", false, ""),
			},
		},
		{
			name: "army moves across multiple water provinces with a convoy chain",
			units: []unitSpec{
				army("eng-a-lon", "eng", "lon"),
				fleet("eng-f-eng", "eng", "eng", "eng"),
				fleet("fra-f-mao", "fra", "mao", "mao"),
			},
			orders: []game.Order{
				game.NewConvoyedMoveOrder("eng-a-lon", "eng", "por"),
				game.NewConvoyOrder("eng-f-eng", "eng", "eng-a-lon", "lon", "por"),
				game.NewConvoyOrder("fra-f-mao", "fra", "eng-a-lon", "lon", "por"),
			},
			expected: []expectedOutcome{
				moveOutcome("eng-a-lon", "lon", "por"),
				holdOutcome("eng-f-eng", "eng", true, adjudicator.ReasonSuccess),
				holdOutcome("fra-f-mao", "mao", true, adjudicator.ReasonSuccess),
			},
		},
	})
}

func runScenarios(t *testing.T, scenarios []scenario) {
	t.Helper()

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gm := loadWesternEuropeMap(t)
			g := newScenarioGame(t, gm, tt.units)
			submitOrders(t, g, gm, tt.orders...)

			got, err := adjudicator.Resolve(g, gm)
			if err != nil {
				t.Fatalf("Resolve failed: %v", err)
			}

			assertResolution(t, got, tt.expected)
		})
	}
}

func newScenarioGame(t *testing.T, gm *gamemap.GameMap, units []unitSpec) *game.Game {
	t.Helper()

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

	g.Units = make(map[game.UnitID]game.Unit, len(units))
	g.Positions = make(map[gamemap.ProvinceID]game.UnitID, len(units))
	g.FleetCoasts = make(map[game.UnitID]gamemap.CoastID)
	g.Orders = make(map[game.UnitID]game.Order)

	for _, spec := range units {
		if _, exists := g.Units[spec.id]; exists {
			t.Fatalf("duplicate unit ID %q", spec.id)
		}
		if occupyingUnit, exists := g.Positions[spec.province]; exists {
			t.Fatalf("province %q occupied by both %q and %q", spec.province, occupyingUnit, spec.id)
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

	return g
}

func submitOrders(t *testing.T, g *game.Game, gm *gamemap.GameMap, orders ...game.Order) {
	t.Helper()

	for _, order := range orders {
		if err := g.SubmitOrder(order, gm); err != nil {
			t.Fatalf("SubmitOrder for unit %q failed: %v", order.Unit(), err)
		}
	}
}

func assertResolution(t *testing.T, got adjudicator.Resolution, expected []expectedOutcome) {
	t.Helper()

	if len(got.OrderOutcomes) != len(expected) {
		t.Errorf("OrderOutcomes count = %d, want %d", len(got.OrderOutcomes), len(expected))
	}
	if len(got.UnitOutcomes) != len(expected) {
		t.Errorf("UnitOutcomes count = %d, want %d", len(got.UnitOutcomes), len(expected))
	}

	for _, want := range expected {
		orderOutcome, ok := got.OrderOutcomes[want.unitID]
		if !ok {
			t.Errorf("order outcome for unit %q not found", want.unitID)
		} else {
			if orderOutcome.Success != want.orderSuccess {
				t.Errorf("order outcome for unit %q success = %v, want %v", want.unitID, orderOutcome.Success, want.orderSuccess)
			}
			if want.reason != "" && orderOutcome.Reason != want.reason {
				t.Errorf("order outcome for unit %q reason = %q, want %q", want.unitID, orderOutcome.Reason, want.reason)
			}
		}

		unitOutcome, ok := got.UnitOutcomes[want.unitID]
		if !ok {
			t.Errorf("unit outcome for unit %q not found", want.unitID)
			continue
		}
		if unitOutcome.Type != want.unitType {
			t.Errorf("unit outcome for unit %q type = %q, want %q", want.unitID, unitOutcome.Type, want.unitType)
		}
		if unitOutcome.From != want.from {
			t.Errorf("unit outcome for unit %q from = %q, want %q", want.unitID, unitOutcome.From, want.from)
		}
		if unitOutcome.To != want.to {
			t.Errorf("unit outcome for unit %q to = %q, want %q", want.unitID, unitOutcome.To, want.to)
		}
	}
}

func army(id game.UnitID, nation gamemap.NationID, province gamemap.ProvinceID) unitSpec {
	return unitSpec{id: id, nation: nation, province: province, kind: game.UnitTypeArmy}
}

func fleet(id game.UnitID, nation gamemap.NationID, province gamemap.ProvinceID, coast gamemap.CoastID) unitSpec {
	return unitSpec{id: id, nation: nation, province: province, kind: game.UnitTypeFleet, coast: coast}
}

func moveOutcome(unitID game.UnitID, from, to gamemap.ProvinceID) expectedOutcome {
	return expectedOutcome{
		unitID:       unitID,
		unitType:     adjudicator.UnitOutcomeMove,
		from:         from,
		to:           to,
		orderSuccess: true,
		reason:       adjudicator.ReasonSuccess,
	}
}

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

func retreatOutcome(unitID game.UnitID, province gamemap.ProvinceID) expectedOutcome {
	return expectedOutcome{
		unitID:       unitID,
		unitType:     adjudicator.UnitOutcomeRetreat,
		from:         province,
		to:           province,
		orderSuccess: false,
	}
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
