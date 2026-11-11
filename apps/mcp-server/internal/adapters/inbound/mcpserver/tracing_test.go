package mcpserver_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
)

// recordingExporter keeps the spans of the test.
//
// The exporter of the SDK clears its spans when it is shut down, so a test
// that reads them after the shutdown reads nothing. This one keeps them.
type recordingExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

// ExportSpans keeps the spans it is given.
func (e *recordingExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.spans = append(e.spans, spans...)

	return nil
}

// Shutdown keeps nothing: the spans of the test are read from the exporter.
func (e *recordingExporter) Shutdown(context.Context) error {
	return nil
}

// theSpans answers the spans the exporter kept.
func (e *recordingExporter) theSpans() []sdktrace.ReadOnlySpan {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]sdktrace.ReadOnlySpan(nil), e.spans...)
}

// theSpanWithID answers the span with the given identifier, or nil.
func (e *recordingExporter) theSpanWithID(id trace.SpanID) sdktrace.ReadOnlySpan {
	for _, span := range e.theSpans() {
		if span.SpanContext().SpanID() == id {
			return span
		}
	}

	return nil
}

// theSpanNamed answers the span with the given name, and whether it was
// there.
func (e *recordingExporter) theSpanNamed(name string) (sdktrace.ReadOnlySpan, bool) {
	for _, span := range e.theSpans() {
		if span.Name() == name {
			return span, true
		}
	}

	return nil, false
}

// tracedSession starts the transport with the tracing on, and connects a
// client to it.
func tracedSession(t *testing.T, onboarding ports.OnboardingCompleter) (*sdk.ClientSession, *recordingExporter) {
	t.Helper()

	exporter := &recordingExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		TracerProvider:      provider,
		MaxRequestBodyBytes: 1 << 20,
		Onboarding:          onboarding,
	})

	// The handler of the transport is wrapped the way the router of the
	// service wraps it: the span of the request is the parent of the span
	// of the call.
	server := httptest.NewServer(otelhttp.NewHandler(transport.Handler(), "http",
		otelhttp.WithTracerProvider(provider)))
	t.Cleanup(server.Close)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)
	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: server.URL}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the transport: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Logf("the session did not close: %v", err)
		}
	})

	return session, exporter
}

func TestTheCallOfAToolLeavesASpanNamedAfterTheTool(t *testing.T) {
	t.Parallel()

	session, exporter := tracedSession(t, &stubOnboarding{completed: theAnswerOfTheOnboarding()})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCompleteOnboarding,
		Arguments: map[string]any{
			"document_id": "1234567",
			"full_name":   "Ana María Pérez",
			"phone":       "+57 321-253-9984",
			"email":       "ana@example.com",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	span, ok := exporter.theSpanNamed(mcpserver.ToolCompleteOnboarding)
	if !ok {
		t.Fatalf("the call of the tool left no span named %q", mcpserver.ToolCompleteOnboarding)
	}

	if span.SpanKind() != trace.SpanKindServer {
		t.Errorf("the kind of the span is %v, want %v", span.SpanKind(), trace.SpanKindServer)
	}

	attributes := theAttributesOf(span)
	if attributes["mcp.method.name"] != "tools/call" {
		t.Errorf("the method of the span is %q, want %q", attributes["mcp.method.name"], "tools/call")
	}
	if attributes["mcp.tool.name"] != mcpserver.ToolCompleteOnboarding {
		t.Errorf("the tool of the span is %q, want %q", attributes["mcp.tool.name"], mcpserver.ToolCompleteOnboarding)
	}

	// The span of the call is the child of the span of the request that
	// carried it: the parent is the span of the transport.
	parent := exporter.theSpanWithID(span.Parent().SpanID())
	if parent == nil {
		t.Fatalf("the span of the call has no parent in the store: %s", span.Parent().SpanID())
	}
	if parent.SpanKind() != trace.SpanKindServer {
		t.Errorf("the kind of the parent of the span of the call is %v, want %v", parent.SpanKind(), trace.SpanKindServer)
	}
	if !strings.Contains(parent.Name(), "POST") {
		t.Errorf("the parent of the span of the call is %q, want the span of the request", parent.Name())
	}

	// The span carries no argument and no result: the data of the people
	// never reaches the store of the traces.
	for key, value := range attributes {
		for _, secret := range []string{"Ana María Pérez", "+57 321-253-9984", "ana@example.com", "1234567"} {
			if strings.Contains(value, secret) {
				t.Errorf("the attribute %q carries the data of the person: %q", key, value)
			}
		}
	}
}

func TestAFailingToolLeavesTheFailureInTheSpan(t *testing.T) {
	t.Parallel()

	session, exporter := tracedSession(t, &stubOnboarding{
		completed: theAnswerOfTheOnboarding(),
		err:       errors.New("the person already holds a personal business"),
	})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCompleteOnboarding,
		Arguments: map[string]any{
			"document_id": "1234567",
			"full_name":   "Ana María Pérez",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatalf("the tool answered no error: %s", textOf(t, result))
	}

	span, ok := exporter.theSpanNamed(mcpserver.ToolCompleteOnboarding)
	if !ok {
		t.Fatalf("the call of the tool left no span named %q", mcpserver.ToolCompleteOnboarding)
	}

	attributes := theAttributesOf(span)
	if attributes["error.type"] == "" {
		t.Errorf("the span of the failed call carries no error type: %v", attributes)
	}
	if span.Status().Code.String() != "Error" {
		t.Errorf("the status of the span is %v, want Error", span.Status())
	}
}

func TestAMethodThatCarriesNoWorkLeavesNoSpan(t *testing.T) {
	t.Parallel()

	session, exporter := tracedSession(t, &stubOnboarding{completed: theAnswerOfTheOnboarding()})

	// The list of the tools is a method of the protocol that carries no
	// work: it reads nothing and writes nothing.
	if _, err := session.ListTools(t.Context(), nil); err != nil {
		t.Fatalf("the list of the tools failed: %v", err)
	}

	for _, span := range exporter.theSpans() {
		if span.Name() == "tools/list" {
			t.Errorf("the list of the tools left a span named %q", span.Name())
		}
		if span.Name() == mcpserver.ToolCompleteOnboarding {
			t.Errorf("the list of the tools left the span of a tool: %q", span.Name())
		}
	}
}

func TestATransportWithoutATracerLeavesNoSpan(t *testing.T) {
	t.Parallel()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		Onboarding:          &stubOnboarding{completed: theAnswerOfTheOnboarding()},
	})

	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)
	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: server.URL}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the transport: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Logf("the session did not close: %v", err)
		}
	})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCompleteOnboarding,
		Arguments: map[string]any{
			"document_id": "1234567",
			"full_name":   "Ana María Pérez",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}
}

// theAttributesOf answers the attributes of a span, as text.
func theAttributesOf(span sdktrace.ReadOnlySpan) map[string]string {
	attributes := make(map[string]string)

	for _, attribute := range span.Attributes() {
		attributes[string(attribute.Key)] = attribute.Value.String()
	}

	return attributes
}
