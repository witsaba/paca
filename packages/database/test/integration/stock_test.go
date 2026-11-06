//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/user"
)

// layersRepository returns the repository of the layers, bound to the
// database of the suite.
func layersRepository(t *testing.T) *postgres.StockRepository {
	t.Helper()

	repo, err := postgres.NewStockRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the layers could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheLayers opens the business that owns the stock of the test.
func theBakeryOfTheLayers(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Stock")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theIngredientOfTheLayers writes the ingredient the stock of the test is
// of.
func theIngredientOfTheLayers(t *testing.T, ctx context.Context, bakery business.Business) ingredient.Ingredient {
	t.Helper()

	built, err := ingredient.New(bakery.ID, "Harina panadera", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	written, err := ingredientsRepository(t).Create(ctx, built)
	if err != nil {
		t.Fatalf("the ingredient could not be written: %v", err)
	}

	return written
}

// thePurchaseOfTheLayers writes the purchase the layers of the test come
// from.
func thePurchaseOfTheLayers(
	t *testing.T,
	ctx context.Context,
	bakery business.Business,
	of ingredient.ID,
) purchase.Purchase {
	t.Helper()

	draft, err := draftsRepository(t).Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	bought := purchase.Purchase{
		DraftID:       draft.ID,
		BusinessID:    bakery.ID,
		Day:           draft.Day,
		Market:        draft.Market,
		Invoice:       draft.Invoice,
		Utterance:     draft.Utterance,
		ConfirmedWith: "sí",
		Lines: []purchase.Line{
			{
				Name:         "Harina",
				Measure:      purchase.Measure{Quantity: 25, Unit: "kg"},
				Price:        purchase.Price{Total: 12000, Unit: 480},
				IngredientID: of,
			},
		},
		Total: 12000,
	}

	written, err := purchasesRecorder(t).Record(ctx, bought)
	if err != nil {
		t.Fatalf("the purchase could not be written: %v", err)
	}

	return written
}

// theLayersOfThePurchase derives the layers of the purchase the tests of
// the table write.
func theLayersOfThePurchase(t *testing.T, bought purchase.Purchase, of ingredient.ID) []purchase.Layer {
	t.Helper()

	layers, err := purchase.LayersOf(bought)
	if err != nil {
		t.Fatalf("the layers were refused: %v", err)
	}

	return layers
}

// TestTheLayersOfThePurchaseAreWrittenAndReadBack proves the rules of the
// table against a real database: the layer is derived from the purchase,
// it is written with the ingredient, the quantity, the unit, the money it
// cost, the day it was bought and the purchase it comes from, and it is
// read back as it was written.
func TestTheLayersOfThePurchaseAreWrittenAndReadBack(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := layersRepository(t)
	bakery := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)
	bought := thePurchaseOfTheLayers(t, ctx, bakery, flour.ID)

	written, err := repo.Record(ctx, theLayersOfThePurchase(t, bought, flour.ID))
	if err != nil {
		t.Fatalf("the layers could not be written: %v", err)
	}

	if len(written) != 1 {
		t.Fatalf("the purchase derived %d layers, want 1", len(written))
	}
	if written[0].ID == 0 {
		t.Error("the layer has no identity, want the one the database generated")
	}
	if written[0].BusinessID != bakery.ID {
		t.Errorf("the business of the layer is %d, want %d", written[0].BusinessID, bakery.ID)
	}
	if written[0].IngredientID != flour.ID {
		t.Errorf("the ingredient of the layer is %d, want %d", written[0].IngredientID, flour.ID)
	}
	if written[0].PurchaseID != bought.ID {
		t.Errorf("the purchase of the layer is %d, want %d", written[0].PurchaseID, bought.ID)
	}
	if written[0].Bought.Quantity != 25 || written[0].Bought.Unit != "kg" {
		t.Errorf("what the layer bought is %+v, want 25 kg", written[0].Bought)
	}
	if written[0].Left.Quantity != 25 {
		t.Errorf("what is left of the layer is %v, want 25", written[0].Left.Quantity)
	}
	if written[0].Cost != ledger.Amount(12000) {
		t.Errorf("the money of the layer is %d, want 12000", written[0].Cost)
	}
	if written[0].Day != bought.Day {
		t.Errorf("the day of the layer is %v, want the day of the purchase", written[0].Day)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the layers could not be read: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the stock holds %d layers, want 1", len(held))
	}
	if held[0].Name != "Harina" {
		t.Errorf("the name of the layer is %q, want %q", held[0].Name, "Harina")
	}
}

// TestTheLayersOfTheBusinessAreTheOnlyOnesItReads proves that the stock of
// a business holds its own layers and no other one.
func TestTheLayersOfTheBusinessAreTheOnlyOnesItReads(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := layersRepository(t)
	bakery := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)
	bought := thePurchaseOfTheLayers(t, ctx, bakery, flour.ID)

	if _, err := repo.Record(ctx, theLayersOfThePurchase(t, bought, flour.ID)); err != nil {
		t.Fatalf("the layers could not be written: %v", err)
	}

	other := theBakeryOfTheLayers(t, ctx)

	held, err := repo.List(ctx, other.ID)
	if err != nil {
		t.Fatalf("the layers could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("the stock of the other business holds %d layers, want none", len(held))
	}
}

// TestTheTableRefusesTheUnitThatIsNotOneOfTheBook proves the rule of the
// domain against a real database: the units of the book are the gram, the
// kilogram, the package and the piece, and nothing else.
func TestTheTableOfTheLayersRefusesTheUnitThatIsNotOneOfTheBook(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)
	bought := thePurchaseOfTheLayers(t, ctx, bakery, flour.ID)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO stock_layers (business_id, ingredient_id, purchase_id, position, name, day, quantity, unit, remaining, cost)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		int64(bakery.ID), int64(flour.ID), int64(bought.ID), 1, "Harina",
		ledger.NewDate(time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)).String(),
		25, "libra", 25, 12000,
	)
	if err == nil {
		t.Fatal("the database stored a unit the book does not know, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "stock_layers_unit_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "stock_layers_unit_check")
	}
}

