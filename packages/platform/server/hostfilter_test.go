package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/witsaba/paca/packages/platform/server"
)

// theHandler answers every request it receives.
func theHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("served"))
	})
}

func TestTheHostFilterServesAHostOfTheAllowlist(t *testing.T) {
	t.Parallel()

	handler := server.HostFilter([]string{"paca-mcp-server", "localhost"})(theHandler())

	request := httptest.NewRequest(http.MethodGet, "http://paca-mcp-server:8081/mcp", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", response.Code, http.StatusOK)
	}
}

func TestTheHostFilterRefusesAForeignHost(t *testing.T) {
	t.Parallel()

	handler := server.HostFilter([]string{"paca-mcp-server"})(theHandler())

	request := httptest.NewRequest(http.MethodGet, "http://another.example.com/mcp", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("the status is %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestTheHostFilterIgnoresTheCaseOfTheHost(t *testing.T) {
	t.Parallel()

	handler := server.HostFilter([]string{"PACA-MCP-Server"})(theHandler())

	request := httptest.NewRequest(http.MethodGet, "http://paca-mcp-server/mcp", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", response.Code, http.StatusOK)
	}
}

// TestTheHostFilterIgnoresThePortOfTheRequest proves that the port of the
// request is not part of the name of the host: the allowlist names the
// hosts of the service, and the port the request carries is the port of the
// service.
func TestTheHostFilterIgnoresThePortOfTheRequest(t *testing.T) {
	t.Parallel()

	handler := server.HostFilter([]string{"paca-mcp-server"})(theHandler())

	request := httptest.NewRequest(http.MethodGet, "http://paca-mcp-server:9090/mcp", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("the status is %d, want %d", response.Code, http.StatusOK)
	}
}

func TestTheHostFilterRefusesARequestWithNoHost(t *testing.T) {
	t.Parallel()

	handler := server.HostFilter([]string{"paca-mcp-server"})(theHandler())

	request := httptest.NewRequest(http.MethodGet, "http://paca-mcp-server/mcp", nil)
	request.Host = ""
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("the status is %d, want %d", response.Code, http.StatusForbidden)
	}
}
