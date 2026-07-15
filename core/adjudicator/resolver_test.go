package adjudicator

import (
	"testing"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// resolvedOutcomes runs the full pipeline and returns the final outcomes.
func resolvedOutcomes(gm *gamemap.GameMap, units []testUnit, orders ...game.Order) map[game.UnitID]Outcome {
	rc := newResolutionContext(newTestGame(gm, units, orders...), gm)
	rc.normalizeOrders()
	rc.categorizeOrders()
	rc.pruneMisalignedOrders()
	rc.buildIntendedEndingPositions()
	rc.resolveOrders()
	return rc.buildResolution(game.StartingTurn()).Outcomes
}

func wantMove(t *testing.T, o Outcome, to gamemap.ProvinceID) {
	t.Helper()
	if o.Unit.Type != UnitOutcomeMove {
		t.Errorf("unit %q: type = %q, want move", o.UnitID, o.Unit.Type)
	}
	if o.Unit.To != to {
		t.Errorf("unit %q: To = %q, want %q", o.UnitID, o.Unit.To, to)
	}
	if !o.Order.Success {
		t.Errorf("unit %q: order failed (%q), want success", o.UnitID, o.Order.Reason)
	}
}

func wantHold(t *testing.T, o Outcome, success bool, reason ReasonCode) {
	t.Helper()
	if o.Unit.Type != UnitOutcomeHold {
		t.Errorf("unit %q: type = %q, want hold", o.UnitID, o.Unit.Type)
	}
	if o.Order.Success != success {
		t.Errorf("unit %q: order success = %v, want %v", o.UnitID, o.Order.Success, success)
	}
	if o.Order.Reason != reason {
		t.Errorf("unit %q: reason = %q, want %q", o.UnitID, o.Order.Reason, reason)
	}
}

func wantRetreat(t *testing.T, o Outcome) {
	t.Helper()
	if o.Unit.Type != UnitOutcomeRetreat {
		t.Errorf("unit %q: type = %q, want retreat", o.UnitID, o.Unit.Type)
	}
	if o.Order.Success {
		t.Errorf("unit %q: order succeeded, want failure", o.UnitID)
	}
	if o.Order.Reason != ReasonDislodged {
		t.Errorf("unit %q: reason = %q, want %q", o.UnitID, o.Order.Reason, ReasonDislodged)
	}
}

func TestResolve_ConvoySucceeds(t *testing.T) {
	gm := loadTestMap(t)

	outcomes := resolvedOutcomes(gm,
		[]testUnit{
			tArmy("a", "fra", "bre"),
			tFleet("f", "fra", "eng", "eng"),
		},
		game.NewConvoyedMoveOrder("a", "fra", "lon"),
		game.NewConvoyOrder("f", "fra", "a", "bre", "lon"),
	)

	wantMove(t, outcomes["a"], "lon")
	wantHold(t, outcomes["f"], true, ReasonSuccess)
}

func TestResolve_ConvoyDisruptedByDislodgement(t *testing.T) {
	gm := loadTestMap(t)

	// The army in spa is convoyed to por by the fleet in mao, but two English
	// fleets dislodge that convoying fleet, breaking the convoy. The army's
	// destination (por) is unrelated to the attack, so this is not a paradox.
	outcomes := resolvedOutcomes(gm,
		[]testUnit{
			tArmy("a", "fra", "spa"),
			tFleet("fmao", "fra", "mao", "mao"),
			tFleet("feng", "eng", "eng", "eng"),
			tFleet("fbre", "eng", "bre", "bre"),
		},
		game.NewConvoyedMoveOrder("a", "fra", "por"),
		game.NewConvoyOrder("fmao", "fra", "a", "spa", "por"),
		game.NewMoveOrder("feng", "eng", "mao", ""),
		game.NewSupportMoveOrder("fbre", "eng", "feng", "mao", ""),
	)

	wantHold(t, outcomes["a"], false, ReasonConvoyFailure) // convoy broken, army holds
	wantRetreat(t, outcomes["fmao"])                       // convoying fleet dislodged
	wantMove(t, outcomes["feng"], "mao")                   // attacker takes the ocean
	wantHold(t, outcomes["fbre"], true, ReasonSuccess)     // supporter
}

func TestResolve_HeadToHeadWithSupport(t *testing.T) {
	gm := loadTestMap(t)

	// par and gas move directly at each other; gas is supported, so it wins the
	// head-to-head and dislodges par.
	outcomes := resolvedOutcomes(gm,
		[]testUnit{
			tArmy("apar", "fra", "par"),
			tArmy("agas", "eng", "gas"),
			tArmy("abre", "eng", "bre"),
		},
		game.NewMoveOrder("apar", "fra", "gas", ""),
		game.NewMoveOrder("agas", "eng", "par", ""),
		game.NewSupportMoveOrder("abre", "eng", "agas", "par", ""),
	)

	wantRetreat(t, outcomes["apar"])                   // dislodged by the stronger side
	wantMove(t, outcomes["agas"], "par")               // wins the head-to-head
	wantHold(t, outcomes["abre"], true, ReasonSuccess) // supporter
}

func TestResolve_ConvoyParadox(t *testing.T) {
	// Known limitation: convoy paradoxes (DATC 6.F) are not yet fully resolved.
	// The convoyed army is not part of the detected dependency cycle (support-cut
	// recurses through the convoy path, not the army's own resolution node), so the
	// Szykman backup rule cannot target it. Tracked in docs/adjudication-enhancements.md.
	// This test documents the intended outcome for when that work lands.
	t.Skip("convoy paradox handling not yet implemented")

	gm := loadTestMap(t)

	// Pandin-style paradox: the fleet in eng convoys the army bre->lon; the army's
	// attack on lon would cut lon's support for the attack (mao->eng) on that very
	// convoying fleet. Under the Szykman rule the convoy fails, so the army holds,
	// does not cut the support, and the convoying fleet is dislodged.
	outcomes := resolvedOutcomes(gm,
		[]testUnit{
			tArmy("a", "fra", "bre"),
			tFleet("feng", "fra", "eng", "eng"),
			tFleet("flon", "eng", "lon", "lon"),
			tFleet("fmao", "eng", "mao", "mao"),
		},
		game.NewConvoyedMoveOrder("a", "fra", "lon"),
		game.NewConvoyOrder("feng", "fra", "a", "bre", "lon"),
		game.NewSupportMoveOrder("flon", "eng", "fmao", "eng", ""),
		game.NewMoveOrder("fmao", "eng", "eng", ""),
	)

	wantHold(t, outcomes["a"], false, ReasonConvoyFailure) // convoy fails (Szykman)
	wantRetreat(t, outcomes["feng"])                       // convoying fleet dislodged
	wantHold(t, outcomes["flon"], true, ReasonSuccess)     // support not cut
	wantMove(t, outcomes["fmao"], "eng")                   // dislodges the convoying fleet
}

func TestResolve_CircularConvoyOfThree(t *testing.T) {
	gm := loadTestMap(t)

	// A three-way rotation resolves via the backup rule: everyone moves.
	outcomes := resolvedOutcomes(gm,
		[]testUnit{
			tArmy("a1", "fra", "par"),
			tArmy("a2", "fra", "bre"),
			tArmy("a3", "fra", "gas"),
		},
		game.NewMoveOrder("a1", "fra", "bre", ""),
		game.NewMoveOrder("a2", "fra", "gas", ""),
		game.NewMoveOrder("a3", "fra", "par", ""),
	)

	wantMove(t, outcomes["a1"], "bre")
	wantMove(t, outcomes["a2"], "gas")
	wantMove(t, outcomes["a3"], "par")
}
