package sqlite

import (
	"context"
	"database/sql"

	"github.com/witsaba/paca/packages/core/domain"
	"github.com/witsaba/paca/packages/core/ports"
)

// AccountRepository is the SQLite implementation of the account port.
//
// It is the reference of the repositories of this package: the others follow
// the same shape, the same naming, and the same rules about the errors.
type AccountRepository struct {
	db *DB
}

// NewAccountRepository returns the account repository over the given
// database.
func NewAccountRepository(db *DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// The repository implements the port of the application.
var _ ports.AccountRepository = (*AccountRepository)(nil)

// The columns of an account, in the order the scanner reads them.
//
// The holder column of the schema is not read: its shape constraint requires
// it to be empty for every kind of account, so it never holds a value.
const accountColumns = `id, kind, name, currency, opened_on, closed_on, is_active, notes,
	phone, bank_code, account_number, account_type, created_at`

// Create stores a new account and returns it with its identity filled in.
func (r *AccountRepository) Create(ctx context.Context, account domain.Account) (domain.Account, error) {
	currency := account.Currency
	if currency == "" {
		currency = defaultCurrency
	}

	var id int64
	var createdAt string

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO accounts
			(kind, name, currency, opened_on, closed_on, is_active, notes,
			 phone, bank_code, account_number, account_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		account.Kind, account.Name, currency, account.OpenedOn,
		optionalText(account.ClosedOn), account.IsActive, optionalText(account.Notes),
		optionalText(account.Phone),
		optionalText(account.BankCode), optionalText(account.AccountNumber),
		optionalText(account.AccountType),
	).Scan(&id, &createdAt)
	if err != nil {
		return domain.Account{}, err
	}

	account.ID = id
	account.Currency = currency
	account.CreatedAt = createdAt

	return account, nil
}

// GetByID returns the account with the given identity.
func (r *AccountRepository) GetByID(ctx context.Context, id int64) (domain.Account, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+accountColumns+` FROM accounts WHERE id = ?`, id)

	account, err := scanAccount(row.Scan)
	if err != nil {
		return domain.Account{}, notFound(err, "the account does not exist")
	}

	return account, nil
}

// List returns every account, ordered by identity.
func (r *AccountRepository) List(ctx context.Context) ([]domain.Account, error) {
	return r.list(ctx, `SELECT `+accountColumns+` FROM accounts ORDER BY id`)
}

// ListActive returns the accounts that are still open, ordered by identity.
func (r *AccountRepository) ListActive(ctx context.Context) ([]domain.Account, error) {
	return r.list(ctx, `SELECT `+accountColumns+` FROM accounts WHERE is_active = 1 ORDER BY id`)
}

// list runs a query over the accounts and reads every row it returns.
func (r *AccountRepository) list(ctx context.Context, query string, args ...any) ([]domain.Account, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	accounts := []domain.Account{}
	for rows.Next() {
		account, err := scanAccount(rows.Scan)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

// scanAccount reads one account from a row.
//
// The optional columns of the schema are read as nullable, and a value that
// is not there is the empty string of the domain: the domain has no nulls,
// only empty values.
func scanAccount(scan func(dest ...any) error) (domain.Account, error) {
	var account domain.Account
	var closedOn, notes, phone, bankCode, accountNumber, accountType sql.NullString

	err := scan(
		&account.ID, &account.Kind, &account.Name, &account.Currency, &account.OpenedOn,
		&closedOn, &account.IsActive, &notes, &phone,
		&bankCode, &accountNumber, &accountType, &account.CreatedAt,
	)
	if err != nil {
		return domain.Account{}, err
	}

	account.ClosedOn = closedOn.String
	account.Notes = notes.String
	account.Phone = phone.String
	account.BankCode = bankCode.String
	account.AccountNumber = accountNumber.String
	account.AccountType = accountType.String

	return account, nil
}
