package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/platform/logging"
)

func TestTheLoggerWritesTheLogsAsJSON(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := logging.New(buffer, slog.LevelInfo)

	logger.Info("the server is serving", slog.String("addr", "127.0.0.1:8081"))

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("the log is not JSON: %v", err)
	}
	if entry["msg"] != "the server is serving" {
		t.Fatalf("message = %v, want the message of the log", entry["msg"])
	}
	if entry["addr"] != "127.0.0.1:8081" {
		t.Fatalf("addr = %v, want the address of the server", entry["addr"])
	}
}

func TestTheLoggerKeepsTheLogsBelowItsLevel(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := logging.New(buffer, slog.LevelInfo)

	logger.Debug("a message the level does not allow")

	if buffer.Len() != 0 {
		t.Fatalf("the log holds %q, want nothing", buffer.String())
	}
}

func TestTheLoggerWritesTheLevelOfTheEntry(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := logging.New(buffer, slog.LevelWarn)

	logger.Warn("a warning")

	if !strings.Contains(buffer.String(), "WARN") {
		t.Fatalf("the log holds %q, want a warning", buffer.String())
	}
}
