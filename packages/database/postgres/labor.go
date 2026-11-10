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
// labor.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	laborsHoursNotNegativeConstraint   = "labors_hours_not_negative_check"
	laborsHoursPresentConstraint       = "labors_hours_present_check"
	laborsHourValuePositiveConstraint  = "labors_hour_value_positive_check"
	laborsConfirmedWithEmptyConstraint = "labors_confirmed_with_not_empty_check"
)

// LaborRepository is the adapter of the storage of the labor.
//
// The labor carries the hours the owner worked, in the three kinds of the
// labor, and the value of the hour she declared. The money of the labor is
// not stored: it is derived from the hours and the value of the hour, the
// way the cost of a batch is derived from the layers of the stock.
//
// The row never decides what a labor is: the model is built by the domain,
// through its own constructors.
type LaborRepository struct {
	pool *Pool
}

// NewLaborRepository returns the repository of the labor, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewLaborRepository(pool *Pool) (*LaborRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &LaborRepository{pool: pool}, nil
}

// Record writes the labor and answers it with the identity that the
// database generated.
//
// The error is recipe.ErrInvalidValue when the row breaks a rule of the
// table, recipe.ErrNotFound when the business is not there, and
// recipe.ErrNoStorage when the storage is not open.
func (r *LaborRepository) Record(ctx context.Context, written recipe.Labor) (recipe.Labor, error) {
	if r == nil || r.pool == nil {
		return recipe.Labor{}, recipe.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreateLabor(ctx, db.CreateLaborParams{
		BusinessID:    int64(written.BusinessID),
		Day:           dateOf(written.Day),
		HandsOnHours:  written.Hours.HandsOn,
		OvenHours:     written.Hours.Oven,
		WaitingHours:  written.Hours.Waiting,
		HourValue:     int64(written.HourValue),
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
	})
	if err != nil {
		return recipe.Labor{}, writeLaborError(err)
	}

	return laborFromRow(row)
}

// List returns the labor of the business, in the order of the day it was
// worked and of its identity.
//
// The list holds the labor of the business and of no other one, and it is
// empty when the business worked no hour yet.
//
// The error is recipe.ErrNoStorage when the storage is not open.
func (r *LaborRepository) List(ctx context.Context, own business.ID) ([]recipe.Labor, error) {
	if r == nil || r.pool == nil {
		return nil, recipe.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListLaborsByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeLaborError(err)
	}

	held := make([]recipe.Labor, 0, len(rows))

	for _, row := range rows {
		found, err := laborFromRow(row)
		if err != nil {
			return nil, err
		}

		held = append(held, found)
	}

	return held, nil
}

// laborFromRow maps a row of the labor onto the model of the domain.
//
// The labor is built by the domain: the row may come from a table that was
// changed by another writer, and the model does not trust a stranger with a
// labor it does not know.
func laborFromRow(row db.Labor) (recipe.Labor, error) {
	hours, err := recipe.NewHours(row.HandsOnHours, row.OvenHours, row.WaitingHours)
	if err != nil {
		return recipe.Labor{}, err
	}

	labour, err := recipe.NewLabor(business.ID(row.BusinessID), recipe.LaborDefinition{
		Hours:         hours,
		HourValue:     ledger.Amount(row.HourValue),
		Day:           ledger.NewDate(row.Day.Time),
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
	})
	if err != nil {
		return recipe.Labor{}, err
	}

	labour.ID = recipe.LaborID(row.ID)
	labour.CreatedAt = timestamp(row.CreatedAt)

	return labour, nil
}

// writeLaborError translates the truth of the database into the errors of
// the port of the labor.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth, and the business that is not there is the refusal of the foreign
// key.
func writeLaborError(err error) error {
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
		return fmt.Errorf("%w: the business of the labor is not there", recipe.ErrNotFound)
	case checkViolation:
		switch pgErr.ConstraintName {
		case laborsHoursNotNegativeConstraint,
			laborsHoursPresentConstraint,
			laborsHourValuePositiveConstraint,
			laborsConfirmedWithEmptyConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row", recipe.ErrInvalidValue, pgErr.ConstraintName)
		}
	}

	return err
}
