package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/packages/database/internal/db"
)

// The names of the attributes the spans of the statements carry.
const (
	// attributeDBSystem names the database the statement ran on.
	attributeDBSystem = "db.system"

	// attributeDBNamespace names the schema of the statement.
	attributeDBNamespace = "db.namespace"

	// attributeDBStatement names the statement itself.
	attributeDBStatement = "db.statement"

	// attributeErrorType names the type of the error that ended it.
	attributeErrorType = "error.type"
)

// The values of the attributes of the statements.
const (
	// systemPostgreSQL is the database the library speaks to.
	systemPostgreSQL = "postgresql"

	// namespacePublic is the schema the tables of the monorepository live in.
	namespacePublic = "public"

	// spanQuery is the name of the span of a statement.
	spanQuery = "pg.query"
)

// tracedDBTX decorates the door of the generated queries with the spans of
// the tracing.
//
// The seam is the one the generator owns: every query of every repository
// of the library goes through it, so one decorator covers the statements
// that exist and the ones that will be written tomorrow.
//
// The statement is the SQL of the generated code, and the generated code
// always binds the values: the span of a statement carries the shape of the
// query and never a value of it.
type tracedDBTX struct {
	inner  db.DBTX
	tracer trace.Tracer
}

// Exec runs the statement and answers its tag.
func (t *tracedDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	ctx, span := t.start(ctx, sql)
	defer span.End()

	tag, err := t.inner.Exec(ctx, sql, args...)
	if err != nil {
		recordFailure(span, err)
	}

	return tag, err
}

// Query runs the statement and answers its rows.
func (t *tracedDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	ctx, span := t.start(ctx, sql)
	defer span.End()

	rows, err := t.inner.Query(ctx, sql, args...)
	if err != nil {
		recordFailure(span, err)
	}

	return rows, err
}

// QueryRow runs the statement and answers its row.
//
// The row is read after the call returns, so the span of the statement
// ends when the row is read: the error of the read belongs to the span.
// The generated code always reads the row it asks for; a row that is never
// read leaves the span open, and the statement it belongs to was never
// finished either.
func (t *tracedDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	ctx, span := t.start(ctx, sql)

	return &tracedRow{row: t.inner.QueryRow(ctx, sql, args...), span: span}
}

// start opens the span of a statement.
func (t *tracedDBTX) start(ctx context.Context, sql string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, spanQuery,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String(attributeDBSystem, systemPostgreSQL),
			attribute.String(attributeDBNamespace, namespacePublic),
			attribute.String(attributeDBStatement, sql),
		),
	)
}

// tracedRow is a row that ends the span of its statement when it is read.
type tracedRow struct {
	row  pgx.Row
	span trace.Span
}

// Scan reads the row and ends the span of the statement.
func (r *tracedRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if err != nil {
		recordFailure(r.span, err)
	}
	r.span.End()

	return err
}

// recordFailure records a failure on a span.
//
// Only the type of the error is recorded: the message of an error of the
// database carries the values it refuses, and the traces are not the place
// for the data of the people who use the product.
func recordFailure(span trace.Span, err error) {
	failure := fmt.Sprintf("%T", err)

	span.SetStatus(codes.Error, failure)
	span.SetAttributes(attribute.String(attributeErrorType, failure))
}
