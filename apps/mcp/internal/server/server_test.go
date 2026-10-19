package server_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp/internal/config"
	"github.com/witsaba/paca/apps/mcp/internal/server"
)

// testConfig is the configuration of a server under test: a loopback bind
// with a database in a temporary directory.
func testConfig(t *testing.T) config.Config {
	t.Helper()

	return config.Config{
		Bind:   config.DefaultBind,
		Port:   config.DefaultPort,
		DBPath: filepath.Join(t.TempDir(), "paca.db"),
	}
}

// newTestServer builds a server over a temporary database and closes it when
// the test ends.
func newTestServer(t *testing.T, cfg config.Config) *server.Server {
	t.Helper()

	srv, err := server.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	t.Cleanup(func() {
		if err := srv.Close(); err != nil {
			t.Errorf("the server was not closed: %v", err)
		}
	})

	return srv
}

func TestHealthzAnswersOK(t *testing.T) {
	srv := newTestServer(t, testConfig(t))

	recorder := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, server.HealthPath, nil))

	res := recorder.Result()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read the body: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	// The body is the literal the session decided on: plain text "ok". The
	// constant of the server is checked apart, so a change of it is caught.
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
	if server.HealthBody != "ok" {
		t.Errorf("the health body = %q, want %q", server.HealthBody, "ok")
	}
	if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("content type = %q, want text/plain", got)
	}
}

// The liveness endpoint answers even when the MCP surface is closed to the
// world: a probe must not need a token.
func TestHealthzIsNotAuthenticated(t *testing.T) {
	cfg := testConfig(t)
	cfg.AuthToken = "s3cret"
	srv := newTestServer(t, cfg)

	recorder := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, server.HealthPath, nil))

	if got := recorder.Result().StatusCode; got != http.StatusOK {
		t.Fatalf("status = %d, want %d", got, http.StatusOK)
	}
}

func TestTheMCPSurfaceRefusesWithoutTheToken(t *testing.T) {
	cfg := testConfig(t)
	cfg.AuthToken = "s3cret"
	srv := newTestServer(t, cfg)

	cases := []struct {
		name   string
		header string
	}{
		{name: "no header"},
		{name: "wrong token", header: "Bearer wrong"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, server.MCPPath, strings.NewReader("{}"))
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			recorder := httptest.NewRecorder()
			srv.Handler().ServeHTTP(recorder, req)

			if got := recorder.Result().StatusCode; got != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", got, http.StatusUnauthorized)
			}
		})
	}
}

// The server serves an empty MCP session: the transport works, and the tools
// arrive in a later session.
func TestTheServerServesAnEmptyMCPSession(t *testing.T) {
	srv := newTestServer(t, testConfig(t))
	httpServer := httptest.NewServer(srv.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   httpServer.URL + server.MCPPath,
		HTTPClient: httpServer.Client(),
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("the session was not closed: %v", err)
		}
	})

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list the tools: %v", err)
	}
	if len(tools.Tools) != 0 {
		t.Errorf("tools = %d, want 0", len(tools.Tools))
	}
}

// The same session, with the bearer token the configuration demands.
func TestTheServerServesAnAuthenticatedMCPSession(t *testing.T) {
	cfg := testConfig(t)
	cfg.AuthToken = "s3cret"
	srv := newTestServer(t, cfg)
	httpServer := httptest.NewServer(srv.Handler())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   httpServer.URL + server.MCPPath,
		HTTPClient: &http.Client{Transport: bearerTransport{token: "s3cret"}},
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("the session was not closed: %v", err)
		}
	})

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list the tools: %v", err)
	}
	if len(tools.Tools) != 0 {
		t.Errorf("tools = %d, want 0", len(tools.Tools))
	}
}

// bearerTransport adds the bearer token to every request.
type bearerTransport struct {
	token string
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)

	return http.DefaultTransport.RoundTrip(clone)
}
