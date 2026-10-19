package domain

// LoanDirection is the side of a loan PACA is on.
type LoanDirection string

const (
	// LoanDirectionBorrowed means PACA owes the contact: an informal credit
	// from the gardener, or a formal loan from a financial institution.
	LoanDirectionBorrowed LoanDirection = "borrowed"

	// LoanDirectionLent means the contact owes PACA.
	LoanDirectionLent LoanDirection = "lent"
)

// LoanKind is the nature of a loan.
type LoanKind string

const (
	// LoanKindInformal is a loan without a written agreement.
	LoanKindInformal LoanKind = "informal"

	// LoanKindFormal is a loan with a written agreement.
	LoanKindFormal LoanKind = "formal"
)

// LoanStatus is the state of a loan.
type LoanStatus string

const (
	// LoanStatusActive is a loan that is still running.
	LoanStatusActive LoanStatus = "active"

	// LoanStatusSettled is a loan that is paid off.
	LoanStatusSettled LoanStatus = "settled"

	// LoanStatusCancelled is a loan that was retired. A loan is never
	// deleted: it is cancelled.
	LoanStatusCancelled LoanStatus = "cancelled"
)

// Loan is a debt contract with a contact.
//
// The direction says which side of it PACA is on, and the principal is the
// amount that was lent. The status is kept by the application and reconciled
// against the outstanding balance, which the movements determine.
type Loan struct {
	// ID is the identity of the loan. It is zero before the loan is stored.
	ID int64

	// ContactID is the contact the loan is with.
	ContactID int64

	// Direction is the side of the loan PACA is on.
	Direction LoanDirection

	// Kind is the nature of the loan.
	Kind LoanKind

	// Principal is the amount that was lent, in Colombian pesos.
	Principal int64

	// InterestRateBP is the agreed annual rate, in basis points: 100 is
	// one percent. It is nil when no rate is recorded.
	InterestRateBP *int64

	// GrantedOn is the day the loan was granted, as YYYY-MM-DD.
	GrantedOn string

	// DueOn is the day the loan falls due, as YYYY-MM-DD, and is empty when
	// no due date was agreed.
	DueOn string

	// Status is the state of the loan.
	Status LoanStatus

	// Notes is the free text of the loan.
	Notes string

	// CreatedAt is the moment the loan was stored, as an ISO-8601
	// timestamp in UTC.
	CreatedAt string
}
