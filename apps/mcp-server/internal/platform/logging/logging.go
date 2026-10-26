// Package logging builds the structured logger of the service.
//
// The logs go to the standard output as JSON, which is what a container
// runtime collects.
package logging

import (
	"io"
	"log/slog"
)

// New builds the logger of the service.
func New(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}
