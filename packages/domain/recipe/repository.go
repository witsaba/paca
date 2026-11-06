package recipe

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
)

// Repository is the port of the storage of the recipes.
//
// The port is what a use case needs, not the shape of a table: it writes a
// recipe with its ingredients and its steps, and it answers it with the
// identity that the storage generated.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Record writes the recipe with its ingredients and its steps, and
	// answers it with the identity that the storage generated.
	//
	// The ingredients of a recipe and its steps are rows of their own
	// tables, and the storage writes them with it: a recipe without its
	// ingredients and its steps is not a recipe.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the business or the ingredient of the
	// recipe is not there, and ErrNoStorage when the storage is not open.
	Record(ctx context.Context, recipe Recipe) (Recipe, error)
}

// Reader is the port of the storage that reads the recipes back.
//
// The port is what a use case needs: the cost of a batch is derived from
// the recipe and from the layers of the stock, and the use case reads the
// recipe before it values it.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Reader interface {
	// Find returns the recipe that carries that identity, with its
	// ingredients and its steps.
	//
	// A recipe without its ingredients and its steps is not a recipe, and
	// the storage reads them with it.
	//
	// The error is ErrNotFound when no recipe carries that identity, and
	// ErrNoStorage when the storage is not open.
	Find(ctx context.Context, id ID) (Recipe, error)

	// List returns the recipes of the business, with their ingredients and
	// their steps, in the order of their identity.
	//
	// The list holds the recipes of the business and of no other one, and
	// it is empty when the business wrote no recipe yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Recipe, error)
}

// LaborRepository is the port of the storage of the labor.
//
// The port is what a use case needs: the tool of the labor writes the hours
// the owner worked and the value of the hour she declared, and the
// storage keeps them in the book of the business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type LaborRepository interface {
	// Record writes the labor of the business and answers it with the
	// identity that the storage generated.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the business is not there, and
	// ErrNoStorage when the storage is not open.
	Record(ctx context.Context, labor Labor) (Labor, error)
}

// LaborReader is the port of the storage that reads the labor back.
//
// The port is what a use case needs: the value of the hour the owner
// declared is read from the labor of the business, and the labor of a
// batch is valued with it.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type LaborReader interface {
	// List returns the labor of the business, in the order of the day it
	// was worked and of its identity.
	//
	// The list holds the labor of the business and of no other one, and
	// it is empty when the business worked no hour yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Labor, error)
}
