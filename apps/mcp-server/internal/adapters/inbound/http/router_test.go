package http_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	adapter "github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/http"
	"github.com/witsaba/paca/apps/mcp-server/internal/domain/health"
)

// countingHandler is an MCP transport that only counts the calls.
type countingHandler struct {
	calls int
}

// ServeHTTP implements http.Handler.
func (h *countingHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.calls++
	w.WriteHeader(http.StatusAccepted)
}

// panickingHandler is an MCP transport that breaks.
type panickingHandler struct{}

// ServeHTTP implements http.Handler.
func (h *panickingHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
	panic("the transport broke")
}

// discardLogger is the logger of the test.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestTheRouterServesTheHealthEndpoint(t *testing.T) {
	t.Parallel()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{report: health.Report{
			Identity: health.Identity{Name: "paca-mcp-server", Version: "0.1.0"},
			Status:   health.StatusOK,
		}},
		MCP:    &countingHandler{},
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
		MCP:    &countingHandler{},
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
		MCP:    &countingHandler{},
		Logger: discardLogger(),
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestTheRouterForwardsTheMCPTransport(t *testing.T) {
	t.Parallel()

	transport := &countingHandler{}
	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{},
		MCP:    transport,
		Logger: discardLogger(),
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, adapter.MCPPath, nil))

	if transport.calls != 1 {
		t.Fatalf("the transport was called %d times, want 1", transport.calls)
	}
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
}

func TestTheRouterTurnsABrokenTransportIntoAnInternalServerError(t *testing.T) {
	t.Parallel()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{},
		MCP:    &panickingHandler{},
		Logger: discardLogger(),
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, adapter.MCPPath, nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestTheRouterSetsTheSecurityHeaders(t *testing.T) {
	t.Parallel()

	router := adapter.NewRouter(adapter.RouterConfig{
		Health: &fakeQuerier{},
		MCP:    &countingHandler{},
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
