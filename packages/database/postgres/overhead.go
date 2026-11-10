package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names the database gives to the constraints of the table of the
// overhead.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	overheadsElectricityPositiveConstraint = "overheads_electricity_positive_check"
	overheadsOvenHoursPositiveConstraint   = "overheads_oven_hours_positive_check"
	overheadsAllocationShareConstraint     = "overheads_allocation_share_check"
	overheadsConfirmedWithEmptyConstraint  = "overheads_confirmed_with_not_empty_check"
)

// OverheadRepository is the adapter of the storage of the overhead.
//
// The overhead carries the electricity the owner paid, the hours the oven
// ran and the allocation she declared. The money of the overhead is not
// stored: it is derived from the electricity and the allocation, the way
// the money of the labor is derived from the hours and the value of the
// hour.
//
// The row never decides what an overhead is: the model is built by the
// domain, through its own constructors.
type OverheadRepository struct {
	pool *Pool
}

// NewOverheadRepository returns the repository of the overhead, bound to
// the pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewOverheadRepository(pool *Pool) (*OverheadRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &OverheadRepository{pool: pool}, nil
}

// Record writes the overhead and answers it with the identity that the
// database generated.
//
// The error is recipe.ErrInvalidValue when the row breaks a rule of the
// table, recipe.ErrNotFound when the business is not there, and
// recipe.ErrNoStorage when the storage is not open.
func (r *OverheadRepository) Record(ctx context.Context, written recipe.Overhead) (recipe.Overhead, error) {
	if r == nil || r.pool == nil {
		return recipe.Overhead{}, recipe.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreateOverhead(ctx, db.CreateOverheadParams{
		BusinessID:    int64(written.BusinessID),
		Day:           dateOf(written.Day),
		Electricity:   int64(written.Electricity),
		OvenHours:     written.OvenHours,
		Allocation:    written.Allocation,
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
	})
	if err != nil {
		return recipe.Overhead{}, writeOverheadError(err)
	}

	return overheadFromRow(row)
}

// List returns the overhead of the business, in the order of the day it was
// declared and of its identity.
//
// The list holds the overhead of the business and of no other one, and it
// is empty when the business declared none yet.
//
// The error is recipe.ErrNoStorage when the storage is not open.
func (r *OverheadRepository) List(ctx context.Context, own business.ID) ([]recipe.Overhead, error) {
	if r == nil || r.pool == nil {
		return nil, recipe.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListOverheadsByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeOverheadError(err)
	}

	held := make([]recipe.Overhead, 0, len(rows))

	for _, row := range rows {
		found, err := overheadFromRow(row)
		if err != nil {
			return nil, err
		}

		held = append(held, found)
	}

	return held, nil
}

// overheadFromRow maps a row of the overhead onto the model of the domain.
//
// The overhead is built by the domain: the row may come from a table that
// was changed by another writer, and the model does not trust a stranger
// with an overhead it does not know.
func overheadFromRow(row db.Overhead) (recipe.Overhead, error) {
	overhead, err := recipe.NewOverhead(business.ID(row.BusinessID), recipe.OverheadDefinition{
		Electricity:   ledger.Amount(row.Electricity),
		OvenHours:     row.OvenHours,
		Allocation:    row.Allocation,
		Day:           ledger.NewDate(row.Day.Time),
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
	})
	if err != nil {
		return recipe.Overhead{}, err
	}

	overhead.ID = recipe.OverheadID(row.ID)
	overhead.CreatedAt = timestamp(row.CreatedAt)

	return overhead, nil
}

// writeOverheadError translates the truth of the database into the errors
// of the port of the overhead.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth, and the business that is not there is the refusal of the foreign
// key.
func writeOverheadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return recipe.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", recipe.ErrInvalidValue, pgErr.ConstraintName)
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business of the overhead is not there", recipe.ErrNotFound)
	case checkViolation:
		switch pgErr.ConstraintName {
		case overheadsElectricityPositiveConstraint,
			overheadsOvenHoursPositiveConstraint,
			overheadsAllocationShareConstraint,
			overheadsConfirmedWithEmptyConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row", recipe.ErrInvalidValue, pgErr.ConstraintName)
		}
	}

	return err
}
