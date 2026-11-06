package recipe_test

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theDayOfTheProduction is the day the production of the test was made.
func theDayOfTheProduction() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 14, 8, 0, 0, 0, time.UTC))
}

// theRecipeOfTheProduction is the recipe the production was made of: five
// hundred grams of flour, ten units of ninety grams, and three days of
// shelf life.
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

// theDefinitionOfTheProduction is what the owner said about the production.
func theDefinitionOfTheProduction() recipe.ProductionDefinition {
	return recipe.ProductionDefinition{
		Day:           theDayOfTheProduction(),
		Utterance:     "Hice pan de molde el sábado",
		ConfirmedWith: "sí",
	}
}

// TestTheProductionCarriesTheRecipeTheDayTheGramsAndTheUnits proves the
// rule of the domain: the production carries the recipe it was made of, the
// day it was made, the grams that went in and the units that came out.
func TestTheProductionCarriesTheRecipeTheDayTheGramsAndTheUnits(t *testing.T) {
	t.Parallel()

	production, err := recipe.ProductionOf(
		theRecipeOfTheProduction(t),
		theLayersOfTheProduction(),
		theCatalogOfTheTest(t),
		theDefinitionOfTheProduction(),
	)
	if err != nil {
		t.Fatalf("the production was refused: %v", err)
	}

	if production.RecipeID != recipe.ID(9) {
		t.Errorf("the recipe of the production is %d, want 9", production.RecipeID)
	}
	if production.RecipeName != "Pan de molde" {
		t.Errorf("the name of the recipe is %q, want %q", production.RecipeName, "Pan de molde")
	}
	if production.BusinessID != business.ID(77) {
		t.Errorf("the business of the production is %d, want 77", production.BusinessID)
	}
	if production.Day.String() != "2026-03-14" {
		t.Errorf("the day of the production is %q, want %q", production.Day.String(), "2026-03-14")
	}
	if production.GramsIn != 500 {
		t.Errorf("the grams that went in are %v, want 500", production.GramsIn)
	}
	if len(production.Units) != 10 {
		t.Fatalf("the production made %d units, want 10", len(production.Units))
	}
	if production.Units[0].Grams != 90 {
		t.Errorf("the weight of a unit is %v, want 90", production.Units[0].Grams)
	}
	if production.Utterance != "Hice pan de molde el sábado" {
		t.Errorf("the sentence of the owner is %q, want the one she said", production.Utterance)
	}
	if production.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", production.ConfirmedWith, "sí")
	}
}

// TestTheProductionConsumesTheLayersInTheOrderOfTheFIFO proves the rule
// of the domain: the recipe consumes the layers of the stock in the order
// of the FIFO, and the oldest layer is the first one that leaves.
func TestTheProductionConsumesTheLayersInTheOrderOfTheFIFO(t *testing.T) {
	t.Parallel()

	production, err := recipe.ProductionOf(
		theRecipeOfTheProduction(t),
		theLayersOfTheProduction(),
		theCatalogOfTheTest(t),
		theDefinitionOfTheProduction(),
	)
	if err != nil {
		t.Fatalf("the production was refused: %v", err)
	}

	if len(production.Takes) != 2 {
		t.Fatalf("the production took from %d layers, want 2", len(production.Takes))
	}
	if production.Takes[0].LayerID != purchase.LayerID(1) {
		t.Errorf("the first layer the production took from is %d, want 1: the oldest leaves first",
			production.Takes[0].LayerID)
	}
	if production.Takes[0].Quantity != 0.2 {
		t.Errorf("the quantity taken from the oldest layer is %v, want 0.2", production.Takes[0].Quantity)
	}
	if production.Takes[1].LayerID != purchase.LayerID(2) {
		t.Errorf("the second layer the production took from is %d, want 2", production.Takes[1].LayerID)
	}
	if production.Takes[1].Quantity != 0.3 {
		t.Errorf("the quantity taken from the second layer is %v, want 0.3", production.Takes[1].Quantity)
	}
	if production.Takes[1].Grams != 300 {
		t.Errorf("the grams taken from the second layer are %v, want 300", production.Takes[1].Grams)
	}
}

// TestTheMoneyOfTheProductionIsTheMoneyOfTheLayers proves the rule of the
// domain: the money of the production is the money of the layers it
// consumed, pro rata the quantity that was taken.
func TestTheMoneyOfTheProductionIsTheMoneyOfTheLayers(t *testing.T) {
	t.Parallel()

	production, err := recipe.ProductionOf(
		theRecipeOfTheProduction(t),
		theLayersOfTheProduction(),
		theCatalogOfTheTest(t),
		theDefinitionOfTheProduction(),
	)
	if err != nil {
		t.Fatalf("the production was refused: %v", err)
	}

	// Two hundred grams of a layer of two hundred grams that cost four
	// hundred pesos, and three hundred grams of a layer of twenty-five
	// kilos that cost ten thousand pesos: four hundred pesos and a hundred
	// and twenty pesos.
	if production.Money != ledger.Amount(520) {
		t.Errorf("the money of the production is %d, want 520", production.Money)
	}
}

// TestEveryUnitOfTheProductionCarriesTheDayItExpires proves the rule of
// the domain: every unit the production made carries the day it expires,
// and the day is the day of the production plus the shelf life of the
// recipe.
func TestEveryUnitOfTheProductionCarriesTheDayItExpires(t *testing.T) {
	t.Parallel()

	production, err := recipe.ProductionOf(
		theRecipeOfTheProduction(t),
		theLayersOfTheProduction(),
		theCatalogOfTheTest(t),
		theDefinitionOfTheProduction(),
	)
	if err != nil {
		t.Fatalf("the production was refused: %v", err)
	}

	for index, one := range production.Units {
		if one.ExpiresOn.String() != "2026-03-17" {
			t.Errorf("the unit %d expires on %q, want %q", index+1, one.ExpiresOn.String(), "2026-03-17")
		}
		if one.Position != index+1 {
			t.Errorf("the unit %d is the unit %d of the production", one.Position, index+1)
		}
	}
}

