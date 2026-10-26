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

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
)

// MCPPath is the path of the MCP transport.
const MCPPath = "/mcp"

// RouterConfig holds what the router needs to serve the traffic.
type RouterConfig struct {
	// Health is the use case behind the health endpoint.
	Health ports.HealthQuerier
	// MCP is the handler of the MCP transport.
	MCP http.Handler
	// Logger is the logger of the router.
	Logger *slog.Logger
}

// NewRouter builds the HTTP handler of the service.
//
// The health endpoint and the MCP transport live on the same listener, so
// they share the same reachability: whoever can reach the MCP transport can
// reach the health endpoint too.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET "+HealthPath, NewHealthHandler(cfg.Health))
	mux.Handle(MCPPath, cfg.MCP)

	return recoverPanic(cfg.Logger, requestLogger(cfg.Logger, securityHeaders(mux)))
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
func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		logger.Info("request",
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
				logger.Error("the request panicked",
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
