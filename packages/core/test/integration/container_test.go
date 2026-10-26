//go:build integration

// Package integration_test holds the tests of the core that run against a
// real PostgreSQL.
//
// The suite starts one container, applies the migrations through the single
// writer of the schema, and asserts every rule of the tables against that
// database. The tests need Docker.
//
// The suite shares one database: it is started once, the migrations are
// applied once, and every test writes the rows it needs. The identifiers of
// those rows come from a counter, so the tests can run in parallel against
// the same database without stepping on each other.
package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/witsaba/paca/packages/core"
	"github.com/witsaba/paca/packages/core/postgres"
)

// The settings of the database the tests run against.
const (
	// defaultPostgresImage is the PostgreSQL the tests run against. It is the
	// same version the stack of the monorepository runs, and it is the one
	// the tests find on a machine that keeps the image around.
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
)

// suite holds what the whole suite shares: the database with the schema of
// the migrations applied, and the migrator that applied it.
type suite struct {
	// db is the database the assertions run against.
	db *sql.DB

	// migrator is the single writer of the schema.
	migrator *core.Migrator

	// url is the connection string of the database.
	url string
}

// theSuite is the database every test of the suite runs against.
var theSuite *suite

// identifierCounter hands out the identifiers of the rows the tests write.
// The identifiers of the tests start far from the ones a person carries, so
// a row of the test is never the row of a real document.
var identifierCounter atomic.Int64

// TestMain starts the database of the suite, applies the migrations through
// the single writer of the schema, and runs the tests against it.
func TestMain(m *testing.M) {
	os.Exit(runSuite(m))
}

// runSuite holds the life of the suite: the container comes up, the
// migrations are applied, and the container is thrown away with the suite.
func runSuite(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	container, err := startPostgres(ctx)
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

	url, err := connectionString(ctx, container)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the connection string of the database could not be built: %v\n", err)
		return 1
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the database could not be opened: %v\n", err)
		return 1
	}
	defer db.Close()

	// The pool of the adapter is opened against the same database, so the
	// package that talks to PostgreSQL is exercised against the schema the
	// migrations describe.
	cfg := postgres.DefaultConfig()
	cfg.URL = url
	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the pool of the adapter could not be opened: %v\n", err)
		return 1
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "the database did not answer: %v\n", err)
		return 1
	}

	// The schema has one writer, and the tests use it: the migrations are
	// applied here, and never by a query of the suite.
	migrator, err := core.NewMigrator(db, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "the migrator could not be created: %v\n", err)
		return 1
	}

	applied, err := migrator.Up(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the migrations could not be applied: %v\n", err)
		return 1
	}
	if len(applied) == 0 {
		fmt.Fprintln(os.Stderr, "the migrations applied nothing, the schema was already there")
		return 1
	}

	theSuite = &suite{db: db, migrator: migrator, url: url}

	return m.Run()
}

// startPostgres starts the PostgreSQL the tests run against.
//
// The container is the same database the stack of the monorepository runs,
// so the schema is proven against the database it will live in.
func startPostgres(ctx context.Context) (testcontainers.Container, error) {
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
func connectionString(ctx context.Context, container testcontainers.Container) (string, error) {
	port, err := container.MappedPort(ctx, postgresPort)
	if err != nil {
		return "", fmt.Errorf("the container has no port: %w", err)
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser, postgresPassword, "127.0.0.1", port.Port(), postgresDatabase), nil
}

// nextIdentifier returns an identifier no other test uses, so the tests can
// run in parallel against the same database.
func nextIdentifier() int64 {
	return 100_000_000_000 + identifierCounter.Add(1)
}

// postgresError returns the PostgreSQL error behind an error of a query.
//
// The assertions name the constraint and the column the database refused,
// and the driver is the one that carries them.
func postgresError(t *testing.T, err error) *pgconn.PgError {
	t.Helper()

	if err == nil {
		t.Fatal("the query did not fail, want a PostgreSQL error")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("the error is %v, want a PostgreSQL error", err)
	}

	return pgErr
}
