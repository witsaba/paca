package application_test

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

func TestTheClosingOfAMonthReadsTheMoneyAndTheDebts(t *testing.T) {
	t.Parallel()

	owed := theDebtOfTheTests()
	accounts := &fakeDebtAccounts{existing: []account.Account{owed}}
	book := &fakeBook{
		summary: ledger.NewSummary(1000000, 400000),
		balances: []ledger.Balance{{
			AccountID: owed.ID,
			Kind:      account.KindLiability,
			Name:      owed.Name,
			Credits:   500000,
		}},
	}
	service := application.NewSummaryService(accounts, book, theClock())

	closed, err := service.Month(t.Context(), business.ID(77), 2026, 3)
	if err != nil {
		t.Fatalf("the month could not be closed: %v", err)
	}

	if closed.From.String() != "2026-03-01" {
		t.Errorf("the month starts on %q, want %q", closed.From.String(), "2026-03-01")
	}
	if closed.To.String() != "2026-03-31" {
		t.Errorf("the month ends on %q, want %q", closed.To.String(), "2026-03-31")
	}
	if closed.Summary.Income != 1000000 {
		t.Errorf("the income of the month is %d, want 1000000", closed.Summary.Income)
	}
	if closed.Summary.Expenses != 400000 {
		t.Errorf("the expenses of the month are %d, want 400000", closed.Summary.Expenses)
	}
	if closed.Summary.Net() != 600000 {
		t.Errorf("the net of the month is %d, want 600000", closed.Summary.Net())
	}
	if len(closed.Debts) != 1 {
		t.Fatalf("the month holds %d debts, want 1", len(closed.Debts))
	}
	if closed.Debts[0].Amount != 500000 {
		t.Errorf("the debt holds %d, want 500000", closed.Debts[0].Amount)
	}
}

func TestTheClosingOfTheMonthThatIsNotNamedIsTheMonthThatJustEnded(t *testing.T) {
	t.Parallel()

	book := &fakeBook{}
	service := application.NewSummaryService(&fakeDebtAccounts{}, book, theClock())

	closed, err := service.Month(t.Context(), business.ID(77), 0, 0)
	if err != nil {
		t.Fatalf("the month could not be closed: %v", err)
	}

	// The clock of the tests is the 15th of March of 2026, so the month
	// that just ended is the one of February.
	if closed.From.String() != "2026-02-01" {
		t.Errorf("the month starts on %q, want %q", closed.From.String(), "2026-02-01")
	}
	if closed.To.String() != "2026-02-28" {
		t.Errorf("the month ends on %q, want %q", closed.To.String(), "2026-02-28")
	}
}

func TestTheClosingRefusesAMonthThatDoesNotExist(t *testing.T) {
	t.Parallel()

	service := application.NewSummaryService(&fakeDebtAccounts{}, &fakeBook{}, theClock())

	if _, err := service.Month(t.Context(), business.ID(77), 2026, 13); !errors.Is(err, ledger.ErrInvalidDate) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidDate)
	}
	if _, err := service.Month(t.Context(), business.ID(77), 2026, 0); !errors.Is(err, ledger.ErrInvalidDate) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidDate)
	}
}

func TestTheClosingWithoutADatabaseRefusesToWork(t *testing.T) {
	t.Parallel()

	service := application.NewSummaryService(nil, nil, nil)

	if _, err := service.Month(t.Context(), business.ID(77), 2026, 3); !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
