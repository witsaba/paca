package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/commitment"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// The names the database gives to the constraints of the table of the
// commitments.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	// commitmentsBusinessConstraint is the name of the foreign key that
	// refuses the promise of a business that is not there.
	commitmentsBusinessConstraint = "commitments_business_id_fkey"
)

// CommitmentRepository is the adapter of the storage of the commitments.
//
// The promise is one row: what is due, the day it is due and the day it was
// met when it was met. There is no state column, because the day of meeting
// IS the state of the promise.
//
// The row never decides what a commitment is: the model is built by the
// domain, through its own constructors.
type CommitmentRepository struct {
	pool *Pool
}

// NewCommitmentRepository returns the repository of the commitments, bound to
// the pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewCommitmentRepository(pool *Pool) (*CommitmentRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &CommitmentRepository{pool: pool}, nil
}

// Create writes the promise and answers it with the identity that the
// database generated.
//
// The promise is written OPEN: the day of meeting is not a column of the
// insert, and the book never writes one at the creation.
//
// The error is commitment.ErrBusinessNotFound when the business of the
// promise is not there, commitment.ErrInvalidValue when the row breaks a rule
// of the table, and commitment.ErrNoStorage when the storage is not open.
func (r *CommitmentRepository) Create(ctx context.Context, written commitment.Commitment) (commitment.Commitment, error) {
	if r == nil || r.pool == nil {
		return commitment.Commitment{}, commitment.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreateCommitment(ctx, db.CreateCommitmentParams{
		BusinessID: int64(written.BusinessID),
		What:       written.What,
		DueOn:      dateOf(written.DueOn),
	})
	if err != nil {
		return commitment.Commitment{}, writeCommitmentError(err)
	}

	return commitmentOf(row)
}

// Meet writes the day the promise was met and answers the promise that was
// kept.
//
// The meeting moves a promise that is OPEN, and only one: the day of meeting
// is written where no day of meeting is held, so a caller that bypasses the
// domain never meets a promise twice and never hides the day the book holds.
//
// The error is commitment.ErrAlreadyMet when the promise carries a day of
// meeting already, commitment.ErrNotFound when the promise is not there,
// commitment.ErrInvalidValue when the row breaks a rule of the table, and
// commitment.ErrNoStorage when the storage is not open.
func (r *CommitmentRepository) Meet(ctx context.Context, met commitment.Commitment) (commitment.Commitment, error) {
	if r == nil || r.pool == nil {
		return commitment.Commitment{}, commitment.ErrNoStorage
	}

	queries := r.pool.queriesFor(ctx)

	row, err := queries.MeetCommitment(ctx, db.MeetCommitmentParams{
		MetOn:      dateOf(met.MetOn),
		ID:         int64(met.ID),
		BusinessID: int64(met.BusinessID),
	})
	if err != nil {
		return commitment.Commitment{}, theMeetingError(ctx, queries, met, err)
	}

	return commitmentOf(row)
}

// List returns the promises of the business, in the order of their identity,
// with the day each one was met when it was met.
//
// The list holds the promises of the business and of no other one, and it is
// empty when the business promised nothing yet. A promise is never deleted:
// the list holds the ones that were kept too.
//
// The error is commitment.ErrNoStorage when the storage is not open.
func (r *CommitmentRepository) List(ctx context.Context, own business.ID) ([]commitment.Commitment, error) {
	if r == nil || r.pool == nil {
		return nil, commitment.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListCommitmentsByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeCommitmentError(err)
	}

	promises := make([]commitment.Commitment, 0, len(rows))

	for _, row := range rows {
		built, err := commitmentOf(row)
		if err != nil {
			return nil, err
		}

		promises = append(promises, built)
	}

	return promises, nil
}

// commitmentOf builds the promise of a row and asks the domain whether it is
// whole.
//
// The row never decides what a commitment is: a row that another writer broke
// is refused by the domain, and the adapter says it with
// commitment.ErrInvalidValue.
func commitmentOf(row db.Commitment) (commitment.Commitment, error) {
	built, err := commitment.NewFromFacts(commitment.Facts{
		ID:         commitment.ID(row.ID),
		BusinessID: business.ID(row.BusinessID),
		What:       row.What,
		DueOn:      ledger.NewDate(row.DueOn.Time),
		MetOn:      ledger.NewDate(row.MetOn.Time),
	})
	if err != nil {
		return commitment.Commitment{}, fmt.Errorf("%w: the row %d of the commitments breaks the domain: %w",
			commitment.ErrInvalidValue, row.ID, err)
	}

	built.CreatedAt = timestamp(row.CreatedAt)
	built.UpdatedAt = timestamp(row.UpdatedAt)

	return built, nil
}

// theMeetingError translates the refusal of the table into the errors of the
// port of the commitments.
//
// The meeting of a promise is written once: the promise that is not open any
// more is refused with the error of the domain, and the adapter never hides
// the day the book holds. The row is read back only when the meeting found no
// promise to meet, so the adapter can tell the promise that is not there from
// the promise that was met already.
func theMeetingError(ctx context.Context, queries *db.Queries, met commitment.Commitment, err error) error {
	if !errors.Is(err, pgx.ErrNoRows) {
		return writeCommitmentError(err)
	}

	held, err := queries.TheMeetingOfTheCommitment(ctx, db.TheMeetingOfTheCommitmentParams{
		ID:         int64(met.ID),
		BusinessID: int64(met.BusinessID),
	})
	if err != nil {
		return writeCommitmentError(err)
	}

	if held.Valid {
		return fmt.Errorf("%w: the commitment %d was met on %s", commitment.ErrAlreadyMet, met.ID, held.Time.Format("2006-01-02"))
	}

	return fmt.Errorf("%w: the commitment %d was not met by this call", commitment.ErrInvalidValue, met.ID)
}

// writeCommitmentError translates the truth of the database into the errors
// of the port of the commitments.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth: the business of the promise is a business that is there, and the
// promise says what is due.
func writeCommitmentError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return commitment.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case foreignKeyViolation:
		if pgErr.ConstraintName == commitmentsBusinessConstraint {
			return fmt.Errorf("%w: the constraint %q refused the row",
				commitment.ErrBusinessNotFound, pgErr.ConstraintName)
		}

		return fmt.Errorf("%w: the business of the commitment is not there", commitment.ErrBusinessNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", commitment.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
