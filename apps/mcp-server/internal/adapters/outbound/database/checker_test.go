package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/outbound/database"
	"github.com/witsaba/paca/packages/core/postgres"
)

func TestTheCheckerNamesTheDatabase(t *testing.T) {
	t.Parallel()

	checker := database.NewChecker(nil)

	if checker.Name() != "database" {
		t.Errorf("the name is %q, want %q", checker.Name(), "database")
	}
}

func TestTheCheckerAsksTheDatabase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Nothing listens on this port: the pool is lazy, so it opens anyway,
	// and the check is the thing that fails.
	cfg := postgres.DefaultConfig()
	cfg.URL = "postgres://paca:paca@127.0.0.1:1/paca?sslmode=disable"
	cfg.ConnectTimeout = 2 * time.Second

	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("the pool did not open: %v", err)
	}
	defer pool.Close()

	checker := database.NewChecker(pool)

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := checker.Check(checkCtx); err == nil {
		t.Fatal("the check of a database that is not there passed, want an error")
	}
}
