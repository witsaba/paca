package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The fake of the storage of the recipes.
//
// It remembers the recipe it wrote: the use case writes one, and the test
// proves what it wrote.
type fakeRecipes struct {
	written []recipe.Recipe
	held    []recipe.Recipe
	err     error
	found   recipe.ID
	listed  business.ID
}

// Record writes the recipe and remembers it.
func (f *fakeRecipes) Record(_ context.Context, written recipe.Recipe) (recipe.Recipe, error) {
	if f.err != nil {
		return recipe.Recipe{}, f.err
	}

	written.ID = recipe.ID(9)
	f.written = append(f.written, written)

	return written, nil
}

// Find answers the recipe that carries that identity, and remembers it.
func (f *fakeRecipes) Find(_ context.Context, id recipe.ID) (recipe.Recipe, error) {
	f.found = id

	if f.err != nil {
		return recipe.Recipe{}, f.err
	}

	for _, one := range f.held {
		if one.ID == id {
			return one, nil
		}
	}

	return recipe.Recipe{}, recipe.ErrNotFound
}

// List answers the recipes of the business, and remembers it.
func (f *fakeRecipes) List(_ context.Context, own business.ID) ([]recipe.Recipe, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]recipe.Recipe, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// theCatalogOfTheTest answers the catalog the tests of the recipe use.
func theCatalogOfTheTest(t *testing.T) *fakeIngredients {
	t.Helper()

	return &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera"),
		anIngredientOfTheCatalog(t, 20, "Sal"),
	}}
}

// theRecipeOfTheTest is the recipe the owner confirms.
func theRecipeOfTheTest() ports.RecordRecipe {
	return ports.RecordRecipe{
		BusinessID: business.ID(77),
		Name:       "Pan de molde",
		Ingredients: []ports.RecipeIngredient{
			{IngredientID: 10, Grams: 500},
			{IngredientID: 20, Grams: 10},
		},
		Steps: []ports.RecipeStep{
			{Instruction: "Mezclar la harina con el agua y amasar.", Cue: "la masa que holds la forma", Minutes: 20, Temperature: 24},
			{Instruction: "Hornear.", Minutes: 45, Temperature: 220},
		},
		YieldUnits:    10,
		YieldGrams:    90,
		ShelfLifeDays: 3,
		Utterance:     "El pan de molde lleva 500 gramos de harina",
		ConfirmedWith: "sí",
	}
}

// TestTheRecipeIsWrittenWithTheConfirmationOfTheOwner proves that the use
// case writes the recipe the owner confirmed, with its ingredients, its
// steps, its yield and its shelf life.
func TestTheRecipeIsWrittenWithTheConfirmationOfTheOwner(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{}
	service := application.NewRecipeService(recipes, theCatalogOfTheTest(t))

	written, err := service.Record(t.Context(), theRecipeOfTheTest())
	if err != nil {
		t.Fatalf("the recipe could not be written: %v", err)
	}

	if len(recipes.written) != 1 {
		t.Fatalf("the storage holds %d recipes, want 1", len(recipes.written))
	}

	kept := recipes.written[0]
	if kept.ID != recipe.ID(9) {
		t.Errorf("the identity of the recipe is %d, want 9", kept.ID)
	}
	if kept.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", kept.BusinessID)
	}
	if kept.Name != "Pan de molde" {
		t.Errorf("the name is %q, want %q", kept.Name, "Pan de molde")
	}
	if len(kept.Ingredients) != 2 {
		t.Fatalf("the recipe carries %d ingredients, want 2", len(kept.Ingredients))
	}
	if kept.Ingredients[0].IngredientID != ingredient.ID(10) {
		t.Errorf("the first ingredient is %d, want 10", kept.Ingredients[0].IngredientID)
	}
	if kept.Ingredients[0].Grams != 500 {
		t.Errorf("the grams of the first ingredient are %v, want 500", kept.Ingredients[0].Grams)
	}
	if len(kept.Steps) != 2 {
		t.Fatalf("the recipe carries %d steps, want 2", len(kept.Steps))
	}
	if kept.Steps[0].Instruction != "Mezclar la harina con el agua y amasar." {
		t.Errorf("the first step is %q, want the one of the owner", kept.Steps[0].Instruction)
	}
	if kept.Steps[0].Cue != "la masa que holds la forma" {
		t.Errorf("the cue of the first step is %q, want the one of the owner", kept.Steps[0].Cue)
	}
	if kept.Steps[1].Minutes != 45 || kept.Steps[1].Temperature != 220 {
		t.Errorf("the second step is %+v, want 45 minutes at 220 degrees", kept.Steps[1])
	}
	if kept.Yield.Units != 10 || kept.Yield.Grams != 90 {
		t.Errorf("the yield is %+v, want 10 units of 90 grams", kept.Yield)
	}
	if kept.ShelfLifeDays != 3 {
		t.Errorf("the shelf life is %d days, want 3", kept.ShelfLifeDays)
	}
	if kept.Utterance != "El pan de molde lleva 500 gramos de harina" {
		t.Errorf("the sentence is %q, want the one of the owner", kept.Utterance)
	}
	if kept.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", kept.ConfirmedWith, "sí")
	}
	if written.ID != recipe.ID(9) {
		t.Errorf("the identity of the answer is %d, want 9", written.ID)
	}
}

