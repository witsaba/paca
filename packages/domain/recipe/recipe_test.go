package recipe_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theIngredientOfTheTest is the ingredient of the catalog the recipe names.
func theIngredientOfTheTest() recipe.Ingredient {
	flour, err := recipe.NewIngredient(ingredient.ID(10), 500)
	if err != nil {
		panic(err)
	}

	return flour
}

// theDefinitionOfTheTest is the recipe the owner describes.
func theDefinitionOfTheTest() recipe.Definition {
	return recipe.Definition{
		Name: "Pan de molde",
		Ingredients: []recipe.Ingredient{
			theIngredientOfTheTest(),
		},
		Steps: []recipe.Step{
			{
				Instruction: "Mezclar la harina con el agua y amasar.",
				Cue:         "la masa que holds la forma",
				Minutes:     20,
				Temperature: 24,
			},
			{
				Instruction: "Hornear.",
				Minutes:     45,
				Temperature: 220,
			},
		},
		Yield: recipe.Yield{
			Units: 10,
			Grams: 90,
		},
		ShelfLifeDays: 3,
		Utterance:     "El pan de molde lleva 500 gramos de harina y se hornea a 220 grados",
		ConfirmedWith: "sí",
	}
}

// TestTheRecipeIsWhole proves that the recipe carries what the guide
// carries: the name the owner reads, the ingredients with their grams, the
// steps with their times, their temperatures and their cues, the yield and
// the shelf life of what it makes.
func TestTheRecipeIsWhole(t *testing.T) {
	t.Parallel()

	built, err := recipe.New(business.ID(77), theDefinitionOfTheTest())
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	if built.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", built.BusinessID)
	}
	if built.Name != "Pan de molde" {
		t.Errorf("the name is %q, want %q", built.Name, "Pan de molde")
	}
	if len(built.Ingredients) != 1 {
		t.Fatalf("the recipe carries %d ingredients, want 1", len(built.Ingredients))
	}
	if built.Ingredients[0].IngredientID != ingredient.ID(10) {
		t.Errorf("the ingredient is %d, want 10", built.Ingredients[0].IngredientID)
	}
	if built.Ingredients[0].Grams != 500 {
		t.Errorf("the grams of the ingredient are %v, want 500", built.Ingredients[0].Grams)
	}
	if built.Yield.Units != 10 {
		t.Errorf("the yield is %d units, want 10", built.Yield.Units)
	}
	if built.Yield.Grams != 90 {
		t.Errorf("the weight of the unit is %v grams, want 90", built.Yield.Grams)
	}
	if built.ShelfLifeDays != 3 {
		t.Errorf("the shelf life is %d days, want 3", built.ShelfLifeDays)
	}
	if built.Utterance != "El pan de molde lleva 500 gramos de harina y se hornea a 220 grados" {
		t.Errorf("the sentence is %q, want the one of the owner", built.Utterance)
	}
	if built.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", built.ConfirmedWith, "sí")
	}
}

// TestTheStepsKeepTheirOrder proves that the steps of a recipe are read in
// the order the owner said them: a guide whose steps move is not a guide.
func TestTheStepsKeepTheirOrder(t *testing.T) {
	t.Parallel()

	built, err := recipe.New(business.ID(77), theDefinitionOfTheTest())
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	if len(built.Steps) != 2 {
		t.Fatalf("the recipe carries %d steps, want 2", len(built.Steps))
	}
	if built.Steps[0].Instruction != "Mezclar la harina con el agua y amasar." {
		t.Errorf("the first step is %q, want the one of the owner", built.Steps[0].Instruction)
	}
	if built.Steps[0].Cue != "la masa que holds la forma" {
		t.Errorf("the cue of the first step is %q, want the one of the owner", built.Steps[0].Cue)
	}
	if built.Steps[0].Minutes != 20 {
		t.Errorf("the time of the first step is %d minutes, want 20", built.Steps[0].Minutes)
	}
	if built.Steps[0].Temperature != 24 {
		t.Errorf("the temperature of the first step is %v, want 24", built.Steps[0].Temperature)
	}
	if built.Steps[1].Instruction != "Hornear." {
		t.Errorf("the second step is %q, want %q", built.Steps[1].Instruction, "Hornear.")
	}
}

// TestTheStepWithoutATimeNorATemperatureCarriesNeither proves that a step
// carries its time and its temperature only when the owner named them: a
// number PACA invented is a lie in the guide.
func TestTheStepWithoutATimeNorATemperatureCarriesNeither(t *testing.T) {
	t.Parallel()

	step, err := recipe.NewStep("Reposo de la masa", "", 0, 0)
	if err != nil {
		t.Fatalf("the step was refused: %v", err)
	}

	if step.Minutes != 0 {
		t.Errorf("the time of the step is %d minutes, want none", step.Minutes)
	}
	if step.Temperature != 0 {
		t.Errorf("the temperature of the step is %v, want none", step.Temperature)
	}
	if step.Cue != "" {
		t.Errorf("the cue of the step is %q, want none", step.Cue)
	}
}

// TestTheRecipeWithoutANameIsRefused proves that the rule of the domain
// holds: a recipe without a name has nothing to show in the book of the
// business.
func TestTheRecipeWithoutANameIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheTest()
	definition.Name = "  "

	if _, err := recipe.New(business.ID(77), definition); !errors.Is(err, recipe.ErrMissingName) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingName)
	}
}

