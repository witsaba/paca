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
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/bootstrap"
	"github.com/witsaba/paca/apps/db-admin/internal/platform/config"
	"github.com/witsaba/paca/packages/platform/healthcheck"
	"github.com/witsaba/paca/packages/platform/logging"
	"github.com/witsaba/paca/packages/platform/telemetry"
)

// version is the version of the binary. The build stamps it.
var version = "dev"

// shutdownTimeout is how long the telemetry has to leave the process. The
// spans that are still in the batcher are sent during the shutdown; a
// collector that does not answer must not hold the process forever.
const shutdownTimeout = 5 * time.Second

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

	providers, err := telemetry.Setup(ctx, telemetry.Options{
		Enabled:        cfg.Tracing.Enabled,
		ServiceName:    telemetryName(cfg),
		ServiceVersion: cfg.ServiceVersion,
		Endpoint:       cfg.Tracing.Endpoint,
		Protocol:       cfg.Tracing.Protocol,
		Insecure:       cfg.Tracing.Insecure,
		Sampler:        cfg.Tracing.Sampler,
		SamplerArg:     cfg.Tracing.SamplerArg,
	})
	if err != nil {
		return fmt.Errorf("set the telemetry up: %w", err)
	}
	defer shutdownTelemetry(logger, providers)

	logger.Info("the service starts",
		slog.Bool("tracing", cfg.Tracing.Enabled),
		slog.String("collector", cfg.Tracing.Endpoint),
	)

	return bootstrap.Run(ctx, cfg, logger, providers.TracerProvider, version)
}

// telemetryName returns the name the telemetry carries. It is the name of
// the service, unless the environment of the OpenTelemetry specification
// names it otherwise.
func telemetryName(cfg config.Config) string {
	if cfg.Tracing.ServiceName != "" {
		return cfg.Tracing.ServiceName
	}
	return config.ServiceName
}

// shutdownTelemetry sends the spans that are still in the batcher.
func shutdownTelemetry(logger *slog.Logger, providers *telemetry.Providers) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := providers.Shutdown(ctx); err != nil {
		logger.Error("the telemetry did not leave the process cleanly",
			slog.Any("error", err))
	}
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
