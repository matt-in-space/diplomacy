package game

type Season string

const (
	Spring Season = "spring"
	Fall   Season = "fall"
)

type Phase string

const (
	AcceptOrders       Phase = "accept_orders"
	ResolveOrders      Phase = "resolve_orders"
	AcceptRetreats     Phase = "accept_retreats"
	ResolveRetreats    Phase = "resolve_retreats"
	UpdateOwnership    Phase = "update_ownership"
	AcceptAdjustments  Phase = "accept_adjustments"
	ResolveAdjustments Phase = "resolve_adjustments"
	Completed          Phase = "completed"
)

// Turn represents a complete turn in the game with alternating player input
// and rule resolution phases.
type Turn struct {
	Season Season
	Phase  Phase
	Year   int
}

// StartingTurn returns the starting turn of the game in Spring of Year 1.
func StartingTurn() Turn {
	return Turn{
		Season: Spring,
		Phase:  AcceptOrders,
		Year:   1,
	}
}

// AcceptsOrders reports whether the phase is one in which players submit and
// commit orders (as opposed to a resolution phase, which the game itself
// drives).
func (t Turn) AcceptsOrders() bool {
	switch t.Phase {
	case AcceptOrders, AcceptRetreats, AcceptAdjustments:
		return true
	default:
		return false
	}
}

func (t Turn) Next() Turn {
	switch t.Phase {
	case AcceptOrders:
		t.Phase = ResolveOrders
	case ResolveOrders:
		t.Phase = AcceptRetreats
	case AcceptRetreats:
		t.Phase = ResolveRetreats
	case ResolveRetreats:
		if t.Season == Spring {
			t.Season = Fall
			t.Phase = AcceptOrders
		} else {
			t.Phase = UpdateOwnership
		}
	case UpdateOwnership:
		t.Phase = AcceptAdjustments
	case AcceptAdjustments:
		t.Phase = ResolveAdjustments
	case ResolveAdjustments:
		t.Season = Spring
		t.Phase = AcceptOrders
		t.Year++
	case Completed:
		return t
	}

	return t
}
