package domain_test

import (
	"testing"

	"github.com/witsaba/paca/packages/core/domain"
)

// The vocabulary of the domain is the vocabulary of the schema: a value that
// the schema rejects is a value the domain must not name.
func TestAccountKindsAreTheOnesOfTheSchema(t *testing.T) {
	cases := map[domain.AccountKind]string{
		domain.AccountKindNequi: "nequi",
		domain.AccountKindBank:  "bank",
		domain.AccountKindCash:  "cash",
	}

	for kind, want := range cases {
		if string(kind) != want {
			t.Errorf("the account kind is %q, want %q", kind, want)
		}
	}

	if len(cases) != 3 {
		t.Errorf("the domain names %d account kinds, want 3", len(cases))
	}
}

func TestContactKindsAreTheOnesOfTheSchema(t *testing.T) {
	cases := map[domain.ContactKind]string{
		domain.ContactKindPerson: "person",
		domain.ContactKindEntity: "entity",
	}

	for kind, want := range cases {
		if string(kind) != want {
			t.Errorf("the contact kind is %q, want %q", kind, want)
		}
	}

	if len(cases) != 2 {
		t.Errorf("the domain names %d contact kinds, want 2", len(cases))
	}
}

func TestLoanVocabulariesAreTheOnesOfTheSchema(t *testing.T) {
	directions := map[domain.LoanDirection]string{
		domain.LoanDirectionBorrowed: "borrowed",
		domain.LoanDirectionLent:     "lent",
	}

	for direction, want := range directions {
		if string(direction) != want {
			t.Errorf("the loan direction is %q, want %q", direction, want)
		}
	}

	kinds := map[domain.LoanKind]string{
		domain.LoanKindInformal: "informal",
		domain.LoanKindFormal:   "formal",
	}

	for kind, want := range kinds {
		if string(kind) != want {
			t.Errorf("the loan kind is %q, want %q", kind, want)
		}
	}

	statuses := map[domain.LoanStatus]string{
		domain.LoanStatusActive:    "active",
		domain.LoanStatusSettled:   "settled",
		domain.LoanStatusCancelled: "cancelled",
	}

	for status, want := range statuses {
		if string(status) != want {
			t.Errorf("the loan status is %q, want %q", status, want)
		}
	}
}

func TestMovementKindsAreTheOnesOfTheSchema(t *testing.T) {
	cases := map[domain.MovementKind]string{
		domain.MovementKindIncome:        "income",
		domain.MovementKindExpense:       "expense",
		domain.MovementKindTransferIn:    "transfer_in",
		domain.MovementKindTransferOut:   "transfer_out",
		domain.MovementKindLoanReceived:  "loan_received",
		domain.MovementKindLoanLent:      "loan_lent",
		domain.MovementKindLoanRepayment: "loan_repayment",
		domain.MovementKindLoanRecovered: "loan_recovered",
	}

	for kind, want := range cases {
		if string(kind) != want {
			t.Errorf("the movement kind is %q, want %q", kind, want)
		}
	}

	if len(cases) != 8 {
		t.Errorf("the domain names %d movement kinds, want 8", len(cases))
	}
}
