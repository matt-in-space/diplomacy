package game

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

// Resolution represents the outcome of a turn's order adjudication.
type Resolution map[UnitID]Outcome

// Outcome describes the result for a single unit after adjudication.
type Outcome struct {
	UnitID UnitID
	Unit   UnitTransform
	Order  OrderOutcome
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
	Order   Order
	Success bool
	Reason  ReasonCode
}

func CreateOrderFailOutcome(order Order, reason ReasonCode) OrderOutcome {
	return OrderOutcome{
		Order:   order,
		Success: false,
		Reason:  reason,
	}
}

func CreateOrderSuccessOutcome(order Order) OrderOutcome {
	return OrderOutcome{
		Order:   order,
		Success: true,
		Reason:  ReasonSuccess,
	}
}

func (g *Game) BeginOrderResolution() (progressed bool, err error) {
	if g.Turn.Phase != AcceptOrders {
		return false, fmt.Errorf("order resolution can only be started in the accept orders phase")
	}
	if !g.allOrdersSubmitted() {
		return false, nil
	}

	g.Turn = g.Turn.Next()

	return true, nil
}

func (g *Game) allOrdersSubmitted() bool {
	for nationID := range g.Assignments {
		if _, ok := g.CommittedOrders[nationID]; !ok {
			return false
		}
	}
	return true
}

// BeginRetreatResolution advances the game out of the accept retreats phase
// once every nation with a dislodged unit has committed. A turn with no
// dislodgements has no nations to wait for and advances immediately.
func (g *Game) BeginRetreatResolution() (progressed bool, err error) {
	if g.Turn.Phase != AcceptRetreats {
		return false, fmt.Errorf("retreat resolution can only be started in the accept retreats phase")
	}

	for nationID := range g.nationsAwaitingRetreats() {
		if _, ok := g.CommittedOrders[nationID]; !ok {
			return false, nil
		}
	}

	g.Turn = g.Turn.Next()

	return true, nil
}

// nationsAwaitingRetreats returns the set of nations with at least one
// dislodged unit, which must commit before the retreat phase can advance.
func (g *Game) nationsAwaitingRetreats() map[gamemap.NationID]struct{} {
	awaiting := make(map[gamemap.NationID]struct{})
	for _, unit := range g.Units {
		if unit.Dislodged() {
			awaiting[unit.NationID] = struct{}{}
		}
	}
	return awaiting
}

func (g *Game) CompleteOrderResolution(res Resolution) error {
	if g.Turn.Phase != ResolveOrders {
		return fmt.Errorf("order resolution can only be completed in the resolve orders phase")
	}

	transforms := make([]UnitTransform, 0, len(res))
	for _, t := range res {
		transforms = append(transforms, t.Unit)
	}

	if err := g.applyUnitTransforms(transforms); err != nil {
		return err
	}

	g.CommittedOrders = make(map[gamemap.NationID]struct{})
	g.Orders = make(map[UnitID]Order)
	g.LastOrderResolution = res
	g.Turn = g.Turn.Next()

	return nil
}
