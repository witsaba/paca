package domain

// MovementKind is the kind of a movement, and it decides the sign of its
// effect on the balance of an account.
type MovementKind string

const (
	// MovementKindIncome is money in from outside.
	MovementKindIncome MovementKind = "income"

	// MovementKindExpense is money out to outside.
	MovementKindExpense MovementKind = "expense"

	// MovementKindTransferIn is money in from another account.
	MovementKindTransferIn MovementKind = "transfer_in"

	// MovementKindTransferOut is money out to another account.
	MovementKindTransferOut MovementKind = "transfer_out"

	// MovementKindLoanReceived is cash in for a loan PACA owes.
	MovementKindLoanReceived MovementKind = "loan_received"

	// MovementKindLoanLent is cash out for a loan PACA made.
	MovementKindLoanLent MovementKind = "loan_lent"

	// MovementKindLoanRepayment is cash out to repay a loan.
	MovementKindLoanRepayment MovementKind = "loan_repayment"

	// MovementKindLoanRecovered is cash in from a loan PACA made.
	MovementKindLoanRecovered MovementKind = "loan_recovered"
)

// Sign returns the sign of the effect of the kind on the balance of an
// account: 1 when the money comes in, -1 when it goes out.
//
// The sign is a rule of the business and not of the storage: the amount of a
// movement is always positive, and the kind says which way the money went.
func (k MovementKind) Sign() int {
	switch k {
	case MovementKindIncome, MovementKindTransferIn, MovementKindLoanReceived, MovementKindLoanRecovered:
		return 1
	case MovementKindExpense, MovementKindTransferOut, MovementKindLoanLent, MovementKindLoanRepayment:
		return -1
	default:
		return 0
	}
}

// Movement is a money event: the table that grows.
//
// The amount is always positive, so the direction of the flow is never
// ambiguous: the kind decides whether the money comes in or goes out.
//
// A movement may only be attached to a loan for a loan-related kind. A
// repayment movement and its loan payment describe one event from two
// angles: the movement is the cash that moved, the payment is the
// obligation. The two amounts may differ, which is how a partial payment is
// recorded.
type Movement struct {
	// ID is the identity of the movement. It is zero before the movement is
	// stored.
	ID int64

	// AccountID is the account the money moved through.
	AccountID int64

	// Kind is the kind of the movement.
	Kind MovementKind

	// Amount is the amount of the movement, in Colombian pesos. It is
	// always positive.
	Amount int64

	// OccurredOn is the day the money moved, as YYYY-MM-DD.
	OccurredOn string

	// Category is the label of the movement. It is free text.
	Category string

	// CounterpartyContactID is the contact the movement is with, and is nil
	// when there is none.
	CounterpartyContactID *int64

	// LoanID is the loan the movement belongs to, and is nil when it
	// belongs to none.
	LoanID *int64

	// Notes is the free text of the movement.
	Notes string

	// CreatedAt is the moment the movement was stored, as an ISO-8601
	// timestamp in UTC.
	CreatedAt string
}
