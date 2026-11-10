package recipe

import (
	"fmt"
	"math"

	"github.com/witsaba/paca/packages/domain/business"
)

// CapacityOfARecipe is what the hours of the owner carry of one recipe of
// the bakery.
//
// The capacity is derived from the labor of the recipe and from the hours
// the owner declared: it is never typed, and it is never a promise of the
// calendar.
type CapacityOfARecipe struct {
	// RecipeID is the recipe the capacity is of.
	RecipeID ID

	// RecipeName is the name the owner reads of the recipe.
	RecipeName string

	// Minutes is the labor of a batch of that recipe that needs the
	// owner: the hands-on minutes of its steps. The oven and the waiting
	// are not hers, because the batch bakes and rests while she does the
	// next thing.
	Minutes int

	// Batches is the whole number of batches the declared hours carry.
	// The book never splits a batch.
	Batches int

	// Units is the units those batches make: the batches times the yield
	// of the recipe.
	Units int

	// Limited says that the hours bound the recipe. It is false when the
	// recipe carries no hands-on labor at all: the hours do not limit it,
	// and the answer says so instead of lying with a zero.
	Limited bool
}

// Capacity is the answer of the book: the hours the owner declared for the
// weekend, and what the weekend can carry with them.
//
// The capacity is never a promise of the calendar: it is what the hours can
// carry, and the stock of the ingredients may carry less.
type Capacity struct {
	// Hours is the number of hours the owner declared: the hours of her
	// own hands.
	Hours float64

	// Recipes is the capacity of every recipe of the business.
	Recipes []CapacityOfARecipe
}

// TheCapacity derives what the hours of the owner carry of the recipes of
// the business.
//
// The rules of the domain hold here:
//
//   - the hours the owner declares are positive: she declares the hours of
//     her weekend, and zero hours are not a weekend;
//   - every recipe is a recipe of the business that asks for the capacity;
//   - the labor of a batch that needs the owner is the sum of the minutes of
//     the steps of her hands, and the oven and the waiting are not counted:
//     the batch bakes and rests while she does the next thing;
//   - the batches are whole: the book never splits a batch, so the capacity
//     carries the whole number of batches the hours carry, and the units are
//     those batches times the yield of the recipe;
//   - a recipe whose hands-on labor is zero is not limited by the hours: its
//     capacity carries no batch and no unit, and it says that the hours do
//     not bound it.
//
// The error is ErrInvalidCapacityHours when the hours are not positive,
// ErrWrongBusiness when a recipe belongs to another business, and the error
// of the recipe when a step of it is not a step of the book.
func TheCapacity(own business.ID, hours float64, recipes []Recipe) (Capacity, error) {
	if hours <= 0 {
		return Capacity{}, fmt.Errorf("%w: the weekend carries %v hours", ErrInvalidCapacityHours, hours)
	}

	capacity := Capacity{Hours: hours, Recipes: make([]CapacityOfARecipe, 0, len(recipes))}

	for _, one := range recipes {
		if one.BusinessID != own {
			return Capacity{}, fmt.Errorf("%w: the recipe %d belongs to the business %d",
				ErrWrongBusiness, one.ID, one.BusinessID)
		}

		minutes, err := theHandsOnMinutesOf(one)
		if err != nil {
			return Capacity{}, err
		}

		carried := CapacityOfARecipe{
			RecipeID:   one.ID,
			RecipeName: one.Name,
			Minutes:    minutes,
		}

		// The recipe the hours do not bound is not a recipe of no
		// capacity: the answer says that the hours do not limit it.
		if minutes > 0 {
			batches := int(math.Floor(hours * MinutesPerHour / float64(minutes)))

			carried.Limited = true
			carried.Batches = batches
			carried.Units = batches * one.Yield.Units
		}

		capacity.Recipes = append(capacity.Recipes, carried)
	}

	return capacity, nil
}

// theHandsOnMinutesOf answers the minutes of the steps of the recipe that
// need the hands of the owner.
//
// The rule of the domain: the step of the hands is the time the owner spends
// on the dough, the step the owner did not qualify is a step of her hands,
// and the oven and the waiting are not the labor of a batch she carries.
//
// The error is the error of the recipe: a step of negative minutes is not a
// time, and a step of a kind the book does not know is never guessed.
func theHandsOnMinutesOf(baked Recipe) (int, error) {
	minutes := 0

	for _, one := range baked.Steps {
		if one.Minutes < 0 {
			return 0, fmt.Errorf("%w: the step carries %d minutes", ErrInvalidMinutes, one.Minutes)
		}

		kind, err := ParseLaborKind(string(one.Kind))
		if err != nil {
			return 0, err
		}

		if kind != LaborHandsOn {
			continue
		}

		minutes += one.Minutes
	}

	return minutes, nil
}
