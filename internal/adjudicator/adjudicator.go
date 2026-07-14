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
	ReasonMisalignedConvoy  ReasonCode = "misaligned_convoy"
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

	ctx := newResolutionContext(g, gm)

	ctx.normalizeOrders()
	ctx.categorizeOrders()
	ctx.buildDefaultResolutions()
	ctx.buildIntendedEndingPositions()
	ctx.pruneMisalignedOrders()

	return Resolution{}, nil
}

type resolutionContext struct {
	gm                *gamemap.GameMap
	units             map[game.UnitID]game.Unit
	fleetCoasts       map[game.UnitID]gamemap.CoastID
	intendedPositions map[gamemap.ProvinceID][]game.UnitID

	// allOrders holds every unit's order (submitted, or a defaulted hold).
	allOrders map[game.UnitID]game.Order

	// Categorized orders
	moveOrders        map[game.UnitID]game.MoveOrder
	holdOrders        map[game.UnitID]game.HoldOrder
	supportHoldOrders map[game.UnitID]game.SupportHoldOrder
	supportMoveOrders map[game.UnitID]game.SupportMoveOrder
	convoyOrders      map[game.UnitID]game.ConvoyOrder

	// Effective orders: the orders that survive pruning and feed the dependency
	// graph. Together these partition every unit into exactly one behavior.
	effectiveHoldOrders        map[game.UnitID]game.HoldOrder
	effectiveMoveOrders        map[game.UnitID]game.MoveOrder
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

func newResolutionContext(g *game.Game, gm *gamemap.GameMap) resolutionContext {
	return resolutionContext{
		gm:                         gm,
		units:                      maps.Clone(g.Units),
		fleetCoasts:                maps.Clone(g.FleetCoasts),
		intendedPositions:          make(map[gamemap.ProvinceID][]game.UnitID),
		allOrders:                  maps.Clone(g.Orders),
		moveOrders:                 make(map[game.UnitID]game.MoveOrder),
		holdOrders:                 make(map[game.UnitID]game.HoldOrder),
		supportHoldOrders:          make(map[game.UnitID]game.SupportHoldOrder),
		supportMoveOrders:          make(map[game.UnitID]game.SupportMoveOrder),
		convoyOrders:               make(map[game.UnitID]game.ConvoyOrder),
		effectiveHoldOrders:        make(map[game.UnitID]game.HoldOrder),
		effectiveMoveOrders:        make(map[game.UnitID]game.MoveOrder),
		effectiveSupportHoldOrders: make(map[game.UnitID]game.SupportHoldOrder),
		effectiveSupportMoveOrders: make(map[game.UnitID]game.SupportMoveOrder),
		effectiveConvoyOrders:      make(map[game.UnitID]game.ConvoyOrder),
		orderOutcomes:              make(map[game.UnitID]OrderOutcome),
		unitOutcomes:               make(map[game.UnitID]UnitOutcome),
	}
}

func (rc *resolutionContext) normalizeOrders() {
	for id, unit := range rc.units {
		if _, ok := rc.allOrders[id]; !ok {
			rc.allOrders[id] = game.NewHoldOrder(unit.ID, unit.NationID)
		}
	}
}

func (rc *resolutionContext) buildIntendedEndingPositions() {
	for _, order := range rc.allOrders {
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
	for _, order := range rc.allOrders {
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

func (rc *resolutionContext) buildDefaultResolutions() {
	for unitID, unit := range rc.units {
		coast := rc.fleetCoasts[unitID]
		rc.unitOutcomes[unitID] = createUnitHoldOutcome(unit, coast)
	}
}

func (rc *resolutionContext) pruneMisalignedOrders() {
	// Holds always survive pruning.
	maps.Copy(rc.effectiveHoldOrders, rc.holdOrders)

	// Direct (non-convoyed) moves always survive pruning; their success is
	// decided later during adjudication. Convoyed moves are handled below, once
	// we know which convoys are aligned.
	for id, order := range rc.moveOrders {
		if !order.ViaConvoy {
			rc.effectiveMoveOrders[id] = order
		}
	}

	// A supporting unit attempting to support a held province fails if the unit
	// being provided support does not hold in that province. The order was already
	// validated so a holding supported unit is holding in the intended support
	// province.
	for _, order := range rc.supportHoldOrders {
		if _, ok := rc.holdOrders[order.SupportedUnit]; !ok {
			rc.demoteToHold(order, ReasonMisalignedSupport)
			continue
		}
		rc.effectiveSupportHoldOrders[order.UnitID] = order
	}

	// Similarly, supporting a move order fails if that unit either does not move,
	// or it moves to a province other than the one the support expects.
	for _, order := range rc.supportMoveOrders {
		move, ok := rc.moveOrders[order.SupportedUnit]
		if !ok || order.Target != move.Target {
			rc.demoteToHold(order, ReasonMisalignedSupport)
			continue
		}
		rc.effectiveSupportMoveOrders[order.UnitID] = order
	}

	// Convoys can fail in two directions. A convoying fleet fails if the unit it
	// names is not moving via convoy to the destination this fleet expects to
	// carry it to.
	convoysByUnit := make(map[game.UnitID][]game.ConvoyOrder)
	for _, order := range rc.convoyOrders {
		move, ok := rc.moveOrders[order.ConvoyedUnit]
		if !ok || !move.ViaConvoy || move.Target != order.To {
			rc.demoteToHold(order, ReasonMisalignedConvoy)
			continue
		}
		rc.effectiveConvoyOrders[order.UnitID] = order
		convoysByUnit[order.ConvoyedUnit] = append(convoysByUnit[order.ConvoyedUnit], order)
	}

	// A convoyed move fails if its aligned carriers cannot form a complete water
	// path from origin to destination. When it fails, the army and every carrying
	// fleet are demoted to holds so none of them enter the dependency graph as
	// part of a doomed convoy. The fleets kept valid convoy orders, so their
	// outcome is a convoy failure rather than a misalignment.
	for id, move := range rc.moveOrders {
		if !move.ViaConvoy {
			continue
		}
		carriers := convoysByUnit[id]
		if rc.convoyPathExists(move, carriers) {
			rc.effectiveMoveOrders[id] = move
			continue
		}
		rc.demoteToHold(move, ReasonConvoyFailure)
		for _, carrier := range carriers {
			delete(rc.effectiveConvoyOrders, carrier.UnitID)
			rc.demoteToHold(carrier, ReasonConvoyFailure)
		}
	}
}

// demoteToHold records a failed outcome for an order and makes the unit hold in
// place, so it still participates in the dependency graph as a holder.
func (rc *resolutionContext) demoteToHold(order game.Order, reason ReasonCode) {
	id := order.Unit()
	rc.effectiveHoldOrders[id] = game.NewHoldOrder(id, order.Nation())
	rc.orderOutcomes[id] = createOrderFailOutcome(order, reason)
}

// convoyPathExists reports whether the aligned carrier fleets form an unbroken
// chain of adjacent water provinces linking the convoyed army's origin to its
// destination. Dislodgement of the carriers is resolved later; this only rules
// out convoys that cannot possibly succeed.
func (rc *resolutionContext) convoyPathExists(move game.MoveOrder, carriers []game.ConvoyOrder) bool {
	if len(carriers) == 0 {
		return false
	}

	unit, ok := rc.units[move.Unit()]
	if !ok {
		return false
	}

	fleetCoasts := make(map[gamemap.CoastID]bool, len(carriers))
	for _, carrier := range carriers {
		fleetCoasts[rc.fleetCoasts[carrier.UnitID]] = true
	}

	toCoasts := rc.gm.CoastsFor(move.Target)
	visited := make(map[gamemap.CoastID]bool)
	var queue []gamemap.CoastID

	enqueueAdjacent := func(from gamemap.CoastID) {
		for _, neighbor := range rc.gm.FleetNeighbors(from) {
			if fleetCoasts[neighbor] && !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	for _, coast := range rc.gm.CoastsFor(unit.ProvinceID) {
		enqueueAdjacent(coast)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, toCoast := range toCoasts {
			if rc.gm.FleetAdjacent(current, toCoast) {
				return true
			}
		}
		enqueueAdjacent(current)
	}

	return false
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

func createUnitHoldOutcome(unit game.Unit, coast gamemap.CoastID) UnitOutcome {
	return UnitOutcome{
		UnitID: unit.ID,
		Type:   UnitOutcomeHold,
		From:   unit.ProvinceID,
		To:     unit.ProvinceID,
		Coast:  coast,
	}
}
