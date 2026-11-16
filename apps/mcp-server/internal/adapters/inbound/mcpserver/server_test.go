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
	if len(tools.Tools) != 52 {
		t.Fatalf("the server exposes %d tools, want the five of the people and of the onboarding, the two of the businesses, the three of the book, the four of the debts, the four of the utterance, the three of the catalog, the one of the recipes, the two of the reference prices, the one of the stock, the one of the cost of a batch, the one of the capacity of a weekend, the one of the labor, the one of the overhead, the one of the real price, the one of the margin, the one of the production, the two of the expiry, the one of the waste, the one of the sale, the one of the exit, the one of the rotation, the two of the parties, the two of the invoices, the two of the price list, the two of the orders, the one of the route, the three of the commitments and the two of the log of the lessons", len(tools.Tools))
	}
}

// TestTheTransportAnswersOnlyThePostMethod proves the method contract of the
// transport: it is stateless, so it serves the messages of the protocol with
// POST, and every other method is answered with 405 and the header that names
// the method that is allowed, as RFC 9110 §15.5.6 asks.
//
// The protocol of MCP lets a server offer a stream of server-initiated
// messages with GET, and asks a server that does not offer it to answer 405.
// A stateless server has no server-initiated message to send and no session to
// end, so it does not offer the stream and refuses DELETE the same way.
func TestTheTransportAnswersOnlyThePostMethod(t *testing.T) {
	t.Parallel()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
	})

	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			request, err := http.NewRequest(method, server.URL+"/mcp", nil)
			if err != nil {
				t.Fatalf("the request could not be built: %v", err)
			}

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatalf("the request failed: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("the status is %d, want %d", response.StatusCode, http.StatusMethodNotAllowed)
			}
			if got := response.Header.Get("Allow"); got != http.MethodPost {
				t.Fatalf("the Allow header is %q, want %q", got, http.MethodPost)
			}
		})
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
