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

// The names the database gives to the constraints of the tables of the
// wastes.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	wasteUnitsOnePositionIndex = "waste_units_one_position"
	wasteUnitsOneUnitIndex     = "waste_units_one_unit"
)

// WasteRepository is the adapter of the storage of the wastes.
//
// A waste is a fact of the business: the product it lost, the quantity it
// lost, the unit, the reason, the day it happened and the word of the owner.
// The units that left the book are rows of their own table, and the adapter
// writes them with it: a waste without its units is not a waste.
//
// The row never decides what a waste is: the model is built by the domain,
// through its own constructors.
type WasteRepository struct {
	pool *Pool
}

// NewWasteRepository returns the repository of the wastes, bound to the
// database.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewWasteRepository(pool *Pool) (*WasteRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &WasteRepository{pool: pool}, nil
}

// Record writes the waste with the units that left the book, and answers it
// with the identity that the storage generated.
//
// The waste and its units are written in one transaction: a waste whose
// units did not leave the book is not a waste.
//
// The error is recipe.ErrInvalidValue when the row breaks a rule of the
// table, recipe.ErrNotFound when the business or the recipe is not there,
// and recipe.ErrNoStorage when the storage is not open.
func (r *WasteRepository) Record(ctx context.Context, written recipe.Waste) (recipe.Waste, error) {
	if r == nil || r.pool == nil {
		return recipe.Waste{}, recipe.ErrNoStorage
	}

	checked, err := written.Checked()
	if err != nil {
		return recipe.Waste{}, err
	}

	// The set joins the transaction of the caller when the caller opened
	// one, and opens its own when it did not: the waste and the units it
	// took are written whole or not at all.
	if _, ok := transactionIn(ctx); ok {
		return r.write(ctx, r.pool.queriesFor(ctx), checked)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return recipe.Waste{}, fmt.Errorf("%w: the transaction of the wastes could not be opened: %w",
			recipe.ErrInvalidValue, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	recorded, err := r.write(ctx, r.pool.queriesFor(ctx).WithTx(tx), checked)
	if err != nil {
		return recipe.Waste{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return recipe.Waste{}, fmt.Errorf("%w: the transaction of the wastes could not be committed: %w",
			recipe.ErrInvalidValue, err)
	}

	return recorded, nil
}

// write writes the waste and the units that left the book, and answers the
// waste that was written.
func (r *WasteRepository) write(
	ctx context.Context,
	queries *db.Queries,
	written recipe.Waste,
) (recipe.Waste, error) {
	row, err := queries.CreateWaste(ctx, db.CreateWasteParams{
		BusinessID:    int64(written.BusinessID),
		RecipeID:      int64(written.RecipeID),
		Day:           dateOf(written.Day),
		Quantity:      written.Quantity,
		Unit:          written.Unit,
		Reason:        written.Reason.String(),
		Money:         int64(written.Money),
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
	})
	if err != nil {
		return recipe.Waste{}, writeWasteError(err)
	}

	units := make([]recipe.WastedUnit, 0, len(written.Units))

	for _, one := range written.Units {
		place, err := positionOf(one.Position)
		if err != nil {
			return recipe.Waste{}, fmt.Errorf("%w: the position of the unit of the waste: %w",
				recipe.ErrInvalidValue, err)
		}

		if _, err := queries.CreateWasteUnit(ctx, db.CreateWasteUnitParams{
			WasteID:      row.ID,
			UnitID:       int64(one.UnitID),
			ProductionID: int64(one.ProductionID),
			Position:     place,
			Grams:        one.Grams,
			ExpiresOn:    dateOf(one.ExpiresOn),
		}); err != nil {
			return recipe.Waste{}, writeWasteError(err)
		}

		units = append(units, recipe.WastedUnit{
			UnitID:       one.UnitID,
			ProductionID: one.ProductionID,
			Position:     one.Position,
			Grams:        one.Grams,
			ExpiresOn:    one.ExpiresOn,
		})
	}

	// The model is built by the domain: the row never decides what a waste
	// is, and a row another writer broke is never read as one.
	recorded, err := recipe.NewWaste(business.ID(row.BusinessID), recipe.WasteFact{
		RecipeID:      recipe.ID(row.RecipeID),
		RecipeName:    row.RecipeName,
		Day:           ledger.NewDate(row.Day.Time),
		Quantity:      row.Quantity,
		Unit:          row.Unit,
		Reason:        recipe.WasteReason(row.Reason),
		Money:         ledger.Amount(row.Money),
		Units:         units,
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
	})
	if err != nil {
		return recipe.Waste{}, err
	}

	recorded.ID = recipe.WasteID(row.ID)
	recorded.CreatedAt = timestamp(row.CreatedAt)

	return recorded, nil
}

// writeWasteError translates the truth of the database into the errors of
// the port of the wastes.
//
// The constraints of the tables hold the rules of the domain as a defense in
// depth, and the business, the recipe or the unit that is not there is the
// refusal of the foreign key.
func writeWasteError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return recipe.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		if pgErr.ConstraintName == wasteUnitsOnePositionIndex ||
			pgErr.ConstraintName == wasteUnitsOneUnitIndex {
			return fmt.Errorf("%w: the constraint %q refused the row: a unit leaves the book only once",
				recipe.ErrInvalidValue, pgErr.ConstraintName)
		}
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business, the recipe or the unit of the waste is not there",
			recipe.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row",
			recipe.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
