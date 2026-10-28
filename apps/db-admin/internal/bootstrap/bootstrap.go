// Package bootstrap wires the layers of the service together.
//
// It is the only place that knows every layer at once: the domain rules, the
// use cases, the adapters and the infrastructure. Everything else depends on
// a single layer.
//
// The order of the start-up is the order of the service: the configuration
// is read, the telemetry is set up, the database is opened, the pending
// migrations are applied, and only then the service starts serving. A
// service that cannot apply the migrations does not serve at all: it is the
// writer of the schema, and a writer that did not write is not a writer.
package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/apps/db-admin/internal/adapters/inbound/http"
	"github.com/witsaba/paca/apps/db-admin/internal/adapters/outbound/database"
	"github.com/witsaba/paca/apps/db-admin/internal/application"
	"github.com/witsaba/paca/apps/db-admin/internal/domain/health"
	"github.com/witsaba/paca/apps/db-admin/internal/domain/schema"
	"github.com/witsaba/paca/apps/db-admin/internal/platform/clock"
	"github.com/witsaba/paca/apps/db-admin/internal/platform/config"
	"github.com/witsaba/paca/apps/db-admin/internal/platform/server"
	"github.com/witsaba/paca/apps/db-admin/internal/ports"
)

// App is the assembled service: the HTTP server that hosts the health
// endpoint, and the connection to the database it wrote the schema with.
type App struct {
	server *server.Server
	logger *slog.Logger
	db     *sql.DB
}

// New assembles the service.
//
// The version is the one stamped into the binary at build time. The
// environment overrides it, so a deployment can report the version it wants.
//
// The tracer provider creates the spans of the requests. A service without
// one does not trace.
func New(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	tracerProvider trace.TracerProvider,
	version string,
) (*App, error) {
	serviceVersion := cfg.ServiceVersion
	if serviceVersion == "" {
		serviceVersion = version
	}

	startedAt := time.Now()

	db, err := openDatabase(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	// The migrator of the core is the single writer of the schema: the
	// service never writes a migration of its own.
	migrator, err := database.NewMigrator(db, logger)
	if err != nil {
		closeDatabase(logger, db)
		return nil, err
	}

	migrateService, err := application.NewMigrateService(migrator)
	if err != nil {
		closeDatabase(logger, db)
		return nil, err
	}

	state, err := migrateService.Migrate(ctx)
	if err != nil {
		closeDatabase(logger, db)
		return nil, fmt.Errorf("the migrations could not be applied: %w", err)
	}
	logSchema(logger, state)

	// The database is the only component of the service: the service does
	// nothing but the database, so the health report carries one check.
	checker := database.NewChecker(database.NewPinger(db), migrator)

	healthService, err := application.NewHealthService(
		health.Identity{Name: config.ServiceName, Version: serviceVersion},
		[]ports.Checker{checker},
		clock.System{},
		startedAt,
		cfg.CheckTimeout,
	)
	if err != nil {
		closeDatabase(logger, db)
		return nil, err
	}

	router := http.NewRouter(http.RouterConfig{
		Health:         healthService,
		Logger:         logger,
		TracerProvider: tracerProvider,
	})

	return &App{
		server: server.New(cfg.Addr(), router, logger, cfg.ShutdownTimeout),
		logger: logger,
		db:     db,
	}, nil
}

// openDatabase opens the connection of the service to the database, and it
// waits for the database to answer.
//
// The connection is opened with the driver of the core, the same one the
// migrator of the core uses: the service and the schema speak the same
// protocol.
//
// The wait is the rule: the database of the monorepository and the writer
// that owns its schema start at the same moment, and the database accepts
// the connections a couple of seconds after its container starts. The
// writer waits for it instead of asking once and exiting: a container that
// exits is restarted by the runtime of Docker, and the health check of the
// container declares it unhealthy before the restart succeeds.
func openDatabase(ctx context.Context, cfg config.Config, logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("the database could not be opened: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, cfg.Database.WaitTimeout)
	defer cancel()

	waiter := database.NewWaiter(
		database.NewPinger(db),
		logger,
		database.WaitInterval,
		database.Sleep,
	)

	if err := waiter.Wait(waitCtx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// closeDatabase closes the connection of the service to the database.
func closeDatabase(logger *slog.Logger, db *sql.DB) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		logger.Error("the database did not close cleanly", slog.Any("error", err))
	}
}

// logSchema says what the migrations did.
//
// A run that applied nothing is not a failure: the schema was already up to
// date, and the service says so.
func logSchema(logger *slog.Logger, state schema.State) {
	if len(state.Applied) == 0 {
		logger.Info("the schema was already up to date",
			slog.Int64("version", state.Version),
		)
		return
	}

	for _, migration := range state.Applied {
		logger.Info("the migration was applied",
			slog.Int64("version", migration.Version),
			slog.String("name", migration.Name),
		)
	}

	logger.Info("the schema is up to date", slog.Int64("version", state.Version))
}

// Addr returns the address the server listens on. It is known after Listen.
func (a *App) Addr() string {
	return a.server.Addr()
}

// Listen binds the listener of the server. It is separate from Serve so the
// caller knows the address of a server that binds port zero.
func (a *App) Listen() error {
	return a.server.Listen()
}

// Serve serves the requests until the context is cancelled.
func (a *App) Serve(ctx context.Context) error {
	a.logger.Info("the database admin is serving",
		slog.String("addr", a.server.Addr()),
		slog.String("health_path", http.HealthPath),
	)
	defer closeDatabase(a.logger, a.db)

	return a.server.Serve(ctx)
}

// Run assembles the service and serves it.
func Run(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	tracerProvider trace.TracerProvider,
	version string,
) error {
	app, err := New(ctx, cfg, logger, tracerProvider, version)
	if err != nil {
		return err
	}
	if err := app.Listen(); err != nil {
		return err
	}
	return app.Serve(ctx)
}
