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

	// ErrInvalidHours is the error of hours of the labor that are not a
	// number of hours: a negative number of hours is a number nobody
	// worked.
	ErrInvalidHours = errors.New("the hours of the labor are not negative")

	// ErrMissingHours is the error of a labor without hours: the hours of
	// the owner are the fact, and a labor without them is nothing.
	ErrMissingHours = errors.New("the labor carries the hours of the owner")

	// ErrMissingHourValue is the error of a labor whose value of the hour
	// was not declared: the money of the work of a person is never
	// inferred.
	ErrMissingHourValue = errors.New("the labor carries the value of the hour the owner declared")

	// ErrMissingDay is the error of a labor without the day it was
	// worked: a labor of no day is a labor of no time.
	ErrMissingDay = errors.New("the labor carries the day it was worked")

	// ErrUnknownLaborKind is the error of a kind of the labor the book
	// does not know: the kinds of the labor are the hands, the oven and
	// the waiting, and no other one is guessed.
	ErrUnknownLaborKind = errors.New("the kind of the labor is one of the three kinds of the book")

	// ErrMissingBusiness is the error of a labor or of an overhead of no
	// business: the labor of the bakery and the overhead of its house
	// belong to the business that works it.
	ErrMissingBusiness = errors.New("the labor and the overhead belong to a business")

	// ErrMissingElectricity is the error of an overhead without the
	// electricity of the house: the money of the electricity is declared
	// by the owner, and it is never inferred.
	ErrMissingElectricity = errors.New("the overhead carries the electricity of the house the owner declared")

	// ErrMissingOvenHours is the error of an overhead without the hours of
	// the oven: without them the money of the oven has no rate, and a
	// rate nobody declared is a rate PACA invented.
	ErrMissingOvenHours = errors.New("the overhead carries the hours of the oven")

	// ErrInvalidAllocation is the error of an allocation that is not a
	// share of the electricity: the owner declares the share of the
	// electricity of the house that belongs to the oven, and it is greater
	// than zero and not greater than the electricity itself.
	ErrInvalidAllocation = errors.New("the allocation of the overhead is a share of the electricity")

	// ErrMissingRecipe is the error of a production that names no recipe:
	// the production is the fact of a recipe being made, and a production
	// of no recipe is a production of nothing.
	ErrMissingRecipe = errors.New("the production names the recipe it was made of")

	// ErrAmbiguousRecipe is the error of a name of a recipe that more
	// than one recipe of the business carries: the owner chooses with one
	// word, and the book never guesses which recipe she made.
	ErrAmbiguousRecipe = errors.New("the business holds more than one recipe with that name")

	// ErrUnknownRecipe is the error of a name of a recipe that no recipe
	// of the business carries: the owner named a recipe the book does not
	// hold, and the book says so instead of writing a production of
	// nothing.
	ErrUnknownRecipe = errors.New("the business holds no recipe with that name")

	// ErrMissingProductionDay is the error of a production of no day: a
	// production of no time is a production of nothing.
	ErrMissingProductionDay = errors.New("the production carries the day it was made")

	// ErrMissingProductionConfirmation is the error of a production without
	// the word the owner used to confirm it: the word is kept with the
	// record.
	ErrMissingProductionConfirmation = errors.New("the production carries the word of the owner")

	// ErrMissingProductionGrams is the error of a production that carries
	// no grams that went in: the production consumed the layers of the
	// stock, and the grams it consumed are what it carries.
	ErrMissingProductionGrams = errors.New("the production carries the grams that went in")

	// ErrMissingProductionUnits is the error of a production that made no
	// unit: the units are what came out of the oven.
	ErrMissingProductionUnits = errors.New("the production made no unit")

	// ErrInvalidProductionMoney is the error of a production whose money
	// is negative: the money of a production is the money of the layers it
	// consumed, and the money of the book is never negative.
	ErrInvalidProductionMoney = errors.New("the money of the production is not negative")

	// ErrMissingStock is the error of a production the stock cannot cover:
	// a batch that cannot be baked is not a batch.
	ErrMissingStock = errors.New("the stock does not hold what the recipe needs")

	// ErrUnknownStock is the error of a production whose ingredients the
	// book cannot weigh: the grams that went in are never assumed.
	ErrUnknownStock = errors.New("the stock holds an ingredient the book cannot weigh")

	// ErrExpiredUnit is the error of a unit that expired: a unit that
	// expired is never sold, and the sale of it is refused.
	ErrExpiredUnit = errors.New("the unit expired and is never sold")

	// ErrMissingSaleDay is the error of a sale of no day: the day is what
	// says whether the unit expired, and a sale of no day says nothing.
	ErrMissingSaleDay = errors.New("the sale carries the day it was made")

	// ErrMissingExpiry is the error of a unit of the production that
	// carries no day of expiry: the expiry is what the owner asks about,
	// and a unit without it is not a unit of the book.
	ErrMissingExpiry = errors.New("the unit of the production carries the day it expires")
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
		ErrInvalidHours,
		ErrMissingHours,
		ErrMissingHourValue,
		ErrMissingDay,
		ErrUnknownLaborKind,
		ErrMissingBusiness,
		ErrMissingElectricity,
		ErrMissingOvenHours,
		ErrInvalidAllocation,
		ErrMissingRecipe,
		ErrAmbiguousRecipe,
		ErrUnknownRecipe,
		ErrMissingProductionDay,
		ErrMissingProductionConfirmation,
		ErrMissingProductionGrams,
		ErrMissingProductionUnits,
		ErrInvalidProductionMoney,
		ErrMissingStock,
		ErrUnknownStock,
		ErrExpiredUnit,
		ErrMissingSaleDay,
		ErrMissingExpiry,
	}
}
