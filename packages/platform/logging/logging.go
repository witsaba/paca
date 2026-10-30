// Package logging builds the structured logger of the service.
//
// The logs go to the standard output as JSON, which is what a container
// runtime collects.
//
// Every record carries the identifiers of the trace it belongs to when the
// context of the caller holds an active span. That is what ties a line of
// the log to the trace it comes from: the identifiers of the trace open it
// in the backend, and the backend shows the whole request the line belongs
// to.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// Level is the name of the variable that carries the level of the logs.
const Level = "PACA_LOG_LEVEL"

// DefaultLevel is the level of the logs of a service that names none.
const DefaultLevel = slog.LevelInfo

// ParseLevel reads the level of the logs.
//
// The error says what the level must be.
func ParseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("%s: %q is not a log level", Level, raw)
	}
}

// New builds the logger of the service.
func New(w io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(newTraceHandler(handler))
}

// traceHandler adds the identifiers of the trace to the records.
//
// The identifiers are the ones of the span that is active in the context of
// the caller. A record written without a context, or outside of a trace,
// carries none: there is no trace to point at.
type traceHandler struct {
	slog.Handler
}

// newTraceHandler wraps a handler with the identifiers of the trace.
func newTraceHandler(handler slog.Handler) slog.Handler {
	return &traceHandler{Handler: handler}
}

// Handle writes the record, with the identifiers of the active span.
func (h *traceHandler) Handle(ctx context.Context, record slog.Record) error {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return h.Handler.Handle(ctx, record)
	}

	// The record is copied before it is changed: the caller owns it, and
	// the attributes of the copy are the ones of the log.
	record = record.Clone()
	record.AddAttrs(
		slog.String("trace_id", spanContext.TraceID().String()),
		slog.String("span_id", spanContext.SpanID().String()),
	)

	return h.Handler.Handle(ctx, record)
}

// WithAttrs returns a handler that writes the records with the attributes.
func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup returns a handler that writes the records in the group.
func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithGroup(name)}
}