// TestTheProductionIsRefusedWhenTheStockDoesNotHoldWhatTheRecipeNeeds
// proves the rule of the domain: a production that cannot be baked from
// what the business holds is not a production.
func TestTheProductionIsRefusedWhenTheStockDoesNotHoldWhatTheRecipeNeeds(t *testing.T) {
	t.Parallel()

	layers := []purchase.Layer{
		{
			ID:                 purchase.LayerID(1),
			BusinessID:         business.ID(77),
			PurchaseBusinessID: business.ID(77),
			IngredientID:       ingredient.ID(10),
			Name:               "Harina panadera",
			PurchaseID:         purchase.ID(9),
			Position:           1,
			Day:                theDayOfTheCost(),
			Bought:             purchase.Measure{Quantity: 1, Unit: "kg"},
			Left:               purchase.Measure{Quantity: 0.2, Unit: "kg"},
			Cost:               ledger.Amount(1200),
		},
	}

	_, err := recipe.ProductionOf(
		theRecipeOfTheProduction(t),
		layers,
		theCatalogOfTheTest(t),
		theDefinitionOfTheProduction(),
	)
	if !errors.Is(err, recipe.ErrMissingStock) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingStock)
	}
}

// TestTheProductionIsRefusedWhenTheStockHoldsAnIngredientInAUnitTheBook
// NeverConverts proves the rule of the domain: the grams that went in are
// never assumed, and a stock the book cannot read into grams cannot produce
// a production.
func TestTheProductionIsRefusedWhenTheStockHoldsAnIngredientInAUnitTheBookNeverConverts(t *testing.T) {
	t.Parallel()

	layers := []purchase.Layer{
		{
			ID:                 purchase.LayerID(1),
			BusinessID:         business.ID(77),
			PurchaseBusinessID: business.ID(77),
			IngredientID:       ingredient.ID(10),
			Name:               "Harina panadera",
			PurchaseID:         purchase.ID(9),
			Position:           1,
			Day:                theDayOfTheCost(),
			Bought:             purchase.Measure{Quantity: 10, Unit: "unidad"},
			Left:               purchase.Measure{Quantity: 10, Unit: "unidad"},
			Cost:               ledger.Amount(1200),
		},
	}

	_, err := recipe.ProductionOf(
		theRecipeOfTheProduction(t),
		layers,
		theCatalogOfTheTest(t),
		theDefinitionOfTheProduction(),
	)
	if !errors.Is(err, recipe.ErrUnknownStock) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownStock)
	}
}

// TestTheProductionIsRefusedWithoutADay proves the rule of the domain: a
// production of no day is a production of no time.
func TestTheProductionIsRefusedWithoutADay(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheProduction()
	definition.Day = ledger.Date{}

	_, err := recipe.ProductionOf(
		theRecipeOfTheProduction(t),
		theLayersOfTheProduction(),
		theCatalogOfTheTest(t),
		definition,
	)
	if !errors.Is(err, recipe.ErrMissingProductionDay) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingProductionDay)
	}
}

// TestTheProductionIsRefusedWithoutTheWordOfTheOwner proves the rule of
// the domain: the production is written with the confirmation of the owner,
// and the word she used is kept with the record.
func TestTheProductionIsRefusedWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheProduction()
	definition.ConfirmedWith = ""

	_, err := recipe.ProductionOf(
		theRecipeOfTheProduction(t),
		theLayersOfTheProduction(),
		theCatalogOfTheTest(t),
		definition,
	)
	if !errors.Is(err, recipe.ErrMissingProductionConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingProductionConfirmation)
	}
}

// TestTheProductionIsRefusedWhenTheRecipeNamesAnIngredientTheCatalogDoes
// NotHold proves the rule of the domain: the layers the production consumes
// are the layers of the ingredients of the catalog, and an ingredient the
// catalog does not hold has no layer to take from.
func TestTheProductionIsRefusedWhenTheRecipeNamesAnIngredientTheCatalogDoesNotHold(t *testing.T) {
	t.Parallel()

	catalog, err := ingredient.NewCatalog(business.ID(77), []ingredient.Ingredient{})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	_, err = recipe.ProductionOf(
		theRecipeOfTheProduction(t),
		theLayersOfTheProduction(),
		catalog,
		theDefinitionOfTheProduction(),
	)
	if !errors.Is(err, recipe.ErrUnknownIngredient) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownIngredient)
	}
}

// TestTheRecipeNamedAnswersTheOneRecipeTheOwnerNamed proves the rule of
// the domain: the name of a recipe is not unique inside the business, and a
// production names one recipe: the book never guesses which one.
func TestTheRecipeNamedAnswersTheOneRecipeTheOwnerNamed(t *testing.T) {
	t.Parallel()

	other := theRecipeOfTheProduction(t)
	other.ID = recipe.ID(10)

	found, err := recipe.TheRecipeNamed([]recipe.Recipe{other}, "Pan de molde")
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}
	if found.ID != recipe.ID(10) {
		t.Errorf("the recipe is %d, want 10", found.ID)
	}

	_, err = recipe.TheRecipeNamed([]recipe.Recipe{other, theRecipeOfTheProduction(t)}, "Pan de molde")
	if !errors.Is(err, recipe.ErrAmbiguousRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrAmbiguousRecipe)
	}

	_, err = recipe.TheRecipeNamed([]recipe.Recipe{other}, "Pan de yeast")
	if !errors.Is(err, recipe.ErrUnknownRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownRecipe)
	}
}
