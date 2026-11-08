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

// SaleRepository is the adapter of the storage of the sales.
//
// A sale is a fact of the business: the product it sold, the quantity it
// sold, the unit, the money the client paid, the cost the sale released,
// the day it happened and the word of the owner. The units that left the
// book are rows of the table of the exits, and the adapter writes them
// with it: a sale without its units is not a sale.
//
// The row never decides what a sale is: the model is built by the domain,
// through its own constructors.
type SaleRepository struct {
	pool *Pool
}

// NewSaleRepository returns the repository of the sales, bound to the
// database.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewSaleRepository(pool *Pool) (*SaleRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &SaleRepository{pool: pool}, nil
}

// Record writes the sale with the units that left the book, and answers it
// with the identity that the storage generated.
//
// The sale and its units are written in one transaction: a sale whose units
// did not leave the book is not a sale.
//
// The error is recipe.ErrInvalidValue when the row breaks a rule of the
// table, recipe.ErrNotFound when the business or the recipe is not there,
// and recipe.ErrNoStorage when the storage is not open.
func (r *SaleRepository) Record(ctx context.Context, written recipe.Sale) (recipe.Sale, error) {
	if r == nil || r.pool == nil {
		return recipe.Sale{}, recipe.ErrNoStorage
	}

	checked, err := written.Checked()
	if err != nil {
		return recipe.Sale{}, err
	}

	// The set joins the transaction of the caller when the caller opened
	// one, and opens its own when it did not: the sale and the units it
	// took are written whole or not at all.
	if _, ok := transactionIn(ctx); ok {
		return r.write(ctx, r.pool.queriesFor(ctx), checked)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return recipe.Sale{}, fmt.Errorf("%w: the transaction of the sales could not be opened: %w",
			recipe.ErrInvalidValue, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	recorded, err := r.write(ctx, r.pool.queriesFor(ctx).WithTx(tx), checked)
	if err != nil {
		return recipe.Sale{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return recipe.Sale{}, fmt.Errorf("%w: the transaction of the sales could not be committed: %w",
			recipe.ErrInvalidValue, err)
	}

	return recorded, nil
}

// write writes the sale and the units that left the book, and answers the
// sale that was written.
func (r *SaleRepository) write(
	ctx context.Context,
	queries *db.Queries,
	written recipe.Sale,
) (recipe.Sale, error) {
	row, err := queries.CreateSale(ctx, db.CreateSaleParams{
		BusinessID:    int64(written.BusinessID),
		RecipeID:      int64(written.RecipeID),
		Day:           dateOf(written.Day),
		Quantity:      written.Quantity,
		Unit:          written.Unit,
		Money:         int64(written.Money),
		Cost:          int64(written.Cost),
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
	})
	if err != nil {
		return recipe.Sale{}, writeSaleError(err)
	}

	units := make([]recipe.SoldUnit, 0, len(written.Units))

	for _, one := range written.Units {
		place, err := positionOf(one.Position)
		if err != nil {
			return recipe.Sale{}, fmt.Errorf("%w: the position of the unit of the sale: %w",
				recipe.ErrInvalidValue, err)
		}

		if _, err := queries.CreateUnitExit(ctx, db.CreateUnitExitParams{
			UnitID:       int64(one.UnitID),
			ExitKind:     exitKindSale,
			ExitID:       row.ID,
			ProductionID: int64(one.ProductionID),
			Position:     place,
			Grams:        one.Grams,
			ExpiresOn:    dateOf(one.ExpiresOn),
		}); err != nil {
			return recipe.Sale{}, writeSaleError(err)
		}

		units = append(units, recipe.SoldUnit{
			UnitID:       one.UnitID,
			ProductionID: one.ProductionID,
			Position:     one.Position,
			Grams:        one.Grams,
			ExpiresOn:    one.ExpiresOn,
		})
	}

	// The model is built by the domain: the row never decides what a sale
	// is, and a row another writer broke is never read as one.
	recorded, err := recipe.NewSale(business.ID(row.BusinessID), recipe.SaleFact{
		RecipeID:      recipe.ID(row.RecipeID),
		RecipeName:    row.RecipeName,
		Day:           ledger.NewDate(row.Day.Time),
		Quantity:      row.Quantity,
		Unit:          row.Unit,
		Money:         ledger.Amount(row.Money),
		Cost:          ledger.Amount(row.Cost),
		Units:         units,
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
	})
	if err != nil {
		return recipe.Sale{}, err
	}

	recorded.ID = recipe.SaleID(row.ID)
	recorded.CreatedAt = timestamp(row.CreatedAt)

	return recorded, nil
}

// writeSaleError translates the truth of the database into the errors of
// the port of the sales.
//
// The constraints of the tables hold the rules of the domain as a defense in
// depth, and the business, the recipe or the unit that is not there is the
// refusal of the foreign key.
func writeSaleError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return recipe.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		if pgErr.ConstraintName == unitExitsPrimaryKey ||
			pgErr.ConstraintName == unitExitsOnePositionIndex {
			return fmt.Errorf("%w: the constraint %q refused the row: a unit leaves the book only once",
				recipe.ErrInvalidValue, pgErr.ConstraintName)
		}
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business, the recipe or the unit of the sale is not there",
			recipe.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row",
			recipe.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
