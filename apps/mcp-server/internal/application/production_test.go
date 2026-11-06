package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The fake of the storage of the productions.
//
// It remembers the production it wrote, and the layers it was told are
// left.
type fakeProductions struct {
	written []recipe.Production
	err     error
}

// Record writes the production and remembers it.
func (f *fakeProductions) Record(_ context.Context, written recipe.Production) (recipe.Production, error) {
	if f.err != nil {
		return recipe.Production{}, f.err
	}

	written.ID = recipe.ProductionID(3)
	f.written = append(f.written, written)

	return written, nil
}

// The fake of the transaction of the storage of the production.
//
// It keeps what the work wrote, and it gives it back when the work fails:
// a transaction that rolls back leaves the storage as it was.
type fakeProductionWork struct {
	productions *fakeProductions
	layers      *fakeLayers
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakeProductionWork) Do(ctx context.Context, work func(context.Context) error) error {
	productions := len(f.productions.written)
	layers := len(f.layers.layers)

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.productions.written = f.productions.written[:productions]
	f.layers.layers = f.layers.layers[:layers]

	return err
}

// theRecipeOfTheProduction is the recipe the owner made: five hundred grams
// of flour, ten units of ninety grams, and three days of shelf life.
func theRecipeOfTheProduction(t *testing.T) recipe.Recipe {
	t.Helper()

	flour, err := recipe.NewIngredient(ingredient.ID(10), 500)
	if err != nil {
		t.Fatalf("the ingredient of the recipe was refused: %v", err)
	}

	built, err := recipe.New(business.ID(77), recipe.Definition{
		Name:        "Pan de molde",
		Ingredients: []recipe.Ingredient{flour},
		Steps: []recipe.Step{
			{Instruction: "Amasar."},
		},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	built.ID = recipe.ID(9)

	return built
}

// theLayersOfTheProduction are the two layers of the stock the production
// consumes: the older one comes first in the FIFO, and it holds only two
// hundred grams of the five hundred the recipe needs.
func theLayersOfTheProduction() []purchase.Layer {
	older := ledger.NewDate(time.Date(2026, time.March, 1, 10, 0, 0, 0, time.UTC))
	newer := ledger.NewDate(time.Date(2026, time.March, 5, 10, 0, 0, 0, time.UTC))

	return []purchase.Layer{
		{
			ID:                 purchase.LayerID(1),
			BusinessID:         business.ID(77),
			PurchaseBusinessID: business.ID(77),
			IngredientID:       ingredient.ID(10),
			Name:               "Harina panadera",
			PurchaseID:         purchase.ID(9),
			Position:           1,
			Day:                older,
			Bought:             purchase.Measure{Quantity: 0.2, Unit: "kg"},
			Left:               purchase.Measure{Quantity: 0.2, Unit: "kg"},
			Cost:               ledger.Amount(400),
		},
		{
			ID:                 purchase.LayerID(2),
			BusinessID:         business.ID(77),
			PurchaseBusinessID: business.ID(77),
			IngredientID:       ingredient.ID(10),
			Name:               "Harina panadera",
			PurchaseID:         purchase.ID(10),
			Position:           1,
			Day:                newer,
			Bought:             purchase.Measure{Quantity: 25, Unit: "kg"},
			Left:               purchase.Measure{Quantity: 25, Unit: "kg"},
			Cost:               ledger.Amount(10000),
		},
	}
}

// theProductionService builds the use case of the productions with the
// fakes of the test.
func theProductionService(
	t *testing.T,
	productions *fakeProductions,
	layers *fakeLayers,
) *application.ProductionService {
	t.Helper()

	return application.NewProductionService(
		&fakeRecipes{held: []recipe.Recipe{theRecipeOfTheProduction(t)}},
		layers,
		theCatalogOfTheTest(t),
		productions,
		&fakeProductionWork{productions: productions, layers: layers},
		theClock(),
	)
}

// TestTheProductionIsWrittenWithTheRecipeTheDayAndTheUnits proves that
// the use case writes the production the owner confirmed: the recipe she
// made, the day she made it, the grams that went in and the units that
// came out.
func TestTheProductionIsWrittenWithTheRecipeTheDayAndTheUnits(t *testing.T) {
	t.Parallel()

	productions := &fakeProductions{}
	layers := &fakeLayers{layers: theLayersOfTheProduction()}
	service := theProductionService(t, productions, layers)

	written, err := service.Record(t.Context(), ports.RecordProduction{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Day:           "2026-03-14",
		Utterance:     "Hice pan de molde el sábado",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the production was refused: %v", err)
	}

	if len(productions.written) != 1 {
		t.Fatalf("the use case wrote %d productions, want 1", len(productions.written))
	}

	if written.RecipeID != recipe.ID(9) {
		t.Errorf("the recipe of the production is %d, want 9", written.RecipeID)
	}
	if written.Day.String() != "2026-03-14" {
		t.Errorf("the day of the production is %q, want %q", written.Day.String(), "2026-03-14")
	}
	if written.GramsIn != 500 {
		t.Errorf("the grams that went in are %v, want 500", written.GramsIn)
	}
	if written.Money != ledger.Amount(520) {
		t.Errorf("the money of the production is %d, want 520", written.Money)
	}
	if len(written.Units) != 10 {
		t.Fatalf("the production made %d units, want 10", len(written.Units))
	}
	if written.Units[0].ExpiresOn.String() != "2026-03-17" {
		t.Errorf("the first unit expires on %q, want %q", written.Units[0].ExpiresOn.String(), "2026-03-17")
	}
	if written.Utterance != "Hice pan de molde el sábado" {
		t.Errorf("the sentence of the owner is %q, want the one she said", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
}

// TestTheProductionConsumesTheLayersInTheOrderOfTheFIFO proves the rule
// of the use case: the production consumes the layers of the stock in the
// order of the FIFO, and the oldest layer is the first one that leaves.
func TestTheProductionConsumesTheLayersInTheOrderOfTheFIFO(t *testing.T) {
	t.Parallel()

	productions := &fakeProductions{}
	layers := &fakeLayers{layers: theLayersOfTheProduction()}
	service := theProductionService(t, productions, layers)

	written, err := service.Record(t.Context(), ports.RecordProduction{
		BusinessID:    business.ID(77),
		RecipeID:      9,
		Day:           "2026-03-14",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the production was refused: %v", err)
	}

	if len(written.Takes) != 2 {
		t.Fatalf("the production took from %d layers, want 2", len(written.Takes))
	}
	if written.Takes[0].LayerID != purchase.LayerID(1) {
		t.Errorf("the first layer the production took from is %d, want 1: the oldest leaves first",
			written.Takes[0].LayerID)
	}
	if written.Takes[1].LayerID != purchase.LayerID(2) {
		t.Errorf("the second layer the production took from is %d, want 2",
			written.Takes[1].LayerID)
	}
}

// TestTheQuantityThatIsLeftOfTheLayersIsWritten proves the rule of the use
// case: the quantity that is left of each layer the production consumed is
// written, and it is the quantity the layer held minus what the production
// took from it.
func TestTheQuantityThatIsLeftOfTheLayersIsWritten(t *testing.T) {
	t.Parallel()

	productions := &fakeProductions{}
	layers := &fakeLayers{layers: theLayersOfTheProduction()}
	service := theProductionService(t, productions, layers)

	if _, err := service.Record(t.Context(), ports.RecordProduction{
		BusinessID:    business.ID(77),
		RecipeID:      9,
		Day:           "2026-03-14",
		ConfirmedWith: "sí",
	}); err != nil {
		t.Fatalf("the production was refused: %v", err)
	}

	if len(layers.layers) != 2 {
		t.Fatalf("the storage holds %d layers, want 2", len(layers.layers))
	}

	if layers.layers[0].Left.Quantity != 0 {
		t.Errorf("the quantity that is left of the first layer is %v, want 0", layers.layers[0].Left.Quantity)
	}
	if layers.layers[1].Left.Quantity != 24.7 {
		t.Errorf("the quantity that is left of the second layer is %v, want 24.7", layers.layers[1].Left.Quantity)
	}
	if layers.layers[1].Left.Unit != "kg" {
		t.Errorf("the unit of what is left of the second layer is %q, want %q", layers.layers[1].Left.Unit, "kg")
	}
}

// TestTheProductionIsRefusedWhenTheStockDoesNotHoldWhatTheRecipeNeeds
// proves the rule of the use case: a production the stock cannot cover is
// refused, and it leaves nothing behind.
func TestTheProductionIsRefusedWhenTheStockDoesNotHoldWhatTheRecipeNeeds(t *testing.T) {
	t.Parallel()

	productions := &fakeProductions{}
	layers := &fakeLayers{layers: []purchase.Layer{theLayersOfTheProduction()[0]}}
	service := theProductionService(t, productions, layers)

	_, err := service.Record(t.Context(), ports.RecordProduction{
		BusinessID:    business.ID(77),
		RecipeID:      9,
		Day:           "2026-03-14",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingStock) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingStock)
	}

	if len(productions.written) != 0 {
		t.Errorf("the use case wrote %d productions, want none", len(productions.written))
	}
}

// TestTheProductionIsRefusedWithoutTheWordOfTheOwner proves the rule of
// the use case: the production is written with the confirmation of the
// owner, and it is refused without her word.
func TestTheProductionIsRefusedWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	productions := &fakeProductions{}
	layers := &fakeLayers{layers: theLayersOfTheProduction()}
	service := theProductionService(t, productions, layers)

	_, err := service.Record(t.Context(), ports.RecordProduction{
		BusinessID: business.ID(77),
		RecipeID:   9,
		Day:        "2026-03-14",
	})
	if !errors.Is(err, recipe.ErrMissingProductionConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingProductionConfirmation)
	}

	if len(productions.written) != 0 {
		t.Errorf("the use case wrote %d productions, want none", len(productions.written))
	}
}

// TestTheProductionIsRefusedWhenTheNameCarriesMoreThanOneRecipe proves the
// rule of the use case: the name of a recipe is not unique inside the
// business, and the book never guesses which recipe the owner made.
func TestTheProductionIsRefusedWhenTheNameCarriesMoreThanOneRecipe(t *testing.T) {
	t.Parallel()

	other := theRecipeOfTheProduction(t)
	other.ID = recipe.ID(10)

	productions := &fakeProductions{}
	layers := &fakeLayers{layers: theLayersOfTheProduction()}
	service := application.NewProductionService(
		&fakeRecipes{held: []recipe.Recipe{theRecipeOfTheProduction(t), other}},
		layers,
		theCatalogOfTheTest(t),
		productions,
		&fakeProductionWork{productions: productions, layers: layers},
		theClock(),
	)

	_, err := service.Record(t.Context(), ports.RecordProduction{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Day:           "2026-03-14",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrAmbiguousRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrAmbiguousRecipe)
	}

	if len(productions.written) != 0 {
		t.Errorf("the use case wrote %d productions, want none", len(productions.written))
	}
}

// TestTheProductionIsRefusedWhenTheBusinessHoldsNoRecipeWithThatName
// proves the rule of the use case: the book says that it does not hold the
// recipe the owner named, and it never writes a production of nothing.
func TestTheProductionIsRefusedWhenTheBusinessHoldsNoRecipeWithThatName(t *testing.T) {
	t.Parallel()

	productions := &fakeProductions{}
	layers := &fakeLayers{layers: theLayersOfTheProduction()}
	service := theProductionService(t, productions, layers)

	_, err := service.Record(t.Context(), ports.RecordProduction{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de yeast",
		Day:           "2026-03-14",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrUnknownRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownRecipe)
	}

	if len(productions.written) != 0 {
		t.Errorf("the use case wrote %d productions, want none", len(productions.written))
	}
}

// TestTheProductionIsRefusedWhenTheRecipeIsOfAnotherBusiness proves the
// rule of the use case: a recipe belongs to the business that owns it, and
// the production of another business is refused.
func TestTheProductionIsRefusedWhenTheRecipeIsOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	other := theRecipeOfTheProduction(t)
	other.BusinessID = business.ID(78)

	productions := &fakeProductions{}
	layers := &fakeLayers{layers: theLayersOfTheProduction()}
	service := application.NewProductionService(
		&fakeRecipes{held: []recipe.Recipe{other}},
		layers,
		theCatalogOfTheTest(t),
		productions,
		&fakeProductionWork{productions: productions, layers: layers},
		theClock(),
	)

	_, err := service.Record(t.Context(), ports.RecordProduction{
		BusinessID:    business.ID(77),
		RecipeID:      9,
		Day:           "2026-03-14",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrWrongBusiness)
	}

	if len(productions.written) != 0 {
		t.Errorf("the use case wrote %d productions, want none", len(productions.written))
	}
}
