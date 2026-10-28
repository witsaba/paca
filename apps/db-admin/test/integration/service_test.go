//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// startService starts the service against the database of the suite.
//
// The environment of the container is the one the compose file gives to the
// service: the credentials of the database travel to it through the
// environment, and the service assembles the connection string out of them.
func startService(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: imageName + ":" + imageTag,
			Env: map[string]string{
				// The service binds the interface of the container, so the
				// test reaches it the way the other containers of the
				// stack do.
				"PACA_DB_ADMIN_HOST": "0.0.0.0",
				"PACA_DB_ADMIN_PORT": containerPort,
				// The credentials of the database, the way the compose
				// file of the service gives them.
				"POSTGRES_USER":      postgresUser,
				"POSTGRES_PASSWORD":  postgresPassword,
				"POSTGRES_DB":        postgresDatabase,
				"PACA_DATABASE_HOST": postgresAlias,
				"PACA_DATABASE_PORT": "5432",
			},
			ExposedPorts: []string{containerPort + "/tcp"},
			WaitingFor: wait.ForHTTP("/healthz").
				WithPort(containerPort + "/tcp").
				WithStatusCodeMatcher(func(status int) bool {
					return status == http.StatusOK
				}).
				WithStartupTimeout(3 * time.Minute),
			Networks: []string{theSuite.network.Name},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("the service container did not start: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("the service container did not terminate: %v", err)
		}
	})

	port, err := container.MappedPort(ctx, containerPort+"/tcp")
	if err != nil {
		t.Fatalf("the service container has no port: %v", err)
	}

	return "http://" + net.JoinHostPort("127.0.0.1", port.Port())
}

// startServiceWithURL starts the service against the database of the suite,
// naming the database with the whole connection string.
func startServiceWithURL(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: imageName + ":" + imageTag,
			Env: map[string]string{
				"PACA_DB_ADMIN_HOST": "0.0.0.0",
				"PACA_DB_ADMIN_PORT": containerPort,
				// The whole connection string wins over everything else.
				"PACA_DATABASE_URL": fmt.Sprintf(
					"postgres://%s:%s@%s:5432/%s?sslmode=disable",
					postgresUser, postgresPassword, postgresAlias, postgresDatabase),
			},
			ExposedPorts: []string{containerPort + "/tcp"},
			WaitingFor: wait.ForHTTP("/healthz").
				WithPort(containerPort + "/tcp").
				WithStatusCodeMatcher(func(status int) bool {
					return status == http.StatusOK
				}).
				WithStartupTimeout(3 * time.Minute),
			Networks: []string{theSuite.network.Name},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("the service container did not start: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("the service container did not terminate: %v", err)
		}
	})

	port, err := container.MappedPort(ctx, containerPort+"/tcp")
	if err != nil {
		t.Fatalf("the service container has no port: %v", err)
	}

	return "http://" + net.JoinHostPort("127.0.0.1", port.Port())
}

// healthOf asks the service for its health report.
func healthOf(t *testing.T, address string) (int, healthBody) {
	t.Helper()

	response, err := http.Get(address + "/healthz")
	if err != nil {
		t.Fatalf("the health request failed: %v", err)
	}
	defer response.Body.Close()

	var body healthBody
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}

	return response.StatusCode, body
}

// healthBody is the body of the health endpoint.
type healthBody struct {
	Status  string `json:"status"`
	Service struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"service"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	CheckedAt     string  `json:"checked_at"`
	Checks        []struct {
		Name      string `json:"name"`
		Status    string `json:"status"`
		LatencyMS int64  `json:"latency_ms"`
		Detail    string `json:"detail"`
	} `json:"checks"`
}

func TestTheServiceAppliesTheMigrationsAndAnswersItsHealth(t *testing.T) {
	address := startService(t)

	status, body := healthOf(t, address)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}

	if body.Status != "ok" {
		t.Fatalf("status = %q, want %q", body.Status, "ok")
	}
	if body.Service.Name != "paca-db-admin" {
		t.Fatalf("service name = %q, want %q", body.Service.Name, "paca-db-admin")
	}
	if body.Service.Version == "" {
		t.Fatal("the service does not report its version")
	}
	if len(body.Checks) != 1 {
		t.Fatalf("checks = %+v, want the single check of the service", body.Checks)
	}
	if body.Checks[0].Name != "database" {
		t.Fatalf("check = %+v, want the database to be checked", body.Checks[0])
	}
	if body.Checks[0].Status != "ok" {
		t.Fatalf("the database is %q, want %q", body.Checks[0].Status, "ok")
	}

	// The schema is in place: the table of the users exists.
	if !hasTable(t, "users") {
		t.Fatal("the table of the users does not exist, want the migrations to be applied")
	}
}

func TestTheServiceAppliesNothingOnASecondRun(t *testing.T) {
	// The first run applies the migrations of the schema.
	startService(t)

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: imageName + ":" + imageTag,
			Env: map[string]string{
				"PACA_DB_ADMIN_HOST": "0.0.0.0",
				"PACA_DB_ADMIN_PORT": containerPort,
				"POSTGRES_USER":      postgresUser,
				"POSTGRES_PASSWORD":  postgresPassword,
				"POSTGRES_DB":        postgresDatabase,
				"PACA_DATABASE_HOST": postgresAlias,
				"PACA_DATABASE_PORT": "5432",
			},
			ExposedPorts: []string{containerPort + "/tcp"},
			WaitingFor: wait.ForHTTP("/healthz").
				WithPort(containerPort + "/tcp").
				WithStatusCodeMatcher(func(status int) bool {
					return status == http.StatusOK
				}).
				WithStartupTimeout(3 * time.Minute),
			Networks: []string{theSuite.network.Name},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("the service container did not start: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("the service container did not terminate: %v", err)
		}
	})

	logs := logsOf(ctx, container)
	if !contains(logs, "the schema was already up to date") {
		t.Fatalf("the logs of the second run do not say that there was nothing to do:\n%s", logs)
	}
}

func TestTheServiceNamesTheDatabaseWithTheConnectionString(t *testing.T) {
	address := startServiceWithURL(t)

	status, body := healthOf(t, address)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q, want %q", body.Status, "ok")
	}
}

func TestTheServiceFailsWithoutADatabase(t *testing.T) {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: imageName + ":" + imageTag,
			Env: map[string]string{
				"PACA_DB_ADMIN_HOST": "0.0.0.0",
				"PACA_DB_ADMIN_PORT": containerPort,
				// The service names no database at all.
			},
			ExposedPorts: []string{containerPort + "/tcp"},
			WaitingFor: wait.ForExit().
				WithExitTimeout(3 * time.Minute),
			Networks: []string{theSuite.network.Name},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("the service container did not stop: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("the service container did not terminate: %v", err)
		}
	})

	state, err := container.State(ctx)
	if err != nil {
		t.Fatalf("the state of the container could not be read: %v", err)
	}
	if state.ExitCode == 0 {
		t.Fatal("the service exited with 0, want a failure")
	}

	logs := logsOf(ctx, container)
	if !contains(logs, "cannot run without a database") {
		t.Fatalf("the logs of the service do not explain the failure:\n%s", logs)
	}
}
