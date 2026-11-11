package bootstrap

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/apps/db-admin/internal/application"
	"github.com/witsaba/paca/apps/db-admin/internal/ports"
	"github.com/witsaba/paca/packages/domain/schema"
)

// fakeMigrator is the migrator of the test.
//
// It answers the state of the schema and it tells whether the context it
// received carries the span of the boot.
type fakeMigrator struct {
	calls   int
	spanCtx bool
}

// Up applies the migrations of the test.
func (m *fakeMigrator) Up(ctx context.Context) ([]schema.Migration, error) {
	m.calls++
	m.spanCtx = trace.SpanContextFromContext(ctx).IsValid()

	return []schema.Migration{{Version: 36, Name: "add_the_orders.sql", Applied: true}}, nil
}

// Pending answers the pending migrations of the test.
func (m *fakeMigrator) Pending(context.Context) (bool, error) {
	return false, nil
}

// Version answers the version of the schema of the test.
func (m *fakeMigrator) Version(context.Context) (int64, error) {
	return 36, nil
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

func TestTheBootOfTheSchemaLeavesARootTrace(t *testing.T) {
	t.Parallel()

	exporter := &recordingExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	migrator := &fakeMigrator{}
	service, err := application.NewMigrateService(migrator)
	if err != nil {
		t.Fatalf("the migration service could not be built: %v", err)
	}

	state, err := applySchema(t.Context(), provider.Tracer("paca/db-admin"), service)
	if err != nil {
		t.Fatalf("the schema could not be applied: %v", err)
	}
	if state.Version != 36 {
		t.Errorf("the version of the schema is %d, want 36", state.Version)
	}

	if len(exporter.spans) != 1 {
		t.Fatalf("the boot left %d spans, want 1", len(exporter.spans))
	}

	span := exporter.spans[0]
	if span.Name() != spanMigrate {
		t.Errorf("the name of the span is %q, want %q", span.Name(), spanMigrate)
	}
	if span.Parent().IsValid() {
		t.Errorf("the span of the boot has a parent: %s", span.Parent().SpanID())
	}

	// The work of the migrations runs under the span of the boot: the
	// migrator receives a context that carries it.
	if !migrator.spanCtx {
		t.Errorf("the migrator received a context without the span of the boot")
	}
	if migrator.calls != 1 {
		t.Errorf("the migrator ran %d times, want 1", migrator.calls)
	}
}

func TestTheBootWithoutATracerLeavesNoSpan(t *testing.T) {
	t.Parallel()

	migrator := &fakeMigrator{}
	service, err := application.NewMigrateService(migrator)
	if err != nil {
		t.Fatalf("the migration service could not be built: %v", err)
	}

	tracer := tracerProviderOf(nil).Tracer("paca/db-admin")

	if _, err := applySchema(t.Context(), tracer, service); err != nil {
		t.Fatalf("the schema could not be applied: %v", err)
	}

	if migrator.calls != 1 {
		t.Errorf("the migrator ran %d times, want 1", migrator.calls)
	}
}

// theMigratorIsThePort keeps the port of the service in the test.
var _ ports.Migrator = (*fakeMigrator)(nil)
