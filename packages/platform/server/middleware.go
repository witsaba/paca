package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/filters"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// MiddlewareConfig holds what the middleware of the HTTP server needs.
//
// The middleware belongs to the platform because it belongs to the server:
// every service that serves HTTP needs it the same way, and a copy of it in
// two services is two promises that drift apart.
type MiddlewareConfig struct {
	// Logger writes the line of every request. A middleware without one
	// writes the lines with the default logger of the process.
	Logger *slog.Logger
	// TracerProvider creates the spans of the requests. A middleware
	// without one does not trace the requests it serves.
	TracerProvider trace.TracerProvider
	// UntracedPaths are the paths that leave no span behind. The
	// container runtime asks the health endpoint every few seconds, and
	// those spans would fill the traces of the work without telling
	// anything about it.
	UntracedPaths []string
}

// NewMiddleware wraps a handler with the middleware of the HTTP server.
//
// The chain is the same for every service: the tracing is the outer layer,
// so the span of a request covers the recovery of a panic and the line of
// the log; the recovery is the outer layer of the work, so a broken request
// never takes the server down; the line of the log knows the status of the
// response, so the recorder sits between the log and the handler; and the
// security headers are set before the handler writes anything.
func NewMiddleware(cfg MiddlewareConfig, next http.Handler) http.Handler {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	handler := recoverPanic(logger, requestLogger(logger, securityHeaders(next)))

	options := []otelhttp.Option{
		otelhttp.WithTracerProvider(tracerProvider(cfg.TracerProvider)),
	}
	if filter := untracedFilter(cfg.UntracedPaths); filter != nil {
		options = append(options, otelhttp.WithFilter(filter))
	}

	return otelhttp.NewHandler(handler, "http", options...)
}

// untracedFilter returns the filter that keeps the named paths out of the
// traces, or nil when no path is named: the filter of the tracing then
// traces every request.
func untracedFilter(paths []string) otelhttp.Filter {
	if len(paths) == 0 {
		return nil
	}

	excluded := make([]otelhttp.Filter, 0, len(paths))
	for _, path := range paths {
		excluded = append(excluded, filters.Path(path))
	}

	return filters.Not(filters.Any(excluded...))
}

// tracerProvider returns the provider of the middleware, or the no-op one
// when the middleware has none.
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
