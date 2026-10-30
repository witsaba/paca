// Package http adapts the use cases of the service to the HTTP protocol.
//
// The adapters translate: they read a request, they call the use case, and
// they write a response. They hold no rule of their own.
package http

import (
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/apps/db-admin/internal/ports"
	"github.com/witsaba/paca/packages/platform/server"
)

// RouterConfig holds what the router needs to serve the traffic.
type RouterConfig struct {
	// Health is the use case behind the health endpoint.
	Health ports.HealthQuerier
	// Logger is the logger of the router.
	Logger *slog.Logger
	// TracerProvider creates the spans of the requests. A router without
	// one does not trace the requests it serves.
	TracerProvider trace.TracerProvider
}

// NewRouter builds the HTTP handler of the service.
//
// The health endpoint is the only route of the service: the service does
// nothing but the database, and the health endpoint is how it says so.
//
// The middleware of the server belongs to the platform: the tracing, the
// security headers, the line of the log, the recovery of a panic and the
// recorder of the status are the same in every service.
//
// Every request leaves a span behind, named after the route it took. The
// health endpoint is the exception: the container runtime asks it every few
// seconds, and those spans would fill the traces of the work without
// telling anything about it.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET "+HealthPath, NewHealthHandler(cfg.Health))

	return server.NewMiddleware(server.MiddlewareConfig{
		Logger:         cfg.Logger,
		TracerProvider: cfg.TracerProvider,
		UntracedPaths:  []string{HealthPath},
	}, mux)
}
