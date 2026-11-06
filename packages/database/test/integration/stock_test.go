//go:build integration

package integration_test

import (
	"context"
	"errors"
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

// thePurchaseWithTheUnitOfTheOwner builds the purchase whose line names
// the unit the way the owner said it.
//
// The sentence of the owner keeps the exact wording, and the line of the
// purchase carries the unit the book knows: the domain is what turns the
// wording into the unit, and the layer that is derived from it carries the
// same unit.
func thePurchaseWithTheUnitOfTheOwner(
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

	measure, err := purchase.NewMeasure(25, "kilos")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	line, err := purchase.NewLine("Harina", measure, purchase.Price{Total: 12000})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	return purchase.Purchase{
		DraftID:       draft.ID,
		BusinessID:    bakery.ID,
		Day:           draft.Day,
		Utterance:     draft.Utterance,
		ConfirmedWith: "sí",
		Lines: []purchase.Line{
			line.WithIngredient(of),
		},
		Total: 12000,
	}
}

// TestThePurchaseKeepsTheSentenceOfTheOwnerAndTheUnitOfTheBook proves that
// the sentence of the owner keeps the exact wording and that the purchase
// and its layer carry the unit of the book: the layer is derived, and the
// stock never adds a gram to a kilogram.
//
// The rule that changed: the purchase keeps the unit of the book too, and
// not only the layer.
func TestThePurchaseKeepsTheSentenceOfTheOwnerAndTheUnitOfTheBook(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := layersRepository(t)
	bakery := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)

	bought, err := purchasesRecorder(t).Record(ctx, thePurchaseWithTheUnitOfTheOwner(t, ctx, bakery, flour.ID))
	if err != nil {
		t.Fatalf("the purchase could not be written: %v", err)
	}

	if bought.Lines[0].Measure.Unit != "kg" {
		t.Errorf("the unit of the line of the purchase is %q, want %q: the book stores the unit of the book",
			bought.Lines[0].Measure.Unit, "kg")
	}
	if bought.Utterance != "Compré 25 kilos de harina" {
		t.Errorf("the sentence is %q, want %q: the sentence of the owner keeps the exact wording",
			bought.Utterance, "Compré 25 kilos de harina")
	}

	written, err := repo.Record(ctx, theLayersOfThePurchase(t, bought, flour.ID))
	if err != nil {
		t.Fatalf("the layers could not be written: %v", err)
	}
	if written[0].Bought.Unit != "kg" {
		t.Errorf("the unit of what the layer bought is %q, want %q", written[0].Bought.Unit, "kg")
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the layers could not be read: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the stock holds %d layers, want 1", len(held))
	}
	if held[0].Bought.Unit != "kg" {
		t.Errorf("the unit of what the layer bought is %q, want %q", held[0].Bought.Unit, "kg")
	}
}

