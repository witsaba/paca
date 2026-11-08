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

// OverheadRepository is the port of the storage of the overhead.
//
// The port is what a use case needs: the tool of the overhead writes the
// electricity of the house, the hours of the oven and the allocation the
// owner declared, and the storage keeps them in the book of the business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type OverheadRepository interface {
	// Record writes the overhead of the business and answers it with the
	// identity that the storage generated.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the business is not there, and
	// ErrNoStorage when the storage is not open.
	Record(ctx context.Context, overhead Overhead) (Overhead, error)
}

// OverheadReader is the port of the storage that reads the overhead back.
//
// The port is what a use case needs: the rate of the overhead of the house
// is read from the overhead of the business, and the overhead of a batch
// is valued with it.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type OverheadReader interface {
	// List returns the overhead of the business, in the order of the day
	// it was declared and of its identity.
	//
	// The list holds the overhead of the business and of no other one,
	// and it is empty when the business declared none yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Overhead, error)
}

// ProductionRepository is the port of the storage of the productions.//
// The port is what a use case needs: the tool of the production writes the
// fact of what the bakery made on a day, with the units that came out of
// it, and the storage keeps them in the book of the business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type ProductionRepository interface {
	// Record writes the production of the business with the units that
	// came out of it, and answers it with the identity that the storage
	// generated.
	//
	// The units of a production are rows of their own table, and the
	// storage writes them with it: a production without its units is not a
	// production.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the business or the recipe is not there,
	// and ErrNoStorage when the storage is not open.
	Record(ctx context.Context, production Production) (Production, error)
}

// ShelfLifeRepository is the port of the storage of the corrections of the
// shelf life.
//
// The port is what a use case needs: the tool of the expiry writes the
// correction the owner declared, and the storage keeps it in the book of the
// business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type ShelfLifeRepository interface {
	// Record writes the correction of the shelf life and answers it with
	// the identity that the storage generated.
	//
	// The correction is kept: every one of them is a row of its own, and
	// the last one the owner declared is the shelf life that counts.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the business or the recipe is not there,
	// and ErrNoStorage when the storage is not open.
	Record(ctx context.Context, correction ShelfLifeCorrection) (ShelfLifeCorrection, error)
}

// ShelfLifeReader is the port of the storage that reads the corrections of
// the shelf life back.
//
// The port is what a use case needs: the shelf life that counts is the last
// correction the owner declared, and the production reads it before it
// writes the day the units expire.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type ShelfLifeReader interface {
	// List returns the corrections of the business, in the order of the day
	// they were declared and of their identity.
	//
	// The list holds the corrections of the business and of no other one,
	// and it is empty when the owner corrected nothing.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]ShelfLifeCorrection, error)
}

// UnitRepository is the port of the storage of the units of the productions.
//
// The unit is a row of the production that made it: the storage reads it
// with the production and the recipe it belongs to, because the expiry is
// read for one business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type UnitRepository interface {
	// List returns the units of the productions of the business, with the
	// production they came out of and the recipe they were made of.
	//
	// The list holds the units of the business and of no other one, and
	// it is empty when the business made no production yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]ExpiringUnit, error)

	// WriteExpiry writes the day the units expire.
	//
	// The rule of the domain: the storage writes the day the domain
	// computed, and it writes it only when the unit still expires on the
	// day the domain read: a unit that changed under the correction is
	// refused.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the unit is not there, and ErrNoStorage
	// when the storage is not open.
	WriteExpiry(ctx context.Context, units []CorrectedUnit) error
}

// ProductionReader is the port of the storage that reads the productions
// back.
//
// The port is what a use case needs: the money of a unit is the money of the
// production it came out of, and the waste releases that money.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type ProductionReader interface {
	// List returns the productions of the business, with the units that
	// came out of them, in the order of the day they were made and of
	// their identity.
	//
	// The list holds the productions of the business and of no other one,
	// and it is empty when the business made no production yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Production, error)
}

// UnitInTheBookReader is the port of the storage that reads the units that
// are still in the book.
//
// The port is what a use case needs: the waste takes the units of the
// product that expire first, and a unit that left the book is never taken
// again.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type UnitInTheBookReader interface {
	// ListInTheBook returns the units of the productions of the business
	// that are still in the book, in the order they expire.
	//
	// The list holds the units of the business and of no other one, and
	// it is empty when the business made no production yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	ListInTheBook(ctx context.Context, own business.ID) ([]ExpiringUnit, error)
}

// WasteRepository is the port of the storage of the wastes.
//
// The port is what a use case needs: the tool of the waste writes the fact
// of what the bakery lost, with the units that left the book, and the
// storage keeps it in the book of the business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type WasteRepository interface {
	// Record writes the waste of the business with the units that left
	// the book, and answers it with the identity that the storage
	// generated.
	//
	// The units of a waste are rows of their own table, and the storage
	// writes them with it: a waste without its units is not a waste.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the business or the recipe is not there,
	// and ErrNoStorage when the storage is not open.
	Record(ctx context.Context, waste Waste) (Waste, error)
}

// SaleRepository is the port of the storage of the sales.
//
// The port is what a use case needs: the tool of the sale writes the fact
// of what the bakery sold, with the units that left the book, and the
// storage keeps it in the book of the business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type SaleRepository interface {
	// Record writes the sale of the business with the units that left
	// the book, and answers it with the identity that the storage
	// generated.
	//
	// The units of a sale are rows of the table of the exits of the
	// book, and the storage writes them with it: a sale without its
	// units is not a sale, and a unit that left the book never leaves
	// it again.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the business or the recipe is not there,
	// and ErrNoStorage when the storage is not open.
	Record(ctx context.Context, sale Sale) (Sale, error)
}
