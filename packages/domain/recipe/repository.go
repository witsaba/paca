package recipe

import "context"

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
