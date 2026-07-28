package game

import (
	"fmt"

	"github.com/matt-in-space/diplomacy/core/gamemap"
)

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

func (g *Game) CompleteOrderResolution(transforms []UnitTransform) error {
	if g.Turn.Phase != ResolveOrders {
		return fmt.Errorf("order resolution can only be completed in the resolve orders phase")
	}
	if err := g.ApplyUnitTransforms(transforms); err != nil {
		return err
	}

	g.CommittedOrders = make(map[gamemap.NationID]struct{})
	g.Orders = make(map[UnitID]Order)
	g.Turn = g.Turn.Next()

	return nil
}
