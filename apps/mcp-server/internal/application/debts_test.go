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

// The fake of the accounts of the debts.
//
// It answers the accounts it holds and it remembers the ones it wrote.
type fakeDebtAccounts struct {
	existing []account.Account
	written  []account.Account
	found    account.Account
	err      error
}

// Create writes the account and answers it with an identity.
func (f *fakeDebtAccounts) Create(_ context.Context, written account.Account) (account.Account, error) {
	if f.err != nil {
		return account.Account{}, f.err
	}

	written.ID = account.ID(9 + len(f.written))
	f.written = append(f.written, written)

	return written, nil
}

// CreateMany writes the set of accounts and remembers it.
func (f *fakeDebtAccounts) CreateMany(_ context.Context, accounts []account.Account) ([]account.Account, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.written = append(f.written, accounts...)

	return accounts, nil
}

// Find answers the account of the identity.
func (f *fakeDebtAccounts) Find(_ context.Context, id account.ID) (account.Account, error) {
	if f.err != nil {
		return account.Account{}, f.err
	}

	if f.found.ID == 0 {
		return account.Account{}, account.ErrNotFound
	}

	return f.found, nil
}

// List answers the accounts of the business.
func (f *fakeDebtAccounts) List(_ context.Context, _ business.ID) ([]account.Account, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.existing, nil
}

// The fake of the transaction of the debts.
//
// It runs the work, and it remembers that it ran: the debt and the money
// that opened it are written whole or not at all.
type fakeDebtWork struct {
	ran  int
	fail error
}

// Do runs the work of the transaction.
func (f *fakeDebtWork) Do(ctx context.Context, work func(context.Context) error) error {
	f.ran++

	if f.fail != nil {
		return f.fail
	}

	return work(ctx)
}

// The debt of the tests: the account of a friend and what is owed.
func theDebtOfTheTests() account.Account {
	due := time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC)

	return account.Account{
		ID:           account.ID(9),
		BusinessID:   business.ID(77),
		Kind:         account.KindLiability,
		Name:         "Deuda con Ana",
		Counterparty: "Ana",
		DueDate:      &due,
	}
}

func TestTheDebtIsOpenedWithTheMoneyThatOpenedIt(t *testing.T) {
	t.Parallel()

	accounts := &fakeDebtAccounts{}
	book := &fakeBook{}
	work := &fakeDebtWork{}
	service := application.NewDebtService(accounts, book, work, theClock())

	opened, err := service.Record(t.Context(), business.ID(77), "Ana", 500000, account.ID(1), "2026-03-01", "", "2026-04-30", "")
	if err != nil {
		t.Fatalf("the debt could not be opened: %v", err)
	}

	if len(accounts.written) != 1 {
		t.Fatalf("the storage holds %d accounts, want 1", len(accounts.written))
	}

	written := accounts.written[0]
	if written.Kind != account.KindLiability {
		t.Errorf("the kind of the account is %q, want %q", written.Kind, account.KindLiability)
	}
	if written.Counterparty != "Ana" {
		t.Errorf("the counterparty is %q, want %q", written.Counterparty, "Ana")
	}
	if written.DueDate == nil || written.DueDate.Format("2006-01-02") != "2026-04-30" {
		t.Errorf("the due date is %v, want the 30th of April", written.DueDate)
	}
	if written.Name != "Deuda con Ana" {
		t.Errorf("the name is %q, want %q", written.Name, "Deuda con Ana")
	}

	if len(book.recorded) != 1 {
		t.Fatalf("the storage holds %d movements, want 1", len(book.recorded))
	}

	movement := book.recorded[0]
	if movement.DebitAccountID != account.ID(1) {
		t.Errorf("the money lands in the account %d, want %d", movement.DebitAccountID, account.ID(1))
	}
	if movement.CreditAccountID != written.ID {
		t.Errorf("the money comes from the account %d, want %d", movement.CreditAccountID, written.ID)
	}
	if movement.Amount != 500000 {
		t.Errorf("the amount is %d, want 500000", movement.Amount)
	}
	if movement.Description != "Deuda con Ana" {
		t.Errorf("the description is %q, want %q", movement.Description, "Deuda con Ana")
	}

	if opened.Amount != 500000 {
		t.Errorf("the debt holds %d, want 500000", opened.Amount)
	}
	if work.ran != 1 {
		t.Errorf("the transaction ran %d times, want 1", work.ran)
	}
}

func TestTheDebtWithoutTheOtherSideOfTheMoneyIsRefused(t *testing.T) {
	t.Parallel()

	service := application.NewDebtService(&fakeDebtAccounts{}, &fakeBook{}, &fakeDebtWork{}, theClock())

	if _, err := service.Record(t.Context(), business.ID(77), "  ", 500000, account.ID(1), "2026-03-01", "", "", ""); !errors.Is(err, ports.ErrCounterpartyRequired) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrCounterpartyRequired)
	}
}

func TestTheDebtTakesTheDayOfTodayWhenTheDayIsNotNamed(t *testing.T) {
	t.Parallel()

	book := &fakeBook{}
	service := application.NewDebtService(&fakeDebtAccounts{}, book, &fakeDebtWork{}, theClock())

	if _, err := service.Record(t.Context(), business.ID(77), "Ana", 500000, account.ID(1), "", "", "", ""); err != nil {
		t.Fatalf("the debt could not be opened: %v", err)
	}

	if book.recorded[0].OccurredOn.String() != "2026-03-15" {
		t.Errorf("the day is %q, want %q", book.recorded[0].OccurredOn.String(), "2026-03-15")
	}
}

