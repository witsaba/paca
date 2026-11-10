package recipe_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theRecipeOfTheCapacity is a recipe of the bakery that carries the minutes
// of its steps and the yield of what it makes.
//
// The hands-on step is the one the owner does with her hands, the oven is
// the one the batch bakes in and the waiting is the one the dough rests in:
// the capacity counts the first one alone.
func theRecipeOfTheCapacity(
	t *testing.T,
	id recipe.ID,
	name string,
	handsOn, oven, waiting, units int,
) recipe.Recipe {
	t.Helper()

	flour, err := recipe.NewIngredient(ingredient.ID(10), 500)
	if err != nil {
		t.Fatalf("the ingredient of the recipe was refused: %v", err)
	}

	amasar, err := recipe.NewStep("Amasar.", "", handsOn, 24, recipe.LaborHandsOn)
	if err != nil {
		t.Fatalf("the step of the hands was refused: %v", err)
	}

	hornear, err := recipe.NewStep("Hornear.", "", oven, 220, recipe.LaborOven)
	if err != nil {
		t.Fatalf("the step of the oven was refused: %v", err)
	}

	reposar, err := recipe.NewStep("Reposar.", "", waiting, 24, recipe.LaborWaiting)
	if err != nil {
		t.Fatalf("the step of the waiting was refused: %v", err)
	}

	built, err := recipe.New(business.ID(77), recipe.Definition{
		Name:          name,
		Ingredients:   []recipe.Ingredient{flour},
		Steps:         []recipe.Step{amasar, hornear, reposar},
		Yield:         recipe.Yield{Units: units, Grams: 90},
		ShelfLifeDays: 3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	built.ID = id

	return built
}

// TestTheCapacityCountsTheHandsAndNotTheOvenNorTheWaiting proves the rule of
// the domain: the labor of a batch that needs the owner is the minutes of
// the steps of her hands alone.
//
// The batch bakes and rests while she does the next thing, so the oven and
// the waiting are not hers: a recipe with thirty minutes of oven, ten of her
// hands and two hours of waiting has a labor of ten minutes.
func TestTheCapacityCountsTheHandsAndNotTheOvenNorTheWaiting(t *testing.T) {
	t.Parallel()

	baked := theRecipeOfTheCapacity(t, recipe.ID(9), "Pan de molde", 10, 30, 120, 10)

	capacity, err := recipe.TheCapacity(business.ID(77), 10, []recipe.Recipe{baked})
	if err != nil {
		t.Fatalf("the capacity was refused: %v", err)
	}

	if len(capacity.Recipes) != 1 {
		t.Fatalf("the capacity carries %d recipes, want 1", len(capacity.Recipes))
	}

	one := capacity.Recipes[0]
	if one.Minutes != 10 {
		t.Errorf("the minutes of the batch are %d, want the 10 of the hands alone", one.Minutes)
	}
	if one.RecipeID != recipe.ID(9) {
		t.Errorf("the recipe is %d, want 9", one.RecipeID)
	}
	if one.RecipeName != "Pan de molde" {
		t.Errorf("the name of the recipe is %q, want %q", one.RecipeName, "Pan de molde")
	}
	if capacity.Hours != 10 {
		t.Errorf("the hours of the capacity are %v, want the 10 the owner declared", capacity.Hours)
	}
}

// TestTheStepTheOwnerDidNotQualifyIsTheLaborOfHerHands proves the rule of the
// domain: the kind of the labor the owner did not name is the labor of the
// hands, and the capacity counts it.
func TestTheStepTheOwnerDidNotQualifyIsTheLaborOfHerHands(t *testing.T) {
	t.Parallel()

	baked := theRecipeOfTheCapacity(t, recipe.ID(9), "Pan de molde", 0, 0, 0, 10)
	baked.Steps = []recipe.Step{{Instruction: "Amasar.", Minutes: 60}}

	capacity, err := recipe.TheCapacity(business.ID(77), 1, []recipe.Recipe{baked})
	if err != nil {
		t.Fatalf("the capacity was refused: %v", err)
	}

	if capacity.Recipes[0].Minutes != 60 {
		t.Errorf("the minutes of the batch are %d, want the 60 of the step nobody qualified",
			capacity.Recipes[0].Minutes)
	}
	if capacity.Recipes[0].Batches != 1 {
		t.Errorf("the batches are %d, want the 1 that the hour carries", capacity.Recipes[0].Batches)
	}
}

// TestTheBatchesAreWhole proves the rule of the domain: the book never
// splits a batch, so the capacity carries the whole number of batches the
// hours carry.
//
// Ten hours against a batch of three hours of hands-on labor give three
// batches and not 3.33, and an hour against the same batch gives no batch at
// all.
func TestTheBatchesAreWhole(t *testing.T) {
	t.Parallel()

	baked := theRecipeOfTheCapacity(t, recipe.ID(9), "Pan de molde", 180, 0, 0, 10)

	capacity, err := recipe.TheCapacity(business.ID(77), 10, []recipe.Recipe{baked})
	if err != nil {
		t.Fatalf("the capacity was refused: %v", err)
	}

	if got := capacity.Recipes[0].Batches; got != 3 {
		t.Fatalf("the batches are %d, want the 3 whole batches that ten hours carry", got)
	}
	if !capacity.Recipes[0].Limited {
		t.Error("the recipe has hands-on labor and the hours do not limit it")
	}

	short, err := recipe.TheCapacity(business.ID(77), 1, []recipe.Recipe{baked})
	if err != nil {
		t.Fatalf("the capacity of one hour was refused: %v", err)
	}

	if short.Recipes[0].Batches != 0 {
		t.Errorf("the batches of one hour are %d, want 0: the book never splits a batch",
			short.Recipes[0].Batches)
	}
	if short.Recipes[0].Units != 0 {
		t.Errorf("the units of one hour are %d, want 0: the book never splits a batch",
			short.Recipes[0].Units)
	}
	if !short.Recipes[0].Limited {
		t.Error("the recipe has hands-on labor and the hours do not limit it")
	}
}

// TestTheUnitsAreTheBatchesTimesTheYield proves the rule of the domain: the
// units of the capacity are the units those batches make, and the yield of
// the recipe is what makes them.
func TestTheUnitsAreTheBatchesTimesTheYield(t *testing.T) {
	t.Parallel()

	baked := theRecipeOfTheCapacity(t, recipe.ID(9), "Pan de molde", 180, 0, 0, 10)

	capacity, err := recipe.TheCapacity(business.ID(77), 10, []recipe.Recipe{baked})
	if err != nil {
		t.Fatalf("the capacity was refused: %v", err)
	}

	if got := capacity.Recipes[0].Units; got != 30 {
		t.Errorf("the units are %d, want the 30 of the 3 batches of 10 units", got)
	}
}

// TestARecipeWithNoLaborOfTheHandsIsNotLimited proves the rule of the
// domain: a recipe whose hands-on labor is zero is not bounded by the hours,
// and the answer says so instead of lying with a zero.
func TestARecipeWithNoLaborOfTheHandsIsNotLimited(t *testing.T) {
	t.Parallel()

	baked := theRecipeOfTheCapacity(t, recipe.ID(9), "Pan de molde", 0, 45, 120, 10)

	capacity, err := recipe.TheCapacity(business.ID(77), 10, []recipe.Recipe{baked})
	if err != nil {
		t.Fatalf("the capacity was refused: %v", err)
	}

	one := capacity.Recipes[0]
	if one.Limited {
		t.Error("the recipe carries no labor of the hands and the answer says that the hours limit it")
	}
	if one.Minutes != 0 {
		t.Errorf("the minutes of the batch are %d, want 0: the oven and the waiting are not hers", one.Minutes)
	}
	if one.Batches != 0 || one.Units != 0 {
		t.Errorf("the batch carries %d batches and %d units, want none: the hours do not bound it",
			one.Batches, one.Units)
	}
}

// TestTheHoursOfTheWeekendArePositive proves the rule of the domain: the
// owner declares the hours of her weekend, and zero hours are not a weekend.
func TestTheHoursOfTheWeekendArePositive(t *testing.T) {
	t.Parallel()

	baked := theRecipeOfTheCapacity(t, recipe.ID(9), "Pan de molde", 60, 0, 0, 10)

	tests := []struct {
		name  string
		hours float64
	}{
		{name: "no hours at all", hours: 0},
		{name: "less than nothing", hours: -3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := recipe.TheCapacity(business.ID(77), test.hours, []recipe.Recipe{baked})
			if !errors.Is(err, recipe.ErrInvalidCapacityHours) {
				t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidCapacityHours)
			}
			if !strings.Contains(err.Error(), "hours") {
				t.Errorf("the error does not name the hours: %v", err)
			}
		})
	}
}

