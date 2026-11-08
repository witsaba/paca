package recipe_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theDayOfTheLabor is the day the labor of the test was worked.
func theDayOfTheLabor() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 7, 9, 0, 0, 0, time.UTC))
}

// theHoursOfTheLabor are the hours the owner worked: two hours with her
// hands, one in the oven and half an hour of waiting.
func theHoursOfTheLabor(t *testing.T) recipe.Hours {
	t.Helper()

	hours, err := recipe.NewHours(2, 1, 0.5)
	if err != nil {
		t.Fatalf("the hours of the labor were refused: %v", err)
	}

	return hours
}

// theRecipeOfTheLabor is the recipe the labor of the batch is derived
// from: thirty minutes with the hands, forty-five in the oven and an hour
// of waiting.
func theRecipeOfTheLabor(t *testing.T) recipe.Recipe {
	t.Helper()

	flour, err := recipe.NewIngredient(ingredient.ID(10), 500)
	if err != nil {
		t.Fatalf("the ingredient of the recipe was refused: %v", err)
	}

	amasar, err := recipe.NewStep("Amasar.", "", 30, 24, recipe.LaborHandsOn)
	if err != nil {
		t.Fatalf("the step was refused: %v", err)
	}

	hornear, err := recipe.NewStep("Hornear.", "", 45, 220, recipe.LaborOven)
	if err != nil {
		t.Fatalf("the step was refused: %v", err)
	}

	reposar, err := recipe.NewStep("Reposar.", "", 60, 24, recipe.LaborWaiting)
	if err != nil {
		t.Fatalf("the step was refused: %v", err)
	}

	built, err := recipe.New(business.ID(77), recipe.Definition{
		Name:          "Pan de molde",
		Ingredients:   []recipe.Ingredient{flour},
		Steps:         []recipe.Step{amasar, hornear, reposar},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the recipe was refused: %v", err)
	}

	return built
}

// TestTheLaborIsMeasuredInHoursAndInThreeKinds proves the rule of the
// domain: the labor is measured in hours, and the hours are held apart in
// the three kinds of the labor - the hands, the oven and the waiting.
func TestTheLaborIsMeasuredInHoursAndInThreeKinds(t *testing.T) {
	t.Parallel()

	labour, err := recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:         theHoursOfTheLabor(t),
		HourValue:     ledger.Amount(20000),
		Day:           theDayOfTheLabor(),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the labor was refused: %v", err)
	}

	if labour.Hours.HandsOn != 2 {
		t.Errorf("the hours of the hands are %v, want 2", labour.Hours.HandsOn)
	}
	if labour.Hours.Oven != 1 {
		t.Errorf("the hours of the oven are %v, want 1", labour.Hours.Oven)
	}
	if labour.Hours.Waiting != 0.5 {
		t.Errorf("the hours of the waiting are %v, want 0.5", labour.Hours.Waiting)
	}
	if labour.Hours.Total() != 3.5 {
		t.Errorf("the hours of the labor are %v, want 3.5", labour.Hours.Total())
	}
}

// TestTheHoursThatAreNotHoursAreRefused proves the rule of the domain: the
// hours of the labor are not negative, because a negative number of hours
// is a number nobody worked.
func TestTheHoursThatAreNotHoursAreRefused(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewHours(-1, 0, 0)
	if !errors.Is(err, recipe.ErrInvalidHours) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidHours)
	}
}

// TestTheLaborWithoutHoursIsRefused proves the rule of the domain: the
// hours of the owner are the fact, and a labor without hours is nothing.
func TestTheLaborWithoutHoursIsRefused(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:         recipe.Hours{},
		HourValue:     ledger.Amount(20000),
		Day:           theDayOfTheLabor(),
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingHours) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingHours)
	}
}

// TestTheValueOfTheHourIsDeclaredByTheOwner proves the rule of the domain:
// the value of the hour is declared, and it is never inferred.
func TestTheValueOfTheHourIsDeclaredByTheOwner(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:         theHoursOfTheLabor(t),
		HourValue:     ledger.Amount(0),
		Day:           theDayOfTheLabor(),
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingHourValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingHourValue)
	}
}

// TestTheValueOfTheHourThatWasInferredIsRefused proves the rule of the
// domain: the money of the work of a person is never inferred, and a
// labor whose value of the hour was filled by the agent is not built.
//
// The rule is the one of the purchase: the money of a line that was
// inferred is refused, and the book asks the owner for it.
func TestTheValueOfTheHourThatWasInferredIsRefused(t *testing.T) {
	t.Parallel()

	labour, err := recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:             theHoursOfTheLabor(t),
		HourValue:         ledger.Amount(20000),
		HourValueInferred: true,
		Day:               theDayOfTheLabor(),
		ConfirmedWith:     "sí",
	})
	if !errors.Is(err, recipe.ErrInferredHourValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInferredHourValue)
	}
	if !reflect.DeepEqual(labour, recipe.Labor{}) {
		t.Errorf("the labor was built: %v", labour)
	}
}

