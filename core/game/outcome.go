package game

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

// Outcome describes the result for a single unit after adjudication.
type Outcome struct {
	UnitID UnitID
	Unit   UnitTransform
	Order  OrderOutcome
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
