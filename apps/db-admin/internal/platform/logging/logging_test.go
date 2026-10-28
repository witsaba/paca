package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/apps/db-admin/internal/platform/logging"
)

// traceContext returns a context that holds an active span.
func traceContext() context.Context {
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: trace.FlagsSampled,
	})

	return trace.ContextWithSpanContext(context.Background(), spanContext)
}

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

func TestTheLoggerWritesTheIdentifiersOfTheTrace(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := logging.New(buffer, slog.LevelInfo)

	logger.InfoContext(traceContext(), "a message")

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("the log is not JSON: %v", err)
	}
	if entry["trace_id"] != "01000000000000000000000000000000" {
		t.Fatalf("trace_id = %v, want the identifier of the trace", entry["trace_id"])
	}
	if entry["span_id"] != "0200000000000000" {
		t.Fatalf("span_id = %v, want the identifier of the span", entry["span_id"])
	}
}

func TestTheLoggerWritesNoIdentifierOutsideOfATrace(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := logging.New(buffer, slog.LevelInfo)

	logger.Info("a message")

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("the log is not JSON: %v", err)
	}
	if _, ok := entry["trace_id"]; ok {
		t.Fatalf("the log holds %v, want no trace", entry)
	}
}

func TestTheLoggerKeepsTheIdentifiersWithTheAttributes(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := logging.New(buffer, slog.LevelInfo).With(slog.String("service", "paca"))

	logger.InfoContext(traceContext(), "a message")

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("the log is not JSON: %v", err)
	}
	if entry["service"] != "paca" {
		t.Fatalf("service = %v, want %q", entry["service"], "paca")
	}
	if entry["trace_id"] != "01000000000000000000000000000000" {
		t.Fatalf("trace_id = %v, want the identifier of the trace", entry["trace_id"])
	}
}
