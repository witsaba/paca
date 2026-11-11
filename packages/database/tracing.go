package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3/lock"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// The names of the spans of the migrator of the schema.
const (
	// spanMigrate is the span of the application of the pending migrations.
	spanMigrate = "schema.migrate"

	// spanLock is the span of the wait for the lock of the migrations.
	spanLock = "schema.lock"

	// spanVersion is the span of the reading of the version of the schema.
	spanVersion = "schema.version"

	// spanPending is the span of the reading of the pending migrations.
	spanPending = "schema.pending"
)

// The names of the attributes the spans of the migrator carry.
const (
	// attributeSchemaApplied names the migrations the run applied.
	attributeSchemaApplied = "schema.applied"

	// attributeSchemaVersion names the version of the schema.
	attributeSchemaVersion = "schema.version"

	// attributeSchemaPending names the migrations that are still waiting.
	attributeSchemaPending = "schema.pending"

	// attributeErrorType names the type of the error that ended the work.
	attributeErrorType = "error.type"
)

// MigratorOption configures the migrator of the schema.
type MigratorOption func(*Migrator)

// WithMigratorTracer makes the migrator trace the work it does.
//
// The tracer is the API of the tracing: the library builds no provider,
// exports no span and reads no environment variable. The composition root
// of the service owns that, and it hands the tracer over here.
//
// A migrator without a tracer traces nothing: the work of the schema
// leaves no span behind, and the behaviour of the library is the same.
func WithMigratorTracer(tracer trace.Tracer) MigratorOption {
	return func(m *Migrator) {
		m.tracer = tracer
	}
}

// tracedSessionLocker is the lock of the migrations, with the span of the
// wait.
//
// The lock is where the service waits: two replicas of the writer take
// turns, and the one that waits says nothing until the other one is done.
// The span says how long the wait was.
type tracedSessionLocker struct {
	inner  lock.SessionLocker
	tracer trace.Tracer
}

// SessionLock takes the lock and traces the wait.
func (l *tracedSessionLocker) SessionLock(ctx context.Context, conn *sql.Conn) error {
	ctx, span := l.tracer.Start(ctx, spanLock, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	err := l.inner.SessionLock(ctx, conn)
	if err != nil {
		recordFailure(span, err)
	}

	return err
}

// SessionUnlock releases the lock.
func (l *tracedSessionLocker) SessionUnlock(ctx context.Context, conn *sql.Conn) error {
	return l.inner.SessionUnlock(ctx, conn)
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
