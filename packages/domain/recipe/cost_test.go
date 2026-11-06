package recipe_test

import (
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theCatalogOfTheTest is the catalog of the names the business buys.
func theCatalogOfTheTest(t *testing.T) ingredient.Catalog {
	t.Helper()

	flour, err := ingredient.New(business.ID(77), "Harina panadera", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}
	flour.ID = ingredient.ID(10)

	salt, err := ingredient.New(business.ID(77), "Sal", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}
	salt.ID = ingredient.ID(11)

	catalog, err := ingredient.NewCatalog(business.ID(77), []ingredient.Ingredient{flour, salt})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	return catalog
}

// theLayerOfTheTest is a layer of the stock the cost of a batch is derived
// from.
func theLayerOfTheTest(
	id ingredient.ID,
	name string,
	quantity float64,
	unit string,
	cost ledger.Amount,
	day ledger.Date,
	purchaseID purchase.ID,
) purchase.Layer {
	return purchase.Layer{
		ID:                 purchase.LayerID(int(purchaseID)),
		BusinessID:         business.ID(77),
		PurchaseBusinessID: business.ID(77),
		IngredientID:       id,
		Name:               name,
		PurchaseID:         purchaseID,
		Position:           1,
		Day:                day,
		Bought:             purchase.Measure{Quantity: quantity, Unit: unit},
		Left:               purchase.Measure{Quantity: quantity, Unit: unit},
		Cost:               cost,
	}
}

// theDayOfTheCost is the day the layers of the test were bought.
func theDayOfTheCost() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 1, 10, 0, 0, 0, time.UTC))
}

