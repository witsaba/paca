package bootstrap_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/bootstrap"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/config"
)

// discardLogger is the logger of the test.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// startService assembles the service and serves it on a free port.
func startService(t *testing.T, version string) string {
	t.Helper()

	cfg := config.Config{
		Host:            "127.0.0.1",
		Port:            0,
		ShutdownTimeout: 5 * time.Second,
		CheckTimeout:    time.Second,
		MaxRequestBytes: 1 << 20,
		LogLevel:        slog.LevelInfo,
	}

	app, err := bootstrap.New(cfg, discardLogger(), version)
	if err != nil {
		t.Fatalf("the service could not be assembled: %v", err)
	}

	if err := app.Listen(); err != nil {
		t.Fatalf("the service could not listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() {
		served <- app.Serve(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-served:
		case <-time.After(10 * time.Second):
			t.Error("the service did not stop")
		}
	})

	return app.Addr()
}

func TestTheServiceAnswersItsHealthEndpoint(t *testing.T) {
	t.Parallel()

	addr := startService(t, "0.1.0")

	response, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("the health request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var body struct {
		Status  string `json:"status"`
		Service struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"service"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("status = %q, want %q", body.Status, "ok")
	}
	if body.Service.Name != config.ServiceName {
		t.Fatalf("service name = %q, want %q", body.Service.Name, config.ServiceName)
	}
	if body.Service.Version != "0.1.0" {
		t.Fatalf("service version = %q, want %q", body.Service.Version, "0.1.0")
	}
	if len(body.Checks) != 1 || body.Checks[0].Name != "mcp-transport" {
		t.Fatalf("checks = %+v, want the transport to be checked", body.Checks)
	}
}

func TestTheServiceSpeaksTheMCPProtocol(t *testing.T) {
	t.Parallel()

	addr := startService(t, "0.1.0")

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)

	ctx := context.Background()
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: "http://" + addr + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the service: %v", err)
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
