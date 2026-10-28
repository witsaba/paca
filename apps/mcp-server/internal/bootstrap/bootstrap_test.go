package bootstrap_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/witsaba/paca/apps/mcp-server/internal/bootstrap"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/config"
)

// discardLogger is the logger of the test.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// startService assembles the service and serves it on a free port.
//
// The mutator changes the configuration before the service is assembled, so
// a test can name a database or leave it out.
func startService(t *testing.T, version string, mutate func(*config.Config)) string {
	t.Helper()

	cfg := config.Config{
		Host:            "127.0.0.1",
		Port:            0,
		ShutdownTimeout: 5 * time.Second,
		CheckTimeout:    time.Second,
		MaxRequestBytes: 1 << 20,
		LogLevel:        slog.LevelInfo,
	}
	if mutate != nil {
		mutate(&cfg)
	}

	app, err := bootstrap.New(context.Background(), cfg, discardLogger(), noop.NewTracerProvider(), version)
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

	addr := startService(t, "0.1.0", nil)

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

func TestTheServiceChecksTheDatabaseWhenItNamesOne(t *testing.T) {
	t.Parallel()

	// Nothing listens on this port: the pool is lazy, so the service starts
	// anyway, and the check of the database is the thing that fails.
	addr := startService(t, "0.1.0", func(cfg *config.Config) {
		cfg.Database = config.DatabaseConfig{
			URL:            "postgres://paca:paca@127.0.0.1:1/paca?sslmode=disable",
			MaxConns:       1,
			ConnectTimeout: time.Second,
		}
	})

	response, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("the health request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusServiceUnavailable)
	}

	var body struct {
		Status string `json:"status"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}

	if body.Status != "unhealthy" {
		t.Fatalf("status = %q, want %q", body.Status, "unhealthy")
	}

	found := false
	for _, check := range body.Checks {
		if check.Name == "database" {
			found = true
			if check.Status != "unhealthy" {
				t.Errorf("the check of the database is %q, want %q", check.Status, "unhealthy")
			}
		}
	}
	if !found {
		t.Fatalf("checks = %+v, want the database to be checked", body.Checks)
	}
}

// TestTheServiceSaysItHasNoDatabaseWhenItHasNone proves the wiring of the
// service: with no database configured, the tools are still there, and the
// call answers that the database is not configured — in the words of the
// service, not in the words of the adapter.
//
// The test drives the real composition root, because the trap it guards
// against lives there: a repository that is nil inside an interface is not a
// nil interface.
func TestTheServiceSaysItHasNoDatabaseWhenItHasNone(t *testing.T) {
	t.Parallel()

	addr := startService(t, "0.1.0", nil)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)

	ctx := context.Background()
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: "http://" + addr + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the service: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "onboard_user",
		Arguments: map[string]any{"document_id": "1234567"},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	text := textOf(t, result)
	want := "the service has no database configured: the users cannot be reached"
	if !strings.Contains(text, want) {
		t.Errorf("the message is %q, want it to hold %q", text, want)
	}
}

// textOf returns the text of the answer of a tool.
func textOf(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()

	var text strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(*sdk.TextContent); ok {
			text.WriteString(textContent.Text)
		}
	}

	return text.String()
}

func TestTheServiceSpeaksTheMCPProtocol(t *testing.T) {
	t.Parallel()

	addr := startService(t, "0.1.0", nil)

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
	if len(tools.Tools) != 3 {
		t.Fatalf("the server exposes %d tools, want the three of the users", len(tools.Tools))
	}
}