// TestTheCapacityOfARecipeOfAnotherBusinessIsRefused proves the rule of the
// domain: every recipe of the capacity is a recipe of the business, and the
// recipe of another business is refused.
func TestTheCapacityOfARecipeOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	mine := theRecipeOfTheCapacity(t, recipe.ID(9), "Pan de molde", 60, 0, 0, 10)

	theirs := theRecipeOfTheCapacity(t, recipe.ID(10), "Pan integral", 60, 0, 0, 10)
	theirs.BusinessID = business.ID(88)

	capacity, err := recipe.TheCapacity(business.ID(77), 10, []recipe.Recipe{theirs})
	if !errors.Is(err, recipe.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrWrongBusiness)
	}
	if !reflect.DeepEqual(capacity, recipe.Capacity{}) {
		t.Errorf("the capacity is %+v, want the empty answer of a refusal", capacity)
	}

	if _, err := recipe.TheCapacity(business.ID(77), 10, []recipe.Recipe{mine, theirs}); !errors.Is(err, recipe.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrWrongBusiness)
	}
}

// TestTheStepOfAnUnknownKindIsRefused proves the rule of the domain: the
// kind of the labor is one of the three kinds of the book, and a kind the
// book does not know is never guessed by the capacity.
func TestTheStepOfAnUnknownKindIsRefused(t *testing.T) {
	t.Parallel()

	baked := theRecipeOfTheCapacity(t, recipe.ID(9), "Pan de molde", 60, 0, 0, 10)
	baked.Steps = []recipe.Step{{Instruction: "Amasar.", Minutes: 60, Kind: recipe.LaborKind("mixing")}}

	_, err := recipe.TheCapacity(business.ID(77), 10, []recipe.Recipe{baked})
	if !errors.Is(err, recipe.ErrUnknownLaborKind) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownLaborKind)
	}
}

