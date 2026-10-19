package sqlite

import (
	"context"
	"database/sql"

	"github.com/witsaba/paca/packages/core/domain"
	"github.com/witsaba/paca/packages/core/ports"
)

// LoanPaymentRepository is the SQLite implementation of the loan payment
// port.
//
// It follows the shape of the account repository: the same naming, the same
// rules about the errors, and the same reading of the optional columns.
type LoanPaymentRepository struct {
	db *DB
}

// NewLoanPaymentRepository returns the loan payment repository over the given
// database.
func NewLoanPaymentRepository(db *DB) *LoanPaymentRepository {
	return &LoanPaymentRepository{db: db}
}

// The repository implements the port of the application.
var _ ports.LoanPaymentRepository = (*LoanPaymentRepository)(nil)

// The columns of a loan payment, in the order the scanner reads them.
const loanPaymentColumns = `id, loan_id, seq, amount, due_on, paid_on, movement_id, notes, created_at`

// Create stores a new obligation and returns it with its identity filled in.
func (r *LoanPaymentRepository) Create(ctx context.Context, payment domain.LoanPayment) (domain.LoanPayment, error) {
	var id int64
	var createdAt string

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO loan_payments
			(loan_id, seq, amount, due_on, paid_on, movement_id, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		payment.LoanID, payment.Seq, payment.Amount, payment.DueOn,
		optionalText(payment.PaidOn), optionalMovement(payment.MovementID),
		optionalText(payment.Notes),
	).Scan(&id, &createdAt)
	if err != nil {
		return domain.LoanPayment{}, err
	}

	payment.ID = id
	payment.CreatedAt = createdAt

	return payment, nil
}

// GetByID returns the obligation with the given identity.
func (r *LoanPaymentRepository) GetByID(ctx context.Context, id int64) (domain.LoanPayment, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+loanPaymentColumns+` FROM loan_payments WHERE id = ?`, id)

	payment, err := scanLoanPayment(row.Scan)
	if err != nil {
		return domain.LoanPayment{}, notFound(err, "the obligation does not exist")
	}

	return payment, nil
}

// ListByLoan returns the obligations of a loan, ordered by their position in
// the schedule.
func (r *LoanPaymentRepository) ListByLoan(ctx context.Context, loanID int64) ([]domain.LoanPayment, error) {
	return r.list(ctx, `SELECT `+loanPaymentColumns+` FROM loan_payments WHERE loan_id = ? ORDER BY seq`, loanID)
}

// ListUnpaid returns the obligations that are still outstanding, ordered by
// the day they fall due.
//
// The order is total: two obligations that fall due on the same day keep the
// order of their position in the schedule.
func (r *LoanPaymentRepository) ListUnpaid(ctx context.Context) ([]domain.LoanPayment, error) {
	return r.list(ctx, `SELECT `+loanPaymentColumns+` FROM loan_payments WHERE paid_on IS NULL ORDER BY due_on, seq`)
}

// MarkPaid settles an obligation: it records the day it was paid and the
// movement that settled it.
func (r *LoanPaymentRepository) MarkPaid(ctx context.Context, id int64, paidOn string, movementID *int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE loan_payments SET paid_on = ?, movement_id = ? WHERE id = ?`,
		optionalText(paidOn), optionalMovement(movementID), id,
	)
	if err != nil {
		return err
	}

	settled, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if settled == 0 {
		return domain.New(domain.CodeNotFound, "the obligation does not exist")
	}

	return nil
}

// list runs a query over the obligations and reads every row it returns.
func (r *LoanPaymentRepository) list(ctx context.Context, query string, args ...any) ([]domain.LoanPayment, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	payments := []domain.LoanPayment{}
	for rows.Next() {
		payment, err := scanLoanPayment(rows.Scan)
		if err != nil {
			return nil, err
		}

		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return payments, nil
}

// scanLoanPayment reads one obligation from a row.
//
// The optional columns of the schema are read as nullable, and a value that
// is not there is the empty value of the domain: the domain has no nulls,
// only empty values. The movement is read as nullable too, because the
// domain keeps the identity of the movement as a pointer.
func scanLoanPayment(scan func(dest ...any) error) (domain.LoanPayment, error) {
	var payment domain.LoanPayment
	var paidOn, notes sql.NullString
	var movementID sql.NullInt64

	err := scan(
		&payment.ID, &payment.LoanID, &payment.Seq, &payment.Amount,
		&payment.DueOn, &paidOn, &movementID, &notes, &payment.CreatedAt,
	)
	if err != nil {
		return domain.LoanPayment{}, err
	}

	payment.PaidOn = paidOn.String
	payment.Notes = notes.String

	if movementID.Valid {
		id := movementID.Int64
		payment.MovementID = &id
	}

	return payment, nil
}

// optionalMovement returns the identity of a movement, or nil when the
// obligation has no movement.
//
// The schema stores the absence of a movement as NULL: an obligation that is
// not settled by a tracked movement is not a movement of zero.
func optionalMovement(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}
