// Package server runs the HTTP server that hosts the MCP transport.
//
// The server owns the listener, the timeouts and the graceful shutdown. It
// knows nothing about the MCP protocol.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Server runs the HTTP server of the service.
type Server struct {
	http            *http.Server
	listener        net.Listener
	addr            string
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// New builds the HTTP server.
//
// The write timeout is deliberately left unset: the MCP transport keeps
// server-sent event streams open, and a write timeout would cut them in the
// middle. The header timeout still bounds a slow client.
func New(addr string, handler http.Handler, logger *slog.Logger, shutdownTimeout time.Duration) *Server {
	return &Server{
		http: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       2 * time.Minute,
			MaxHeaderBytes:    32 << 10,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		},
		addr:            addr,
		logger:          logger,
		shutdownTimeout: shutdownTimeout,
	}
}

// Listen binds the listener of the server. It is separate from Serve so the
// caller knows the address of a server that binds port zero.
func (s *Server) Listen() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.addr, err)
	}
	s.listener = listener
	s.http.Addr = listener.Addr().String()
	s.addr = listener.Addr().String()
	return nil
}

// Addr returns the address the server listens on.
func (s *Server) Addr() string {
	return s.addr
}

// Serve serves the requests until the context is cancelled, then shuts the
// server down and waits for the requests in flight.
func (s *Server) Serve(ctx context.Context) error {
	served := make(chan error, 1)
	go func() {
		err := s.http.Serve(s.listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		served <- err
	}()

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}

	s.logger.Info("shutting down the server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut the server down: %w", err)
	}

	return <-served
}

// App is the service the server of the platform runs.
//
// The composition root of a service builds its own application and hands it
// over: the platform binds the listener and serves the requests, and it
// knows nothing about what the service does with them.
type App interface {
	// Listen binds the listener of the server.
	Listen() error
	// Serve serves the requests until the context is cancelled.
	Serve(context.Context) error
}

// Run binds the listener of the application and serves the requests until
// the context is cancelled.
//
// The sequence is the same in every service: the listener is bound before
// the requests are served, so a service that cannot bind fails before it
// accepts anything. A service that writes the sequence itself writes it
// twice.
func Run(ctx context.Context, app App) error {
	if err := app.Listen(); err != nil {
		return err
	}
	return app.Serve(ctx)
}
