package main

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestRunWithoutADatabase(t *testing.T) {
	t.Setenv(databaseURLEnv, "")

	err := run(context.Background(), slog.Default(), false)
	if err == nil {
		t.Fatal("the command ran without a database, want an error")
	}
	if !strings.Contains(err.Error(), databaseURLEnv) {
		t.Errorf("the error %q does not name the variable %q", err, databaseURLEnv)
	}
}

func TestRunWithoutADatabaseInStatus(t *testing.T) {
	t.Setenv(databaseURLEnv, "")

	err := run(context.Background(), slog.Default(), true)
	if err == nil {
		t.Fatal("the command ran without a database, want an error")
	}
	if !strings.Contains(err.Error(), databaseURLEnv) {
		t.Errorf("the error %q does not name the variable %q", err, databaseURLEnv)
	}
}

func TestRunWithAnUnreachableDatabase(t *testing.T) {
	// Nothing listens on this port: the command must say so instead of
	// waiting for the timeout.
	t.Setenv(databaseURLEnv, "postgres://paca:paca@127.0.0.1:1/paca?sslmode=disable")

	previous := connectTimeout
	connectTimeout = 2 * time.Second
	t.Cleanup(func() { connectTimeout = previous })

	err := run(context.Background(), slog.Default(), false)
	if err == nil {
		t.Fatal("the command ran against a database that is not there, want an error")
	}
}
