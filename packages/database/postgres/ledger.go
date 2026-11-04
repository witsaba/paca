package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// The names the database gives to the constraints of the table of the
// movements.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	ledgerAmountConstraint      = "ledger_entries_amount_positive_check"
	ledgerDistinctConstraint    = "ledger_entries_distinct_accounts_check"
	ledgerDescriptionConstraint = "ledger_entries_description_not_empty_check"
)

// LedgerRepository is the adapter of the storage of the book.
//
// It writes one statement at a time. It does not read a row to find out
// whether it may write it: the constraints of the table tell the truth by
// themselves, and the adapter maps that truth to the errors of the port.
//
// The balance of the accounts is a sum, and the database does it: the money
// of a person is read from the rows of the book.
type LedgerRepository struct {
	pool *Pool
}

// NewLedgerRepository returns the repository of the book, bound to the pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil.
//
// The error is ErrNoPool when the pool is not there.
func NewLedgerRepository(pool *Pool) (*LedgerRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &LedgerRepository{pool: pool}, nil
}

// Record writes a movement of the book and answers it with the identity
// that the database generated.
//
// The movement is cleaned by the domain before it is written, so the row
// never carries a description with its spaces, whatever the caller sends.
//
// The error is ledger.ErrInvalidAmount, ledger.ErrSameAccount or
// ledger.ErrInvalidDescription when the movement breaks a rule of the
// domain, and ledger.ErrNotFound when the business or one of the accounts
// is not there.
func (r *LedgerRepository) Record(ctx context.Context, written ledger.Entry) (ledger.Entry, error) {
	clean, err := ledger.NewEntry(
		written.BusinessID,
		written.DebitAccountID,
		written.CreditAccountID,
		written.Amount,
		written.OccurredOn,
		written.Description,
	)
	if err != nil {
		return ledger.Entry{}, err
	}

	if r == nil || r.pool == nil {
		return ledger.Entry{}, ledger.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).RecordEntry(ctx, db.RecordEntryParams{
		BusinessID:      int64(clean.BusinessID),
		DebitAccountID:  int64(clean.DebitAccountID),
		CreditAccountID: int64(clean.CreditAccountID),
		Amount:          int64(clean.Amount),
		OccurredOn:      dateOf(clean.OccurredOn),
		Description:     clean.Description,
	})
	if err != nil {
		return ledger.Entry{}, writeLedgerError(err)
	}

	return ledgerEntryFromRow(row)
}

// List returns the movements of a business between two days, in the order of
// the day and of the identity.
//
// The error is ledger.ErrNotFound when no business carries that identity.
func (r *LedgerRepository) List(ctx context.Context, businessID business.ID, from, to ledger.Date) ([]ledger.Entry, error) {
	if r == nil || r.pool == nil {
		return nil, ledger.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListEntriesByBusiness(ctx, db.ListEntriesByBusinessParams{
		BusinessID: int64(businessID),
		FromDate:   dateOf(from),
		ToDate:     dateOf(to),
	})
	if err != nil {
		return nil, writeLedgerError(err)
	}

	entries := make([]ledger.Entry, 0, len(rows))
	for _, row := range rows {
		mapped, err := ledgerEntryFromRow(row)
		if err != nil {
			return nil, err
		}
		entries = append(entries, mapped)
	}

	return entries, nil
}

// Balances returns the balance of every account of a business, as of a day.
//
// The account that never moved is in the answer with a balance of zero: the
// person reads the accounts of their book, and an account that is missing
// reads like an account that does not exist.
//
// The error is ledger.ErrNotFound when no business carries that identity.
func (r *LedgerRepository) Balances(ctx context.Context, businessID business.ID, asOf ledger.Date) ([]ledger.Balance, error) {
	if r == nil || r.pool == nil {
		return nil, ledger.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListAccountBalances(ctx, db.ListAccountBalancesParams{
		AsOf:       dateOf(asOf),
		BusinessID: int64(businessID),
	})
	if err != nil {
		return nil, writeLedgerError(err)
	}

	balances := make([]ledger.Balance, 0, len(rows))
	for _, row := range rows {
		kind, err := account.ParseKind(row.Kind)
		if err != nil {
			return nil, fmt.Errorf("the row carries an unknown kind: %w", err)
		}

		balances = append(balances, ledger.Balance{
			AccountID: account.ID(row.AccountID),
			Kind:      kind,
			Name:      row.Name,
			Debits:    ledger.Amount(row.Debits),
			Credits:   ledger.Amount(row.Credits),
		})
	}

	return balances, nil
}

// ledgerEntryFromRow maps a row of the movements table onto the model of the
// domain.
func ledgerEntryFromRow(row db.LedgerEntry) (ledger.Entry, error) {
	return ledger.Entry{
		ID:              ledger.EntryID(row.ID),
		BusinessID:      business.ID(row.BusinessID),
		DebitAccountID:  account.ID(row.DebitAccountID),
		CreditAccountID: account.ID(row.CreditAccountID),
		Amount:          ledger.Amount(row.Amount),
		OccurredOn:      ledger.NewDate(row.OccurredOn.Time),
		Description:     row.Description,
		CreatedAt:       timestamp(row.CreatedAt),
	}, nil
}

// dateOf returns the day of the book as the column of the table holds it.
func dateOf(day ledger.Date) pgtype.Date {
	return pgtype.Date{Time: day.Time(), Valid: true}
}

// writeLedgerError translates the truth of the database into the errors of
// the port of the book.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth, and the business or the account that is not there is the refusal
// of the foreign key.
func writeLedgerError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case checkViolation:
		switch pgErr.ConstraintName {
		case ledgerAmountConstraint:
			return fmt.Errorf("%w: the amount of the movement is positive", ledger.ErrInvalidAmount)
		case ledgerDistinctConstraint:
			return fmt.Errorf("%w: the two sides of the movement are not the same account", ledger.ErrSameAccount)
		case ledgerDescriptionConstraint:
			return fmt.Errorf("%w: the description of the movement is not empty", ledger.ErrInvalidDescription)
		}
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business or one of the accounts is not there", ledger.ErrNotFound)
	}

	return err
}
