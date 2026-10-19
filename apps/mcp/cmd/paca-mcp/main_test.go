package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A refused configuration ends the process with the usage status, and the
// message names the reason.
func TestRunRefusesAConfiguration(t *testing.T) {
	var stderr bytes.Buffer

	status := run(context.Background(), []string{"--bind", "192.168.1.20", "--db", filepath.Join(t.TempDir(), "paca.db")}, &stderr)

	if status != exitCode {
		t.Errorf("status = %d, want %d", status, exitCode)
	}
	if !strings.Contains(stderr.String(), "paca-mcp:") {
		t.Errorf("the message is %q, want it to name the program", stderr.String())
	}
	if !strings.Contains(stderr.String(), "token") {
		t.Errorf("the message is %q, want it to name the reason", stderr.String())
	}
}

// The token never reaches the message of a refusal.
func TestRunNeverPrintsTheToken(t *testing.T) {
	const token = "s3cret-token-value"

	var stderr bytes.Buffer
	status := run(context.Background(), []string{
		"--db", filepath.Join(t.TempDir(), "paca.db"),
		"--bind", "192.168.1.20",
		"--auth-token", token,
	}, &stderr)

	if status != exitCode {
		t.Errorf("status = %d, want %d", status, exitCode)
	}
	if strings.Contains(stderr.String(), token) {
		t.Errorf("the message carries the token: %q", stderr.String())
	}
}

// A configuration that is refused before the server starts leaves the
// database untouched.
func TestRunFailsWhenTheDatabaseCannotBeOpened(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "paca.db")

	var stderr bytes.Buffer
	status := run(context.Background(), []string{"--db", path}, &stderr)

	if status != 1 {
		t.Errorf("status = %d, want %d", status, 1)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the database was created: %v", err)
	}
}

// A context that is already cancelled stops the server without serving.
func TestRunStopsWithTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stderr bytes.Buffer
	status := run(ctx, []string{"--db", filepath.Join(t.TempDir(), "paca.db")}, &stderr)

	if status != 0 {
		t.Errorf("status = %d, want %d", status, 0)
	}
}

// The body of the process installs the signal handler and refuses a
// configuration the same way the binary does.
func TestExecuteRefusesAConfiguration(t *testing.T) {
	var stderr bytes.Buffer

	status := execute([]string{"--bind", "0.0.0.0", "--db", filepath.Join(t.TempDir(), "paca.db")}, &stderr)

	if status != exitCode {
		t.Errorf("status = %d, want %d", status, exitCode)
	}
}
