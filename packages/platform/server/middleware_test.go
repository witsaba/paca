package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/packages/platform/server"
)

// theHealthPath is the path the container runtime asks.
const theHealthPath = "/healthz"

// theWorkPath is the path of the work of the service.
const theWorkPath = "/work"

// quietLogger is the logger of the tests that do not read the lines.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// loggedRequest is the line the middleware writes for a request.
type loggedRequest struct {
	Method   string  `json:"method"`
	Path     string  `json:"path"`
	Status   int     `json:"status"`
	Duration float64 `json:"duration"`
}

// recordingLogger returns the logger of the test and the buffer that holds
// the lines it writes.
func recordingLogger() (*slog.Logger, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	return slog.New(slog.NewJSONHandler(buffer, nil)), buffer
}

func TestTheMiddlewareSetsTheSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := server.NewMiddleware(server.MiddlewareConfig{
		Logger: quietLogger(),
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, theWorkPath, nil))

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

func TestTheMiddlewareLogsTheRequest(t *testing.T) {
	t.Parallel()

	logger, buffer := recordingLogger()
	handler := server.NewMiddleware(server.MiddlewareConfig{
		Logger: logger,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, theWorkPath, nil))

	var logged loggedRequest
	if err := json.Unmarshal(buffer.Bytes(), &logged); err != nil {
		t.Fatalf("the line of the request is not a JSON object: %v", err)
	}
	if logged.Method != http.MethodPost {
		t.Fatalf("method = %q, want %q", logged.Method, http.MethodPost)
	}
	if logged.Path != theWorkPath {
		t.Fatalf("path = %q, want %q", logged.Path, theWorkPath)
	}
	if logged.Status != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", logged.Status, http.StatusAccepted)
	}
	if logged.Duration <= 0 {
		t.Fatalf("duration = %v, want a positive duration", logged.Duration)
	}
}

func TestTheMiddlewareTurnsAPanicIntoAnInternalServerError(t *testing.T) {
	t.Parallel()

	handler := server.NewMiddleware(server.MiddlewareConfig{
		Logger: quietLogger(),
	}, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("the handler broke")
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, theWorkPath, nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestTheMiddlewareWithoutATracerProviderDoesNotTraceTheRequest(t *testing.T) {
	t.Parallel()

	var recording bool
	var valid bool
	handler := server.NewMiddleware(server.MiddlewareConfig{
		Logger: quietLogger(),
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		recording = span.IsRecording()
		valid = span.SpanContext().IsValid()
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, theWorkPath, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recording {
		t.Fatal("the request left a recording span, want the no-op provider")
	}
	if valid {
		t.Fatal("the request left a valid span context, want the no-op provider")
	}
}

func TestTheMiddlewareDoesNotTraceTheExcludedPath(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = provider.Shutdown(context.Background()) }()

	handler := server.NewMiddleware(server.MiddlewareConfig{
		Logger:         discardLogger(),
		TracerProvider: provider,
		UntracedPaths:  []string{theHealthPath},
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, theHealthPath, nil))

	if spans := exporter.GetSpans(); len(spans) != 0 {
		t.Fatalf("the exporter holds %d spans, want none", len(spans))
	}
}

func TestTheMiddlewareTracesTheOtherPaths(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = provider.Shutdown(context.Background()) }()

	handler := server.NewMiddleware(server.MiddlewareConfig{
		Logger:         discardLogger(),
		TracerProvider: provider,
		UntracedPaths:  []string{theHealthPath},
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, theWorkPath, nil))

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("the exporter holds %d spans, want one", len(spans))
	}
	if spans[0].SpanKind != trace.SpanKindServer {
		t.Fatalf("span kind = %v, want %v", spans[0].SpanKind, trace.SpanKindServer)
	}
}