// TestTheCapacityCarriesNoMoney proves the rule of the domain: the capacity
// is what the weekend can carry, in hours, batches and units, and it carries
// no money - the payroll of the employees is out of scope.
func TestTheCapacityCarriesNoMoney(t *testing.T) {
	t.Parallel()

	for _, model := range []reflect.Type{reflect.TypeOf(recipe.Capacity{}), reflect.TypeOf(recipe.CapacityOfARecipe{})} {
		for i := 0; i < model.NumField(); i++ {
			field := strings.ToLower(model.Field(i).Name)

			for _, money := range []string{"money", "amount", "price", "currency", "cost"} {
				if strings.Contains(field, money) {
					t.Errorf("the field %s of %s carries money, and the capacity carries none",
						model.Field(i).Name, model.Name())
				}
			}
		}
	}
}

// TestTheCapacityOfABusinessWithoutRecipesIsEmpty proves the rule of the
// domain: a business that wrote no recipe yet has no capacity to answer, and
// the hours it declared are still the hours of the answer.
func TestTheCapacityOfABusinessWithoutRecipesIsEmpty(t *testing.T) {
	t.Parallel()

	capacity, err := recipe.TheCapacity(business.ID(77), 8, nil)
	if err != nil {
		t.Fatalf("the capacity was refused: %v", err)
	}

	if capacity.Hours != 8 {
		t.Errorf("the hours of the capacity are %v, want 8", capacity.Hours)
	}
	if len(capacity.Recipes) != 0 {
		t.Errorf("the capacity carries %d recipes, want none", len(capacity.Recipes))
	}
}
