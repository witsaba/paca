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

// The exit of the book is a movement of the book, and the units that left
// it are rows of the table of the exits: the table of the exits owns the
// rule that a unit leaves the book once, and it is shared with the other
// exits.

// ExitRepository is the adapter of the storage of the exits.
//
// An exit is a fact of the business: the kind of the exit, the product it
// took, the quantity it took, the unit, the cost it released, the tax
// treatment it waits for, the day it happened and the word of the owner.
// The units that left the book are rows of the table of the exits, and the
// adapter writes them with it: an exit without its units is not an exit.
//
// The row never decides what an exit is: the model is built by the domain,
// through its own constructors.
type ExitRepository struct {
	pool *Pool
}

// NewExitRepository returns the repository of the exits, bound to the
// database.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewExitRepository(pool *Pool) (*ExitRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &ExitRepository{pool: pool}, nil
}

// Record writes the exit with the units that left the book, and answers it
// with the identity that the storage generated.
//
// The exit and its units are written in one transaction: an exit whose
// units did not leave the book is not an exit.
//
// The error is recipe.ErrInvalidValue when the row breaks a rule of the
// table, recipe.ErrNotFound when the business or the recipe is not there,
// and recipe.ErrNoStorage when the storage is not open.
func (r *ExitRepository) Record(ctx context.Context, written recipe.Exit) (recipe.Exit, error) {
	if r == nil || r.pool == nil {
		return recipe.Exit{}, recipe.ErrNoStorage
	}

	checked, err := written.Checked()
	if err != nil {
		return recipe.Exit{}, err
	}

	// The set joins the transaction of the caller when the caller opened
	// one, and opens its own when it did not: the exit and the units it
	// took are written whole or not at all.
	if _, ok := transactionIn(ctx); ok {
		return r.write(ctx, r.pool.queriesFor(ctx), checked)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return recipe.Exit{}, fmt.Errorf("%w: the transaction of the exits could not be opened: %w",
			recipe.ErrInvalidValue, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	recorded, err := r.write(ctx, r.pool.queriesFor(ctx).WithTx(tx), checked)
	if err != nil {
		return recipe.Exit{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return recipe.Exit{}, fmt.Errorf("%w: the transaction of the exits could not be committed: %w",
			recipe.ErrInvalidValue, err)
	}

	return recorded, nil
}

// write writes the exit and the units that left the book, and answers the
// exit that was written.
func (r *ExitRepository) write(
	ctx context.Context,
	queries *db.Queries,
	written recipe.Exit,
) (recipe.Exit, error) {
	row, err := queries.CreateExit(ctx, db.CreateExitParams{
		BusinessID:    int64(written.BusinessID),
		RecipeID:      int64(written.RecipeID),
		Kind:          written.Kind.String(),
		Day:           dateOf(written.Day),
		Quantity:      written.Quantity,
		Unit:          written.Unit,
		Cost:          int64(written.Cost),
		TaxTreatment:  written.TaxTreatment.String(),
		TaxQuestion:   written.TaxQuestion,
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
	})
	if err != nil {
		return recipe.Exit{}, writeExitError(err)
	}

	units := make([]recipe.ExitedUnit, 0, len(written.Units))

	for _, one := range written.Units {
		place, err := positionOf(one.Position)
		if err != nil {
			return recipe.Exit{}, fmt.Errorf("%w: the position of the unit of the exit: %w",
				recipe.ErrInvalidValue, err)
		}

		if _, err := queries.CreateUnitExit(ctx, db.CreateUnitExitParams{
			UnitID:       int64(one.UnitID),
			ExitKind:     exitKindOf(written.Kind),
			ExitID:       row.ID,
			ProductionID: int64(one.ProductionID),
			Position:     place,
			Grams:        one.Grams,
			ExpiresOn:    dateOf(one.ExpiresOn),
		}); err != nil {
			return recipe.Exit{}, writeExitError(err)
		}

		units = append(units, recipe.ExitedUnit{
			UnitID:       one.UnitID,
			ProductionID: one.ProductionID,
			Position:     one.Position,
			Grams:        one.Grams,
			ExpiresOn:    one.ExpiresOn,
		})
	}

	// The model is built by the domain: the row never decides what an exit
	// is, and a row another writer broke is never read as one.
	recorded, err := recipe.NewExit(business.ID(row.BusinessID), recipe.ExitFact{
		RecipeID:      recipe.ID(row.RecipeID),
		RecipeName:    row.RecipeName,
		Kind:          recipe.ExitKind(row.Kind),
		Day:           ledger.NewDate(row.Day.Time),
		Quantity:      row.Quantity,
		Unit:          row.Unit,
		Cost:          ledger.Amount(row.Cost),
		TaxTreatment:  recipe.TaxTreatment(row.TaxTreatment),
		TaxQuestion:   row.TaxQuestion,
		Units:         units,
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
	})
	if err != nil {
		return recipe.Exit{}, err
	}

	recorded.ID = recipe.ExitID(row.ID)
	recorded.CreatedAt = timestamp(row.CreatedAt)

	return recorded, nil
}

// exitKindOf answers the kind of the exit the table of the exits carries.
//
// The table of the exits holds the four kinds of the book - the waste, the
// sale, the consumption and the donation - and the kind of the exit of the
// domain is the one it carries.
//
// The rule of the domain refuses the kind that is not one of the two exits
// of the book before the adapter is asked, so the answer of the kind that
// is not the donation is the consumption.
func exitKindOf(kind recipe.ExitKind) string {
	if kind == recipe.ExitDonation {
		return exitKindDonation
	}

	return exitKindConsumption
}

// writeExitError translates the truth of the database into the errors of the
// port of the exits.
//
// The constraints of the tables hold the rules of the domain as a defense in
// depth, and the business, the recipe or the unit that is not there is the
// refusal of the foreign key.
func writeExitError(err error) error {
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
		return fmt.Errorf("%w: the business, the recipe or the unit of the exit is not there",
			recipe.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row",
			recipe.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
