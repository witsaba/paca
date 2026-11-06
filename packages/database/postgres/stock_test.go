package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

func TestTheRepositoryOfTheLayersNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewStockRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRepositoryOfTheLayersWithoutADatabaseRefusesToWork(t *testing.T) {
	t.Parallel()

	layers := &StockRepository{}

	if _, err := layers.List(t.Context(), business.ID(77)); !errors.Is(err, purchase.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrNoStorage)
	}
	if _, err := layers.Record(t.Context(), nil); !errors.Is(err, purchase.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrNoStorage)
	}
}

// TestTheRowOfALayerIsMappedOntoTheModel proves that the row is mapped
// onto the model of the domain, and that the model decides what a layer
// is.
func TestTheRowOfALayerIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	mapped, err := layerFromRow(db.StockLayer{
		ID:           3,
		BusinessID:   77,
		IngredientID: 10,
		PurchaseID:   9,
		Position:     1,
		Name:         "Harina panadera",
		Quantity:     25,
		Unit:         "kg",
		Remaining:    25,
		Cost:         12000,
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if mapped.ID != purchase.LayerID(3) {
		t.Errorf("the identity is %d, want 3", mapped.ID)
	}
	if mapped.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", mapped.BusinessID)
	}
	if mapped.IngredientID != ingredient.ID(10) {
		t.Errorf("the ingredient is %d, want 10", mapped.IngredientID)
	}
	if mapped.PurchaseID != purchase.ID(9) {
		t.Errorf("the purchase is %d, want 9", mapped.PurchaseID)
	}
	if mapped.Name != "Harina panadera" {
		t.Errorf("the name is %q, want %q", mapped.Name, "Harina panadera")
	}
	if mapped.Bought.Quantity != 25 || mapped.Bought.Unit != "kg" {
		t.Errorf("what it bought is %v %s, want 25 kg", mapped.Bought.Quantity, mapped.Bought.Unit)
	}
	if mapped.Left.Quantity != 25 || mapped.Left.Unit != "kg" {
		t.Errorf("what is left is %v %s, want 25 kg", mapped.Left.Quantity, mapped.Left.Unit)
	}
	if mapped.Cost != ledger.Amount(12000) {
		t.Errorf("the money of the layer is %d, want 12000", mapped.Cost)
	}
}

// TestTheRowWithAUnitTheBookDoesNotKnowIsRefused proves that the model
// refuses a row that another writer broke: the units of the book are a
// rule.
func TestTheRowOfALayerWithAUnitTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	_, err := layerFromRow(db.StockLayer{
		ID:           3,
		BusinessID:   77,
		IngredientID: 10,
		PurchaseID:   9,
		Position:     1,
		Name:         "Harina panadera",
		Quantity:     25,
		Unit:         "libra",
		Remaining:    25,
		Cost:         12000,
	})
	if !errors.Is(err, ingredient.ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrUnknownUnit)
	}
}

// TestTheRowThatHoldsMoreThanItBoughtIsRefused proves that the model
// refuses a row that another writer broke: what is left of a layer is
// never more than what it bought.
func TestTheRowThatHoldsMoreThanItBoughtIsRefused(t *testing.T) {
	t.Parallel()

	_, err := layerFromRow(db.StockLayer{
		ID:           3,
		BusinessID:   77,
		IngredientID: 10,
		PurchaseID:   9,
		Position:     1,
		Name:         "Harina panadera",
		Quantity:     25,
		Unit:         "kg",
		Remaining:    30,
		Cost:         12000,
	})
	if !errors.Is(err, purchase.ErrInvalidLayer) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrInvalidLayer)
	}
}

// TestTheRefusalsOfTheDatabaseAreTranslatedForTheLayer proves that the
// truth of the database reaches the caller as the errors of the port.
func TestTheRefusalsOfTheDatabaseAreTranslatedForTheLayer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  *pgconn.PgError
		want error
	}{
		{
			name: "the purchase that is not there",
			err:  &pgconn.PgError{Code: foreignKeyViolation},
			want: purchase.ErrNotFound,
		},
		{
			name: "the rule the table holds",
			err:  &pgconn.PgError{Code: checkViolation, ConstraintName: "stock_layers_unit_check"},
			want: purchase.ErrInvalidValue,
		},
		{
			name: "the layer that is already there",
			err:  &pgconn.PgError{Code: uniqueViolation},
			want: purchase.ErrInvalidValue,
		},
	}

	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			t.Parallel()

			err := writeLayerError(one.err)
			if !errors.Is(err, one.want) {
				t.Fatalf("the error is %v, want %v", err, one.want)
			}
		})
	}
}
