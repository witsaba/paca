// Package bootstrap wires the layers of the service together.
//
// It is the only place that knows every layer at once: the domain rules, the
// use cases, the adapters and the infrastructure. Everything else depends on
// a single layer.
package bootstrap

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/http"
	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/outbound/serving"
	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/domain/health"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/clock"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/config"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/server"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
)

// Instructions tells the MCP clients what the server is for.
const Instructions = "The PACA MCP server. It speaks the Model Context Protocol " +
	"and exposes no tool yet."

// App is the assembled service: the HTTP server that hosts the MCP
// transport, and the health use case that reports on it.
type App struct {
	server           *server.Server
	logger           *slog.Logger
	transportChecker *serving.Checker
}

// New assembles the service.
//
// The version is the one stamped into the binary at build time. The
// environment overrides it, so a deployment can report the version it wants.
//
// The tracer provider creates the spans of the requests. A service without
// one does not trace.
func New(
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
	healthService, err := application.NewHealthService(
		health.Identity{Name: config.ServiceName, Version: serviceVersion},
		[]ports.Checker{transportChecker},
		clock.System{},
		startedAt,
		cfg.CheckTimeout,
	)
	if err != nil {
		return nil, err
	}

	mcpServer := mcpserver.New(mcpserver.Options{
		ServiceName:         config.ServiceName,
		ServiceVersion:      serviceVersion,
		Instructions:        Instructions,
		Logger:              logger,
		MaxRequestBodyBytes: cfg.MaxRequestBytes,
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
	}, nil
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
	app, err := New(cfg, logger, tracerProvider, version)
	if err != nil {
		return err
	}
	if err := app.Listen(); err != nil {
		return err
	}
	return app.Serve(ctx)
}
