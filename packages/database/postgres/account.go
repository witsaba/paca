package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
)

// The names the database gives to the constraints of the table of the
// accounts.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	accountsNameUniqueIndex = "accounts_name_unique"
)

// AccountRepository is the adapter of the storage of the accounts.
//
// It writes one statement at a time, and the set of accounts of a business
// is written in one transaction: a book that starts with half of its
// accounts is a book that lies about what the person has.
type AccountRepository struct {
	pool *Pool
}

// NewAccountRepository returns the repository of the accounts, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewAccountRepository(pool *Pool) (*AccountRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &AccountRepository{pool: pool}, nil
}

// Create writes one account and answers it with the identity that the
// database generated.
//
// The data is cleaned by the domain before it is written, so the row never
// carries a name with its spaces, whatever the caller sends.
//
// The error is account.ErrAlreadyExists when the name of the account is
// taken inside the business that owns it.
func (r *AccountRepository) Create(ctx context.Context, written account.Account) (account.Account, error) {
	clean, err := account.New(written.BusinessID, written.Kind, written.Name, written.Counterparty, written.DueDate)
	if err != nil {
		return account.Account{}, err
	}

	if r == nil || r.pool == nil {
		return account.Account{}, account.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreateAccount(ctx, db.CreateAccountParams{
		BusinessID:   int64(clean.BusinessID),
		Kind:         clean.Kind.String(),
		Name:         clean.Name,
		Counterparty: optionalText(clean.Counterparty),
		DueDate:      optionalDate(clean.DueDate),
	})
	if err != nil {
		return account.Account{}, writeAccountError(err)
	}

	return accountFromRow(row)
}

// CreateMany writes the accounts of a business in one transaction, and
// answers them with the identities that the database generated.
//
// The set of accounts of a business is written whole or not at all: a book
// that starts with half of its accounts is a book that lies about what the
// person has.
//
// The error is account.ErrAlreadyExists when the name of one of the accounts
// is taken inside the business that owns them, and the accounts of the
// business that were written before the refusal are gone: the transaction
// rolls back.
func (r *AccountRepository) CreateMany(ctx context.Context, accounts []account.Account) ([]account.Account, error) {
	if r == nil || r.pool == nil {
		return nil, account.ErrNoStorage
	}

	if len(accounts) == 0 {
		return []account.Account{}, nil
	}

	clean := make([]account.Account, 0, len(accounts))
	for _, written := range accounts {
		value, err := account.New(written.BusinessID, written.Kind, written.Name, written.Counterparty, written.DueDate)
		if err != nil {
			return nil, err
		}
		clean = append(clean, value)
	}

	// The set joins the transaction of the caller when the caller opened
	// one: the accounts of a business and the business itself are written
	// in the same transaction when the onboarding writes them.
	if _, ok := transactionIn(ctx); ok {
		return r.writeMany(ctx, r.pool.queriesFor(ctx), clean)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("the transaction of the accounts could not be opened: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created, err := r.writeMany(ctx, r.pool.queriesFor(ctx).WithTx(tx), clean)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("the transaction of the accounts could not be committed: %w", err)
	}

	return created, nil
}

// writeMany writes the accounts one by one and answers them with the
// identities that the database generated.
func (r *AccountRepository) writeMany(ctx context.Context, queries *db.Queries, accounts []account.Account) ([]account.Account, error) {
	created := make([]account.Account, 0, len(accounts))
	for _, value := range accounts {
		row, err := queries.CreateAccount(ctx, db.CreateAccountParams{
			BusinessID:   int64(value.BusinessID),
			Kind:         value.Kind.String(),
			Name:         value.Name,
			Counterparty: optionalText(value.Counterparty),
			DueDate:      optionalDate(value.DueDate),
		})
		if err != nil {
			return nil, writeAccountError(err)
		}

		mapped, err := accountFromRow(row)
		if err != nil {
			return nil, err
		}
		created = append(created, mapped)
	}

	return created, nil
}

// Find returns the account of the identity.
//
// The error is account.ErrNotFound when no account carries that identity.
func (r *AccountRepository) Find(ctx context.Context, id account.ID) (account.Account, error) {
	if r == nil || r.pool == nil {
		return account.Account{}, account.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).GetAccountByID(ctx, int64(id))
	if err != nil {
		return account.Account{}, writeAccountError(err)
	}

	return accountFromRow(row)
}

// List returns the accounts of a business, in the order of their identity.
//
// The error is account.ErrNotFound when no business carries that identity.
func (r *AccountRepository) List(ctx context.Context, businessID business.ID) ([]account.Account, error) {
	if r == nil || r.pool == nil {
		return nil, account.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListAccountsByBusiness(ctx, int64(businessID))
	if err != nil {
		return nil, writeAccountError(err)
	}

	accounts := make([]account.Account, 0, len(rows))
	for _, row := range rows {
		mapped, err := accountFromRow(row)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, mapped)
	}

	return accounts, nil
}

// accountFromRow maps a row of the accounts table onto the model of the
// domain.
//
// The nullable columns of the row become the absent values of the model, and
// the kind is read again by the domain: the row may come from a table that
// was changed by another writer, and the model does not trust a stranger
// with a kind it does not know.
func accountFromRow(row db.Account) (account.Account, error) {
	kind, err := account.ParseKind(row.Kind)
	if err != nil {
		return account.Account{}, fmt.Errorf("the row carries an unknown kind: %w", err)
	}

	return account.Account{
		ID:           account.ID(row.ID),
		BusinessID:   business.ID(row.BusinessID),
		Kind:         kind,
		Name:         row.Name,
		Counterparty: text(row.Counterparty),
		DueDate:      date(row.DueDate),
		CreatedAt:    timestamp(row.CreatedAt),
		UpdatedAt:    timestamp(row.UpdatedAt),
	}, nil
}

// optionalDate returns the value of a date that may be absent.
//
// The domain and the table agree on the rule: a value that is not set is
// NULL in the table and absent in the model.
func optionalDate(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{}
	}

	return pgtype.Date{Time: *value, Valid: true}
}

// date returns the value of a nullable date, or nil when the column holds
// NULL.
func date(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}

	moment := value.Time
	return &moment
}

// writeAccountError translates the truth of the database into the errors of
// the port of the accounts.
//
// The name of an account is unique inside the business that owns it, and the
// business that does not exist is the refusal of the foreign key.
func writeAccountError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return account.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		if pgErr.ConstraintName == accountsNameUniqueIndex {
			return fmt.Errorf("%w: that name is taken inside the business", account.ErrAlreadyExists)
		}
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business is not there", account.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", account.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