// TestTheTableRefusesALayerThatHoldsMoreThanItBought proves the rule of
// the domain against a real database: what is left of a layer is never
// more than what it bought.
func TestTheTableOfTheLayersRefusesALayerThatHoldsMoreThanItBought(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)
	bought := thePurchaseOfTheLayers(t, ctx, bakery, flour.ID)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO stock_layers (business_id, ingredient_id, purchase_id, position, name, day, quantity, unit, remaining, cost)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		int64(bakery.ID), int64(flour.ID), int64(bought.ID), 1, "Harina",
		ledger.NewDate(time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)).String(),
		25, "kg", 30, 12000,
	)
	if err == nil {
		t.Fatal("the database stored a layer that holds more than it bought, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "stock_layers_remaining_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "stock_layers_remaining_check")
	}
}

// TestTheTableRefusesTwoLayersOfTheSameLine proves that a layer is derived
// from one line of the purchase and from no other one.
func TestTheTableOfTheLayersRefusesTwoLayersOfTheSameLine(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)
	bought := thePurchaseOfTheLayers(t, ctx, bakery, flour.ID)

	day := ledger.NewDate(time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)).String()
	insert := `INSERT INTO stock_layers (business_id, ingredient_id, purchase_id, position, name, day, quantity, unit, remaining, cost)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	if _, err := theSuite.db.ExecContext(ctx, insert,
		int64(bakery.ID), int64(flour.ID), int64(bought.ID), 1, "Harina",
		day, 25, "kg", 25, 12000,
	); err != nil {
		t.Fatalf("the layer could not be written: %v", err)
	}

	_, err := theSuite.db.ExecContext(ctx, insert,
		int64(bakery.ID), int64(flour.ID), int64(bought.ID), 1, "Harina",
		day, 25, "kg", 25, 12000,
	)
	if err == nil {
		t.Fatal("the database stored two layers of the same line, want the second one refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != uniqueViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, uniqueViolation)
	}
	if pgErr.ConstraintName != "stock_layers_one_per_line" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "stock_layers_one_per_line")
	}
}