// TestTheLineOfAPurchaseWithAUnitTheBookDoesNotKnowIsRefused proves the
// rule of the table: the unit of a line is one of the four units of the
// book, and the table refuses the row another writer wrote with a unit that
// is not one of them.
func TestTheLineOfAPurchaseWithAUnitTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)
	bought := thePurchaseOfTheLayers(t, ctx, bakery, flour.ID)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO purchase_lines (purchase_id, position, name, quantity, unit, total, unit_price)
		 VALUES ($1, 1, 'Harina', 25, 'libra', 12000, 0)`,
		int64(bought.ID),
	)
	if err == nil {
		t.Fatal("the row with the unit of a stranger was written, want the refusal of the table")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "purchase_lines_unit_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "purchase_lines_unit_check")
	}
}

// TestTheLayerOfAnotherBusinessIsNotReadAsTheStockOfABusiness proves that
// the stock of a business never holds the layer of a purchase of another
// one: the layer belongs to the business of its purchase.
func TestTheLayerOfAnotherBusinessIsNotReadAsTheStockOfABusiness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := layersRepository(t)
	bakery := theBakeryOfTheLayers(t, ctx)
	other := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)
	bought := thePurchaseOfTheLayers(t, ctx, bakery, flour.ID)

	// The row another writer broke: the layer claims the other business,
	// and it comes from a purchase of the bakery.
	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO stock_layers (business_id, ingredient_id, purchase_id, position, name, day, quantity, unit, remaining, cost)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		int64(other.ID), int64(flour.ID), int64(bought.ID), 1, "Harina",
		ledger.NewDate(time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)).String(),
		25, "kg", 25, 12000,
	)
	if err != nil {
		t.Fatalf("the row could not be written: %v", err)
	}

	held, err := repo.List(ctx, other.ID)
	if err != nil {
		t.Fatalf("the layers could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("the stock of the other business holds %d layers, want none", len(held))
	}
}

// TestTheStorageRefusesALayerOfAnotherBusiness proves that the domain
// refuses a layer that claims another business than the purchase it comes
// from, and that nothing is written.
func TestTheStorageRefusesALayerOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := layersRepository(t)
	bakery := theBakeryOfTheLayers(t, ctx)
	other := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)
	bought := thePurchaseOfTheLayers(t, ctx, bakery, flour.ID)

	layers := theLayersOfThePurchase(t, bought, flour.ID)
	layers[0].BusinessID = other.ID

	if _, err := repo.Record(ctx, layers); !errors.Is(err, purchase.ErrLayerOfAnotherBusiness) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrLayerOfAnotherBusiness)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the layers could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("the stock holds %d layers, want none", len(held))
	}
}

// TestTheRefusedLayerLeavesNeitherThePurchaseNorTheLayerBehind proves that
// the purchase and its layers are written in one transaction: a layer the
// storage refuses takes the purchase with it, and nothing is left behind.
func TestTheRefusedLayerLeavesNeitherThePurchaseNorTheLayerBehind(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)
	bought := thePurchaseWithTheUnitOfTheOwner(t, ctx, bakery, flour.ID)

	recorder := purchasesRecorder(t)
	repo := layersRepository(t)
	work, err := postgres.NewUnitOfWork(theSuite.pool)
	if err != nil {
		t.Fatalf("the unit of work could not be created: %v", err)
	}

	err = work.Do(ctx, func(ctx context.Context) error {
		recorded, err := recorder.Record(ctx, bought)
		if err != nil {
			return err
		}

		layers, err := purchase.LayersOf(recorded)
		if err != nil {
			return err
		}

		// The layer names an ingredient that is not there: the domain
		// cannot know it, and the table refuses the row.
		layers[0].IngredientID = ingredient.ID(999_999_999)

		_, err = repo.Record(ctx, layers)

		return err
	})
	if err == nil {
		t.Fatal("the transaction was committed, want the layer refused")
	}

	var purchases int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM purchases WHERE business_id = $1`, int64(bakery.ID)).Scan(&purchases); err != nil {
		t.Fatalf("the purchases could not be counted: %v", err)
	}
	if purchases != 0 {
		t.Errorf("the purchase was left behind: %d purchases of the business", purchases)
	}

	var layers int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM stock_layers WHERE business_id = $1`, int64(bakery.ID)).Scan(&layers); err != nil {
		t.Fatalf("the layers could not be counted: %v", err)
	}
	if layers != 0 {
		t.Errorf("the layers were left behind: %d layers of the business", layers)
	}
}

// TestThePurchaseAndItsLayersAreOneTransaction proves that the purchase and
// the layers that come from it are written in the same transaction: a
// failure that comes after them leaves neither the purchase nor the layer
// behind.
func TestThePurchaseAndItsLayersAreOneTransaction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLayers(t, ctx)
	flour := theIngredientOfTheLayers(t, ctx, bakery)

	// The purchase that is already written: its layer is written inside
	// the transaction of the test, and it goes back with it.
	written := thePurchaseOfTheLayers(t, ctx, bakery, flour.ID)

	// The purchase that the transaction writes: it goes back with it too.
	bought := thePurchaseWithTheUnitOfTheOwner(t, ctx, bakery, flour.ID)

	recorder := purchasesRecorder(t)
	repo := layersRepository(t)
	work, err := postgres.NewUnitOfWork(theSuite.pool)
	if err != nil {
		t.Fatalf("the unit of work could not be created: %v", err)
	}

	err = work.Do(ctx, func(ctx context.Context) error {
		if _, err := repo.Record(ctx, theLayersOfThePurchase(t, written, flour.ID)); err != nil {
			return err
		}

		recorded, err := recorder.Record(ctx, bought)
		if err != nil {
			return err
		}

		if _, err := repo.Record(ctx, theLayersOfThePurchase(t, recorded, flour.ID)); err != nil {
			return err
		}

		// The movement of the money of the payment failed: the purchase
		// and the layers that come from it go back with it.
		return errors.New("the movement of the money of the payment failed")
	})
	if err == nil {
		t.Fatal("the transaction was committed, want the failure of the movement of the money")
	}

	var purchases int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM purchases WHERE business_id = $1`, int64(bakery.ID)).Scan(&purchases); err != nil {
		t.Fatalf("the purchases could not be counted: %v", err)
	}
	if purchases != 1 {
		t.Errorf("the business holds %d purchases, want the one that was written before the transaction", purchases)
	}

	var layers int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM stock_layers WHERE business_id = $1`, int64(bakery.ID)).Scan(&layers); err != nil {
		t.Fatalf("the layers could not be counted: %v", err)
	}
	if layers != 0 {
		t.Errorf("the layers were left behind: %d layers of the business", layers)
	}
}