// TestTheRecipeWithoutANameIsRefusedAndNothingIsWritten proves that the
// rule of the domain holds: a recipe that is not whole is refused, and a
// refused recipe leaves nothing behind.
func TestTheRecipeWithoutANameIsRefusedAndNothingIsWritten(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{}
	service := application.NewRecipeService(recipes, theCatalogOfTheTest(t))

	request := theRecipeOfTheTest()
	request.Name = "  "

	if _, err := service.Record(t.Context(), request); !errors.Is(err, recipe.ErrMissingName) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingName)
	}
	if len(recipes.written) != 0 {
		t.Errorf("the storage holds %d recipes, want none", len(recipes.written))
	}
}

// TestTheRecipeWithoutAYieldIsRefusedAndNothingIsWritten proves that the
// rule of the domain holds: the yield is a declared constant, and a recipe
// without it says nothing about what it makes.
func TestTheRecipeWithoutAYieldIsRefusedAndNothingIsWritten(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{}
	service := application.NewRecipeService(recipes, theCatalogOfTheTest(t))

	request := theRecipeOfTheTest()
	request.YieldUnits = 0

	if _, err := service.Record(t.Context(), request); !errors.Is(err, recipe.ErrInvalidYield) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidYield)
	}
	if len(recipes.written) != 0 {
		t.Errorf("the storage holds %d recipes, want none", len(recipes.written))
	}
}

// TestTheIngredientWithoutItsGramsIsRefusedAndNothingIsWritten proves that
// the rule of the domain holds: an ingredient of a recipe carries the grams
// of it.
func TestTheIngredientWithoutItsGramsIsRefusedAndNothingIsWritten(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{}
	service := application.NewRecipeService(recipes, theCatalogOfTheTest(t))

	request := theRecipeOfTheTest()
	request.Ingredients[0].Grams = 0

	if _, err := service.Record(t.Context(), request); !errors.Is(err, recipe.ErrMissingGrams) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingGrams)
	}
	if len(recipes.written) != 0 {
		t.Errorf("the storage holds %d recipes, want none", len(recipes.written))
	}
}

// TestTheStepWithoutItsInstructionIsRefusedAndNothingIsWritten proves that
// the rule of the domain holds: a step without its instruction teaches
// nothing.
func TestTheStepWithoutItsInstructionIsRefusedAndNothingIsWritten(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{}
	service := application.NewRecipeService(recipes, theCatalogOfTheTest(t))

	request := theRecipeOfTheTest()
	request.Steps[0].Instruction = ""

	if _, err := service.Record(t.Context(), request); !errors.Is(err, recipe.ErrMissingInstruction) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingInstruction)
	}
	if len(recipes.written) != 0 {
		t.Errorf("the storage holds %d recipes, want none", len(recipes.written))
	}
}

// TestTheIngredientTheCatalogDoesNotHoldIsRefused proves that the rule of
// the domain holds: the ingredient of a recipe is the ingredient of the
// catalog, and a recipe never carries a second copy of the flour.
func TestTheIngredientTheCatalogDoesNotHoldIsRefused(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{}
	service := application.NewRecipeService(recipes, theCatalogOfTheTest(t))

	request := theRecipeOfTheTest()
	request.Ingredients[0].IngredientID = 999

	if _, err := service.Record(t.Context(), request); !errors.Is(err, ingredient.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrNotFound)
	}
	if len(recipes.written) != 0 {
		t.Errorf("the storage holds %d recipes, want none", len(recipes.written))
	}
}

// TestTheRecipeWithoutADatabaseIsRefused proves that the use case says that
// the database is not there, and that it writes nothing.
func TestTheRecipeWithoutADatabaseIsRefused(t *testing.T) {
	t.Parallel()

	service := application.NewRecipeService(nil, theCatalogOfTheTest(t))

	if _, err := service.Record(t.Context(), theRecipeOfTheTest()); !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
