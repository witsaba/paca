// Package http adapts the use cases of the service to the HTTP protocol.
//
// The adapters translate: they read a request, they call the use case, and
// they write a response. They hold no rule of their own.
package http

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/filters"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/witsaba/paca/apps/db-admin/internal/ports"
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
// Every request leaves a span behind, named after the route it took. The
// health endpoint is the exception: the container runtime asks it every few
// seconds, and those spans would fill the traces of the work without
// telling anything about it.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET "+HealthPath, NewHealthHandler(cfg.Health))

	handler := recoverPanic(cfg.Logger, requestLogger(cfg.Logger, securityHeaders(mux)))

	return otelhttp.NewHandler(handler, "http",
		otelhttp.WithTracerProvider(tracerProvider(cfg.TracerProvider)),
		otelhttp.WithFilter(filters.Not(filters.Path(HealthPath))),
	)
}

// tracerProvider returns the provider of the router, or the no-op one when
// the router has none.
func tracerProvider(provider trace.TracerProvider) trace.TracerProvider {
	if provider == nil {
		return noop.NewTracerProvider()
	}
	return provider
}

// securityHeaders sets the headers that keep a browser away from the
// responses of the service.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// requestLogger writes one line per request.
//
// The line carries the context of the request, so the logger adds the
// identifiers of the trace the request belongs to.
func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		logger.InfoContext(r.Context(), "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Duration("duration", time.Since(started)),
		)
	})
}

// recoverPanic turns a panic into a 500, so one broken request never takes
// the server down.
func recoverPanic(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(r.Context(), "the request panicked",
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
				)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// statusRecorder remembers the status of a response.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

// WriteHeader remembers the status of the response.
func (s *statusRecorder) WriteHeader(status int) {
	if !s.written {
		s.status = status
		s.written = true
	}
	s.ResponseWriter.WriteHeader(status)
}

// Write remembers the status of the response, which defaults to 200.
func (s *statusRecorder) Write(body []byte) (int, error) {
	if !s.written {
		s.status = http.StatusOK
		s.written = true
	}
	return s.ResponseWriter.Write(body)
}
