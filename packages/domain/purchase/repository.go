package purchase

import "context"

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
	Find(ctx context.Context, id ID) (Draft, error)
}
