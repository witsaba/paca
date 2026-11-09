package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/product"
	"github.com/witsaba/paca/packages/domain/purchase"
)

func TestTheRepositoryOfTheProductsNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewProductRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRepositoryOfThePriceListNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewPriceRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRepositoriesOfThePriceListWithoutADatabaseRefuseToWork(t *testing.T) {
	t.Parallel()

	products := &ProductRepository{}
	prices := &PriceRepository{}

	if _, err := products.List(t.Context(), business.ID(77)); !errors.Is(err, product.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, product.ErrNoStorage)
	}
	if _, err := products.Create(t.Context(), product.Product{}); !errors.Is(err, product.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, product.ErrNoStorage)
	}
	if _, err := products.State(t.Context(), product.Product{}); !errors.Is(err, product.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, product.ErrNoStorage)
	}
	if _, err := prices.List(t.Context(), business.ID(77)); !errors.Is(err, product.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, product.ErrNoStorage)
	}
	if _, err := prices.Set(t.Context(), product.Price{}); !errors.Is(err, product.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, product.ErrNoStorage)
	}
}

// TestTheRowOfAProductIsMappedOntoTheModel proves that the row is mapped
// onto the model of the domain, and that the model decides what a product
// is.
func TestTheRowOfAProductIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	mapped, err := productFromRow(db.Product{
		ID:           10,
		BusinessID:   77,
		RecipeID:     5,
		Name:         "Pan de molde",
		TaxTreatment: pgtype.Text{String: "Grammy 19", Valid: true},
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if mapped.ID != product.ID(10) {
		t.Errorf("the identity is %d, want 10", mapped.ID)
	}
	if mapped.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", mapped.BusinessID)
	}
	if mapped.RecipeID != 5 {
		t.Errorf("the recipe is %d, want 5", mapped.RecipeID)
	}
	if mapped.Name != "Pan de molde" {
		t.Errorf("the name is %q, want %q", mapped.Name, "Pan de molde")
	}
	if mapped.TaxTreatment != "Grammy 19" {
		t.Errorf("the tax treatment is %q, want %q", mapped.TaxTreatment, "Grammy 19")
	}
}

