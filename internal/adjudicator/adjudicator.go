package adjudicator

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

type Resolution struct{}

func Resolve(g *game.Game, gm *gamemap.GameMap) (Resolution, error) {
	err := validateInputs(g, gm)

	if err != nil {
		return Resolution{}, err
	}

	// 2. Build a resolution context from the current game state and map.
	//    ctx, err := newContext(g, gm)
	//    Errors: inconsistent game state, units missing positions, fleets missing coasts.
	//
	// 3. Normalize missing unit orders into implicit hold orders.
	//    effectiveOrders := normalizeOrders(ctx)
	//    No error expected: missing orders are legal and become holds.
	//
	// 4. Categorize effective orders by type: hold, move, support, and convoy.
	//    categorized, err := categorizeOrders(effectiveOrders)
	//    Errors: unsupported order implementation. This should be rare because SubmitOrder validates order types.
	//
	// 5. Build potential attacks from move orders.
	//    attacks, err := buildAttacks(ctx, categorized.MoveOrders)
	//    Errors: invalid move order references if state was loaded from storage incorrectly.
	//
	// 6. Determine which support orders match the supported unit's actual order.
	//    supportIntents := buildSupportIntents(ctx, categorized)
	//    No error expected: mismatched support intent simply does not apply.
	//
	// 7. Determine which matching supports are cut by attacks.
	//    cutSupports := determineCutSupports(ctx, attacks, supportIntents)
	//    No error expected: this is derived from known attacks/supports.
	//
	// 8. Compute attack and defense strengths using uncut supports.
	//    strengths := computeStrengths(ctx, attacks, supportIntents, cutSupports)
	//    No error expected: strength is derived data.
	//
	// 9. Resolve move contests, bounces, successful moves, and dislodgements.
	//    moveResults := resolveMoves(ctx, attacks, strengths)
	//    No error expected for normal resolution; unresolved paradoxes may later require explicit result states.
	//
	// 10. Compute retreat requirements for dislodged units.
	//     retreats := buildRetreatRequirements(ctx, moveResults)
	//     No error expected: retreat options are derived from the resolved board state.
	//
	// 11. Build and return a resolution result without mutating the game or map.
	//     return buildResolution(ctx, categorized, supportIntents, cutSupports, strengths, moveResults, retreats)
	return Resolution{}, nil
}

func validateInputs(g *game.Game, gm *gamemap.GameMap) error {
	if g == nil {
		return fmt.Errorf("game is nil")
	}
	if gm == nil {
		return fmt.Errorf("map is nil")
	}
	if g.MapID != gm.ID {
		return fmt.Errorf("map ID mismatch")
	}
	if g.Turn.Phase != game.ResolveOrders {
		return fmt.Errorf("wrong turn phase")
	}
	return nil
}
