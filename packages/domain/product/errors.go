package product

import "errors"

// The errors of the domain of the product.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
var (
	// ErrNotFound is the error of a product that the storage does not
	// hold.
	ErrNotFound = errors.New("there is no product with that identity")

	// ErrInvalidName is the error of a product without a name.
	ErrInvalidName = errors.New("the name of a product is not empty")

	// ErrDuplicateName is the error of a product whose name is taken
	// inside the business.
	ErrDuplicateName = errors.New("the business already holds a product with that name")

	// ErrDuplicateRecipe is the error of a recipe that backs another
	// product of the business.
	ErrDuplicateRecipe = errors.New("the recipe already backs another product of the business")

	// ErrMissingRecipe is the error of a product that names no recipe.
	ErrMissingRecipe = errors.New("the product of the bakery names no recipe")

	// ErrUnknownRecipe is the error of a name that is neither a product
	// nor a recipe of the business.
	ErrUnknownRecipe = errors.New("the business holds no recipe with that name")

	// ErrMissingProduct is the error of a price that names no product.
	ErrMissingProduct = errors.New("the price names no product")

	// ErrUnknownChannel is the error of a channel that is not one of the
	// channels of the book.
	ErrUnknownChannel = errors.New("the channel of a price is not one of the channels of the book")

	// ErrInvalidPrice is the error of a price that is not a positive
	// number of whole pesos.
	ErrInvalidPrice = errors.New("the price of a product is not a positive number of pesos")

	// ErrMissingDay is the error of a price without the day it was set.
	ErrMissingDay = errors.New("the price carries no day")

	// ErrWrongBusiness is the error of a price list of a business that
	// does not hold price lists.
	ErrWrongBusiness = errors.New("the price list belongs to a business of its own")

	// ErrOfAnotherBusiness is the error of a product backed by the recipe
	// of another business, and of a price of a product of another
	// business: the product and the price belong to the business that
	// owns them, and never to the one of another.
	ErrOfAnotherBusiness = errors.New("the recipe and the product belong to the business that owns them")

	// ErrInvalidValue is the error of a row the database refused because
	// it breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no
	// storage behind it.
	ErrNoStorage = errors.New("the storage of the price list is not open")
)

// Errors returns the errors of the domain of the product.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrInvalidName,
		ErrDuplicateName,
		ErrDuplicateRecipe,
		ErrMissingRecipe,
		ErrUnknownRecipe,
		ErrMissingProduct,
		ErrUnknownChannel,
		ErrInvalidPrice,
		ErrMissingDay,
		ErrWrongBusiness,
		ErrOfAnotherBusiness,
		ErrInvalidValue,
		ErrNoStorage,
	}
}
