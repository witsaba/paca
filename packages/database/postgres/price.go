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
	"github.com/witsaba/paca/packages/domain/product"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names the database gives to the indexes of the price list.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	// productsNameConstraint is the name of the index that refuses a
	// second product with the same name inside the same business.
	productsNameConstraint = "products_one_name_per_business"

	// productsRecipeConstraint is the name of the index that refuses a
	// second product of the same recipe inside the same business.
	productsRecipeConstraint = "products_one_recipe_per_business"
)

// ProductRepository is the adapter of the storage of the products.
//
// The product is the thing the bakery sells: it carries the recipe it
// comes from and the name the owner reads. The adapter builds it through
// the domain: the row never decides what a product is.
//
// The tax treatment of a product is an OPTIONAL fact of it: the column is
// NULL when the owner has not decided it, and the book never derives it.
type ProductRepository struct {
	pool *Pool
}

// NewProductRepository returns the repository of the products, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewProductRepository(pool *Pool) (*ProductRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &ProductRepository{pool: pool}, nil
}

// Create writes a product of the business, and answers it with the
// identity that the database generated.
//
// The rules of the domain travel with the port: the name of a product is
// taken inside the business, one recipe backs one product, and the tax
// treatment is optional.
//
// The error is product.ErrNotFound when the business or the recipe of the
// product is not there, product.ErrDuplicateName when the business already
// holds a product with that name, product.ErrDuplicateRecipe when the
// recipe already backs another product, product.ErrInvalidValue when the
// row breaks a rule of the table, and product.ErrNoStorage when the storage
// is not open.
func (r *ProductRepository) Create(ctx context.Context, p product.Product) (product.Product, error) {
	if r == nil || r.pool == nil {
		return product.Product{}, product.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreateProduct(ctx, db.CreateProductParams{
		BusinessID:   int64(p.BusinessID),
		RecipeID:     int64(p.RecipeID),
		Name:         p.Name,
		TaxTreatment: optionalText(p.TaxTreatment),
	})
	if err != nil {
		return product.Product{}, writePriceError(err)
	}

	return productFromRow(row)
}

// State writes the tax treatment the owner stated for the product, and
// answers it with the product as it is now.
//
// The rule of the domain: the tax treatment is an OPTIONAL fact of the
// product, and the name and the recipe of the product never change: only
// the treatment the owner stated is written.
//
// The error is product.ErrNotFound when the product is not there, and
// product.ErrNoStorage when the storage is not open.
func (r *ProductRepository) State(ctx context.Context, p product.Product) (product.Product, error) {
	if r == nil || r.pool == nil {
		return product.Product{}, product.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).StateProduct(ctx, db.StateProductParams{
		ID:           int64(p.ID),
		BusinessID:   int64(p.BusinessID),
		TaxTreatment: optionalText(p.TaxTreatment),
	})
	if err != nil {
		return product.Product{}, writePriceError(err)
	}

	return productFromRow(row)
}

// List returns the products of the business, in the order of their
// identity.
//
// The error is product.ErrNoStorage when the storage is not open.
func (r *ProductRepository) List(ctx context.Context, own business.ID) ([]product.Product, error) {
	if r == nil || r.pool == nil {
		return nil, product.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListProductsByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writePriceError(err)
	}

	held := make([]product.Product, 0, len(rows))
	for _, row := range rows {
		found, err := productFromRow(row)
		if err != nil {
			return nil, err
		}

		held = append(held, found)
	}

	return held, nil
}

// PriceRepository is the adapter of the storage of the price list.
//
// The price is a fact of the business: it is written once, and it is never
// rewritten. The channel is already in the price from the first row, and
// the history of the prices of a channel is kept.
//
// The money of a price travels in whole pesos, and the adapter reads it
// back through the domain: the row never decides what a price is.
type PriceRepository struct {
	pool *Pool
}

// NewPriceRepository returns the repository of the price list, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewPriceRepository(pool *Pool) (*PriceRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &PriceRepository{pool: pool}, nil
}

