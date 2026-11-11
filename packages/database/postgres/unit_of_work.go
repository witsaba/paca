package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/witsaba/paca/packages/database/internal/db"
)

// transactionKey is the key of the transaction of the storage in the
// context.
//
// The transaction travels in the context because the repositories of the
// library are bound to the pool and not to a transaction: the unit of work
// opens the transaction, and the repositories join the one they find.
type transactionKey struct{}

// UnitOfWork is the adapter of the transaction of the storage.
//
// It is the only door of the library to a transaction: a use case that
// writes more than one row in more than one table hands its work over to
// it, and the rows are written whole or not at all.
type UnitOfWork struct {
	pool *Pool
}

// NewUnitOfWork returns the transaction of the storage, bound to the pool.
//
// The unit of work needs an open pool, and the constructor refuses anything
// else: a unit of work built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil.
//
// The error is ErrNoPool when the pool is not there.
func NewUnitOfWork(pool *Pool) (*UnitOfWork, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &UnitOfWork{pool: pool}, nil
}

// Do runs the work in one transaction.
//
// The work that fails leaves the storage as it was: the transaction is
// rolled back, and the error of the work travels as it is.
func (u *UnitOfWork) Do(ctx context.Context, work func(context.Context) error) error {
	if u == nil || u.pool == nil || u.pool.pool == nil {
		return ErrNoPool
	}

	tx, err := u.pool.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("the transaction could not be opened: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := work(context.WithValue(ctx, transactionKey{}, tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("the transaction could not be committed: %w", err)
	}

	return nil
}

// transactionIn returns the transaction of the context, when it carries one.
func transactionIn(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(transactionKey{}).(pgx.Tx)
	return tx, ok
}

// queriesFor returns the queries bound to the transaction of the context, or
// to the pool when the context carries none.
//
// The repositories of the library always ask for their queries this way: a
// repository that is called inside a unit of work joins the transaction of
// that work, and a repository that is called outside one writes by itself.
func (p *Pool) queriesFor(ctx context.Context) *db.Queries {
	if p == nil || p.pool == nil {
		return nil
	}

	if tx, ok := transactionIn(ctx); ok {
		return db.New(p.door(tx))
	}

	return db.New(p.door(nil))
}
