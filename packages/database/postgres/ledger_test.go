package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

func TestTheRepositoryOfTheBookNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewLedgerRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRepositoryOfTheBookWithoutADatabaseRefusesToWork(t *testing.T) {
	t.Parallel()

	repo := &LedgerRepository{}

	day := ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	entry, err := ledger.NewEntry(business.ID(77), account.ID(1), account.ID(2), 5000, day, "Lunch")
	if err != nil {
		t.Fatalf("the movement could not be built: %v", err)
	}

	if _, err := repo.Record(t.Context(), entry); !errors.Is(err, ledger.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, ledger.ErrNoStorage)
	}
	if _, err := repo.List(t.Context(), 1, day, day); !errors.Is(err, ledger.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, ledger.ErrNoStorage)
	}
	if _, err := repo.Balances(t.Context(), 1, day); !errors.Is(err, ledger.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, ledger.ErrNoStorage)
	}
}

func TestTheRowOfAMovementIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	moment := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	mapped, err := ledgerEntryFromRow(db.LedgerEntry{
		ID:              7,
		BusinessID:      77,
		DebitAccountID:  1,
		CreditAccountID: 2,
		Amount:          5000,
		OccurredOn:      pgtype.Date{Time: moment, Valid: true},
		Description:     "Lunch",
		CreatedAt:       pgtype.Timestamptz{Time: moment, Valid: true},
	})
	if err != nil {
		t.Fatalf("the row could not be mapped: %v", err)
	}

	if mapped.ID != ledger.EntryID(7) {
		t.Errorf("the identity is %d, want 7", mapped.ID)
	}
	if mapped.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", mapped.BusinessID)
	}
	if mapped.DebitAccountID != account.ID(1) {
		t.Errorf("the debit account is %d, want 1", mapped.DebitAccountID)
	}
	if mapped.CreditAccountID != account.ID(2) {
		t.Errorf("the credit account is %d, want 2", mapped.CreditAccountID)
	}
	if mapped.Amount != 5000 {
		t.Errorf("the amount is %d, want 5000", mapped.Amount)
	}
	if mapped.OccurredOn.String() != "2026-03-01" {
		t.Errorf("the day is %q, want %q", mapped.OccurredOn.String(), "2026-03-01")
	}
	if mapped.Description != "Lunch" {
		t.Errorf("the description is %q, want %q", mapped.Description, "Lunch")
	}
}

func TestTheRefusalsOfTheDatabaseAreTranslatedForTheBook(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "an amount that is not positive",
			err:  theRefusal(checkViolation, ledgerAmountConstraint),
			want: ledger.ErrInvalidAmount,
		},
		{
			name: "the two sides of the movement are the same account",
			err:  theRefusal(checkViolation, ledgerDistinctConstraint),
			want: ledger.ErrSameAccount,
		},
		{
			name: "a movement without a description",
			err:  theRefusal(checkViolation, ledgerDescriptionConstraint),
			want: ledger.ErrInvalidDescription,
		},
		{
			name: "a business that is not there",
			err:  theRefusal(foreignKeyViolation, "ledger_entries_business_id_fkey"),
			want: ledger.ErrNotFound,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := writeLedgerError(testCase.err)
			if !errors.Is(got, testCase.want) {
				t.Fatalf("the error is %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestAMovementThatIsNotThereIsNotFound(t *testing.T) {
	t.Parallel()

	if !errors.Is(writeLedgerError(pgxErrNoRows()), ledger.ErrNotFound) {
		t.Error("the error of the book is not the error of a row that is not there")
	}
}
