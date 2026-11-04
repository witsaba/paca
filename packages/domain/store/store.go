// Package store holds the port of a transaction of the storage.
//
// The work of a use case that spans more than one row of more than one table
// is written whole or not at all: a person whose book is open and whose
// accounts are half of them is a person PACA cannot tell the truth about.
//
// The port is what a use case needs: it hands the work over and the storage
// decides when the transaction opens and when it closes. The caller never
// sees the transaction, and never commits or rolls back by itself.
package store

import "context"

// UnitOfWork is the port of a transaction of the storage.
//
// The work the caller hands over runs inside one transaction. The error of
// the work is the error of the call: the transaction is rolled back and the
// error travels as it is.
type UnitOfWork interface {
	// Do runs the work in one transaction.
	//
	// The error is the error of the work, and the error of a transaction
	// that could not be opened or committed.
	Do(ctx context.Context, work func(ctx context.Context) error) error
}
