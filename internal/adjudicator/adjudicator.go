package adjudicator

import (
	"errors"
	"maps"

	"github.com/matt-in-space/diplomacy/internal/game"
	"github.com/matt-in-space/diplomacy/internal/gamemap"
)

// Resolution represents the outcome of a turn's order adjudication.
type Resolution struct {
	Turn game.Turn
	// Outcomes maps each UnitID to its resolution Outcome.
	Outcomes map[game.UnitID]Outcome
}

// Outcome describes the result for a single unit after adjudication.
type Outcome struct {
	UnitID game.UnitID
	Unit   UnitOutcome
	Order  OrderOutcome
}

type UnitOutcomeType string

const (
	UnitOutcomeMove    UnitOutcomeType = "move"
	UnitOutcomeHold    UnitOutcomeType = "hold"
	UnitOutcomeRetreat UnitOutcomeType = "retreat"
)

// UnitOutcome details the unit's final position and type.
type UnitOutcome struct {
	UnitID game.UnitID
	Type   UnitOutcomeType
	From   gamemap.ProvinceID
	To     gamemap.ProvinceID
	Coast  gamemap.CoastID
}

type ReasonCode string

const (
	ReasonSuccess           ReasonCode = "success"
	ReasonWeakAttack        ReasonCode = "weak_attack" // e.g., bounce, draw
	ReasonDislodged         ReasonCode = "dislodged"
	ReasonSupportCut        ReasonCode = "support_cut"
	ReasonConvoyFailure     ReasonCode = "convoy_failure"
	ReasonMisalignedSupport ReasonCode = "misaligned_support"
)

// OrderOutcome details whether an order succeeded and why.
type OrderOutcome struct {
	Order   game.Order
	Success bool
	Reason  ReasonCode
}

// Resolve determines the outcome of all unit orders for a given turn phase.
func Resolve(g *game.Game, gm *gamemap.GameMap) (Resolution, error) {
	if g.MapID != gm.ID {
		return Resolution{}, errors.New("unexpected game map provided")
	}

	ctx := newResolutionContext(g)
	ctx.normalizeOrders()
	ctx.categorizeOrders()
	ctx.buildIntendedEndingPositions()
	ctx.pruneMisalignedOrders()

	return Resolution{}, nil
}

type resolutionContext struct {
	units             map[game.UnitID]game.Unit
	fleetCoasts       map[game.UnitID]gamemap.CoastID
	intendedPositions map[gamemap.ProvinceID][]game.UnitID

	// Categorized orders
	moveOrders        map[game.UnitID]game.MoveOrder
	holdOrders        map[game.UnitID]game.HoldOrder
	supportHoldOrders map[game.UnitID]game.SupportHoldOrder
	supportMoveOrders map[game.UnitID]game.SupportMoveOrder
	convoyOrders      map[game.UnitID]game.ConvoyOrder

	// Effective orders
	effectiveOrders            map[game.UnitID]game.Order
	effectiveSupportHoldOrders map[game.UnitID]game.SupportHoldOrder
	effectiveSupportMoveOrders map[game.UnitID]game.SupportMoveOrder
	effectiveConvoyOrders      map[game.UnitID]game.ConvoyOrder

	// Graph data
	dependents map[game.UnitID][]game.UnitID
	indegree   map[game.UnitID]int

	// Outcomes
	orderOutcomes map[game.UnitID]OrderOutcome
	unitOutcomes  map[game.UnitID]UnitOutcome
}

func newResolutionContext(g *game.Game) resolutionContext {
	return resolutionContext{
		units:                      maps.Clone(g.Units),
		fleetCoasts:                maps.Clone(g.FleetCoasts),
		intendedPositions:          make(map[gamemap.ProvinceID][]game.UnitID),
		moveOrders:                 make(map[game.UnitID]game.MoveOrder),
		holdOrders:                 make(map[game.UnitID]game.HoldOrder),
		supportHoldOrders:          make(map[game.UnitID]game.SupportHoldOrder),
		supportMoveOrders:          make(map[game.UnitID]game.SupportMoveOrder),
		convoyOrders:               make(map[game.UnitID]game.ConvoyOrder),
		effectiveOrders:            make(map[game.UnitID]game.Order),
		effectiveSupportHoldOrders: make(map[game.UnitID]game.SupportHoldOrder),
		effectiveSupportMoveOrders: make(map[game.UnitID]game.SupportMoveOrder),
		effectiveConvoyOrders:      make(map[game.UnitID]game.ConvoyOrder),
		orderOutcomes:              make(map[game.UnitID]OrderOutcome),
		unitOutcomes:               make(map[game.UnitID]UnitOutcome),
	}
}

func (rc *resolutionContext) normalizeOrders() {
	for id, unit := range rc.units {
		if _, ok := rc.effectiveOrders[id]; !ok {
			rc.effectiveOrders[id] = game.NewHoldOrder(unit.ID, unit.NationID)
		}
	}
}

func (rc *resolutionContext) buildIntendedEndingPositions() {
	for _, order := range rc.effectiveOrders {
		if moveOrder, ok := order.(game.MoveOrder); ok {
			rc.intendedPositions[moveOrder.Target] = append(rc.intendedPositions[moveOrder.Target], order.Unit())
		} else {
			unit, ok := rc.units[order.Unit()]
			if !ok {
				panic("Invalid UnitID")
			}
			rc.intendedPositions[unit.ProvinceID] = append(rc.intendedPositions[unit.ProvinceID], order.Unit())
		}
	}
}

func (rc *resolutionContext) categorizeOrders() {
	for _, order := range rc.effectiveOrders {
		switch order := order.(type) {
		case game.MoveOrder:
			rc.moveOrders[order.Unit()] = order
		case game.SupportHoldOrder:
			rc.supportHoldOrders[order.Unit()] = order
		case game.SupportMoveOrder:
			rc.supportMoveOrders[order.Unit()] = order
		case game.HoldOrder:
			rc.holdOrders[order.Unit()] = order
		case game.ConvoyOrder:
			rc.convoyOrders[order.Unit()] = order
		}
	}
}

func (rc *resolutionContext) pruneMisalignedOrders() {
	for _, order := range rc.supportHoldOrders {
		_, ok := rc.holdOrders[order.SupportedUnit]
		if !ok {
			rc.orderOutcomes[order.UnitID] = createOrderFailOutcome(order, ReasonMisalignedSupport)
			continue
		}
		// The order was already validated so we can assume since the supported unit
		// is holding that it is holding in the intended support province.
		rc.effectiveSupportHoldOrders[order.UnitID] = order
	}

	for _, order := range rc.supportMoveOrders {
		move, ok := rc.moveOrders[order.SupportedUnit]
		if !ok {
			rc.orderOutcomes[order.UnitID] = createOrderFailOutcome(order, ReasonMisalignedSupport)
			continue
		}
		// In this case we also need to check that the moving unit is ordered to move to the
		// province expected by the support
		if order.Target != move.Target {
			rc.orderOutcomes[order.UnitID] = createOrderFailOutcome(order, ReasonMisalignedSupport)
			continue
		}
		rc.effectiveSupportMoveOrders[order.UnitID] = order
	}
}

func createOrderFailOutcome(order game.Order, reason ReasonCode) OrderOutcome {
	return OrderOutcome{
		Order:   order,
		Success: false,
		Reason:  reason,
	}
}

func createOrderSuccessOutcome(order game.Order) OrderOutcome {
	return OrderOutcome{
		Order:   order,
		Success: true,
		Reason:  ReasonSuccess,
	}
}
