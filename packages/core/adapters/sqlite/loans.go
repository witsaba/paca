package sqlite

import (
	"context"
	"database/sql"

	"github.com/witsaba/paca/packages/core/domain"
	"github.com/witsaba/paca/packages/core/ports"
)

// LoanRepository is the SQLite implementation of the loan port.
type LoanRepository struct {
	db *DB
}

// NewLoanRepository returns the loan repository over the given database.
func NewLoanRepository(db *DB) *LoanRepository {
	return &LoanRepository{db: db}
}

// The repository implements the port of the application.
var _ ports.LoanRepository = (*LoanRepository)(nil)

// The columns of a loan, in the order the scanner reads them.
const loanColumns = `id, contact_id, direction, kind, principal, interest_rate_bp,
	granted_on, due_on, status, notes, created_at`

// The status a loan has when the caller names none. It is the default of the
// schema: a loan is active until it is settled or cancelled.
const defaultLoanStatus = domain.LoanStatusActive

// Create stores a new loan and returns it with its identity filled in.
func (r *LoanRepository) Create(ctx context.Context, loan domain.Loan) (domain.Loan, error) {
	var id int64
	var createdAt string

	status := loan.Status
	if status == "" {
		status = defaultLoanStatus
	}

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO loans
			(contact_id, direction, kind, principal, interest_rate_bp,
			 granted_on, due_on, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		loan.ContactID, loan.Direction, loan.Kind, loan.Principal,
		optionalRate(loan.InterestRateBP), loan.GrantedOn,
		optionalText(loan.DueOn), status, optionalText(loan.Notes),
	).Scan(&id, &createdAt)
	if err != nil {
		return domain.Loan{}, err
	}

	loan.ID = id
	loan.Status = status
	loan.CreatedAt = createdAt

	return loan, nil
}

// GetByID returns the loan with the given identity.
func (r *LoanRepository) GetByID(ctx context.Context, id int64) (domain.Loan, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+loanColumns+` FROM loans WHERE id = ?`, id)

	loan, err := scanLoan(row.Scan)
	if err != nil {
		return domain.Loan{}, notFound(err, "the loan does not exist")
	}

	return loan, nil
}

// List returns every loan, ordered by identity.
func (r *LoanRepository) List(ctx context.Context) ([]domain.Loan, error) {
	return r.list(ctx, `SELECT `+loanColumns+` FROM loans ORDER BY id`)
}

// ListByStatus returns the loans with the given status, ordered by identity.
func (r *LoanRepository) ListByStatus(ctx context.Context, status domain.LoanStatus) ([]domain.Loan, error) {
	return r.list(ctx, `SELECT `+loanColumns+` FROM loans WHERE status = ? ORDER BY id`, status)
}

// ListByContact returns the loans with the given contact, ordered by
// identity.
func (r *LoanRepository) ListByContact(ctx context.Context, contactID int64) ([]domain.Loan, error) {
	return r.list(ctx, `SELECT `+loanColumns+` FROM loans WHERE contact_id = ? ORDER BY id`, contactID)
}

// Update stores the changes of an existing loan.
func (r *LoanRepository) Update(ctx context.Context, loan domain.Loan) error {
	row := r.db.QueryRowContext(ctx, `
		UPDATE loans
		SET contact_id = ?, direction = ?, kind = ?, principal = ?,
			interest_rate_bp = ?, granted_on = ?, due_on = ?, status = ?, notes = ?
		WHERE id = ?
		RETURNING id`,
		loan.ContactID, loan.Direction, loan.Kind, loan.Principal,
		optionalRate(loan.InterestRateBP), loan.GrantedOn,
		optionalText(loan.DueOn), loan.Status, optionalText(loan.Notes),
		loan.ID,
	)

	var id int64
	if err := row.Scan(&id); err != nil {
		return notFound(err, "the loan does not exist")
	}

	return nil
}

// list runs a query over the loans and reads every row it returns.
func (r *LoanRepository) list(ctx context.Context, query string, args ...any) ([]domain.Loan, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	loans := []domain.Loan{}
	for rows.Next() {
		loan, err := scanLoan(rows.Scan)
		if err != nil {
			return nil, err
		}

		loans = append(loans, loan)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return loans, nil
}

// scanLoan reads one loan from a row.
//
// The optional columns of the schema are read as nullable, and a value that
// is not there is the empty value of the domain: the domain has no nulls,
// only empty values. The interest rate is a pointer, so an absent rate stays
// absent instead of becoming a zero.
func scanLoan(scan func(dest ...any) error) (domain.Loan, error) {
	var loan domain.Loan
	var interestRate sql.NullInt64
	var dueOn, notes sql.NullString

	err := scan(
		&loan.ID, &loan.ContactID, &loan.Direction, &loan.Kind, &loan.Principal,
		&interestRate, &loan.GrantedOn, &dueOn, &loan.Status, &notes, &loan.CreatedAt,
	)
	if err != nil {
		return domain.Loan{}, err
	}

	if interestRate.Valid {
		rate := interestRate.Int64
		loan.InterestRateBP = &rate
	}

	loan.DueOn = dueOn.String
	loan.Notes = notes.String

	return loan, nil
}

// optionalRate returns the value of an interest rate, or nil when no rate is
// recorded.
//
// The schema stores the absence of a rate as NULL, and the domain models it
// as an absent value: a loan without a rate is not a loan with a rate of
// zero.
func optionalRate(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}
