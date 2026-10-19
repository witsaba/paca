package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/witsaba/paca/apps/mcp/internal/config"
)

const (
	// shutdownTimeout is how long the server waits for the requests in
	// flight when it is asked to stop.
	shutdownTimeout = 5 * time.Second
	// readHeaderTimeout bounds how long a client may take to send the
	// headers of a request.
	readHeaderTimeout = 10 * time.Second
)

// Run builds the server and serves it until the context is cancelled.
//
// The listener is plain HTTP on a loopback address and TLS everywhere else,
// which the validation of the configuration already guaranteed. The startup
// log names the address and whether the transport and the authentication are
// on; it never names the token.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	if err := ctx.Err(); err != nil {
		return nil
	}

	s, err := New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := s.Close(); closeErr != nil {
			logger.Error("the database was not closed", "error", closeErr)
		}
	}()

	address := net.JoinHostPort(cfg.Bind, strconv.Itoa(cfg.Port))
	httpServer := &http.Server{
		Addr:              address,
		Handler:           s.Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("the server did not shut down cleanly", "error", shutdownErr)
		}
	}()

	logger.Info("paca-mcp is listening",
		"address", address,
		"tls", cfg.TLSEnabled(),
		"auth", cfg.AuthToken != "",
		"database", cfg.DBPath,
	)

	var serveErr error
	if cfg.TLSEnabled() {
		serveErr = httpServer.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
	} else {
		serveErr = httpServer.ListenAndServe()
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		<-stopped
		return nil
	}
	return serveErr
}
