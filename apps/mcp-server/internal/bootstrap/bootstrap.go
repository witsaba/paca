// Package bootstrap wires the layers of the service together.
//
// It is the only place that knows every layer at once: the domain rules, the
// use cases, the adapters and the infrastructure. Everything else depends on
// a single layer.
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/http"
	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/outbound/database"
	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/outbound/serving"
	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/config"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/health"
	"github.com/witsaba/paca/packages/domain/user"
	"github.com/witsaba/paca/packages/platform/clock"
	"github.com/witsaba/paca/packages/platform/server"
)

// Instructions tells the MCP clients what the server is for.
const Instructions = "The PACA MCP server. It speaks the Model Context Protocol " +
	"and it takes care of the people who use PACA: it registers a person and " +
	"starts their onboarding, it moves them to another status, and it " +
	"corrects their data."

// App is the assembled service: the HTTP server that hosts the MCP
// transport, and the health use case that reports on it.
type App struct {
	server           *server.Server
	logger           *slog.Logger
	transportChecker *serving.Checker
	pool             *postgres.Pool
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

	transportChecker := serving.NewChecker()
	checkers := []ports.Checker{transportChecker}

	pool, err := openDatabase(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	if pool != nil {
		checkers = append(checkers, database.NewChecker(pool))
	}

	healthService, err := application.NewHealthService(
		health.Identity{Name: config.ServiceName, Version: serviceVersion},
		checkers,
		clock.System{},
		startedAt,
		cfg.CheckTimeout,
	)
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, err
	}

	// The repository of the users is the adapter of the core: the service
	// writes the people through it, and it never writes a row of its own.
	// Without a database there is no repository at all — not a repository
	// that is closed — so the use case tells the agent that calls it that
	// the database is not configured.
	var repository user.Repository
	if pool != nil {
		bound, err := postgres.NewUserRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the users could not be built: %w", err)
		}
		repository = bound
	}

	usersService := application.NewUsersService(repository)

	mcpServer := mcpserver.New(mcpserver.Options{
		ServiceName:         config.ServiceName,
		ServiceVersion:      serviceVersion,
		Instructions:        Instructions,
		Logger:              logger,
		MaxRequestBodyBytes: cfg.MaxRequestBytes,
		Users:               usersService,
	})

	router := http.NewRouter(http.RouterConfig{
		Health:         healthService,
		MCP:            mcpServer.Handler(),
		Logger:         logger,
		TracerProvider: tracerProvider,
	})

	return &App{
		server:           server.New(cfg.Addr(), router, logger, cfg.ShutdownTimeout),
		logger:           logger,
		transportChecker: transportChecker,
		pool:             pool,
	}, nil
}

// openDatabase opens the pool of the database, when the service names one.
//
// The database is optional: a service that does not name it starts without
// one, and its health report says nothing about it. The pool is lazy, so a
// database that is down does not stop the service from starting.
func openDatabase(ctx context.Context, cfg config.Config, logger *slog.Logger) (*postgres.Pool, error) {
	if cfg.Database.URL == "" {
		return nil, nil
	}

	poolCfg := postgres.DefaultConfig()
	poolCfg.URL = cfg.Database.URL
	poolCfg.MaxConns = cfg.Database.MaxConns
	poolCfg.ConnectTimeout = cfg.Database.ConnectTimeout

	pool, err := postgres.Open(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("the pool of the database could not be opened: %w", err)
	}

	logger.Info("the database is configured",
		slog.Int64("max_conns", int64(cfg.Database.MaxConns)),
	)

	return pool, nil
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
	a.transportChecker.SetServing(true)
	a.logger.Info("the MCP server is serving",
		slog.String("addr", a.server.Addr()),
		slog.String("mcp_path", http.MCPPath),
		slog.String("health_path", http.HealthPath),
	)
	defer a.transportChecker.SetServing(false)
	defer a.pool.Close()

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
