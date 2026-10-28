//go:build integration

// Package integration_test holds the tests of the database admin service
// that run against a real PostgreSQL.
//
// The suite starts one PostgreSQL, builds the image of the service, and
// starts it against that database. The tests assert that the service
// applies the migrations of the schema, that it answers its health endpoint,
// that a second run applies nothing, and that a service without a database
// refuses to start. The tests need Docker.
package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// The settings of the container under test.
const (
	// containerPort is the port the service listens on inside the
	// container.
	containerPort = "8082"

	// imageName is the name of the image the tests build.
	imageName = "paca/db-admin"

	// imageTag is the tag of the image the tests build.
	imageTag = "integration-test"
)

// The settings of the database the tests run against.
const (
	// defaultPostgresImage is the PostgreSQL the tests run against. It is
	// the version the stack of the monorepository runs. The schema of the
	// core needs no extension, so the plain image is enough.
	defaultPostgresImage = "postgres:18.6-bookworm"

	// postgresPort is the port PostgreSQL listens on inside the container.
	postgresPort = "5432/tcp"

	// postgresUser is the owner of the database of the container.
	postgresUser = "paca"

	// postgresPassword is the password of the owner of the database of the
	// container. The database lives inside the container of the test, and
	// it is thrown away with it.
	postgresPassword = "paca"

	// postgresDatabase is the name of the database of the container.
	postgresDatabase = "paca"

	// postgresAlias is the name the database carries on the network of the
	// tests. It is the name the stack of the monorepository uses too.
	postgresAlias = "postgres"
)

// suite holds what the whole suite shares: the database, the network the
// containers of the tests share, and the image of the service.
type suite struct {
	// container is the database the assertions run against.
	container testcontainers.Container

	// network is the network the containers of the tests share.
	network *testcontainers.DockerNetwork

	// db is the handle of the database, from the host.
	db *sql.DB
}

// theSuite is the database every test of the suite runs against.
var theSuite *suite

// TestMain starts the database of the suite, builds the image of the
// service, and runs the tests against them.
func TestMain(m *testing.M) {
	os.Exit(runSuite(m))
}

// runSuite holds the life of the suite: the database comes up, the image is
// built, and both are thrown away with the suite.
func runSuite(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	shared, err := network.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the network of the tests did not start: %v\n", err)
		return 1
	}
	defer func() {
		if err := shared.Remove(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "the network of the tests did not go away: %v\n", err)
		}
	}()

	container, err := startPostgres(ctx, shared)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the database of the tests did not start: %v\n", err)
		return 1
	}
	defer func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), time.Minute)
		defer terminateCancel()
		if err := container.Terminate(terminateCtx); err != nil {
			fmt.Fprintf(os.Stderr, "the database container did not terminate: %v\n", err)
		}
	}()

	db, err := sql.Open("pgx", connectionString(ctx, container))
	if err != nil {
		fmt.Fprintf(os.Stderr, "the database could not be opened: %v\n", err)
		return 1
	}
	defer db.Close()

	if err := waitForTheDatabase(ctx, db); err != nil {
		fmt.Fprintf(os.Stderr, "the database did not answer: %v\n", err)
		return 1
	}

	if err := buildImage(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "the image of the service was not built: %v\n", err)
		return 1
	}

	theSuite = &suite{container: container, network: shared, db: db}

	return m.Run()
}

// waitForTheDatabase asks the database whether it answers, and it asks again
// while the server is still starting up.
//
// The container opens the port of PostgreSQL before the server accepts the
// connections, and the first question of the suite can arrive in that
// window: the database answers with the code 57P03, the server is starting
// up. The suite waits for the answer instead of failing on it.
func waitForTheDatabase(ctx context.Context, db *sql.DB) error {
	deadline := time.Now().Add(2 * time.Minute)

	for {
		err := db.PingContext(ctx)
		if err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// startPostgres starts the PostgreSQL the tests run against.
//
// The container is the same database the stack of the monorepository runs,
// so the schema is proven against the database it will live in.
func startPostgres(ctx context.Context, shared *testcontainers.DockerNetwork) (testcontainers.Container, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: postgresImage(),
			Env: map[string]string{
				"POSTGRES_USER":     postgresUser,
				"POSTGRES_PASSWORD": postgresPassword,
				"POSTGRES_DB":       postgresDatabase,
				// The clock of the database is the clock of the machine, in
				// UTC, so the timestamps of the rows are the timestamps the
				// tests expect.
				"TZ":   "UTC",
				"PGTZ": "UTC",
			},
			ExposedPorts: []string{postgresPort},
			WaitingFor: wait.ForListeningPort(postgresPort).
				WithStartupTimeout(3 * time.Minute),
			Networks: []string{shared.Name},
			NetworkAliases: map[string][]string{
				shared.Name: {postgresAlias},
			},
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("the container could not be started: %w", err)
	}

	return container, nil
}

// postgresImage returns the image of the database the tests run against.
//
// The image can be changed with PACA_TEST_POSTGRES_IMAGE, for a machine that
// keeps another PostgreSQL around.
func postgresImage() string {
	if image := os.Getenv("PACA_TEST_POSTGRES_IMAGE"); image != "" {
		return image
	}
	return defaultPostgresImage
}

// connectionString returns the connection string of the database of the
// container, from the host.
func connectionString(ctx context.Context, container testcontainers.Container) string {
	port, err := container.MappedPort(ctx, postgresPort)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("postgres://%s:%s@127.0.0.1:%s/%s?sslmode=disable",
		postgresUser, postgresPassword, port.Port(), postgresDatabase)
}

// buildImage builds the image of the service from its Dockerfile.
//
// The image is built from the root of the monorepository, because the
// service imports the shared libraries of the repository, and they are
// modules of their own next to it.
func buildImage(ctx context.Context) error {
	root := repositoryRoot()

	command := exec.CommandContext(ctx, "docker", "build",
		"-f", filepath.Join(projectDir(), "Dockerfile"),
		"-t", imageName+":"+imageTag,
		root,
	)

	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("the image was not built: %w: %s", err, string(output))
	}

	return nil
}

// projectDir returns the directory of the service, so the tests build the
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
func repositoryRoot() string {
	return filepath.Clean(filepath.Join(projectDir(), "..", ".."))
}

// logsOf returns the logs of a container.
func logsOf(ctx context.Context, container testcontainers.Container) string {
	reader, err := container.Logs(ctx)
	if err != nil {
		return ""
	}
	defer func() {
		_ = reader.Close()
	}()

	buffer := make([]byte, 0, 4096)
	chunk := make([]byte, 4096)
	for {
		read, err := reader.Read(chunk)
		buffer = append(buffer, chunk[:read]...)
		if err != nil {
			break
		}
	}

	return string(buffer)
}

// hasTable tells whether the table of the schema exists in the database.
func hasTable(t *testing.T, name string) bool {
	t.Helper()

	var exists bool
	query := "SELECT to_regclass($1) IS NOT NULL"
	if err := theSuite.db.QueryRow(query, name).Scan(&exists); err != nil {
		t.Fatalf("the table %q could not be read: %v", name, err)
	}

	return exists
}

// contains tells whether the logs hold the message.
func contains(logs, message string) bool {
	return strings.Contains(logs, message)
}
