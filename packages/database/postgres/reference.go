package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The names the database gives to the constraints of the table of the
// reference prices.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	priceReferencesPriceConstraint         = "price_references_price_positive_check"
	priceReferencesUnitConstraint          = "price_references_unit_not_empty_check"
	priceReferencesCanonicalUnitConstraint = "price_references_unit_check"
	priceReferencesSourceConstraint        = "price_references_source_not_empty_check"
	priceReferencesConfirmedWithConstraint = "price_references_confirmed_with_not_empty_check"
)

// ReferenceRepository is the adapter of the storage of the reference prices.
//
// A reference is what the market was seen at: the ingredient it is about,
// the price, the unit of that price, the market, the source and the day it
// was read.
//
// The row never decides what a reference is: the model is built by the
// domain, through its own constructors.
type ReferenceRepository struct {
	pool *Pool
}

// NewReferenceRepository returns the repository of the reference prices,
// bound to the pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewReferenceRepository(pool *Pool) (*ReferenceRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &ReferenceRepository{pool: pool}, nil
}

// Record writes the reference and answers it with the identity that the
// database generated.
//
// The error is purchase.ErrInvalidValue when the row breaks a rule of the
// table, purchase.ErrNotFound when the business or the ingredient of the
// reference is not there, and purchase.ErrNoStorage when the storage is not
// open.
func (r *ReferenceRepository) Record(ctx context.Context, written purchase.Reference) (purchase.Reference, error) {
	if r == nil || r.pool == nil {
		return purchase.Reference{}, purchase.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreatePriceReference(ctx, db.CreatePriceReferenceParams{
		BusinessID:    int64(written.BusinessID),
		IngredientID:  int64(written.IngredientID),
		Price:         int64(written.Price),
		Unit:          written.Unit,
		Market:        optionalText(written.Market),
		Source:        written.Source,
		SeenOn:        dateOf(written.SeenOn),
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
	})
	if err != nil {
		return purchase.Reference{}, writeReferenceError(err)
	}

	return referenceFromRow(row)
}

// List returns the references of the business, in the order of their
// identity.
//
// The ingredient is optional: the zero identity answers every reference of
// the business, and an identity answers the references of that ingredient
// alone.
//
// The error is purchase.ErrNoStorage when the storage is not open.
func (r *ReferenceRepository) List(ctx context.Context, own business.ID, of ingredient.ID) ([]purchase.Reference, error) {
	if r == nil || r.pool == nil {
		return nil, purchase.ErrNoStorage
	}

	var rows []db.PriceReference

	if of == 0 {
		found, err := r.pool.queriesFor(ctx).ListPriceReferencesByBusiness(ctx, int64(own))
		if err != nil {
			return nil, writeReferenceError(err)
		}

		rows = found
	} else {
		found, err := r.pool.queriesFor(ctx).ListPriceReferencesOfIngredient(ctx, db.ListPriceReferencesOfIngredientParams{
			BusinessID:   int64(own),
			IngredientID: int64(of),
		})
		if err != nil {
			return nil, writeReferenceError(err)
		}

		rows = found
	}

	held := make([]purchase.Reference, 0, len(rows))
	for _, row := range rows {
		found, err := referenceFromRow(row)
		if err != nil {
			return nil, err
		}

		held = append(held, found)
	}

	return held, nil
}

// referenceFromRow maps a row of the references onto the model of the
// domain.
//
// The reference is built by the domain: the row may come from a table that
// was changed by another writer, and the model does not trust a stranger
// with a reference it does not know.
func referenceFromRow(row db.PriceReference) (purchase.Reference, error) {
	reference, err := purchase.NewReference(business.ID(row.BusinessID), purchase.ReferenceDefinition{
		IngredientID:  ingredient.ID(row.IngredientID),
		Price:         ledger.Amount(row.Price),
		Unit:          row.Unit,
		Market:        text(row.Market),
		Source:        row.Source,
		SeenOn:        ledger.NewDate(row.SeenOn.Time),
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
	})
	if err != nil {
		return purchase.Reference{}, err
	}

	reference.ID = purchase.ReferenceID(row.ID)
	reference.CreatedAt = timestamp(row.CreatedAt)

	return reference, nil
}

// writeReferenceError translates the truth of the database into the errors
// of the port of the reference prices.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth, and the business or the ingredient that is not there is the refusal
// of the foreign key.
func writeReferenceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return purchase.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", purchase.ErrInvalidValue, pgErr.ConstraintName)
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business or the ingredient of the reference is not there", purchase.ErrNotFound)
	case checkViolation:
		switch pgErr.ConstraintName {
		case priceReferencesPriceConstraint,
			priceReferencesUnitConstraint,
			priceReferencesCanonicalUnitConstraint,
			priceReferencesSourceConstraint,
			priceReferencesConfirmedWithConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row", purchase.ErrInvalidValue, pgErr.ConstraintName)
		}
	}

	return err
}
