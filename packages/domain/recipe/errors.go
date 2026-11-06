package recipe

import "errors"

// The errors of the domain of the recipe.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
var (
	// ErrNotFound is the error of a recipe that the storage does not hold.
	ErrNotFound = errors.New("there is no recipe with that identity")

	// ErrMissingName is the error of a recipe without a name.
	ErrMissingName = errors.New("the recipe carries the name the owner reads")

	// ErrMissingConfirmation is the error of a recipe without the word the
	// owner used to confirm it: the word is kept with the record.
	ErrMissingConfirmation = errors.New("the recipe carries the word of the owner")

	// ErrNoIngredients is the error of a recipe that names nothing of the
	// catalog.
	ErrNoIngredients = errors.New("the recipe carries at least one ingredient")

	// ErrMissingGrams is the error of an ingredient of a recipe without its
	// grams.
	ErrMissingGrams = errors.New("the ingredient of the recipe carries its grams")

	// ErrUnknownIngredient is the error of an ingredient of a recipe that
	// names nothing of the catalog of the business.
	ErrUnknownIngredient = errors.New("the ingredient of the recipe is not one of the catalog")

	// ErrNoSteps is the error of a recipe without steps: a recipe is a
	// guide to learn and to teach.
	ErrNoSteps = errors.New("the recipe carries at least one step")

	// ErrMissingInstruction is the error of a step of a recipe without its
	// instruction.
	ErrMissingInstruction = errors.New("the step of the recipe carries its instruction")

	// ErrInvalidTemperature is the error of a step of a recipe whose
	// temperature no oven and no fridge can hold.
	ErrInvalidTemperature = errors.New("the temperature of the step is below the absolute zero")

	// ErrInvalidYield is the error of a recipe whose yield is not positive.
	ErrInvalidYield = errors.New("the yield of the recipe is positive")

	// ErrMissingShelfLife is the error of a recipe without the shelf life
	// of what it makes.
	ErrMissingShelfLife = errors.New("the recipe carries the shelf life of what it makes")

	// ErrInvalidValue is the error of a row the database refused because it
	// breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no
	// storage behind it.
	ErrNoStorage = errors.New("the storage of the recipes is not open")

	// ErrWrongBusiness is the error of a recipe of another business: the
	// recipe belongs to the business that owns it, and the cost of a batch
	// is read for one business.
	ErrWrongBusiness = errors.New("the recipe belongs to another business")
)

// Errors returns the errors of the domain of the recipe.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrMissingName,
		ErrMissingConfirmation,
		ErrNoIngredients,
		ErrMissingGrams,
		ErrUnknownIngredient,
		ErrNoSteps,
		ErrMissingInstruction,
		ErrInvalidTemperature,
		ErrInvalidYield,
		ErrMissingShelfLife,
		ErrInvalidValue,
		ErrNoStorage,
		ErrWrongBusiness,
	}
}
