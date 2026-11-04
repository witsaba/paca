package application

import (
	"context"
	"strings"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/store"
)

// DebtService is the use case of the debts of a business.
//
// A debt is an account and never a table of its own: a loan of a friend is
// an account that owes, with the other side of the money named, and the
// balance of that account is what the person still owes.
//
// The service speaks in the model of the domain and it writes through the
// ports of the storage, and it knows nothing about the protocol that
// reaches it.
type DebtService struct {
	accounts account.Repository
	book     ledger.Repository
	work     store.UnitOfWork
	clock    ports.Clock
}

// NewDebtService builds the use case of the debts.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The transaction
// writes the account of the debt and the movement that opens it whole: a
// debt with no money behind it is a debt the person never asked for.
func NewDebtService(
	accounts account.Repository,
	book ledger.Repository,
	work store.UnitOfWork,
	clock ports.Clock,
) *DebtService {
	return &DebtService{
		accounts: accounts,
		book:     book,
		work:     work,
		clock:    clock,
	}
}

// Record opens a debt and writes the money that opened it.
//
// The counterparty is required: it is the other side of the money, and the
// debt without it is not a debt.
//
// The money comes from the debt and lands in the account the caller names:
// the cash when a friend lends the money, the expense when a product is
// paid later. The day is the day the caller names, and the day of today
// when the caller names none.
//
// The name of the account of the debt is derived from the counterparty when
// the caller names none, and the description of the movement follows it.
func (s *DebtService) Record(
	ctx context.Context,
	businessID business.ID,
	counterparty string,
	amount ledger.Amount,
	toAccountID account.ID,
	date, description, dueDate, name string,
) (ledger.Debt, error) {
	if s == nil || s.accounts == nil || s.book == nil || s.work == nil || s.clock == nil {
		return ledger.Debt{}, ports.ErrNoDatabase
	}

	who := strings.TrimSpace(counterparty)
	if who == "" {
		return ledger.Debt{}, ports.ErrCounterpartyRequired
	}

	day, err := s.dayOf(date)
	if err != nil {
		return ledger.Debt{}, err
	}

	falls, err := dueDateOf(dueDate)
	if err != nil {
		return ledger.Debt{}, err
	}

	label := strings.TrimSpace(name)
	if label == "" {
		label = defaultDebtName(who)
	}

	text := strings.TrimSpace(description)
	if text == "" {
		text = label
	}

	value, err := account.New(businessID, account.KindLiability, label, who, falls)
	if err != nil {
		return ledger.Debt{}, err
	}

	var debt ledger.Debt
	err = s.work.Do(ctx, func(ctx context.Context) error {
		created, err := s.accounts.Create(ctx, value)
		if err != nil {
			return err
		}

		entry, err := ledger.NewEntry(businessID, toAccountID, created.ID, amount, day, text)
		if err != nil {
			return err
		}

		if _, err := s.book.Record(ctx, entry); err != nil {
			return err
		}

		debt = ledger.Debt{Account: created, Amount: amount}
		return nil
	})
	if err != nil {
		return ledger.Debt{}, err
	}

	return debt, nil
}

// Debts returns the debts the book still holds: what the person owes, to
// whom, and the day it falls due.
//
// The debts come first by the day they fall due: the person reads what is
// due before what has no date.
func (s *DebtService) Debts(ctx context.Context, businessID business.ID) ([]ledger.Debt, error) {
	if s == nil || s.accounts == nil || s.book == nil || s.clock == nil {
		return nil, ports.ErrNoDatabase
	}

	accounts, err := s.accounts.List(ctx, businessID)
	if err != nil {
		return nil, err
	}

	balances, err := s.book.Balances(ctx, businessID, ledger.NewDate(s.clock.Now()))
	if err != nil {
		return nil, err
	}

	return ledger.Debts(accounts, balances), nil
}

// Pay writes the payment of a debt.
//
// The money leaves the account the caller names and lands in the debt: the
// payment is what the person owes, and it is what the debt of the book
// holds after it.
//
// The payment may not pass what the debt still holds: a payment bigger than
// the debt is money the person does not owe, and the book of the person
// says the truth.
func (s *DebtService) Pay(
	ctx context.Context,
	businessID business.ID,
	debtAccountID account.ID,
	amount ledger.Amount,
	fromAccountID account.ID,
	date, description string,
) (ports.DebtPayment, error) {
	if s == nil || s.accounts == nil || s.book == nil || s.clock == nil {
		return ports.DebtPayment{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(date)
	if err != nil {
		return ports.DebtPayment{}, err
	}

	owed, err := s.accounts.Find(ctx, debtAccountID)
	if err != nil {
		return ports.DebtPayment{}, err
	}

	if owed.BusinessID != businessID {
		return ports.DebtPayment{}, account.ErrNotFound
	}

	if !owed.IsDebt() {
		return ports.DebtPayment{}, ports.ErrNotADebt
	}

	balances, err := s.book.Balances(ctx, businessID, day)
	if err != nil {
		return ports.DebtPayment{}, err
	}

	debt := ledger.Debt{Account: owed, Amount: balanceOf(balances, debtAccountID).Normal()}
	if err := debt.AcceptsPayment(amount); err != nil {
		return ports.DebtPayment{}, err
	}

	entry, err := ledger.NewEntry(businessID, debtAccountID, fromAccountID, amount, day, description)
	if err != nil {
		return ports.DebtPayment{}, err
	}

	written, err := s.book.Record(ctx, entry)
	if err != nil {
		return ports.DebtPayment{}, err
	}

	remaining := debt
	remaining.Amount -= amount

	return ports.DebtPayment{Movement: written, Debt: remaining}, nil
}

// dayOf returns the day of the book that the caller names, and the day of
// today when the caller names none.
//
// The book of a person is written the day it happens, and a caller that
// does not name the day means the day of today: the agent of Paca speaks
// to a person in the present tense.
func (s *DebtService) dayOf(date string) (ledger.Date, error) {
	if strings.TrimSpace(date) == "" {
		return ledger.NewDate(s.clock.Now()), nil
	}

	return ledger.ParseDate(date)
}

// dueDateOf returns the day a debt falls due, and no day when the caller
// names none.
//
// A debt without a day is a debt that does not fall due: it is owed, and
// nobody waits for a day that is not there.
func dueDateOf(dueDate string) (*time.Time, error) {
	if strings.TrimSpace(dueDate) == "" {
		return nil, nil
	}

	day, err := ledger.ParseDate(dueDate)
	if err != nil {
		return nil, err
	}

	moment := day.Time()
	return &moment, nil
}

// defaultDebtName returns the name of the account of a debt when the
// caller names none.
//
// The name is what the person reads in their book, and it is written in the
// language of the person: the debt with the other side of the money.
func defaultDebtName(counterparty string) string {
	return "Deuda con " + counterparty
}

// balanceOf returns the balance of an account, and a balance of zero when
// the account carries none.
func balanceOf(balances []ledger.Balance, id account.ID) ledger.Balance {
	for _, balance := range balances {
		if balance.AccountID == id {
			return balance
		}
	}

	return ledger.Balance{AccountID: id}
}
