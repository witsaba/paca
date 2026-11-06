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
// productions.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	productionLayersOnePositionIndex = "production_layers_one_position"
	productionUnitsOnePositionIndex  = "production_units_one_position"
)

// ProductionRepository is the adapter of the storage of the productions.
//
// A production is a fact of the business: the recipe it was made of, the
// day it was made, the grams that went in and the units that came out. The
// layers it consumed and the units it made are rows of their own tables, and
// the adapter writes them with it: a production without its layers and its
// units is not a production.
//
// The row never decides what a production is: the model is built by the
// domain, through its own constructors.
type ProductionRepository struct {
	pool *Pool
}

// NewProductionRepository returns the repository of the productions, bound
// to the database.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewProductionRepository(pool *Pool) (*ProductionRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &ProductionRepository{pool: pool}, nil
}

// Record writes the production with the layers it consumed and the units it
// made, and answers it with the identity that the storage generated.
//
// The production and its rows are written in one transaction: a production
// whose ingredient is not in the catalog leaves nothing behind, and a
// production of a recipe that is not there is not written either.
//
// The error is recipe.ErrInvalidValue when the row breaks a rule of the
// table, recipe.ErrNotFound when the business or the recipe is not there,
// and recipe.ErrNoStorage when the storage is not open.
func (r *ProductionRepository) Record(ctx context.Context, written recipe.Production) (recipe.Production, error) {
	if r == nil || r.pool == nil {
		return recipe.Production{}, recipe.ErrNoStorage
	}

	// The set joins the transaction of the caller when the caller opened
	// one, and opens its own when it did not: the production, the layers it
	// consumed and the units it made are written whole or not at all.
	if _, ok := transactionIn(ctx); ok {
		return r.write(ctx, r.pool.queriesFor(ctx), written)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return recipe.Production{}, fmt.Errorf("%w: the transaction of the productions could not be opened: %w",
			recipe.ErrInvalidValue, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	recorded, err := r.write(ctx, r.pool.queriesFor(ctx).WithTx(tx), written)
	if err != nil {
		return recipe.Production{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return recipe.Production{}, fmt.Errorf("%w: the transaction of the productions could not be committed: %w",
			recipe.ErrInvalidValue, err)
	}

	return recorded, nil
}

// write writes the production, the layers it consumed and the units it made,
// and answers the production that was written.
func (r *ProductionRepository) write(
	ctx context.Context,
	queries *db.Queries,
	written recipe.Production,
) (recipe.Production, error) {
	row, err := queries.CreateProduction(ctx, db.CreateProductionParams{
		BusinessID:    int64(written.BusinessID),
		RecipeID:      int64(written.RecipeID),
		Day:           dateOf(written.Day),
		GramsIn:       written.GramsIn,
		Money:         int64(written.Money),
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
	})
	if err != nil {
		return recipe.Production{}, writeProductionError(err)
	}

	for position, one := range written.Takes {
		place, err := positionOf(position + 1)
		if err != nil {
			return recipe.Production{}, fmt.Errorf("%w: the position of the layer of the production: %w",
				recipe.ErrInvalidValue, err)
		}

		if _, err := queries.CreateProductionLayer(ctx, db.CreateProductionLayerParams{
			ProductionID: row.ID,
			Position:     place,
			LayerID:      int64(one.LayerID),
			PurchaseID:   int64(one.PurchaseID),
			Quantity:     one.Quantity,
			Unit:         one.Unit,
			Grams:        one.Grams,
			Money:        int64(one.Money),
		}); err != nil {
			return recipe.Production{}, writeProductionError(err)
		}
	}

	units := make([]recipe.ProductionUnit, 0, len(written.Units))

	for _, one := range written.Units {
		place, err := positionOf(one.Position)
		if err != nil {
			return recipe.Production{}, fmt.Errorf("%w: the position of the unit of the production: %w",
				recipe.ErrInvalidValue, err)
		}

		unitRow, err := queries.CreateProductionUnit(ctx, db.CreateProductionUnitParams{
			ProductionID: row.ID,
			Position:     place,
			Grams:        one.Grams,
			ExpiresOn:    dateOf(one.ExpiresOn),
		})
		if err != nil {
			return recipe.Production{}, writeProductionError(err)
		}

		units = append(units, recipe.ProductionUnit{
			ID:           recipe.UnitID(unitRow.ID),
			ProductionID: recipe.ProductionID(unitRow.ProductionID),
			Position:     int(unitRow.Position),
			Grams:        unitRow.Grams,
			ExpiresOn:    ledger.NewDate(unitRow.ExpiresOn.Time),
		})
	}

	// The model is built by the domain: the row never decides what a
	// production is, and a row another writer broke is never read as one.
	recorded, err := recipe.NewProduction(business.ID(row.BusinessID), recipe.ProductionFact{
		RecipeID:      recipe.ID(row.RecipeID),
		RecipeName:    row.RecipeName,
		Day:           ledger.NewDate(row.Day.Time),
		GramsIn:       row.GramsIn,
		Money:         ledger.Amount(row.Money),
		Takes:         written.Takes,
		Remaining:     written.Remaining,
		Units:         units,
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
	})
	if err != nil {
		return recipe.Production{}, err
	}

	recorded.ID = recipe.ProductionID(row.ID)
	recorded.CreatedAt = timestamp(row.CreatedAt)

	return recorded, nil
}

// writeProductionError translates the truth of the database into the errors
// of the port of the productions.
//
// The constraints of the tables hold the rules of the domain as a defense in
// depth, and the business or the recipe that is not there is the refusal of
// the foreign key.
func writeProductionError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return recipe.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		if pgErr.ConstraintName == productionLayersOnePositionIndex ||
			pgErr.ConstraintName == productionUnitsOnePositionIndex {
			return fmt.Errorf("%w: the constraint %q refused the row",
				recipe.ErrInvalidValue, pgErr.ConstraintName)
		}
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business, the recipe or the layer of the production is not there",
			recipe.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row",
			recipe.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
