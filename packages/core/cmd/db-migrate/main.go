// Command db-migrate applies the migrations of the schema of PACA.
//
// The schema has one writer, and this command is it. A service never
// migrates on boot: a service that migrates on boot races with the other
// replicas of itself.
//
// The database is named by PACA_DATABASE_URL. Without it the command refuses
// to run rather than guessing a database.
//
// The command applies every migration that has not been applied yet. Running
// it twice is not an error: the second run finds nothing to do.
//
//	db-migrate          apply the pending migrations
//	db-migrate -status  print the state of every migration and change nothing
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/witsaba/paca/packages/core"
)

// databaseURLEnv names the environment variable that holds the connection
// string of the database.
const databaseURLEnv = "PACA_DATABASE_URL"

// connectTimeout is the bound of the attempt to reach the database. It is
// a variable so that the test of the unreachable database does not wait for
// the whole bound.
var connectTimeout = 10 * time.Second

func main() {
	os.Exit(runMain())
}

// runMain runs the command and returns the status the process exits with.
func runMain() int {
	status := flag.Bool("status", false, "print the state of every migration and change nothing")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(ctx, logger, *status); err != nil {
		fmt.Fprintf(os.Stderr, "db-migrate: %v\n", err)
		return 1
	}

	return 0
}

// run applies the migrations, or prints the state of them.
func run(ctx context.Context, logger *slog.Logger, statusOnly bool) error {
	url := os.Getenv(databaseURLEnv)
	if url == "" {
		return fmt.Errorf("%s is empty: the command refuses to guess a database", databaseURLEnv)
	}

	db, err := openDatabase(ctx, url)
	if err != nil {
		return err
	}
	defer closeDatabase(logger, db)

	migrator, err := core.NewMigrator(db, logger)
	if err != nil {
		return err
	}

	if statusOnly {
		return printStatus(ctx, logger, migrator)
	}

	return apply(ctx, logger, migrator)
}

// openDatabase opens the connection of the command.
func openDatabase(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("the database could not be opened: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := db.PingContext(connectCtx); err != nil {
		closeDatabase(slog.Default(), db)
		return nil, fmt.Errorf("the database did not answer: %w", err)
	}

	return db, nil
}

// closeDatabase closes the connection of the command.
func closeDatabase(logger *slog.Logger, db *sql.DB) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		logger.Error("the database did not close cleanly", slog.Any("error", err))
	}
}

// apply applies the pending migrations and reports what it did.
func apply(ctx context.Context, logger *slog.Logger, migrator *core.Migrator) error {
	applied, err := migrator.Up(ctx)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		logger.Info("the schema was already up to date")
		return nil
	}

	for _, migration := range applied {
		logger.Info("the migration was applied",
			slog.Int64("version", migration.Version),
			slog.String("name", migration.Name),
		)
	}

	return nil
}

// printStatus prints the state of every migration and changes nothing.
func printStatus(ctx context.Context, logger *slog.Logger, migrator *core.Migrator) error {
	migrations, err := migrator.Status(ctx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		logger.Info("the state of the migration",
			slog.Int64("version", migration.Version),
			slog.String("name", migration.Name),
			slog.Bool("applied", migration.Applied),
		)
	}

	version, err := migrator.Version(ctx)
	if err != nil {
		return err
	}
	logger.Info("the version of the schema", slog.Int64("version", version))

	return nil
}