// theRecipeOfTheCost is the recipe the cost is derived from: five hundred
// grams of flour, and ten units of ninety grams.
func theRecipeOfTheCost(t *testing.T) recipe.Recipe {
	t.Helper()

	flour, err := recipe.NewIngredient(ingredient.ID(10), 500)
	if err != nil {
		t.Fatalf("the ingredient of the recipe was refused: %v", err)
	}

	salt, err := recipe.NewIngredient(ingredient.ID(11), 10)
	if err != nil {
		t.Fatalf("the ingredient of the recipe was refused: %v", err)
	}

	built, err := recipe.New(business.ID(77), recipe.Definition{
		Name:        "Pan de molde",
		Ingredients: []recipe.Ingredient{flour, salt},
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

	return built
}

// TestTheCostOfABatchIsDerivedFromTheLayers proves the rule of the domain:
// the cost of a batch is derived from the layers that the recipe consumes,
// and never typed.
//
// The money of what the batch takes is the money of the layer, pro rata
// the quantity that was taken: five hundred grams of a layer of twenty-five
// kilos that cost twelve thousand pesos cost two hundred and forty pesos.
func TestTheCostOfABatchIsDerivedFromTheLayers(t *testing.T) {
	t.Parallel()

	layers := []purchase.Layer{
		theLayerOfTheTest(ingredient.ID(10), "Harina panadera", 25, "kg", 12000,
			theDayOfTheCost(), purchase.ID(9)),
		theLayerOfTheTest(ingredient.ID(11), "Sal", 1, "kg", 3000,
			theDayOfTheCost(), purchase.ID(9)),
	}

	cost, err := recipe.CostOf(theRecipeOfTheCost(t), layers, theCatalogOfTheTest(t), ledger.Amount(0), recipe.OverheadRate{})
	if err != nil {
		t.Fatalf("the cost was refused: %v", err)
	}

	if cost.Direct != ledger.Amount(270) {
		t.Errorf("the money of the batch is %d, want 270", cost.Direct)
	}
	if len(cost.Lines) != 2 {
		t.Fatalf("the cost carries %d lines, want 2", len(cost.Lines))
	}
	if cost.Lines[0].Money != ledger.Amount(240) {
		t.Errorf("the money of the flour is %d, want 240", cost.Lines[0].Money)
	}
	if cost.Lines[0].Name != "Harina panadera" {
		t.Errorf("the name of the line is %q, want %q", cost.Lines[0].Name, "Harina panadera")
	}
	if len(cost.Lines[0].Layers) != 1 {
		t.Fatalf("the line of the flour took from %d layers, want 1", len(cost.Lines[0].Layers))
	}
	if cost.Lines[0].Layers[0].PurchaseID != purchase.ID(9) {
		t.Errorf("the line of the flour took from the purchase %d, want 9", cost.Lines[0].Layers[0].PurchaseID)
	}
	if !cost.Complete {
		t.Errorf("the cost is not complete: %v", cost.Missing)
	}
}

// TestTheCostAnswersPerUnitPerGramAndPerKilogram proves that the cost
// answers the three ways the owner sells: per unit, per gram and per
// kilogram.
//
// The yield is a declared constant: the batch makes ten units of ninety
// grams, and the cost is answered against what it makes.
func TestTheCostAnswersPerUnitPerGramAndPerKilogram(t *testing.T) {
	t.Parallel()

	layers := []purchase.Layer{
		theLayerOfTheTest(ingredient.ID(10), "Harina panadera", 25, "kg", 12000,
			theDayOfTheCost(), purchase.ID(9)),
		theLayerOfTheTest(ingredient.ID(11), "Sal", 1, "kg", 3000,
			theDayOfTheCost(), purchase.ID(9)),
	}

	cost, err := recipe.CostOf(theRecipeOfTheCost(t), layers, theCatalogOfTheTest(t), ledger.Amount(0), recipe.OverheadRate{})
	if err != nil {
		t.Fatalf("the cost was refused: %v", err)
	}

	// The batch makes ten units of ninety grams: nine hundred grams.
	if cost.PerUnit != 27 {
		t.Errorf("the money per unit is %v, want 27", cost.PerUnit)
	}
	if cost.PerGram < 0.3 || cost.PerGram > 0.30001 {
		t.Errorf("the money per gram is %v, want 0.3", cost.PerGram)
	}
	if cost.PerKilogram < 300 || cost.PerKilogram > 300.01 {
		t.Errorf("the money per kilogram is %v, want 300", cost.PerKilogram)
	}
}

// TestTheCostIsNotCompleteWhenTheStockDoesNotHoldTheRecipe proves that the
// cost never invents what the stock does not hold: the answer says what is
// missing.
func TestTheCostIsNotCompleteWhenTheStockDoesNotHoldTheRecipe(t *testing.T) {
	t.Parallel()

	layers := []purchase.Layer{
		theLayerOfTheTest(ingredient.ID(10), "Harina panadera", 1, "kg", 12000,
			theDayOfTheCost(), purchase.ID(9)),
	}

	cost, err := recipe.CostOf(theRecipeOfTheCost(t), layers, theCatalogOfTheTest(t), ledger.Amount(0), recipe.OverheadRate{})
	if err != nil {
		t.Fatalf("the cost was refused: %v", err)
	}

	if cost.Complete {
		t.Fatal("the cost is complete, want the news that the stock does not hold the recipe")
	}
	if cost.Lines[0].Missing != 0 {
		t.Errorf("the flour is missing %v grams, want none", cost.Lines[0].Missing)
	}
	if cost.Lines[1].Missing != 10 {
		t.Errorf("the salt is missing %v grams, want 10", cost.Lines[1].Missing)
	}
	if len(cost.Missing) == 0 {
		t.Error("the cost does not say what it still needs")
	}
}

// TestTheCostOfAnIngredientTheCatalogDoesNotHoldIsNotDerivable proves that
// the recipe never carries a second copy of the flour: an ingredient the
// catalog of the business does not hold has no money in the answer.
func TestTheCostOfAnIngredientTheCatalogDoesNotHoldIsNotDerivable(t *testing.T) {
	t.Parallel()

	built, err := recipe.New(business.ID(77), recipe.Definition{
		Name: "Pan de molde",
		Ingredients: []recipe.Ingredient{
			{IngredientID: ingredient.ID(10), Grams: 500},
			{IngredientID: ingredient.ID(99), Grams: 10},
		},
		Steps:         []recipe.Step{{Instruction: "Amasar."}},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	cost, err := recipe.CostOf(built, nil, theCatalogOfTheTest(t), ledger.Amount(0), recipe.OverheadRate{})
	if err != nil {
		t.Fatalf("the cost was refused: %v", err)
	}

	if cost.Complete {
		t.Fatal("the cost is complete, want the news that an ingredient is not one of the catalog")
	}
	if len(cost.Lines) != 2 {
		t.Fatalf("the cost carries %d lines, want 2", len(cost.Lines))
	}
	if !cost.Lines[1].Unknown {
		t.Error("the money of the unknown ingredient was derived, want the news that it is not derivable")
	}
}

// TestTheRecipesAreFoundByTheNameTheOwnerReads proves the rule of the
// domain: the name is compared the way a person reads it, trimmed and
// without case.
func TestTheRecipesAreFoundByTheNameTheOwnerReads(t *testing.T) {
	t.Parallel()

	pan := theRecipeOfTheCost(t)
	pan.ID = recipe.ID(9)

	other := theRecipeOfTheCost(t)
	other.ID = recipe.ID(10)
	other.Name = "Pan de ronda"

	found := recipe.Named([]recipe.Recipe{pan, other}, " pan de molde ")

	if len(found) != 1 {
		t.Fatalf("the catalog answered %d recipes, want 1", len(found))
	}
	if found[0].ID != recipe.ID(9) {
		t.Errorf("the recipe is %d, want 9", found[0].ID)
	}
}

// TestTheCostOfAnIngredientTheRecipeNamesTwiceIsTheCostOfTheGrams proves
// the rule of the domain: an ingredient the recipe names twice is one
// ingredient of the batch, and the cost of the batch never pays twice for
// the same gram.
func TestTheCostOfAnIngredientTheRecipeNamesTwiceIsTheCostOfTheGrams(t *testing.T) {
	t.Parallel()

	built, err := recipe.New(business.ID(77), recipe.Definition{
		Name: "Pan de molde",
		Ingredients: []recipe.Ingredient{
			{IngredientID: ingredient.ID(10), Grams: 500},
			{IngredientID: ingredient.ID(10), Grams: 500},
		},
		Steps:         []recipe.Step{{Instruction: "Amasar."}},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	layers := []purchase.Layer{
		theLayerOfTheTest(ingredient.ID(10), "Harina panadera", 1, "kg", 1200,
			theDayOfTheCost(), purchase.ID(9)),
	}

	cost, err := recipe.CostOf(built, layers, theCatalogOfTheTest(t), ledger.Amount(0), recipe.OverheadRate{})
	if err != nil {
		t.Fatalf("the cost was refused: %v", err)
	}

	if len(cost.Lines) != 1 {
		t.Fatalf("the cost carries %d lines, want the one of the flour", len(cost.Lines))
	}
	if cost.Lines[0].Grams != 1000 {
		t.Errorf("the batch needs %v grams of the flour, want 1000", cost.Lines[0].Grams)
	}
	if cost.Direct != ledger.Amount(1200) {
		t.Errorf("the money of the batch is %d, want 1200", cost.Direct)
	}
	if cost.Lines[0].Missing != 0 {
		t.Errorf("the flour is missing %v grams, want none", cost.Lines[0].Missing)
	}
}

// TestTheCostCarriesTheLaborBesideTheDirectCost proves the rule of the
// domain: the layers of the cost of a batch are never hidden inside one
// another.
//
// The direct cost is the money of the ingredients, and the labor is the
// money of the hours of the steps: the two are carried apart, and the
// labor is never added to the direct cost.
func TestTheCostCarriesTheLaborBesideTheDirectCost(t *testing.T) {
	t.Parallel()

	layers := []purchase.Layer{
		theLayerOfTheTest(ingredient.ID(10), "Harina panadera", 25, "kg", 12000,
			theDayOfTheCost(), purchase.ID(9)),
	}

	flour, err := recipe.NewIngredient(ingredient.ID(10), 500)
	if err != nil {
		t.Fatalf("the ingredient of the recipe was refused: %v", err)
	}

	hornear, err := recipe.NewStep("Hornear.", "", 45, 220, recipe.LaborOven)
	if err != nil {
		t.Fatalf("the step was refused: %v", err)
	}

	built, err := recipe.New(business.ID(77), recipe.Definition{
		Name:          "Pan de molde",
		Ingredients:   []recipe.Ingredient{flour},
		Steps:         []recipe.Step{hornear},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	cost, err := recipe.CostOf(built, layers, theCatalogOfTheTest(t), ledger.Amount(20000), recipe.OverheadRate{})
	if err != nil {
		t.Fatalf("the cost was refused: %v", err)
	}

	if cost.Direct != ledger.Amount(240) {
		t.Errorf("the direct cost of the batch is %d, want 240", cost.Direct)
	}
	if cost.Labor.Money != ledger.Amount(15000) {
		t.Errorf("the money of the labor is %d, want 15000", cost.Labor.Money)
	}
	if cost.Labor.Unknown {
		t.Error("the money of the labor is not derivable, want the money of the hours")
	}
	if cost.Labor.Hours.Oven != 0.75 {
		t.Errorf("the hours of the oven are %v, want 0.75", cost.Labor.Hours.Oven)
	}
	if cost.Labor.Value != ledger.Amount(20000) {
		t.Errorf("the value of the hour is %d, want 20000", cost.Labor.Value)
	}
}

// TestTheCostCarriesTheOverheadBesideTheDirectCostAndTheLabor proves the
// rule of the domain: the three layers of the cost of a batch are never
// hidden inside one another.
//
// The direct cost is the money of the ingredients, the labor is the money
// of the hours of the steps and the overhead is the money of the hours of
// the oven of the steps: the three are carried apart, and none of them is
// added to the direct cost.
func TestTheCostCarriesTheOverheadBesideTheDirectCostAndTheLabor(t *testing.T) {
	t.Parallel()

	layers := []purchase.Layer{
		theLayerOfTheTest(ingredient.ID(10), "Harina panadera", 25, "kg", 12000,
			theDayOfTheCost(), purchase.ID(9)),
	}

	flour, err := recipe.NewIngredient(ingredient.ID(10), 500)
	if err != nil {
		t.Fatalf("the ingredient of the recipe was refused: %v", err)
	}

	hornear, err := recipe.NewStep("Hornear.", "", 45, 220, recipe.LaborOven)
	if err != nil {
		t.Fatalf("the step was refused: %v", err)
	}

	built, err := recipe.New(business.ID(77), recipe.Definition{
		Name:          "Pan de molde",
		Ingredients:   []recipe.Ingredient{flour},
		Steps:         []recipe.Step{hornear},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	cost, err := recipe.CostOf(built, layers, theCatalogOfTheTest(t),
		ledger.Amount(20000), recipe.OverheadRate{MoneyPerOvenHour: 1500})
	if err != nil {
		t.Fatalf("the cost was refused: %v", err)
	}

	if cost.Direct != ledger.Amount(240) {
		t.Errorf("the direct cost of the batch is %d, want 240", cost.Direct)
	}
	if cost.Labor.Money != ledger.Amount(15000) {
		t.Errorf("the money of the labor is %d, want 15000", cost.Labor.Money)
	}
	if cost.Overhead.Money != ledger.Amount(1125) {
		t.Errorf("the money of the overhead is %d, want 1125", cost.Overhead.Money)
	}
	if cost.Overhead.Unknown {
		t.Error("the money of the overhead is not derivable, want the money of the hours of the oven")
	}
	if cost.Overhead.OvenHours != 0.75 {
		t.Errorf("the hours of the oven are %v, want 0.75", cost.Overhead.OvenHours)
	}
	if cost.Overhead.Rate != 1500 {
		t.Errorf("the money of the hour of the oven is %v, want 1500", cost.Overhead.Rate)
	}
}
