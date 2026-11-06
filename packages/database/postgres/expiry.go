package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// ShelfLifeRepository is the adapter of the storage of the corrections of
// the shelf life.
//
// The correction is a fact of the business: the recipe it corrects, the
// number of days what it makes keeps, the day it was declared and the word
// of the owner. Every correction is a row of its own, and the list is read
// in the order of the day it was declared.
//
// The row never decides what a correction is: the model is built by the
// domain, through its own constructor.
type ShelfLifeRepository struct {
	pool *Pool
}

// NewShelfLifeRepository returns the repository of the corrections, bound
// to the database.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewShelfLifeRepository(pool *Pool) (*ShelfLifeRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &ShelfLifeRepository{pool: pool}, nil
}

// Record writes the correction of the shelf life and answers it with the
// identity that the storage generated.
//
// The correction joins the transaction of the caller when the caller opened
// one, and opens its own when it did not: the correction and the days of the
// units it moves are written whole or not at all.
//
// The error is recipe.ErrInvalidValue when the row breaks a rule of the
// table, recipe.ErrNotFound when the business or the recipe is not there,
// and recipe.ErrNoStorage when the storage is not open.
func (r *ShelfLifeRepository) Record(ctx context.Context, written recipe.ShelfLifeCorrection) (recipe.ShelfLifeCorrection, error) {
	if r == nil || r.pool == nil {
		return recipe.ShelfLifeCorrection{}, recipe.ErrNoStorage
	}

	checked, err := written.Checked()
	if err != nil {
		return recipe.ShelfLifeCorrection{}, err
	}

	if _, ok := transactionIn(ctx); ok {
		return r.write(ctx, r.pool.queriesFor(ctx), checked)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return recipe.ShelfLifeCorrection{}, fmt.Errorf("%w: the transaction of the corrections could not be opened: %w",
			recipe.ErrInvalidValue, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	recorded, err := r.write(ctx, r.pool.queriesFor(ctx).WithTx(tx), checked)
	if err != nil {
		return recipe.ShelfLifeCorrection{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return recipe.ShelfLifeCorrection{}, fmt.Errorf("%w: the transaction of the corrections could not be committed: %w",
			recipe.ErrInvalidValue, err)
	}

	return recorded, nil
}

// write writes the correction and answers the correction that was written.
func (r *ShelfLifeRepository) write(
	ctx context.Context,
	queries *db.Queries,
	written recipe.ShelfLifeCorrection,
) (recipe.ShelfLifeCorrection, error) {
	days, err := daysOfShelfLife(written.ShelfLifeDays)
	if err != nil {
		return recipe.ShelfLifeCorrection{}, err
	}

	row, err := queries.CreateShelfLifeCorrection(ctx, db.CreateShelfLifeCorrectionParams{
		BusinessID:    int64(written.BusinessID),
		RecipeID:      int64(written.RecipeID),
		ShelfLifeDays: days,
		Day:           dateOf(written.Day),
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
	})
	if err != nil {
		return recipe.ShelfLifeCorrection{}, writeExpiryError(err)
	}

	return correctionOf(rowOfACorrection{
		ID:            row.ID,
		BusinessID:    row.BusinessID,
		RecipeID:      row.RecipeID,
		RecipeName:    row.RecipeName,
		ShelfLifeDays: int(row.ShelfLifeDays),
		Day:           row.Day,
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
		CreatedAt:     row.CreatedAt,
	})
}

// List returns the corrections of the business, in the order of the day they
// were declared.
//
// The error is recipe.ErrNoStorage when the storage is not open.
func (r *ShelfLifeRepository) List(ctx context.Context, own business.ID) ([]recipe.ShelfLifeCorrection, error) {
	if r == nil || r.pool == nil {
		return nil, recipe.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListShelfLifeCorrectionsByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeExpiryError(err)
	}

	corrections := make([]recipe.ShelfLifeCorrection, 0, len(rows))

	for _, row := range rows {
		correction, err := correctionOf(rowOfACorrection{
			ID:            row.ID,
			BusinessID:    row.BusinessID,
			RecipeID:      row.RecipeID,
			RecipeName:    row.RecipeName,
			ShelfLifeDays: int(row.ShelfLifeDays),
			Day:           row.Day,
			Utterance:     row.Utterance,
			ConfirmedWith: row.ConfirmedWith,
			CreatedAt:     row.CreatedAt,
		})
		if err != nil {
			return nil, err
		}

		corrections = append(corrections, correction)
	}

	return corrections, nil
}

// rowOfACorrection is the row of a correction, as the storage holds it.
type rowOfACorrection struct {
	ID            int64
	BusinessID    int64
	RecipeID      int64
	RecipeName    string
	ShelfLifeDays int
	Day           pgtype.Date
	Utterance     string
	ConfirmedWith string
	CreatedAt     pgtype.Timestamptz
}

// correctionOf builds the correction of the domain out of a row, and asks
// the domain whether it is whole.
//
// The correction is built by the domain: the row may come from a table that
// was changed by another writer, and the model does not trust a stranger
// with a correction it does not know.
func correctionOf(row rowOfACorrection) (recipe.ShelfLifeCorrection, error) {
	correction := recipe.ShelfLifeCorrection{
		ID:            recipe.CorrectionID(row.ID),
		BusinessID:    business.ID(row.BusinessID),
		RecipeID:      recipe.ID(row.RecipeID),
		RecipeName:    row.RecipeName,
		ShelfLifeDays: row.ShelfLifeDays,
		Day:           ledger.NewDate(row.Day.Time),
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
		CreatedAt:     timestamp(row.CreatedAt),
	}

	return correction.Checked()
}

// UnitRepository is the adapter of the storage of the units of the
// productions.
//
// The unit is a row of the production that made it: the adapter reads it
// with the production and the recipe it belongs to, because the expiry is
// read for one business.
//
// The row never decides what a unit is: the model is built by the domain,
// through its own constructor.
type UnitRepository struct {
	pool *Pool
}

// NewUnitRepository returns the repository of the units, bound to the
// database.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewUnitRepository(pool *Pool) (*UnitRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &UnitRepository{pool: pool}, nil
}

// List returns the units of the productions of the business, in the order
// they expire.
//
// The query reads the production and the recipe of every unit with it: the
// expiry is read for one business, and a unit of another business is never
// read as a unit of it.
//
// The error is recipe.ErrNoStorage when the storage is not open.
func (r *UnitRepository) List(ctx context.Context, own business.ID) ([]recipe.ExpiringUnit, error) {
	if r == nil || r.pool == nil {
		return nil, recipe.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListProductionUnitsByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeExpiryError(err)
	}

	units := make([]recipe.ExpiringUnit, 0, len(rows))

	for _, row := range rows {
		unit, err := unitOf(rowOfAUnit{
			ID:           row.ID,
			ProductionID: row.ProductionID,
			Position:     int(row.Position),
			Grams:        row.Grams,
			ExpiresOn:    row.ExpiresOn,
			RecipeID:     row.RecipeID,
			ProducedOn:   row.ProducedOn,
			RecipeName:   row.RecipeName,
		})
		if err != nil {
			return nil, err
		}

		units = append(units, unit)
	}

	return units, nil
}

// WriteExpiry writes the day the units expire.
//
// The day is written only when the unit still expires on the day the
// correction read: a unit that changed under the correction is refused, and
// the correction leaves the book as it was.
//
// The error is recipe.ErrInvalidValue when the row breaks a rule of the
// table or when the unit changed under the correction, recipe.ErrNotFound
// when the unit is not there, and recipe.ErrNoStorage when the storage is
// not open.
func (r *UnitRepository) WriteExpiry(ctx context.Context, units []recipe.CorrectedUnit) error {
	if r == nil || r.pool == nil {
		return recipe.ErrNoStorage
	}

	for _, one := range units {
		written, err := r.pool.queriesFor(ctx).UpdateProductionUnitExpiry(ctx, db.UpdateProductionUnitExpiryParams{
			ExpiresOn:         dateOf(one.Now),
			ID:                int64(one.UnitID),
			PreviousExpiresOn: dateOf(one.Was),
		})
		if err != nil {
			return writeExpiryError(err)
		}

		if written != 1 {
			return fmt.Errorf("%w: the unit %d no longer expires on %q",
				recipe.ErrInvalidValue, one.UnitID, one.Was.String())
		}
	}

	return nil
}

// rowOfAUnit is the row of a unit, as the storage holds it.
type rowOfAUnit struct {
	ID           int64
	ProductionID int64
	Position     int
	Grams        float64
	ExpiresOn    pgtype.Date
	RecipeID     int64
	ProducedOn   pgtype.Date
	RecipeName   string
}

// unitOf builds the unit of the domain out of a row, and asks the domain
// whether it is whole.
//
// The unit is built by the domain: the row may come from a table that was
// changed by another writer, and the model does not trust a stranger with a
// unit it does not know.
func unitOf(row rowOfAUnit) (recipe.ExpiringUnit, error) {
	unit := recipe.ExpiringUnit{
		ProductionUnit: recipe.ProductionUnit{
			ID:           recipe.UnitID(row.ID),
			ProductionID: recipe.ProductionID(row.ProductionID),
			Position:     row.Position,
			Grams:        row.Grams,
			ExpiresOn:    ledger.NewDate(row.ExpiresOn.Time),
		},
		RecipeID:   recipe.ID(row.RecipeID),
		RecipeName: row.RecipeName,
		ProducedOn: ledger.NewDate(row.ProducedOn.Time),
	}

	return unit.Checked()
}

// daysOfShelfLife returns the number of days of a shelf life as the column
// of the table holds it.
//
// The column is an integer of thirty-two bits, and the number of days is the
// number of days a thing keeps: a number that does not fit the column is a
// number the table cannot hold.
func daysOfShelfLife(days int) (int32, error) {
	if days <= 0 || days > math.MaxInt32 {
		return 0, fmt.Errorf("%w: the number of days of the shelf life is %d",
			recipe.ErrInvalidValue, days)
	}

	return int32(days), nil
}

// writeExpiryError translates the truth of the database into the errors of
// the port of the expiry.
//
// The constraints of the tables hold the rules of the domain as a defense in
// depth, and the business or the recipe that is not there is the refusal of
// the foreign key.
func writeExpiryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return recipe.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business or the recipe of the correction is not there",
			recipe.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row",
			recipe.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
