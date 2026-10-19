package domain

// AccountKind is the kind of an account: where the money is held.
type AccountKind string

const (
	// AccountKindNequi is a Nequi account, identified by its phone number.
	AccountKindNequi AccountKind = "nequi"

	// AccountKindBank is a bank account, identified by the bank and the
	// account number.
	AccountKindBank AccountKind = "bank"

	// AccountKindCash is a cash account: money on hand, with no institution
	// behind it.
	AccountKindCash AccountKind = "cash"
)

// Account is a place where money is held.
//
// The subtype of the account decides which of the subtype fields carry a
// value: a Nequi account carries a phone, a bank account carries the bank
// code, the account number and the account type, and a cash account carries
// none of them. The rule belongs to the schema, which refuses the
// combinations that make no sense.
//
// The schema has a holder column, and its shape constraint requires that
// column to be empty for every kind of account, so the domain does not model
// it. The reference of the repository documents the column as a field of a
// Nequi account: the constraint and the documentation disagree, and the
// constraint is the one that is enforced.
type Account struct {
	// ID is the identity of the account. It is zero before the account is
	// stored.
	ID int64

	// Kind is the kind of the account.
	Kind AccountKind

	// Name is the name the account is known by.
	Name string

	// Currency is the currency of the account. MVP1 is Colombian pesos only.
	Currency string

	// OpenedOn is the day the account was opened, as YYYY-MM-DD.
	OpenedOn string

	// ClosedOn is the day the account was closed, as YYYY-MM-DD, and is
	// empty while the account is open.
	ClosedOn string

	// IsActive reports whether money can still move through the account.
	IsActive bool

	// Notes is the free text of the account.
	Notes string

	// Phone is the phone number of a Nequi account.
	Phone string

	// BankCode is the code of the bank of a bank account.
	BankCode string

	// AccountNumber is the number of a bank account.
	AccountNumber string

	// AccountType is the kind of a bank account: "ahorro" or "corriente".
	AccountType string

	// CreatedAt is the moment the account was stored, as an ISO-8601
	// timestamp in UTC.
	CreatedAt string
}
