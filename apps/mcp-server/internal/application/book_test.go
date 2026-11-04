package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// The fake of the storage of the book.
type fakeBook struct {
	recorded []ledger.Entry
	listed   []ledger.Entry
	balances []ledger.Balance
	summary  ledger.Summary
	from     ledger.Date
	to       ledger.Date
	asOf     ledger.Date
	err      error
}

// Record remembers the movement and answers it.
func (f *fakeBook) Record(_ context.Context, entry ledger.Entry) (ledger.Entry, error) {
	if f.err != nil {
		return ledger.Entry{}, f.err
	}

	entry.ID = 7
	f.recorded = append(f.recorded, entry)

	return entry, nil
}

// List answers the movements of the period and remembers the period.
func (f *fakeBook) List(_ context.Context, _ business.ID, from, to ledger.Date) ([]ledger.Entry, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.from = from
	f.to = to

	return f.listed, nil
}

// Summary answers the money of a period and remembers the two ends of it.
func (f *fakeBook) Summary(_ context.Context, _ business.ID, from, to ledger.Date) (ledger.Summary, error) {
	if f.err != nil {
		return ledger.Summary{}, f.err
	}

	f.from = from
	f.to = to

	return f.summary, nil
}

// Balances answers the balances of the book and remembers the day.
func (f *fakeBook) Balances(_ context.Context, _ business.ID, asOf ledger.Date) ([]ledger.Balance, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.asOf = asOf

	return f.balances, nil
}

// The fake of the clock of the service.
type fakeBookClock struct {
	moment time.Time
}

// Now answers the moment of the test.
func (f *fakeBookClock) Now() time.Time {
	return f.moment
}

// theClock is the clock the tests of the book use.
func theClock() *fakeBookClock {
	return &fakeBookClock{moment: time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC)}
}

func TestTheBookRecordsTheMovementOfTheMoney(t *testing.T) {
	t.Parallel()

	book := &fakeBook{}
	service := application.NewBookService(book, theClock())

	written, err := service.Record(t.Context(), business.ID(77), account.ID(1), account.ID(2), 5000, "2026-03-01", "Lunch")
	if err != nil {
		t.Fatalf("the movement could not be recorded: %v", err)
	}

	if len(book.recorded) != 1 {
		t.Fatalf("the storage holds %d movements, want 1", len(book.recorded))
	}

	entry := book.recorded[0]
	if entry.DebitAccountID != account.ID(2) {
		t.Errorf("the money lands in the account %d, want %d", entry.DebitAccountID, account.ID(2))
	}
	if entry.CreditAccountID != account.ID(1) {
		t.Errorf("the money comes from the account %d, want %d", entry.CreditAccountID, account.ID(1))
	}
	if entry.Amount != 5000 {
		t.Errorf("the amount is %d, want 5000", entry.Amount)
	}
	if entry.OccurredOn.String() != "2026-03-01" {
		t.Errorf("the day is %q, want %q", entry.OccurredOn.String(), "2026-03-01")
	}
	if written.ID != ledger.EntryID(7) {
		t.Errorf("the identity is %d, want 7", written.ID)
	}
}

func TestTheBookRecordsTheMovementOfTodayWhenTheDayIsNotNamed(t *testing.T) {
	t.Parallel()

	book := &fakeBook{}
	service := application.NewBookService(book, theClock())

	if _, err := service.Record(t.Context(), business.ID(77), account.ID(1), account.ID(2), 5000, "", "Lunch"); err != nil {
		t.Fatalf("the movement could not be recorded: %v", err)
	}

	if book.recorded[0].OccurredOn.String() != "2026-03-15" {
		t.Errorf("the day is %q, want %q", book.recorded[0].OccurredOn.String(), "2026-03-15")
	}
}

func TestTheBookRefusesWhatTheDomainRefuses(t *testing.T) {
	t.Parallel()

	book := &fakeBook{}
	service := application.NewBookService(book, theClock())

	cases := []struct {
		name        string
		amount      ledger.Amount
		date        string
		description string
		want        error
	}{
		{
			name:        "an amount that is not positive",
			amount:      0,
			date:        "2026-03-01",
			description: "Lunch",
			want:        ledger.ErrInvalidAmount,
		},
		{
			name:        "a day that is not a day",
			amount:      5000,
			date:        "the first of March",
			description: "Lunch",
			want:        ledger.ErrInvalidDate,
		},
		{
			name:        "a movement without a description",
			amount:      5000,
			date:        "2026-03-01",
			description: "   ",
			want:        ledger.ErrInvalidDescription,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := service.Record(
				t.Context(), business.ID(77), account.ID(1), account.ID(2),
				testCase.amount, testCase.date, testCase.description,
			)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
			if len(book.recorded) != 0 {
				t.Errorf("the movement was written: the storage holds %d", len(book.recorded))
			}
		})
	}
}

func TestTheBookReadsTheMovementsOfAPeriod(t *testing.T) {
	t.Parallel()

	book := &fakeBook{}
	service := application.NewBookService(book, theClock())

	if _, err := service.Movements(t.Context(), business.ID(77), "2026-03-01", "2026-03-31"); err != nil {
		t.Fatalf("the movements could not be read: %v", err)
	}

	if book.from.String() != "2026-03-01" {
		t.Errorf("the first day is %q, want %q", book.from.String(), "2026-03-01")
	}
	if book.to.String() != "2026-03-31" {
		t.Errorf("the last day is %q, want %q", book.to.String(), "2026-03-31")
	}
}

func TestTheBookReadsTheBalancesOfTheBook(t *testing.T) {
	t.Parallel()

	book := &fakeBook{balances: []ledger.Balance{
		{AccountID: 1, Kind: account.KindAsset, Name: "Efectivo", Debits: 100000},
		{AccountID: 2, Kind: account.KindExpense, Name: "Alimentación", Debits: 5000},
	}}
	service := application.NewBookService(book, theClock())

	money, balances, err := service.Balances(t.Context(), business.ID(77), "2026-03-31")
	if err != nil {
		t.Fatalf("the balances could not be read: %v", err)
	}

	if len(balances) != 2 {
		t.Fatalf("the answer holds %d balances, want 2", len(balances))
	}
	if money.Assets != 100000 {
		t.Errorf("the assets are %d, want 100000", money.Assets)
	}
	if money.Expenses != 5000 {
		t.Errorf("the expenses are %d, want 5000", money.Expenses)
	}
	if book.asOf.String() != "2026-03-31" {
		t.Errorf("the day is %q, want %q", book.asOf.String(), "2026-03-31")
	}
}

func TestTheBookWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewBookService(nil, nil)

	if _, err := service.Record(t.Context(), 77, 1, 2, 5000, "", "Lunch"); !errors.Is(err, ports.ErrNoDatabase) {
		t.Errorf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
	if _, err := service.Movements(t.Context(), 77, "", ""); !errors.Is(err, ports.ErrNoDatabase) {
		t.Errorf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
	if _, _, err := service.Balances(t.Context(), 77, ""); !errors.Is(err, ports.ErrNoDatabase) {
		t.Errorf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

func TestTheBookRefusesAPeriodThatEndsBeforeItBegins(t *testing.T) {
	t.Parallel()

	book := &fakeBook{}
	service := application.NewBookService(book, theClock())

	_, err := service.Movements(t.Context(), business.ID(77), "2026-03-31", "2026-03-01")
	if !errors.Is(err, ledger.ErrInvalidPeriod) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidPeriod)
	}

	if len(book.listed) != 0 {
		t.Errorf("the storage holds %d movements, want none", len(book.listed))
	}
}
