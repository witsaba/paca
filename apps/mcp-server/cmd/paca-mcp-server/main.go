// Command paca-mcp-server runs the PACA MCP server.
//
// The server speaks the Model Context Protocol over the official Go SDK and
// exposes a health endpoint. It registers no tool yet: this binary is the
// initialization of the server, not its first feature.
//
// The configuration comes from the environment; see the config package for
// the variables and their defaults.
//
// The binary also answers a "healthcheck" command, which asks the running
// server whether it is alive. The container health check uses it.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/witsaba/paca/apps/mcp-server/internal/bootstrap"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/config"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/healthcheck"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/logging"
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
			fmt.Fprintf(os.Stderr, "paca-mcp-server: %v\n", err)
			return 1
		}
		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "paca-mcp-server: %v\n", err)
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

	return bootstrap.Run(ctx, cfg, logger, version)
}

// runHealthcheck asks the running server whether it is alive.
func runHealthcheck() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	return healthcheck.Check(context.Background(), cfg.Port)
}