func TestTheDebtRefusesADueDateThatIsNotADay(t *testing.T) {
	t.Parallel()

	service := application.NewDebtService(&fakeDebtAccounts{}, &fakeBook{}, &fakeDebtWork{}, theClock())

	_, err := service.Record(t.Context(), business.ID(77), "Ana", 500000, account.ID(1), "2026-03-01", "", "the thirtieth", "")
	if !errors.Is(err, ledger.ErrInvalidDate) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidDate)
	}
}

func TestTheDebtsOfTheBookAreRead(t *testing.T) {
	t.Parallel()

	owed := theDebtOfTheTests()
	accounts := &fakeDebtAccounts{existing: []account.Account{owed}}
	book := &fakeBook{balances: []ledger.Balance{{
		AccountID: owed.ID,
		Kind:      account.KindLiability,
		Name:      owed.Name,
		Credits:   500000,
	}}}
	service := application.NewDebtService(accounts, book, &fakeDebtWork{}, theClock())

	debts, err := service.Debts(t.Context(), business.ID(77))
	if err != nil {
		t.Fatalf("the debts could not be read: %v", err)
	}

	if len(debts) != 1 {
		t.Fatalf("the book holds %d debts, want 1", len(debts))
	}
	if debts[0].Amount != 500000 {
		t.Errorf("the debt holds %d, want 500000", debts[0].Amount)
	}
	if debts[0].Account.Counterparty != "Ana" {
		t.Errorf("the counterparty is %q, want %q", debts[0].Account.Counterparty, "Ana")
	}
}

func TestThePaymentOfADebtReducesWhatItHolds(t *testing.T) {
	t.Parallel()

	owed := theDebtOfTheTests()
	accounts := &fakeDebtAccounts{found: owed}
	book := &fakeBook{balances: []ledger.Balance{{
		AccountID: owed.ID,
		Kind:      account.KindLiability,
		Name:      owed.Name,
		Credits:   500000,
	}}}
	service := application.NewDebtService(accounts, book, &fakeDebtWork{}, theClock())

	paid, err := service.Pay(t.Context(), business.ID(77), owed.ID, 100000, account.ID(1), "2026-03-02", "Payment to Ana")
	if err != nil {
		t.Fatalf("the payment could not be written: %v", err)
	}

	if len(book.recorded) != 1 {
		t.Fatalf("the storage holds %d movements, want 1", len(book.recorded))
	}

	movement := book.recorded[0]
	if movement.DebitAccountID != owed.ID {
		t.Errorf("the money lands in the account %d, want %d", movement.DebitAccountID, owed.ID)
	}
	if movement.CreditAccountID != account.ID(1) {
		t.Errorf("the money comes from the account %d, want %d", movement.CreditAccountID, account.ID(1))
	}
	if movement.Amount != 100000 {
		t.Errorf("the amount is %d, want 100000", movement.Amount)
	}

	if paid.Debt.Amount != 400000 {
		t.Errorf("the debt holds %d, want 400000", paid.Debt.Amount)
	}
}

func TestThePaymentThatPassesTheDebtIsRefused(t *testing.T) {
	t.Parallel()

	owed := theDebtOfTheTests()
	accounts := &fakeDebtAccounts{found: owed}
	book := &fakeBook{balances: []ledger.Balance{{
		AccountID: owed.ID,
		Kind:      account.KindLiability,
		Name:      owed.Name,
		Credits:   500000,
	}}}
	service := application.NewDebtService(accounts, book, &fakeDebtWork{}, theClock())

	_, err := service.Pay(t.Context(), business.ID(77), owed.ID, 600000, account.ID(1), "2026-03-02", "Payment to Ana")
	if !errors.Is(err, ledger.ErrPaymentTooLarge) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrPaymentTooLarge)
	}

	if len(book.recorded) != 0 {
		t.Errorf("the storage holds %d movements, want none", len(book.recorded))
	}
}

func TestThePaymentOfSomethingThatIsNotADebtIsRefused(t *testing.T) {
	t.Parallel()

	cash := account.Account{ID: account.ID(3), BusinessID: business.ID(77), Kind: account.KindAsset, Name: "Efectivo"}
	service := application.NewDebtService(&fakeDebtAccounts{found: cash}, &fakeBook{}, &fakeDebtWork{}, theClock())

	if _, err := service.Pay(t.Context(), business.ID(77), cash.ID, 1000, account.ID(1), "2026-03-02", "Payment"); !errors.Is(err, ports.ErrNotADebt) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNotADebt)
	}
}

func TestTheDebtsWithoutADatabaseRefuseToWork(t *testing.T) {
	t.Parallel()

	service := application.NewDebtService(nil, nil, nil, nil)

	if _, err := service.Record(t.Context(), business.ID(77), "Ana", 500000, account.ID(1), "", "", "", ""); !errors.Is(err, ports.ErrNoDatabase) {
		t.Errorf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
	if _, err := service.Debts(t.Context(), business.ID(77)); !errors.Is(err, ports.ErrNoDatabase) {
		t.Errorf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
	if _, err := service.Pay(t.Context(), business.ID(77), account.ID(9), 1000, account.ID(1), "", ""); !errors.Is(err, ports.ErrNoDatabase) {
		t.Errorf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
