package adjudicator

import (
	"fmt"
	"maps"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

type Resolution struct{}

type resolutionContext struct {
	game          *game.Game
	gameMap       *gamemap.GameMap
	units         map[game.UnitID]game.Unit
	positions     map[gamemap.ProvinceID]game.UnitID
	unitPositions map[game.UnitID]gamemap.ProvinceID
	fleetCoasts   map[game.UnitID]gamemap.CoastID
	orders        map[game.UnitID]game.Order
}

func Resolve(g *game.Game, gm *gamemap.GameMap) (Resolution, error) {
	err := validateInputs(g, gm)
	if err != nil {
		return Resolution{}, err
	}

	ctx := newContext(g, gm)
	effectiveOrders := normalizeOrders(ctx)
	categorized, err := categorizeOrders(effectiveOrders)
	if err != nil {
		return Resolution{}, err
	}

	attacks := buildAttacks(ctx, categorized.moves)
	_ = attacks

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

func newContext(g *game.Game, gm *gamemap.GameMap) resolutionContext {
	ctx := resolutionContext{
		game:          g,
		gameMap:       gm,
		units:         maps.Clone(g.Units),
		positions:     maps.Clone(g.Positions),
		unitPositions: make(map[game.UnitID]gamemap.ProvinceID, len(g.Units)),
		fleetCoasts:   maps.Clone(g.FleetCoasts),
		orders:        maps.Clone(g.Orders),
	}

	for unitID, unit := range ctx.units {
		ctx.unitPositions[unitID] = unit.ProvinceID
	}

	return ctx
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