// TestTheLaborCarriesTheDayAndTheWordOfTheOwner proves the rule of the
// domain: the labor carries the day it was worked and the word the owner
// used to confirm it.
//
// The labor carries its own error: the error of the recipe is the error of
// another record, and the owner is told about the record that failed.
func TestTheLaborCarriesTheDayAndTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:     theHoursOfTheLabor(t),
		HourValue: ledger.Amount(20000),
		Day:       theDayOfTheLabor(),
	})
	if !errors.Is(err, recipe.ErrMissingLaborConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingLaborConfirmation)
	}
	if errors.Is(err, recipe.ErrMissingConfirmation) {
		t.Fatalf("the error is %v, want the error of the labor and not the error of the recipe", err)
	}

	_, err = recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:         theHoursOfTheLabor(t),
		HourValue:     ledger.Amount(20000),
		Day:           ledger.Date{},
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingDay) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingDay)
	}
}

// TestTheMoneyOfTheLaborIsTheHoursTimesTheValueOfTheHour proves the rule
// of the domain: the money of the labor is the hours of it times the value
// of the hour, and the money of the book counts whole pesos.
func TestTheMoneyOfTheLaborIsTheHoursTimesTheValueOfTheHour(t *testing.T) {
	t.Parallel()

	labour, err := recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:         theHoursOfTheLabor(t),
		HourValue:     ledger.Amount(20000),
		Day:           theDayOfTheLabor(),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the labor was refused: %v", err)
	}

	if labour.Money != ledger.Amount(70000) {
		t.Errorf("the money of the labor is %d, want 70000", labour.Money)
	}
}

// TestTheKindOfTheStepIsOneOfTheThreeKinds proves the rule of the domain:
// the kind of the labor of a step is one of the three kinds of the book,
// and the kind the owner did not name is the labor of the hands.
func TestTheKindOfTheStepIsOneOfTheThreeKinds(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewStep("Amasar.", "", 30, 24, recipe.LaborKind("cocina"))
	if !errors.Is(err, recipe.ErrUnknownLaborKind) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownLaborKind)
	}

	step, err := recipe.NewStep("Amasar.", "", 30, 24, recipe.LaborKind(""))
	if err != nil {
		t.Fatalf("the step was refused: %v", err)
	}
	if step.Kind != recipe.LaborHandsOn {
		t.Errorf("the kind of the step is %q, want %q", step.Kind, recipe.LaborHandsOn)
	}
}

// TestTheRecipeRefusesTheStepOfAnUnknownKind proves the rule of the domain:
// the recipe is written with the word of the owner, and it never carries a
// step whose kind of labor the book does not know.
func TestTheRecipeRefusesTheStepOfAnUnknownKind(t *testing.T) {
	t.Parallel()

	flour, err := recipe.NewIngredient(ingredient.ID(10), 500)
	if err != nil {
		t.Fatalf("the ingredient of the recipe was refused: %v", err)
	}

	_, err = recipe.New(business.ID(77), recipe.Definition{
		Name:          "Pan de molde",
		Ingredients:   []recipe.Ingredient{flour},
		Steps:         []recipe.Step{{Instruction: "Amasar.", Kind: recipe.LaborKind("cocina")}},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrUnknownLaborKind) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownLaborKind)
	}
}

// TestTheLaborOfTheBatchIsDerivedFromTheHoursOfItsSteps proves the rule of
// the domain: the labor of a batch is derived from the hours of its steps,
// and never typed.
//
// Thirty minutes with the hands, forty-five in the oven and an hour of
// waiting are two hours and a quarter of labor, and they are worth forty-
// five thousand pesos with an hour of twenty thousand.
func TestTheLaborOfTheBatchIsDerivedFromTheHoursOfItsSteps(t *testing.T) {
	t.Parallel()

	labour, err := recipe.LaborOf(theRecipeOfTheLabor(t), ledger.Amount(20000))
	if err != nil {
		t.Fatalf("the labor of the batch was refused: %v", err)
	}

	if labour.Hours.HandsOn != 0.5 {
		t.Errorf("the hours of the hands are %v, want 0.5", labour.Hours.HandsOn)
	}
	if labour.Hours.Oven != 0.75 {
		t.Errorf("the hours of the oven are %v, want 0.75", labour.Hours.Oven)
	}
	if labour.Hours.Waiting != 1 {
		t.Errorf("the hours of the waiting are %v, want 1", labour.Hours.Waiting)
	}
	if labour.Money != ledger.Amount(45000) {
		t.Errorf("the money of the labor is %d, want 45000", labour.Money)
	}
	if labour.Unknown {
		t.Error("the money of the labor is not derivable, want the money of the hours")
	}
}

// TestTheLaborOfTheBatchSaysThatItDoesNotKnowTheValueOfTheHour proves the
// rule of the domain: the value of the hour is declared by the owner, and
// the answer that does not know says so instead of guessing it.
func TestTheLaborOfTheBatchSaysThatItDoesNotKnowTheValueOfTheHour(t *testing.T) {
	t.Parallel()

	labour, err := recipe.LaborOf(theRecipeOfTheLabor(t), ledger.Amount(0))
	if err != nil {
		t.Fatalf("the labor of the batch was refused: %v", err)
	}

	if !labour.Unknown {
		t.Fatal("the money of the labor is derivable, want the news that it is not")
	}
	if labour.Money != 0 {
		t.Errorf("the money of the labor is %d, want none", labour.Money)
	}
	if labour.Reason == "" {
		t.Error("the labor carries no reason, want the reason it does not know")
	}
}

