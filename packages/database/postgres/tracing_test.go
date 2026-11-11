package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/witsaba/paca/packages/database/internal/db"
)

// fakeDBTX is the door of the generated queries that the test answers.
type fakeDBTX struct {
	execErr  error
	queryErr error
	rowErr   error
	executed int
	queried  int
}

// Exec answers the call of the test.
func (f *fakeDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	f.executed++
	if f.execErr != nil {
		return pgconn.CommandTag{}, f.execErr
	}

	return pgconn.CommandTag{}, nil
}

// Query answers the call of the test.
func (f *fakeDBTX) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	f.queried++
	if f.queryErr != nil {
		return nil, f.queryErr
	}

	return nil, nil
}

// QueryRow answers the call of the test.
func (f *fakeDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return &fakeRow{err: f.rowErr}
}

// fakeRow is the row of the test.
type fakeRow struct {
	err error
}

// Scan answers the read of the test.
func (r *fakeRow) Scan(...any) error {
	return r.err
}

// recordingExporter keeps the spans of the test.
type recordingExporter struct {
	spans []sdktrace.ReadOnlySpan
}

// ExportSpans keeps the spans it is given.
func (e *recordingExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.spans = append(e.spans, spans...)
	return nil
}

// Shutdown keeps nothing: the spans are read from the exporter.
func (e *recordingExporter) Shutdown(context.Context) error {
	return nil
}

// theSpan answers the span the exporter kept, and whether it was there.
func (e *recordingExporter) theSpan(t *testing.T) sdktrace.ReadOnlySpan {
	t.Helper()

	if len(e.spans) != 1 {
		t.Fatalf("the exporter holds %d spans, want 1", len(e.spans))
	}

	return e.spans[0]
}

// theAttributesOf answers the attributes of a span, as text.
func theAttributesOf(span sdktrace.ReadOnlySpan) map[string]string {
	attributes := make(map[string]string)

	for _, attribute := range span.Attributes() {
		attributes[string(attribute.Key)] = attribute.Value.String()
	}

	return attributes
}

func TestAStatementLeavesASpanOfItsOwn(t *testing.T) {
	t.Parallel()

	exporter := &recordingExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tracer := provider.Tracer("paca/database")

	inner := &fakeDBTX{}
	door := &tracedDBTX{inner: inner, tracer: tracer}

	parentCtx, parent := tracer.Start(t.Context(), "the work")
	defer parent.End()

	if _, err := door.Exec(parentCtx, "INSERT INTO accounts (id) VALUES ($1)", 7); err != nil {
		t.Fatalf("the statement failed: %v", err)
	}

	span := exporter.theSpan(t)

	if span.Name() != spanQuery {
		t.Errorf("the name of the span is %q, want %q", span.Name(), spanQuery)
	}
	if span.SpanKind() != trace.SpanKindClient {
		t.Errorf("the kind of the span is %v, want %v", span.SpanKind(), trace.SpanKindClient)
	}
	if span.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Errorf("the parent of the span is %s, want %s", span.Parent().SpanID(), parent.SpanContext().SpanID())
	}

	attributes := theAttributesOf(span)
	if attributes[attributeDBSystem] != systemPostgreSQL {
		t.Errorf("the system of the span is %q, want %q", attributes[attributeDBSystem], systemPostgreSQL)
	}
	if attributes[attributeDBNamespace] != namespacePublic {
		t.Errorf("the namespace of the span is %q, want %q", attributes[attributeDBNamespace], namespacePublic)
	}
	if attributes[attributeDBStatement] != "INSERT INTO accounts (id) VALUES ($1)" {
		t.Errorf("the statement of the span is %q", attributes[attributeDBStatement])
	}

	if inner.executed != 1 {
		t.Errorf("the statement ran %d times, want 1", inner.executed)
	}
}

func TestAFailingStatementLeavesTheTypeOfTheFailureInItsSpan(t *testing.T) {
	t.Parallel()

	exporter := &recordingExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tracer := provider.Tracer("paca/database")

	inner := &fakeDBTX{execErr: errors.New("the relation does not exist")}
	door := &tracedDBTX{inner: inner, tracer: tracer}

	if _, err := door.Exec(t.Context(), "SELECT 1"); err == nil {
		t.Fatalf("the statement answered no error")
	}

	span := exporter.theSpan(t)

	attributes := theAttributesOf(span)
	if attributes[attributeErrorType] == "" {
		t.Errorf("the span of the failed statement carries no error type: %v", attributes)
	}
	if span.Status().Code.String() != "Error" {
		t.Errorf("the status of the span is %v, want Error", span.Status())
	}
}

func TestTheRowEndsTheSpanOfItsStatement(t *testing.T) {
	t.Parallel()

	exporter := &recordingExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tracer := provider.Tracer("paca/database")

	inner := &fakeDBTX{rowErr: errors.New("the row could not be read")}
	door := &tracedDBTX{inner: inner, tracer: tracer}

	row := door.QueryRow(t.Context(), "SELECT id FROM accounts WHERE id = $1", 7)
	if err := row.Scan(nil); err == nil {
		t.Fatalf("the row answered no error")
	}

	span := exporter.theSpan(t)

	attributes := theAttributesOf(span)
	if attributes[attributeErrorType] == "" {
		t.Errorf("the span of the failed row carries no error type: %v", attributes)
	}
}

func TestAPoolWithoutATracerTracesNothing(t *testing.T) {
	t.Parallel()

	inner := &fakeDBTX{}
	pool := &Pool{}

	door := pool.door(inner)
	if door != db.DBTX(inner) {
		t.Errorf("the door of a pool without a tracer is decorated: %T", door)
	}

	withTracer := &Pool{tracer: noop.NewTracerProvider().Tracer("paca/database")}
	if _, ok := withTracer.door(inner).(*tracedDBTX); !ok {
		t.Errorf("the door of a pool with a tracer is not decorated: %T", withTracer.door(inner))
	}
}