// TestTheRecipeWithoutTheWordOfTheOwnerIsRefused proves that the rule of
// the domain holds: the word she used to confirm is kept with the record,
// and a record without it is not a record she confirmed.
func TestTheRecipeWithoutTheWordOfTheOwnerIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheTest()
	definition.ConfirmedWith = ""

	if _, err := recipe.New(business.ID(77), definition); !errors.Is(err, recipe.ErrMissingConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingConfirmation)
	}
}

// TestTheRecipeWithoutIngredientsIsRefused proves that the rule of the
// domain holds: a recipe without ingredients is a list of nothing.
func TestTheRecipeWithoutIngredientsIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheTest()
	definition.Ingredients = nil

	if _, err := recipe.New(business.ID(77), definition); !errors.Is(err, recipe.ErrNoIngredients) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNoIngredients)
	}
}

// TestTheIngredientWithoutItsGramsIsRefused proves that the rule of the
// domain holds: an ingredient of a recipe carries the grams of it, and a
// recipe that says how much of nothing it needs is not a guide.
func TestTheIngredientWithoutItsGramsIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := recipe.NewIngredient(ingredient.ID(10), 0); !errors.Is(err, recipe.ErrMissingGrams) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingGrams)
	}

	definition := theDefinitionOfTheTest()
	definition.Ingredients = []recipe.Ingredient{{IngredientID: ingredient.ID(10)}}

	if _, err := recipe.New(business.ID(77), definition); !errors.Is(err, recipe.ErrMissingGrams) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingGrams)
	}
}

// TestTheRecipeWithoutStepsIsRefused proves that the rule of the domain
// holds: a recipe is a guide to learn and to teach, and a guide without
// steps teaches nothing.
func TestTheRecipeWithoutStepsIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheTest()
	definition.Steps = nil

	if _, err := recipe.New(business.ID(77), definition); !errors.Is(err, recipe.ErrNoSteps) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNoSteps)
	}
}

// TestTheStepWithoutItsInstructionIsRefused proves that the rule of the
// domain holds: a step without its instruction is not a step.
func TestTheStepWithoutItsInstructionIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := recipe.NewStep("  ", "la masa que holds la forma", 20, 24); !errors.Is(err, recipe.ErrMissingInstruction) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingInstruction)
	}

	definition := theDefinitionOfTheTest()
	definition.Steps = []recipe.Step{{Cue: "la masa que holds la forma", Minutes: 20}}

	if _, err := recipe.New(business.ID(77), definition); !errors.Is(err, recipe.ErrMissingInstruction) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingInstruction)
	}
}

// TestTheRecipeWithoutAYieldIsRefused proves that the rule of the domain
// holds: the yield is a declared constant, and a recipe without it says
// nothing about what it makes.
func TestTheRecipeWithoutAYieldIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := recipe.NewYield(0, 90); !errors.Is(err, recipe.ErrInvalidYield) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidYield)
	}
	if _, err := recipe.NewYield(10, 0); !errors.Is(err, recipe.ErrInvalidYield) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidYield)
	}

	definition := theDefinitionOfTheTest()
	definition.Yield = recipe.Yield{}

	if _, err := recipe.New(business.ID(77), definition); !errors.Is(err, recipe.ErrInvalidYield) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidYield)
	}
}

// TestTheRecipeWithoutItsShelfLifeIsRefused proves that the rule of the
// domain holds: the recipe carries the shelf life of what it makes, and the
// owner corrects it later.
func TestTheRecipeWithoutItsShelfLifeIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheTest()
	definition.ShelfLifeDays = 0

	if _, err := recipe.New(business.ID(77), definition); !errors.Is(err, recipe.ErrMissingShelfLife) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingShelfLife)
	}
}

// TestTheStepBelowTheAbsoluteZeroIsRefused proves that the rule of the
// domain holds: a temperature that no oven and no fridge can hold is not a
// temperature.
func TestTheStepBelowTheAbsoluteZeroIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := recipe.NewStep("Hornear", "", 45, -300); !errors.Is(err, recipe.ErrInvalidTemperature) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidTemperature)
	}
}

// TestTheRecipeCarriesTheIdentityOfTheIngredientAndNotACopy proves that
// the rule of the domain holds: the ingredient of a recipe is the
// ingredient of the catalog, and the recipe never carries a second copy of
// the flour.
func TestTheRecipeCarriesTheIdentityOfTheIngredientAndNotACopy(t *testing.T) {
	t.Parallel()

	fields := reflect.TypeOf(recipe.Ingredient{})
	names := make([]string, 0, fields.NumField())
	for index := range fields.NumField() {
		names = append(names, fields.Field(index).Name)
	}

	want := []string{"IngredientID", "Grams"}
	if len(names) != len(want) {
		t.Fatalf("the ingredient of a recipe carries %v, want %v", names, want)
	}
	for index, one := range want {
		if names[index] != one {
			t.Errorf("the field %d of the ingredient of a recipe is %q, want %q", index, names[index], one)
		}
	}
}

// TestTheErrorsNameEveryError proves that the catalogue of the errors of the
// package holds every one of them: an error that no caller maps reaches the
// user unmapped.
func TestTheErrorsNameEveryError(t *testing.T) {
	t.Parallel()

	all := recipe.Errors()
	if len(all) != 13 {
		t.Fatalf("the catalogue holds %d errors, want 13", len(all))
	}

	for _, err := range all {
		if err == nil {
			t.Error("the catalogue holds an empty error")
		}
	}
}
