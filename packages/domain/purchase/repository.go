package purchase

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
)

// Repository is the port of the storage of the drafts.
//
// The port is what a use case needs, not the shape of a table: it keeps a
// draft of the business and reads it back by its identity.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Save writes a draft and answers it with the identity that the
	// storage generated.
	//
	// A draft without an identity is written for the first time, and a
	// draft with one replaces the row it names: the amendment of a draft
	// is the same draft, one revision further.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, and ErrNoStorage when the storage is not open.
	Save(ctx context.Context, draft Draft) (Draft, error)

	// Find returns the draft that carries that identity.
	//
	// The error is ErrNotFound when no draft carries that identity, and
	// ErrNoStorage when the storage is not open.
	Find(ctx context.Context, id DraftID) (Draft, error)

	// Confirm moves the draft to the confirmed state and answers the draft.
	//
	// The revision is the revision the owner saw: the row must carry it,
	// and the storage refuses the confirmation of a draft that changed
	// under her eyes.
	//
	// The error is ErrNotFound when no draft carries that identity, and
	// ErrStaleRevision when the draft is not the one the owner saw.
	Confirm(ctx context.Context, id DraftID, revision int) (Draft, error)

	// Cancel moves the draft to the cancelled state and answers the draft.
	//
	// The rule of the state machine: only a draft that waits is cancelled.
	// The row must carry the state of a draft that waits, and the storage
	// refuses the cancellation of a draft that is confirmed or that is
	// cancelled already.
	//
	// The error is ErrNotFound when no draft carries that identity, and
	// ErrNotCancellable when the draft does not wait any more.
	Cancel(ctx context.Context, id DraftID) (Draft, error)

	// List returns the drafts of the business that wait for the yes of the
	// owner, in the order of their identity.
	//
	// The list holds the drafts of the business and of no other one, and
	// the domain decides which of them are still pending: a draft that
	// expired is not pending any more.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Draft, error)
}

// Recorder is the port of the storage of the purchases.
//
// A purchase is a fact: it is written once, and it never changes. The
// lines of a purchase are rows of their own table, and the storage writes
// them with it.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Recorder interface {
	// Record writes the purchase and its lines, and answers it with the
	// identity that the storage generated.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, and ErrNoStorage when the storage is not open.
	Record(ctx context.Context, purchase Purchase) (Purchase, error)
}

// LayerRepository is the port of the storage of the layers of the stock.
//
// A layer is derived from a purchase: it is written with the purchase it
// comes from, and it is read back to answer what the business holds.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type LayerRepository interface {
	// Record writes the layers of a purchase, and answers them with the
	// identities that the storage generated.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the purchase of the layer is not there, and
	// ErrNoStorage when the storage is not open.
	Record(ctx context.Context, layers []Layer) ([]Layer, error)

	// List returns the layers of a business, in the order they leave the
	// stock: the oldest layer is the first.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Layer, error)

	// WriteRemaining writes the quantity that is left of the layers of the
	// stock.
	//
	// The rule of the domain: the quantity that is left of a layer is the
	// quantity it held minus the quantity the production took from it, and
	// the quantity and its unit travel together: the remaining carries the
	// unit of the layer it belongs to, and the book never subtracts a gram
	// from a kilogram. The layers leave the stock in the order of the FIFO:
	// the oldest layer is the first one that leaves.
	//
	// The storage writes the quantity the domain computed, and it writes it
	// only when the layer still holds what the production read: a layer
	// that changed under the production is refused.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the layer is not there, and ErrNoStorage
	// when the storage is not open.
	WriteRemaining(ctx context.Context, remaining []Remaining) error
}

// ReferenceRepository is the port of the storage of the reference prices.
//
// A reference is a price fact of the business: what the market was seen at,
// where it was read and the day it was read. It is never the price of the
// ingredient: the price the owner pays comes from her own purchases.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type ReferenceRepository interface {
	// Record writes the reference and answers it with the identity that
	// the storage generated.
	//
	// The error is ErrInvalidValue when the row breaks a rule of the
	// table, ErrNotFound when the business or the ingredient of the
	// reference is not there, and ErrNoStorage when the storage is not
	// open.
	Record(ctx context.Context, reference Reference) (Reference, error)

	// List returns the references of a business, in the order of their
	// identity.
	//
	// The ingredient is optional: the zero identity answers every
	// reference of the business, and an identity answers the references of
	// that ingredient alone.
	//
	// The list is empty when the business holds no reference, and the
	// error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID, of ingredient.ID) ([]Reference, error)
}
