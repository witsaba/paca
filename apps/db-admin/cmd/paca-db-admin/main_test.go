package main

import (
	"context"
	"testing"

	"github.com/witsaba/paca/apps/db-admin/internal/platform/config"
	"github.com/witsaba/paca/packages/platform/env"
)

func TestRunFailsWithAnInvalidConfiguration(t *testing.T) {
	t.Setenv(config.EnvPort, "eighty")

	if err := run(context.Background()); err == nil {
		t.Fatal("run() = nil, want an error")
	}
}

func TestRunFailsWithoutADatabase(t *testing.T) {
	// The service is the writer of the schema: it has nothing to do
	// without a database, and it says so.
	t.Setenv(env.DatabaseURL, "")
	t.Setenv(env.PostgresUser, "")
	t.Setenv(env.PostgresPassword, "")
	t.Setenv(env.PostgresDB, "")

	if err := run(context.Background()); err == nil {
		t.Fatal("run() = nil, want an error about the database")
	}
}

func TestRunMainAnswersOneWhenTheConfigurationIsInvalid(t *testing.T) {
	t.Setenv(config.EnvPort, "eighty")

	if status := runMain(); status != 1 {
		t.Fatalf("runMain() = %d, want %d", status, 1)
	}
}

func TestTheHealthcheckFailsWhenTheServiceDoesNotAnswer(t *testing.T) {
	t.Setenv(config.EnvPort, "1")

	if err := runHealthcheck(); err == nil {
		t.Fatal("runHealthcheck() = nil, want an error")
	}
}

func TestTheHealthcheckNeedsNoDatabase(t *testing.T) {
	// The health check of the container asks the service whether it is
	// alive: it needs the port of the service, and nothing else.
	t.Setenv(env.DatabaseURL, "")
	t.Setenv(env.PostgresUser, "")
	t.Setenv(env.PostgresPassword, "")
	t.Setenv(env.PostgresDB, "")
	t.Setenv(config.EnvPort, "1")

	if err := runHealthcheck(); err == nil {
		t.Fatal("runHealthcheck() = nil, want the failure of the check")
	}
}