// TestTheRowOfAProductWithoutItsTaxTreatmentIsMapped proves that the tax
// treatment of a product is optional: the row without one is a product.
func TestTheRowOfAProductWithoutItsTaxTreatmentIsMapped(t *testing.T) {
	t.Parallel()

	mapped, err := productFromRow(db.Product{
		ID:         10,
		BusinessID: 77,
		RecipeID:   5,
		Name:       "Pan de molde",
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if mapped.TaxTreatment != "" {
		t.Errorf("the tax treatment is %q, want none", mapped.TaxTreatment)
	}
}

// TestTheRowOfAProductThatBreaksTheRuleOfTheDomainIsRefused proves that
// the model refuses a row that another writer broke: the name of a product
// is not empty and the product names a recipe.
func TestTheRowOfAProductThatBreaksTheRuleOfTheDomainIsRefused(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		row  db.Product
		want error
	}{
		{
			name: "the product without a name",
			row: db.Product{
				BusinessID: 77,
				RecipeID:   5,
				Name:       "  ",
			},
			want: product.ErrInvalidName,
		},
		{
			name: "the product without a recipe",
			row: db.Product{
				BusinessID: 77,
				Name:       "Pan de molde",
			},
			want: product.ErrMissingRecipe,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := productFromRow(testCase.row)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}

// TestTheRowOfAPriceIsMappedOntoTheModel proves that the row is mapped
// onto the model of the domain, and that the channel of the price is read
// back through the domain.
func TestTheRowOfAPriceIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	mapped, err := priceFromRow(db.Price{
		ID:         100,
		BusinessID: 77,
		ProductID:  10,
		Channel:    "wholesale",
		Amount:     2500,
		Basis:      "unit",
		Day:        pgtype.Date{Time: mustDate(t, "2026-03-01"), Valid: true},
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if mapped.ID != product.PriceID(100) {
		t.Errorf("the identity is %d, want 100", mapped.ID)
	}
	if mapped.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", mapped.BusinessID)
	}
	if mapped.ProductID != product.ID(10) {
		t.Errorf("the product is %d, want 10", mapped.ProductID)
	}
	if mapped.Channel != product.ChannelWholesale {
		t.Errorf("the channel is %q, want %q", mapped.Channel, product.ChannelWholesale)
	}
	if mapped.Amount != 2500 {
		t.Errorf("the money is %d, want 2500", mapped.Amount)
	}
	if mapped.Basis != purchase.PricePerUnit {
		t.Errorf("the basis is %q, want %q", mapped.Basis, purchase.PricePerUnit)
	}
	if mapped.Day.String() != "2026-03-01" {
		t.Errorf("the day is %q, want %q", mapped.Day.String(), "2026-03-01")
	}
}

// TestTheRowOfAPriceThatBreaksTheRuleOfTheDomainIsRefused proves that the
// model refuses a row that another writer broke: the channel is one of the
// two of the book and the money is a positive number of pesos.
func TestTheRowOfAPriceThatBreaksTheRuleOfTheDomainIsRefused(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		row  db.Price
		want error
	}{
		{
			name: "the channel the book does not know",
			row: db.Price{
				BusinessID: 77,
				ProductID:  10,
				Channel:    "store",
				Amount:     2500,
				Basis:      "unit",
			},
			want: product.ErrInvalidValue,
		},
		{
			name: "the way of charging the book does not know",
			row: db.Price{
				BusinessID: 77,
				ProductID:  10,
				Channel:    "retail",
				Amount:     2500,
				Basis:      "loaf",
			},
			want: product.ErrInvalidValue,
		},
		{
			name: "the price that is not a positive number of pesos",
			row: db.Price{
				BusinessID: 77,
				ProductID:  10,
				Channel:    "retail",
				Amount:     0,
				Basis:      "unit",
			},
			want: product.ErrInvalidPrice,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := priceFromRow(testCase.row)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}

// TestTheRefusalsOfTheDatabaseAreTranslatedForThePriceList proves that the
// truth of the database reaches the caller as the errors of the port.
func TestTheRefusalsOfTheDatabaseAreTranslatedForThePriceList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     *pgconn.PgError
		want    error
		message string
	}{
		{
			name: "the business that is not there",
			err: &pgconn.PgError{
				Code:           foreignKeyViolation,
				ConstraintName: "products_business_id_fkey",
			},
			want:    product.ErrNotFound,
			message: "the business or the recipe of the price is not there",
		},
		{
			name: "the name that is taken",
			err: &pgconn.PgError{
				Code:           uniqueViolation,
				ConstraintName: productsNameConstraint,
			},
			want:    product.ErrDuplicateName,
			message: "refused the row",
		},
		{
			name: "the recipe that backs another product",
			err: &pgconn.PgError{
				Code:           uniqueViolation,
				ConstraintName: productsRecipeConstraint,
			},
			want:    product.ErrDuplicateRecipe,
			message: "refused the row",
		},
		{
			name: "the row that breaks a rule of the table",
			err: &pgconn.PgError{
				Code:           checkViolation,
				ConstraintName: "prices_channel_check",
			},
			want:    product.ErrInvalidValue,
			message: "refused the row",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := writePriceError(testCase.err)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
			if !strings.Contains(err.Error(), testCase.message) {
				t.Errorf("the error is %q, want it to say %q", err.Error(), testCase.message)
			}
		})
	}
}

// mustDate reads a day of the book of the bakery.
func mustDate(t *testing.T, day string) time.Time {
	t.Helper()

	read, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatalf("the day %q could not be read: %v", day, err)
	}

	return read
}