// TestTheValueOfTheHourIsTheOneOfTheMostRecentLabor proves the rule of the
// domain: the value of the hour is what the owner declared, and the last
// declaration is the one that counts.
func TestTheValueOfTheHourIsTheOneOfTheMostRecentLabor(t *testing.T) {
	t.Parallel()

	if recipe.TheHourValueOf(nil) != 0 {
		t.Error("the value of the hour of a business that declared none is not zero")
	}

	older, err := recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:         theHoursOfTheLabor(t),
		HourValue:     ledger.Amount(15000),
		Day:           ledger.NewDate(time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the labor was refused: %v", err)
	}

	newer, err := recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:         theHoursOfTheLabor(t),
		HourValue:     ledger.Amount(22000),
		Day:           theDayOfTheLabor(),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the labor was refused: %v", err)
	}

	if got := recipe.TheHourValueOf([]recipe.Labor{older, newer}); got != ledger.Amount(22000) {
		t.Errorf("the value of the hour is %d, want 22000", got)
	}
	if got := recipe.TheHourValueOf([]recipe.Labor{newer, older}); got != ledger.Amount(22000) {
		t.Errorf("the value of the hour is %d, want 22000", got)
	}
}

// TestTheLaborCarriesNoEmployee proves the rule of the domain: the labor
// of the PAC is the labor of the owner alone, and the model of the labor
// of an employee does not exist yet.
func TestTheLaborCarriesNoEmployee(t *testing.T) {
	t.Parallel()

	forbidden := []string{"employee", "person", "worker", "payroll"}

	for _, one := range []reflect.Type{reflect.TypeOf(recipe.Labor{}), reflect.TypeOf(recipe.LaborDefinition{})} {
		for index := 0; index < one.NumField(); index++ {
			name := strings.ToLower(one.Field(index).Name)

			for _, word := range forbidden {
				if strings.Contains(name, word) {
					t.Errorf("the labor carries the field %q: the payroll of the employees is out of scope for now",
						one.Field(index).Name)
				}
			}
		}
	}
}

// TestTheKindsOfTheLaborAreTheThreeOfTheBook proves the rule of the
// domain: the kinds of the labor are the hands, the oven and the waiting,
// and the book knows no other one.
func TestTheKindsOfTheLaborAreTheThreeOfTheBook(t *testing.T) {
	t.Parallel()

	kinds := recipe.LaborKinds()
	if len(kinds) != 3 {
		t.Fatalf("the book knows %d kinds of the labor, want 3", len(kinds))
	}

	want := []recipe.LaborKind{recipe.LaborHandsOn, recipe.LaborOven, recipe.LaborWaiting}
	for index, one := range want {
		if kinds[index] != one {
			t.Errorf("the kind %d is %q, want %q", index, kinds[index], one)
		}
	}

	spoken := recipe.SpokenLaborKinds()
	if len(spoken) == 0 {
		t.Fatal("the book carries no way of saying the kinds of the labor")
	}

	kind, err := recipe.ParseLaborKind("horno")
	if err != nil {
		t.Fatalf("the kind of the labor was refused: %v", err)
	}
	if kind != recipe.LaborOven {
		t.Errorf("the kind of the labor is %q, want %q", kind, recipe.LaborOven)
	}

	if _, err := recipe.ParseLaborKind("lavado"); !errors.Is(err, recipe.ErrUnknownLaborKind) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownLaborKind)
	}
}

// TestTheLaborOfTheBatchRefusesTheStepOfNegativeMinutes proves the rule
// of the domain: the minutes of a step are a time, and the labor of a
// batch is never derived without a step the owner wrote.
func TestTheLaborOfTheBatchRefusesTheStepOfNegativeMinutes(t *testing.T) {
	t.Parallel()

	baked := theRecipeOfTheLabor(t)
	baked.Steps = []recipe.Step{
		{Instruction: "Amasar.", Minutes: -30, Kind: recipe.LaborHandsOn},
	}

	_, err := recipe.LaborOf(baked, ledger.Amount(20000))
	if !errors.Is(err, recipe.ErrInvalidMinutes) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidMinutes)
	}
}

// TestTheLaborOfTheBatchRefusesTheStepOfAnUnknownKind proves the rule of
// the domain: a step whose kind of labor the book does not know is not a
// step of the recipe, and the labor of the batch is refused with it.
func TestTheLaborOfTheBatchRefusesTheStepOfAnUnknownKind(t *testing.T) {
	t.Parallel()

	baked := theRecipeOfTheLabor(t)
	baked.Steps = []recipe.Step{
		{Instruction: "Hornear.", Minutes: 45, Kind: recipe.LaborKind("cocina")},
	}

	_, err := recipe.LaborOf(baked, ledger.Amount(20000))
	if !errors.Is(err, recipe.ErrUnknownLaborKind) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownLaborKind)
	}
}
