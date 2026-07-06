package adjudicator

import (
	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

type Attack struct {
	UnitID    game.UnitID
	From      gamemap.ProvinceID
	FromCoast gamemap.CoastID
	To        gamemap.ProvinceID
	ToCoast   gamemap.CoastID
	ViaConvoy bool
}

func buildAttacks(ctx resolutionContext, moves map[game.UnitID]game.MoveOrder) map[gamemap.ProvinceID][]Attack {
	attacks := make(map[gamemap.ProvinceID][]Attack)
	for unitID, move := range moves {
		pos := ctx.unitPositions[unitID]
		coast := ctx.fleetCoasts[unitID]

		attack := Attack{
			UnitID:    unitID,
			From:      pos,
			FromCoast: coast,
			To:        move.Target,
			ToCoast:   move.TargetCoast,
			ViaConvoy: move.ViaConvoy,
		}
		attacks[move.Target] = append(attacks[move.Target], attack)
	}
	return attacks
}
