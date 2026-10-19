// Package server is the MCP server of PACA: the transport, the liveness
// endpoint and the composition of the two.
//
// The package owns no business rule. It serves the MCP surface of the
// official Go SDK over Streamable HTTP, and it keeps the liveness endpoint
// out of that surface: /healthz answers whether the process is alive, and
// nothing else.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp/internal/auth"
	"github.com/witsaba/paca/apps/mcp/internal/config"
	"github.com/witsaba/paca/packages/core/adapters/sqlite"
)

const (
	// MCPPath is the path the MCP surface is served on.
	MCPPath = "/mcp"
	// HealthPath is the path the liveness endpoint is served on.
	HealthPath = "/healthz"
	// serverVersion is the version the server reports to the MCP client.
	serverVersion = "0.1.0"
)

// Server is the MCP server of PACA.
//
// It holds the database open for the whole life of the process, as ADR 0001
// (C8) requires: the migration procedure is stop server, apply migration,
// start server.
type Server struct {
	cfg     config.Config
	db      *sqlite.DB
	handler http.Handler
}

// New builds the server and opens the database it serves.
//
// The database is migrated at startup, as the session decided: the runner of
// the shared core is idempotent, so an already migrated file is untouched
// and a new file is created with the schema.
func New(ctx context.Context, cfg config.Config) (*Server, error) {
	db, err := sqlite.Open(ctx, cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open the database: %w", err)
	}
	if err := sqlite.MigrateSchema(ctx, db); err != nil {
		closeErr := db.Close()
		return nil, errors.Join(fmt.Errorf("migrate the database: %w", err), closeErr)
	}

	s := &Server{cfg: cfg, db: db}
	s.handler = s.buildHandler()
	return s, nil
}

// buildHandler builds the HTTP surface of the server.
//
// The liveness endpoint is mounted apart from the MCP surface and outside
// the authentication: a probe must answer even when the MCP surface is
// closed to the world.
func (s *Server) buildHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle(HealthPath, HealthHandler())

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "paca-mcp",
		Version: serverVersion,
	}, nil)
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, nil)
	mux.Handle(MCPPath, auth.Middleware(s.cfg.AuthToken)(streamable))

	return mux
}

// Handler returns the HTTP handler of the server.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// Close releases the database the server holds open.
func (s *Server) Close() error {
	return s.db.Close()
}
