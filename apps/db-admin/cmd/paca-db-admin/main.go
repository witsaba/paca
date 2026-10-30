// Command paca-db-admin runs the PACA database admin service.
//
// The service is the writer of the schema of the monorepository: it applies
// the pending migrations when it starts, and it stays alive serving exactly
// one health endpoint.
//
// The database is required. The service is the one that writes the schema,
// so it refuses to start without a database, and it fails with a clear
// message when the migrations cannot be applied.
//
// The configuration comes from the environment; see the config package for
// the variables and their defaults.
//
// The binary also answers a "healthcheck" command, which asks the running
// service whether it is alive. The container health check uses it.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/witsaba/paca/apps/db-admin/internal/bootstrap"
	"github.com/witsaba/paca/apps/db-admin/internal/platform/config"
	"github.com/witsaba/paca/packages/platform/healthcheck"
	"github.com/witsaba/paca/packages/platform/logging"
	"github.com/witsaba/paca/packages/platform/telemetry"
)

// version is the version of the binary. The build stamps it.
var version = "dev"

func main() {
	os.Exit(runMain())
}

// runMain runs the command and returns the status the process exits with.
func runMain() int {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(); err != nil {
			fmt.Fprintf(os.Stderr, "paca-db-admin: %v\n", err)
			return 1
		}
		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "paca-db-admin: %v\n", err)
		return 1
	}

	return 0
}

// run reads the configuration and serves until the context is cancelled.
func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(os.Stdout, cfg.LogLevel)

	providers, err := telemetry.Setup(ctx, telemetry.OptionsOf(
		cfg.Tracing,
		cfg.Tracing.ResolveServiceName(config.ServiceName),
		cfg.ServiceVersion,
	))
	if err != nil {
		return fmt.Errorf("set the telemetry up: %w", err)
	}
	defer telemetry.Stop(logger, providers)

	logger.Info("the service starts",
		slog.Bool("tracing", cfg.Tracing.Enabled),
		slog.String("collector", cfg.Tracing.Endpoint),
	)

	return bootstrap.Run(ctx, cfg, logger, providers.TracerProvider, version)
}

// runHealthcheck asks the running service whether it is alive.
//
// The check needs the port of the service and nothing else: a service that
// cannot reach its database still answers its health endpoint, and the
// answer is what the check is after.
func runHealthcheck() error {
	port, err := config.LoadPort()
	if err != nil {
		return err
	}

	return healthcheck.Check(context.Background(), port)
}
