//go:build integration

// Package integration_test holds the tests that run the server as a
// container.
//
// They build the image of the server, start it, and talk to it the way a
// client of the stack does: over the network of the container, never from
// inside it. The tests need Docker.
package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// The settings of the container under test.
const (
	// containerPort is the port the server listens on inside the container.
	containerPort = "8081"
	// imageName is the name of the image the tests build.
	imageName = "paca/mcp-server"
	// imageTag is the tag of the image the tests build.
	imageTag = "integration-test"
)

// startServer starts the server from the image of the suite.
//
// The image is built once, before the tests run, and every test starts a
// container from it. A test that built the image itself would race with the
// other tests: the builder deletes the image and creates it again with the
// same name, and two builders at the same time fight over the name.
func startServer(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        imageName + ":" + imageTag,
			ExposedPorts: []string{containerPort + "/tcp"},
			Env: map[string]string{
				// The server binds the interface of the container, so the
				// test reaches it the way the other containers of the stack
				// do.
				"PACA_MCP_HOST": "0.0.0.0",
				"PACA_MCP_PORT": containerPort,
			},
			WaitingFor: wait.ForHTTP("/healthz").
				WithPort(containerPort + "/tcp").
				WithStatusCodeMatcher(func(status int) bool {
					return status == http.StatusOK
				}).
				WithStartupTimeout(3 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("the server container did not start: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("the server container did not terminate: %v", err)
		}
	})

	port, err := container.MappedPort(ctx, containerPort+"/tcp")
	if err != nil {
		t.Fatalf("the server container has no port: %v", err)
	}

	return "http://" + net.JoinHostPort("127.0.0.1", port.Port())
}

// projectDir returns the directory of the project, so the tests build the
// image whatever the working directory is.
func projectDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	// The test lives in <project>/test/integration.
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// repositoryRoot returns the root of the monorepository, which is the
// context of the build of the image.
//
// The image of the server is built from the root of the repository, the way
// the stack of the monorepository builds it: the server imports the shared
// libraries of the repository, and they are modules of their own next to it.
func repositoryRoot() string {
	return filepath.Clean(filepath.Join(projectDir(), "..", ".."))
}

// buildImage builds the image of the server from its Dockerfile.
//
// The image is built from the root of the monorepository, because the
// server imports the shared libraries of the repository, and they are
// modules of their own next to it.
//
// The build happens once, before the tests run: the tests prove the image
// that ships works, and they prove it from one image.
func buildImage(ctx context.Context) error {
	command := exec.CommandContext(ctx, "docker", "build",
		"-f", filepath.Join(repositoryRoot(), "apps", "mcp-server", "Dockerfile"),
		"-t", imageName+":"+imageTag,
		repositoryRoot(),
	)

	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("the image was not built: %w: %s", err, string(output))
	}

	return nil
}

func TestTheServerComesUpAndAnswersItsHealth(t *testing.T) {
	t.Parallel()

	address := startServer(t)

	response, err := http.Get(address + "/healthz")
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
	if body.Service.Name != "paca-mcp-server" {
		t.Fatalf("service name = %q, want %q", body.Service.Name, "paca-mcp-server")
	}
	if body.Service.Version == "" {
		t.Fatal("the server does not report its version")
	}
	if len(body.Checks) != 1 || body.Checks[0].Name != "mcp-transport" {
		t.Fatalf("checks = %+v, want the transport to be checked", body.Checks)
	}
	if body.Checks[0].Status != "ok" {
		t.Fatalf("the transport is %q, want %q", body.Checks[0].Status, "ok")
	}
}

func TestTheServerSpeaksTheMCPProtocol(t *testing.T) {
	t.Parallel()

	address := startServer(t)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: address + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the server: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	// The server speaks the protocol and it serves the tools that its
	// documentation promises.
	want := []string{"change_user_status", "get_user", "onboard_user", "update_user_contact_data"}
	got := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		got = append(got, tool.Name)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("the server exposes the tools %v, want %v", got, want)
	}
}

func TestTheServerKeepsItsPortToItself(t *testing.T) {
	t.Parallel()

	address := startServer(t)

	// The server is private: it answers the transport, and nothing else
	// that a browser could be tricked into calling.
	request, err := http.NewRequest(http.MethodPost, address+"/mcp", nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	request.Header.Set("Origin", "https://example.invalid")
	request.Header.Set("Sec-Fetch-Site", "cross-site")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("the request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}
}

// TestMain keeps the output of the containers out of the way of the test
// results, unless the test fails.
//
// The image of the server is built before the tests run: the tests are
// parallel, and a build that happens inside a test fights with the builds
// of the other tests over the name of the image.
func TestMain(m *testing.M) {
	if err := buildImage(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "the image of the server was not built: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
