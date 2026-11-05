package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
)

// The names the database gives to the constraints of the table of the
// catalog.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	ingredientsNamePerBusinessIndex = "ingredients_one_name_per_business"
)

// IngredientRepository is the adapter of the storage of the catalog of the
// names the business buys.
//
// The aliases of an ingredient travel as JSONB: they are the names the owner
// uses for the thing she buys. The adapter encodes them, and the domain
// reads them back through its own constructors: the row never decides what
// an ingredient is.
type IngredientRepository struct {
	pool *Pool
}

// NewIngredientRepository returns the repository of the catalog, bound to
// the pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewIngredientRepository(pool *Pool) (*IngredientRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &IngredientRepository{pool: pool}, nil
}

// List returns the ingredients of a business, in the order of their
// identity.
//
// The list is empty when the business bought nothing yet: the catalog of a
// business begins empty.
//
// The error is ingredient.ErrNoStorage when the storage is not open.
func (r *IngredientRepository) List(ctx context.Context, businessID business.ID) ([]ingredient.Ingredient, error) {
	if r == nil || r.pool == nil {
		return nil, ingredient.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListIngredientsByBusiness(ctx, int64(businessID))
	if err != nil {
		return nil, writeIngredientError(err)
	}

	held := make([]ingredient.Ingredient, 0, len(rows))
	for _, row := range rows {
		found, err := ingredientFromRow(row)
		if err != nil {
			return nil, err
		}

		held = append(held, found)
	}

	return held, nil
}

// Create writes an ingredient of the catalog of a business, and answers it
// with the identity that the database generated.
//
// The name is unique inside the business: the index of the table holds the
// rule of the domain as a defense in depth, and the adapter says so with
// the errors of the port.
//
// The error is ingredient.ErrDuplicateName when the name is taken inside
// the business, ingredient.ErrNotFound when the business is not there, and
// ingredient.ErrNoStorage when the storage is not open.
func (r *IngredientRepository) Create(ctx context.Context, item ingredient.Ingredient) (ingredient.Ingredient, error) {
	if r == nil || r.pool == nil {
		return ingredient.Ingredient{}, ingredient.ErrNoStorage
	}

	aliases, err := encodeAliases(item.Aliases)
	if err != nil {
		return ingredient.Ingredient{}, err
	}

	row, err := r.pool.queriesFor(ctx).CreateIngredient(ctx, db.CreateIngredientParams{
		BusinessID: int64(item.BusinessID),
		Name:       item.Name,
		Aliases:    aliases,
	})
	if err != nil {
		return ingredient.Ingredient{}, writeIngredientError(err)
	}

	return ingredientFromRow(row)
}

// ingredientFromRow maps a row of the catalog onto the model of the domain.
//
// The ingredient is built by the domain: the row may come from a table that
// was changed by another writer, and the model does not trust a stranger
// with an ingredient it does not know.
func ingredientFromRow(row db.Ingredient) (ingredient.Ingredient, error) {
	aliases, err := decodeAliases(row.Aliases)
	if err != nil {
		return ingredient.Ingredient{}, err
	}

	built, err := ingredient.New(business.ID(row.BusinessID), row.Name, aliases)
	if err != nil {
		return ingredient.Ingredient{}, err
	}

	built.ID = ingredient.ID(row.ID)
	built.CreatedAt = timestamp(row.CreatedAt)
	built.UpdatedAt = timestamp(row.UpdatedAt)

	return built, nil
}

// encodeAliases encodes the names the owner uses for the storage.
func encodeAliases(aliases []string) ([]byte, error) {
	encoded, err := json.Marshal(aliases)
	if err != nil {
		return nil, fmt.Errorf("%w: the aliases of the ingredient could not be encoded: %w", ingredient.ErrInvalidValue, err)
	}

	return encoded, nil
}

// decodeAliases decodes the names the owner uses from the storage.
//
// The aliases are built by the domain: the row never decides what an alias
// is.
func decodeAliases(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}

	var decoded []string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("%w: the aliases of the ingredient are not a list: %w", ingredient.ErrInvalidValue, err)
	}

	return decoded, nil
}

// writeIngredientError translates the truth of the database into the errors
// of the port of the catalog.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth, and the business that is not there is the refusal of the foreign
// key.
func writeIngredientError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ingredient.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		if pgErr.ConstraintName == ingredientsNamePerBusinessIndex {
			return fmt.Errorf("%w: the business already holds an ingredient with that name",
				ingredient.ErrDuplicateName)
		}
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business of the ingredient is not there", ingredient.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", ingredient.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
