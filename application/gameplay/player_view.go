package gameplay

import (
	"context"
	"fmt"
	"maps"

	"github.com/matt-in-space/diplomacy/core/game"
	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// PlayerView is the slice of a Game a specific player is allowed to see —
// public board state (units, turn, supply centers, who's committed) plus
// that player's own orders. It deliberately excludes every other nation's
// order content: that's the entire point of this type existing instead of
// callers serializing *game.Game directly.
type PlayerView struct {
	Turn               game.Turn
	Units              []UnitView
	SupplyCenterOwners map[gamemap.ProvinceID]gamemap.NationID
	CommittedNations   []gamemap.NationID
	YourNation         gamemap.NationID // "" if the requester controls no nation in this game
	YourOrders         []OrderView
}

// A UnitView is a JSON-friendly projection of game.Unit. Unit positions are
// public information in Diplomacy — everyone can see the whole board —
// unlike orders, which stay secret until resolution.
type UnitView struct {
	ID            game.UnitID
	NationID      gamemap.NationID
	ProvinceID    gamemap.ProvinceID
	Type          game.UnitType
	Coast         gamemap.CoastID
	Dislodged     bool
	DislodgedFrom gamemap.ProvinceID // "" unless Dislodged
}

// An OrderView is a human-readable projection of a single unit's order —
// deliberately just a description string, not a structured/round-trippable
// shape. This pass is read-only display; an order-building UI will likely
// want a richer representation to pre-fill an editable form, and can define
// that when it needs it rather than this guessing at it now.
type OrderView struct {
	UnitID      game.UnitID
	Description string
}

// GetPlayerView assembles the state playerID is allowed to see for gameID.
// Only orders belonging to playerID's own nation are ever included in
// YourOrders — every other nation's order content is filtered out, full
// stop. That filter is the reason this type exists instead of callers
// serializing *game.Game directly, and is covered by a dedicated test
// (see player_view_test.go), not just incidental coverage.
//
// This assembles the view only; it does not check whether playerID is a
// legitimate participant in this game at all (YourNation just comes back
// "" if not). That authorization check belongs to the HTTP layer, which
// already has the game setup's player list to check against.
func (s *GameplayService) GetPlayerView(ctx context.Context, gameID game.GameID, playerID game.PlayerID) (PlayerView, error) {
	stored, err := s.games.GetGame(ctx, gameID)
	if err != nil {
		return PlayerView{}, err
	}
	g := stored.Game

	gm, err := s.maps.GetMap(g.MapID)
	if err != nil {
		return PlayerView{}, fmt.Errorf("failed to get game map %q: %w", g.MapID, err)
	}

	view := PlayerView{
		Turn:               g.Turn,
		SupplyCenterOwners: maps.Clone(g.SupplyCenterOwners),
	}

	for _, unit := range g.Units {
		view.Units = append(view.Units, UnitView{
			ID:            unit.ID,
			NationID:      unit.NationID,
			ProvinceID:    unit.ProvinceID,
			Type:          unit.Type,
			Coast:         unit.Coast,
			Dislodged:     unit.Dislodged(),
			DislodgedFrom: unit.DislodgedFrom,
		})
	}

	for nation := range g.CommittedOrders {
		view.CommittedNations = append(view.CommittedNations, nation)
	}

	for nation, pid := range g.Assignments {
		if pid == playerID {
			view.YourNation = nation
			break // a player controls exactly one nation per the current assignment logic
		}
	}

	if view.YourNation != "" {
		for _, order := range g.Orders {
			if order.Nation() != view.YourNation {
				continue // the filter — never expose another nation's orders
			}
			unitOrder, ok := order.(game.UnitOrder)
			if !ok {
				continue // build orders etc. — adjustment-phase display isn't built yet
			}
			view.YourOrders = append(view.YourOrders, OrderView{
				UnitID:      unitOrder.Unit(),
				Description: describeOrder(order, g, gm),
			})
		}
	}

	return view, nil
}

// describeOrder renders a single AcceptOrders-phase unit order as a short
// human-readable string. Orders from other phases (retreats, adjustments)
// fall through to a generic fallback — this endpoint targets the
// AcceptOrders phase, matching the stub UI's own scope; revisit when
// retreat/adjustment-phase display is built.
func describeOrder(order game.Order, g *game.Game, gm *gamemap.GameMap) string {
	switch o := order.(type) {
	case game.HoldOrder:
		return "Hold"
	case game.MoveOrder:
		desc := "Move → " + provinceName(o.Target, gm)
		if o.ViaConvoy {
			desc += " (via convoy)"
		}
		return desc
	case game.SupportHoldOrder:
		return "Support " + unitShorthand(o.SupportedUnit, g, gm) + " to hold"
	case game.SupportMoveOrder:
		return "Support " + unitShorthand(o.SupportedUnit, g, gm) + " → " + provinceName(o.Target, gm)
	case game.ConvoyOrder:
		return "Convoy " + unitShorthand(o.ConvoyedUnit, g, gm) + " → " + provinceName(o.To, gm)
	default:
		return fmt.Sprintf("%T", order)
	}
}

func provinceName(id gamemap.ProvinceID, gm *gamemap.GameMap) string {
	if province, ok := gm.Province(id); ok {
		return province.Name
	}
	return string(id)
}

// unitShorthand renders a unit as real Diplomacy notation, e.g. "A Paris" —
// falls back to the raw unit ID if the unit or its province can't be found,
// which shouldn't happen for a well-formed order but shouldn't panic either.
func unitShorthand(id game.UnitID, g *game.Game, gm *gamemap.GameMap) string {
	unit, ok := g.Units[id]
	if !ok {
		return string(id)
	}
	prefix := "A"
	if unit.Type == game.UnitTypeFleet {
		prefix = "F"
	}
	return prefix + " " + provinceName(unit.ProvinceID, gm)
}
