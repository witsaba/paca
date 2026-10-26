package mcpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
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
	if len(tools.Tools) != 0 {
		t.Fatalf("the server exposes %d tools, want none yet", len(tools.Tools))
	}
}

func TestTheTransportExposesTheSDKServer(t *testing.T) {
	t.Parallel()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:    "paca-mcp-server",
		ServiceVersion: "0.1.0",
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	if transport.SDKServer() == nil {
		t.Fatal("the transport does not expose the SDK server")
	}
}
