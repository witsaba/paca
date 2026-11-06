package application_test

import (
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

// theRecipeOfTheCost is the recipe the owner confirmed: five hundred grams
// of flour, and ten units of ninety grams.
func theRecipeOfTheCost(t *testing.T) recipe.Recipe {
	t.Helper()

	flour, err := recipe.NewIngredient(ingredient.ID(10), 500)
	if err != nil {
		t.Fatalf("the ingredient of the recipe was refused: %v", err)
	}

	built, err := recipe.New(business.ID(77), recipe.Definition{
		Name:          "Pan de molde",
		Ingredients:   []recipe.Ingredient{flour},
		Steps:         []recipe.Step{{Instruction: "Amasar."}},
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

// theLayersOfTheCost are the layers the cost of the batch is derived from.
func theLayersOfTheCost() []purchase.Layer {
	day := ledger.NewDate(time.Date(2026, time.March, 1, 10, 0, 0, 0, time.UTC))

	return []purchase.Layer{
		{
			ID:                 purchase.LayerID(1),
			BusinessID:         business.ID(77),
			PurchaseBusinessID: business.ID(77),
			IngredientID:       ingredient.ID(10),
			Name:               "Harina panadera",
			PurchaseID:         purchase.ID(9),
			Position:           1,
			Day:                day,
			Bought:             purchase.Measure{Quantity: 25, Unit: "kg"},
			Left:               purchase.Measure{Quantity: 25, Unit: "kg"},
			Cost:               ledger.Amount(12000),
		},
	}
}

// theCatalogOfTheCost is the catalog of the names the business buys.
func theCatalogOfTheCost(t *testing.T) *fakeIngredients {
	t.Helper()

	return &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera"),
	}}
}

// TestTheCostOfTheBatchIsDerivedFromTheStock proves that the use case
// reads the recipe, the layers and the catalog, and answers the cost of
// the batch.
func TestTheCostOfTheBatchIsDerivedFromTheStock(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheCost(t)}}
	layers := &fakeLayers{layers: theLayersOfTheCost()}

	service := application.NewBatchCostService(recipes, layers, theCatalogOfTheCost(t))

	costs, err := service.Read(t.Context(), ports.ReadBatchCost{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the cost was refused: %v", err)
	}

	if len(costs) != 1 {
		t.Fatalf("the answer carries %d costs, want 1", len(costs))
	}
	if costs[0].Direct != ledger.Amount(240) {
		t.Errorf("the money of the batch is %d, want 240", costs[0].Direct)
	}
	if costs[0].Name != "Pan de molde" {
		t.Errorf("the name of the recipe is %q, want %q", costs[0].Name, "Pan de molde")
	}
	if !costs[0].Complete {
		t.Errorf("the cost is not complete: %v", costs[0].Missing)
	}
}

// TestTheCostOfTheRecipeTheAgentNames proves that the use case answers the
// cost of the recipe the agent names, and of no other one.
func TestTheCostOfTheRecipeTheAgentNames(t *testing.T) {
	t.Parallel()

	other := theRecipeOfTheCost(t)
	other.ID = recipe.ID(10)
	other.Name = "Pan de ronda"

	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheCost(t), other}}
	layers := &fakeLayers{layers: theLayersOfTheCost()}

	service := application.NewBatchCostService(recipes, layers, theCatalogOfTheCost(t))

	costs, err := service.Read(t.Context(), ports.ReadBatchCost{
		BusinessID: business.ID(77),
		RecipeName: "pan de molde",
	})
	if err != nil {
		t.Fatalf("the cost was refused: %v", err)
	}

	if len(costs) != 1 {
		t.Fatalf("the answer carries %d costs, want 1", len(costs))
	}
	if costs[0].RecipeID != recipe.ID(9) {
		t.Errorf("the recipe is %d, want 9", costs[0].RecipeID)
	}
}

// TestTheCostOfTheRecipeOfAnotherBusinessIsRefused proves that the cost of
// a batch belongs to a business: a recipe of another business is refused.
func TestTheCostOfTheRecipeOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	other := theRecipeOfTheCost(t)
	other.BusinessID = business.ID(78)

	recipes := &fakeRecipes{held: []recipe.Recipe{other}}
	layers := &fakeLayers{layers: theLayersOfTheCost()}

	service := application.NewBatchCostService(recipes, layers, theCatalogOfTheCost(t))

	_, err := service.Read(t.Context(), ports.ReadBatchCost{
		BusinessID: business.ID(77),
		RecipeID:   9,
	})
	if !errors.Is(err, recipe.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrWrongBusiness)
	}
}

// TestTheCostOfARecipeThatIsNotThereIsRefused proves that the use case
// refuses a recipe the storage does not hold.
func TestTheCostOfARecipeThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{}
	layers := &fakeLayers{}

	service := application.NewBatchCostService(recipes, layers, theCatalogOfTheCost(t))

	_, err := service.Read(t.Context(), ports.ReadBatchCost{
		BusinessID: business.ID(77),
		RecipeID:   9,
	})
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}
}

// TestTheCostWithoutADatabaseSaysSo proves that a service without a
// database says so, and it says it in the only capability it has.
func TestTheCostWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewBatchCostService(nil, nil, nil)

	_, err := service.Read(t.Context(), ports.ReadBatchCost{BusinessID: business.ID(77)})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// TestTheCostOfABusinessThatIsNotThereIsRefused proves that the cost of a
// batch belongs to a business: a request that names no business is
// refused.
func TestTheCostOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{}
	layers := &fakeLayers{}

	service := application.NewBatchCostService(recipes, layers, theCatalogOfTheCost(t))

	_, err := service.Read(t.Context(), ports.ReadBatchCost{BusinessID: business.ID(0)})
	if !errors.Is(err, purchase.ErrMissingBusiness) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrMissingBusiness)
	}
}
