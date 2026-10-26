package main

import (
	"context"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/platform/config"
)

func TestRunFailsWithAnInvalidConfiguration(t *testing.T) {
	t.Setenv(config.EnvPort, "eighty")

	if err := run(context.Background()); err == nil {
		t.Fatal("run() = nil, want an error")
	}
}
