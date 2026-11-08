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
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/user"
)

// productionsRepository returns the repository of the productions, bound to
// the database of the suite.
func productionsRepository(t *testing.T) *postgres.ProductionRepository {
	t.Helper()

	repo, err := postgres.NewProductionRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the productions could not be created: %v", err)
	}

	return repo
}

// unitOfWork returns the transaction of the storage, bound to the database
// of the suite.
func unitOfWork(t *testing.T) *postgres.UnitOfWork {
	t.Helper()

	work, err := postgres.NewUnitOfWork(theSuite.pool)
	if err != nil {
		t.Fatalf("the transaction of the storage could not be created: %v", err)
	}

	return work
}

// theBakeryOfTheProductions opens the business that owns the productions of
// the test.
func theBakeryOfTheProductions(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Productions")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theStockOfTheProductions writes the ingredient, the purchase and the
// layers the production of the test consumes.
func theStockOfTheProductions(
	t *testing.T,
	ctx context.Context,
	bakery business.Business,
) (ingredient.Ingredient, []purchase.Layer) {
	t.Helper()

	flour, err := ingredient.New(bakery.ID, "Harina panadera", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	written, err := ingredientsRepository(t).Create(ctx, flour)
	if err != nil {
		t.Fatalf("the ingredient could not be written: %v", err)
	}

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
				IngredientID: written.ID,
			},
		},
		Total: 12000,
	}

	recorded, err := purchasesRecorder(t).Record(ctx, bought)
	if err != nil {
		t.Fatalf("the purchase could not be written: %v", err)
	}

	layers, err := purchase.LayersOf(recorded)
	if err != nil {
		t.Fatalf("the layers were refused: %v", err)
	}

	stored, err := layersRepository(t).Record(ctx, layers)
	if err != nil {
		t.Fatalf("the layers could not be written: %v", err)
	}

	return written, stored
}

