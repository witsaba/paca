package sqlite

import (
	"context"
	"database/sql"

	// The pure-Go driver. The package registers itself under the name "sqlite"
	// when it is loaded, which is what sql.Open asks for below.
	_ "modernc.org/sqlite"
)

// The name the driver is registered under.
const driverName = "sqlite"

// DB is the connection to the PACA database.
//
// It is a thin wrapper over the pool of the standard library: every operation
// it offers translates the failures of the driver into the typed errors of the
// domain, so that no caller ever sees a raw driver message.
type DB struct {
	pool *sql.DB
}

// Open opens the database at the given path and returns the connection.
//
// The path is a file name, or ":memory:" for a database that lives only as
// long as the connection does. The database file is created when it does not
// exist.
//
// Every connection of the pool runs with PRAGMA foreign_keys = ON. The pragma
// is set on the connection and never on the schema: SQLite stores it per
// connection, so a connection that did not set it would enforce nothing.
//
// The pool holds a single connection. SQLite allows one writer at a time, and
// a second connection would fail with "database is locked" instead of
// waiting; one connection makes that wait explicit instead of random.
func Open(ctx context.Context, path string) (*DB, error) {
	pool, err := sql.Open(driverName, dsn(path))
	if err != nil {
		return nil, translate(err)
	}

	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)

	if err := pool.PingContext(ctx); err != nil {
		// The pool never opened the database: closing it is a formality, and
		// the failure that matters is the one of the ping.
		_ = pool.Close()

		return nil, translate(err)
	}

	return &DB{pool: pool}, nil
}

// dsn returns the data source name of the database at the given path.
//
// The pragma travels in the query of the name, because the driver applies it
// on every connection it opens. A path that carries a question mark of its
// own cannot be used: the driver would read the rest of the path as a query.
func dsn(path string) string {
	return path + "?_pragma=foreign_keys(1)"
}

// Close closes the database. A closed connection refuses every operation.
func (d *DB) Close() error {
	return d.pool.Close()
}

// ExecContext runs a statement that returns no rows.
func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := d.pool.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, translate(err)
	}

	return result, nil
}

// QueryContext runs a statement that returns rows.
func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := d.pool.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, translate(err)
	}

	return &Rows{rows: rows}, nil
}

// QueryRowContext runs a statement that returns at most one row.
//
// The row may not exist: the failure then comes out of Scan, as a not-found
// error.
func (d *DB) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	return &Row{row: d.pool.QueryRowContext(ctx, query, args...)}
}

// BeginTx starts a transaction.
func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := d.pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, translate(err)
	}

	return &Tx{tx: tx}, nil
}

// Rows is a result set whose failures are translated.
type Rows struct {
	rows *sql.Rows
}

// Next advances to the next row, and reports whether there is one.
func (r *Rows) Next() bool {
	return r.rows.Next()
}

// Scan reads the values of the current row into the destinations.
func (r *Rows) Scan(dest ...any) error {
	return translate(r.rows.Scan(dest...))
}

// Err returns the failure that stopped the iteration, if there was one.
func (r *Rows) Err() error {
	return translate(r.rows.Err())
}

// Close closes the result set.
func (r *Rows) Close() error {
	return r.rows.Close()
}

// Row is a single row of a result set, whose failures are translated.
type Row struct {
	row *sql.Row
}

// Scan reads the values of the row into the destinations.
//
// A row that does not exist is a not-found error.
func (r *Row) Scan(dest ...any) error {
	return translate(r.row.Scan(dest...))
}

// Tx is a transaction whose failures are translated.
type Tx struct {
	tx *sql.Tx
}

// ExecContext runs a statement that returns no rows.
func (t *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, translate(err)
	}

	return result, nil
}

// QueryContext runs a statement that returns rows.
func (t *Tx) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, translate(err)
	}

	return &Rows{rows: rows}, nil
}

// QueryRowContext runs a statement that returns at most one row.
func (t *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	return &Row{row: t.tx.QueryRowContext(ctx, query, args...)}
}

// Commit commits the transaction.
func (t *Tx) Commit() error {
	return translate(t.tx.Commit())
}

// Rollback aborts the transaction.
func (t *Tx) Rollback() error {
	return translate(t.tx.Rollback())
}