// Set writes the price of a product for a channel, and answers it with the
// identity that the database generated.
//
// The rule of the domain: the price is a fact and it is never rewritten.
// The row of a price is kept, and the price that holds is the last one set
// on or before the day that is asked for.
//
// The error is product.ErrNotFound when the business or the product of the
// price is not there, product.ErrInvalidValue when the row breaks a rule
// of the table, and product.ErrNoStorage when the storage is not open.
func (r *PriceRepository) Set(ctx context.Context, price product.Price) (product.Price, error) {
	if r == nil || r.pool == nil {
		return product.Price{}, product.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreatePrice(ctx, db.CreatePriceParams{
		BusinessID: int64(price.BusinessID),
		ProductID:  int64(price.ProductID),
		Channel:    price.Channel.String(),
		Amount:     int64(price.Amount),
		Day:        dateOf(price.Day),
	})
	if err != nil {
		return product.Price{}, writePriceError(err)
	}

	return priceFromRow(row)
}

// List returns the prices of the business, in the order of their identity.
//
// The list holds the whole history of the prices of the business, and the
// prices of another business are never read as prices of it.
//
// The error is product.ErrNoStorage when the storage is not open.
func (r *PriceRepository) List(ctx context.Context, own business.ID) ([]product.Price, error) {
	if r == nil || r.pool == nil {
		return nil, product.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListPricesByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writePriceError(err)
	}

	held := make([]product.Price, 0, len(rows))
	for _, row := range rows {
		found, err := priceFromRow(row)
		if err != nil {
			return nil, err
		}

		held = append(held, found)
	}

	return held, nil
}

// productFromRow maps a row of the products onto the model of the domain.
//
// The product is built by the domain: the row may come from a table that
// was changed by another writer, and the model does not trust a stranger
// with a product it does not know.
func productFromRow(row db.Product) (product.Product, error) {
	built, err := product.New(
		business.ID(row.BusinessID),
		recipe.ID(row.RecipeID),
		row.Name,
		text(row.TaxTreatment),
	)
	if err != nil {
		return product.Product{}, err
	}

	built.ID = product.ID(row.ID)
	built.CreatedAt = timestamp(row.CreatedAt)
	built.UpdatedAt = timestamp(row.UpdatedAt)

	return built, nil
}

// priceFromRow maps a row of the prices onto the model of the domain.
//
// The price is built by the domain: the row may come from a table that was
// changed by another writer, and the model does not trust a stranger with a
// price it does not know.
func priceFromRow(row db.Price) (product.Price, error) {
	channel, err := product.ParseChannel(row.Channel)
	if err != nil {
		return product.Price{}, fmt.Errorf("%w: the channel of the price is %q", product.ErrInvalidValue, row.Channel)
	}

	built, err := product.NewPrice(
		business.ID(row.BusinessID),
		product.ID(row.ProductID),
		channel,
		ledger.Amount(row.Amount),
		ledger.NewDate(row.Day.Time),
	)
	if err != nil {
		return product.Price{}, err
	}

	built.ID = product.PriceID(row.ID)
	built.CreatedAt = timestamp(row.CreatedAt)

	return built, nil
}

// writePriceError translates the truth of the database into the errors of
// the port of the price list.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth: the name of a product is taken inside the business, one recipe
// backs one product, the channel is one of the two of the book and the
// money of a price is a positive number of pesos.
func writePriceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return product.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		switch pgErr.ConstraintName {
		case productsNameConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				product.ErrDuplicateName, pgErr.ConstraintName)
		case productsRecipeConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				product.ErrDuplicateRecipe, pgErr.ConstraintName)
		}

		return fmt.Errorf("%w: the constraint %q refused the row", product.ErrInvalidValue, pgErr.ConstraintName)
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business or the recipe of the price is not there", product.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", product.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