// theRecipeOfTheProductions writes the recipe the production of the test is
// made of.
func theRecipeOfTheProductions(
	t *testing.T,
	ctx context.Context,
	bakery business.Business,
	flour ingredient.ID,
) recipe.Recipe {
	t.Helper()

	built, err := recipe.New(bakery.ID, recipe.Definition{
		Name: "Pan de molde",
		Ingredients: []recipe.Ingredient{
			{IngredientID: flour, Grams: 500},
		},
		Steps: []recipe.Step{
			{Instruction: "Hornear."},
		},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		Utterance:     "El pan de molde lleva 500 gramos de harina",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	written, err := recipesRepository(t).Record(ctx, built)
	if err != nil {
		t.Fatalf("the recipe could not be written: %v", err)
	}

	return written
}

// theProductionOfTheTest derives the production the tests of the table
// write: five hundred grams of the stock of the business, and ten units of
// ninety grams that expire three days after the production.
func theProductionOfTheTest(
	t *testing.T,
	bakery business.Business,
	baked recipe.Recipe,
	layers []purchase.Layer,
	flour ingredient.Ingredient,
) recipe.Production {
	t.Helper()

	catalog, err := ingredient.NewCatalog(bakery.ID, []ingredient.Ingredient{flour})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	production, err := recipe.ProductionOf(baked, layers, catalog, recipe.ProductionDefinition{
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 8, 0, 0, 0, time.UTC)),
		Utterance:     "Hice pan de molde el sábado",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the production was refused: %v", err)
	}

	return production
}

// TestTheProductionIsKeptWithItsRecipeAndItsDay proves the rules of the
// table against a real database: the production is written with the recipe
// it was made of, the day it was made, the grams that went in, the money
// and the word of the owner, and it is read back as it was written.
func TestTheProductionIsKeptWithItsRecipeAndItsDay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := productionsRepository(t)
	bakery := theBakeryOfTheProductions(t, ctx)
	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	written, err := repo.Record(ctx, theProductionOfTheTest(t, bakery, baked, layers, flour))
	if err != nil {
		t.Fatalf("the production could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the production has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the production is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.RecipeID != baked.ID {
		t.Errorf("the recipe of the production is %d, want %d", written.RecipeID, baked.ID)
	}
	if written.RecipeName != "Pan de molde" {
		t.Errorf("the name of the recipe is %q, want %q", written.RecipeName, "Pan de molde")
	}
	if written.Day.String() != "2026-03-14" {
		t.Errorf("the day of the production is %q, want %q", written.Day.String(), "2026-03-14")
	}
	if written.GramsIn != 500 {
		t.Errorf("the grams that went in are %v, want 500", written.GramsIn)
	}
	if written.Utterance != "Hice pan de molde el sábado" {
		t.Errorf("the sentence of the production is %q, want the one of the owner", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if written.CreatedAt.IsZero() {
		t.Error("the production carries no moment of creation, want the one the database filled")
	}
}

// TestTheUnitsOfTheProductionAreKeptWithTheDayTheyExpire proves the rules
// of the table against a real database: every unit the production made is
// kept with its weight and the day it expires, which is the day of the
// production plus the shelf life of the recipe.
func TestTheUnitsOfTheProductionAreKeptWithTheDayTheyExpire(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := productionsRepository(t)
	bakery := theBakeryOfTheProductions(t, ctx)
	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	written, err := repo.Record(ctx, theProductionOfTheTest(t, bakery, baked, layers, flour))
	if err != nil {
		t.Fatalf("the production could not be written: %v", err)
	}

	if len(written.Units) != 10 {
		t.Fatalf("the production made %d units, want 10", len(written.Units))
	}

	for index, one := range written.Units {
		if one.ID == 0 {
			t.Errorf("the unit %d has no identity, want the one the database generated", index+1)
		}
		if one.Position != index+1 {
			t.Errorf("the unit %d is the unit %d of the production", one.Position, index+1)
		}
		if one.Grams != 90 {
			t.Errorf("the weight of the unit %d is %v, want 90", index+1, one.Grams)
		}
		if one.ExpiresOn.String() != "2026-03-17" {
			t.Errorf("the unit %d expires on %q, want %q", index+1, one.ExpiresOn.String(), "2026-03-17")
		}
	}
}

// TestTheLayersTheProductionConsumedAreKept proves the rules of the table
// against a real database: the layers the production consumed are kept with
// the quantity that was taken, the unit of that quantity, the grams and the
// money, in the order the FIFO left the stock in.
func TestTheLayersTheProductionConsumedAreKept(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := productionsRepository(t)
	bakery := theBakeryOfTheProductions(t, ctx)
	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	written, err := repo.Record(ctx, theProductionOfTheTest(t, bakery, baked, layers, flour))
	if err != nil {
		t.Fatalf("the production could not be written: %v", err)
	}

	if len(written.Takes) != 1 {
		t.Fatalf("the production took from %d layers, want 1", len(written.Takes))
	}

	take := written.Takes[0]
	if take.LayerID != layers[0].ID {
		t.Errorf("the layer the production took from is %d, want %d", take.LayerID, layers[0].ID)
	}
	if take.Unit != "kg" {
		t.Errorf("the unit of what was taken is %q, want %q", take.Unit, "kg")
	}
	if take.Grams != 500 {
		t.Errorf("the grams that were taken are %v, want 500", take.Grams)
	}
}

// TestTheQuantityThatIsLeftOfTheLayerIsWritten proves the rules of the
// table against a real database: the quantity that is left of the layer the
// production consumed is written, and the unit of the layer travels with
// it: the storage never writes a quantity of another unit than the one the
// layer holds.
func TestTheQuantityThatIsLeftOfTheLayerIsWritten(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := layersRepository(t)
	bakery := theBakeryOfTheProductions(t, ctx)
	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	production := theProductionOfTheTest(t, bakery, baked, layers, flour)

	if len(production.Remaining) != 1 {
		t.Fatalf("the production left %d layers behind, want 1", len(production.Remaining))
	}

	if err := repo.WriteRemaining(ctx, production.Remaining); err != nil {
		t.Fatalf("the quantity that is left could not be written: %v", err)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the layers could not be read: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the business holds %d layers, want 1", len(held))
	}

	if held[0].Left.Quantity != 24.5 {
		t.Errorf("the quantity that is left of the layer is %v, want 24.5", held[0].Left.Quantity)
	}
	if held[0].Left.Unit != "kg" {
		t.Errorf("the unit of what is left of the layer is %q, want %q", held[0].Left.Unit, "kg")
	}
}

// TestTheProductionAndTheLayersItConsumesAreWrittenWhole proves the rule
// of the transaction against a real database: a production whose layers did
// not leave the stock leaves nothing behind, and the storage is as it was.
func TestTheProductionAndTheLayersItConsumesAreWrittenWhole(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	productions := productionsRepository(t)
	bakery := theBakeryOfTheProductions(t, ctx)
	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	production := theProductionOfTheTest(t, bakery, baked, layers, flour)

	// The layer is refused: the storage does not hold that layer, and the
	// production is written whole or not at all.
	stale := production
	stale.Remaining = []purchase.Remaining{
		{
			LayerID: purchase.LayerID(999999),
			Held:    purchase.Measure{Quantity: 25, Unit: "kg"},
			Left:    purchase.Measure{Quantity: 24.5, Unit: "kg"},
		},
	}

	err := unitOfWork(t).Do(ctx, func(ctx context.Context) error {
		if _, err := productions.Record(ctx, stale); err != nil {
			return err
		}

		return layersRepository(t).WriteRemaining(ctx, stale.Remaining)
	})
	if err == nil {
		t.Fatal("the work was accepted, want the refusal of the layer that is not there")
	}

	held, err := layersRepository(t).List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the layers could not be read: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the business holds %d layers, want 1", len(held))
	}

	if held[0].Left.Quantity != 25 {
		t.Errorf("the quantity that is left of the layer is %v, want 25: the work was rolled back",
			held[0].Left.Quantity)
	}

	// The production and the units it made go back with the work that
	// failed: nothing of them is left behind.
	produced, err := productionsRepository(t).List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the productions could not be read: %v", err)
	}

	if len(produced) != 0 {
		t.Errorf("the business holds %d productions, want none: the work was rolled back", len(produced))
	}

	made, err := unitRepository(t).List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(made) != 0 {
		t.Errorf("the business holds %d units, want none: the work was rolled back", len(made))
	}
}

// TestTheProductionIsRefusedWhenTheRecipeIsNotThere proves the rules of
// the table against a real database: a production of a recipe that is not
// there is not written.
func TestTheProductionIsRefusedWhenTheRecipeIsNotThere(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := productionsRepository(t)
	bakery := theBakeryOfTheProductions(t, ctx)
	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	production := theProductionOfTheTest(t, bakery, baked, layers, flour)
	production.RecipeID = recipe.ID(999999)

	_, err := repo.Record(ctx, production)
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}
}
