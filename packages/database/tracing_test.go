package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// fakeSessionLocker is the lock of the test.
type fakeSessionLocker struct {
	locked   bool
	unlocked bool
	err      error
}

// SessionLock takes the lock of the test.
func (l *fakeSessionLocker) SessionLock(context.Context, *sql.Conn) error {
	if l.err != nil {
		return l.err
	}

	l.locked = true

	return nil
}

// SessionUnlock releases the lock of the test.
func (l *fakeSessionLocker) SessionUnlock(context.Context, *sql.Conn) error {
	l.unlocked = true

	return nil
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

func TestTheWaitForTheLockLeavesASpan(t *testing.T) {
	t.Parallel()

	exporter := &recordingExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	inner := &fakeSessionLocker{}
	locker := &tracedSessionLocker{inner: inner, tracer: provider.Tracer("paca/database")}

	parentCtx, parent := provider.Tracer("paca/database").Start(t.Context(), "the boot")
	defer parent.End()

	if err := locker.SessionLock(parentCtx, nil); err != nil {
		t.Fatalf("the lock could not be taken: %v", err)
	}

	if !inner.locked {
		t.Errorf("the lock was not taken")
	}

	if len(exporter.spans) != 1 {
		t.Fatalf("the wait left %d spans, want 1", len(exporter.spans))
	}

	span := exporter.spans[0]
	if span.Name() != spanLock {
		t.Errorf("the name of the span is %q, want %q", span.Name(), spanLock)
	}
	if span.SpanKind() != trace.SpanKindClient {
		t.Errorf("the kind of the span is %v, want %v", span.SpanKind(), trace.SpanKindClient)
	}
	if span.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Errorf("the parent of the span is %s, want %s", span.Parent().SpanID(), parent.SpanContext().SpanID())
	}
}

func TestAFailingLockLeavesTheTypeOfTheFailureInItsSpan(t *testing.T) {
	t.Parallel()

	exporter := &recordingExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	inner := &fakeSessionLocker{err: errors.New("the lock is held by another writer")}
	locker := &tracedSessionLocker{inner: inner, tracer: provider.Tracer("paca/database")}

	if err := locker.SessionLock(t.Context(), nil); err == nil {
		t.Fatalf("the lock answered no error")
	}

	if len(exporter.spans) != 1 {
		t.Fatalf("the wait left %d spans, want 1", len(exporter.spans))
	}

	span := exporter.spans[0]

	found := false
	for _, attribute := range span.Attributes() {
		if string(attribute.Key) == attributeErrorType && attribute.Value.String() != "" {
			found = true
		}
	}
	if !found {
		t.Errorf("the span of the failed wait carries no error type")
	}
	if span.Status().Code.String() != "Error" {
		t.Errorf("the status of the span is %v, want Error", span.Status())
	}
}
