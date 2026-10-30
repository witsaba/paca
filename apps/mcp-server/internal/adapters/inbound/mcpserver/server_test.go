package mcpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
)

func TestTheTransportAnswersAnMCPClient(t *testing.T) {
	t.Parallel()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
	})

	server := httptest.NewServer(transport.Handler())
	defer server.Close()

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)

	ctx := context.Background()
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: server.URL}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the transport: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}
	if len(tools.Tools) != 5 {
		t.Fatalf("the server exposes %d tools, want the five of the users", len(tools.Tools))
	}
}

// TestTheTransportServesTheOriginsItTrusts proves that the rule of the
// Origin is the one the service states: the origins it names are served, and
// the others are refused.
func TestTheTransportServesTheOriginsItTrusts(t *testing.T) {
	t.Parallel()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:    "paca-mcp-server",
		ServiceVersion: "0.1.0",
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		TrustedOrigins: []string{"https://console.example.com"},
	})

	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)

	tests := []struct {
		name      string
		origin    string
		forbidden bool
	}{
		{name: "an origin the service trusts", origin: "https://console.example.com"},
		{name: "an origin the service does not trust", origin: "https://another.example.com", forbidden: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request, err := http.NewRequest(http.MethodPost, server.URL+"/mcp", strings.NewReader("{}"))
			if err != nil {
				t.Fatalf("the request could not be built: %v", err)
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", test.origin)

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatalf("the request failed: %v", err)
			}
			defer response.Body.Close()

			// The body of the request is not a message of the protocol,
			// so the transport answers that it does not understand it: the
			// answer is not a refusal.
			if test.forbidden && response.StatusCode != http.StatusForbidden {
				t.Fatalf("the status is %d, want %d", response.StatusCode, http.StatusForbidden)
			}
			if !test.forbidden && response.StatusCode == http.StatusForbidden {
				t.Fatalf("the status is %d, want the request to reach the protocol", response.StatusCode)
			}
		})
	}
}
