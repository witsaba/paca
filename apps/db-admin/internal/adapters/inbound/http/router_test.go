package http_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	adapter "github.com/witsaba/paca/apps/db-admin/internal/adapters/inbound/http"
	"github.com/witsaba/paca/packages/domain/health"
)

// discardLogger is the logger of the test.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestTheRouterServesTheHealthEndpoint(t *testing.T) {
	t.Parallel()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{report: health.Report{
			Identity: health.Identity{Name: "paca-db-admin", Version: "0.1.0"},
			Status:   health.StatusOK,
		}},
		Logger: discardLogger(),
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, adapter.HealthPath, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestTheRouterRejectsTheWrongMethodOnTheHealthEndpoint(t *testing.T) {
	t.Parallel()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{},
		Logger: discardLogger(),
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, adapter.HealthPath, nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestTheRouterAnswersNotFoundForAnUnknownPath(t *testing.T) {
	t.Parallel()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{},
		Logger: discardLogger(),
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestTheRouterTurnsABrokenRequestIntoAnInternalServerError(t *testing.T) {
	t.Parallel()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{err: context.Canceled},
		Logger: discardLogger(),
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, adapter.HealthPath, nil))

	// The health endpoint answers 503 when the use case fails, and the
	// router never turns it into a panic.
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestTheRouterTracesTheRequest(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = provider.Shutdown(context.Background()) }()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health:         &fakeQuerier{},
		Logger:         discardLogger(),
		TracerProvider: provider,
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("the exporter holds %d spans, want one", len(spans))
	}
	// The span of a request that matches no route carries the method and
	// no path: there is no route to name it after.
	if spans[0].Name != "GET" {
		t.Fatalf("span name = %q, want %q", spans[0].Name, "GET")
	}
	if spans[0].SpanKind != trace.SpanKindServer {
		t.Fatalf("span kind = %v, want %v", spans[0].SpanKind, trace.SpanKindServer)
	}
}

func TestTheRouterDoesNotTraceTheHealthEndpoint(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = provider.Shutdown(context.Background()) }()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health:         &fakeQuerier{},
		Logger:         discardLogger(),
		TracerProvider: provider,
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, adapter.HealthPath, nil))

	if spans := exporter.GetSpans(); len(spans) != 0 {
		t.Fatalf("the exporter holds %d spans, want none", len(spans))
	}
}

func TestTheRouterKeepsTheTraceOfTheCaller(t *testing.T) {
	// The test is not parallel: it sets the propagator of the process,
	// which the instrumentation of the transport reads.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = provider.Shutdown(context.Background()) }()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health:         &fakeQuerier{},
		Logger:         discardLogger(),
		TracerProvider: provider,
	})

	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("the exporter holds %d spans, want one", len(spans))
	}
	if spans[0].Parent.SpanID() != [8]byte{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7} {
		t.Fatalf("parent span = %v, want the span of the caller", spans[0].Parent.SpanID())
	}
}

func TestTheRouterSetsTheSecurityHeaders(t *testing.T) {
	t.Parallel()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{},
		Logger: discardLogger(),
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, adapter.HealthPath, nil))

	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
	if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want %q", got, "DENY")
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want %q", got, "no-referrer")
	}
}

func TestTheRouterWithoutATracerProviderStillServes(t *testing.T) {
	t.Parallel()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{report: health.Report{
			Identity: health.Identity{Name: "paca-db-admin", Version: "0.1.0"},
			Status:   health.StatusOK,
		}},
		Logger: discardLogger(),
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, adapter.HealthPath, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
