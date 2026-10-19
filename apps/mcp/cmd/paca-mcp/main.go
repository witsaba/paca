// Command paca-mcp is the MCP server of PACA.
//
// This file is the composition root: it reads the configuration, builds the
// server and runs it until the process is asked to stop. It holds no
// business rule and no I/O of its own.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/witsaba/paca/apps/mcp/internal/config"
	"github.com/witsaba/paca/apps/mcp/internal/server"
)

// exitCode is the status the process ends with when the configuration is
// refused. It is not 1: a refused configuration is a usage error, and the
// caller of the binary can tell it apart from a failure at runtime.
const exitCode = 2

func main() {
	os.Exit(execute(os.Args[1:], os.Stderr))
}

// execute is the body of the process: it installs the signal handler that
// stops the server and runs it.
func execute(args []string, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx, args, stderr)
}

// run parses the configuration and serves until the context is cancelled.
//
// The status of the process is the return value: a refused configuration
// ends with the usage status, a failure at runtime with 1, and a clean stop
// with 0.
func run(ctx context.Context, args []string, stderr io.Writer) int {
	cfg, err := config.Parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "paca-mcp: %v\n", err)
		return exitCode
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	if err := server.Run(ctx, cfg, logger); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("the server stopped", "error", err)
		return 1
	}

	return 0
}
