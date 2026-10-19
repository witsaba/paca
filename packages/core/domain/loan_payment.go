package domain

// LoanPayment is one obligation of a loan: what is owed, and when it falls
// due.
//
// The amount is the scheduled obligation. The cash that was actually paid
// lives in the movement that settled it, so a partial payment is
// representable: the payment keeps its scheduled amount, and the movement
// carries what really moved.
type LoanPayment struct {
	// ID is the identity of the payment. It is zero before the payment is
	// stored.
	ID int64

	// LoanID is the loan the payment belongs to.
	LoanID int64

	// Seq is the position of the payment in the schedule of the loan. It
	// starts at one.
	Seq int

	// Amount is the scheduled amount, in Colombian pesos.
	Amount int64

	// DueOn is the day the payment falls due, as YYYY-MM-DD.
	DueOn string

	// PaidOn is the day the payment was settled, as YYYY-MM-DD, and is
	// empty while it is outstanding.
	PaidOn string

	// MovementID is the movement that settled the payment, and is nil when
	// the money did not move through a tracked account, or has not moved
	// yet.
	MovementID *int64

	// Notes is the free text of the payment.
	Notes string

	// CreatedAt is the moment the payment was stored, as an ISO-8601
	// timestamp in UTC.
	CreatedAt string
}
