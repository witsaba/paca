// Package mcpserver adapts the service to the Model Context Protocol.
//
// The transport is the official Go SDK of the protocol. It registers no tool
// yet: this is the initialization of the server, not its first feature.
package mcpserver

import (
	"log/slog"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
)

// Options configures the MCP transport.
type Options struct {
	// ServiceName is the name the MCP clients see.
	ServiceName string
	// ServiceVersion is the version the MCP clients see.
	ServiceVersion string
	// Instructions tells the MCP clients what the server is for.
	Instructions string
	// Logger is the logger of the transport.
	Logger *slog.Logger
	// MaxRequestBodyBytes is the largest request body the transport accepts.
	MaxRequestBodyBytes int64
	// Users is the use case of the users, the one behind the three tools.
	//
	// It may be nil: a server without a database still exposes the tools,
	// and every call answers that the database is not there.
	Users ports.UsersQuerier
}

// Server is the MCP transport of the service.
type Server struct {
	server  *sdk.Server
	handler http.Handler
	logger  *slog.Logger
}

// New builds the MCP transport.
//
// The transport is stateless: the protocol revision the SDK negotiates has no
// protocol-level session, and a stateless server cannot leak one.
//
// The protections of the SDK stay on. The DNS-rebinding protection rejects a
// request that arrives through the loopback interface with a foreign Host
// header, and the body size limit rejects an oversized request. The cross
// origin protection of the standard library rejects a cross-origin request
// from a browser, and lets the requests of a non-browser client through.
//
// The three tools of the users are registered here: the list of the tools is
// a property of the transport, and it does not depend on the database.
func New(opts Options) *Server {
	server := sdk.NewServer(
		&sdk.Implementation{
			Name:    opts.ServiceName,
			Title:   opts.ServiceName,
			Version: opts.ServiceVersion,
		},
		&sdk.ServerOptions{
			Logger:       opts.Logger,
			Instructions: opts.Instructions,
		},
	)

	transport := &Server{server: server, logger: opts.Logger}
	if transport.logger == nil {
		transport.logger = slog.Default()
	}

	transport.registerUserTools(opts.Users)

	handler := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server { return server },
		&sdk.StreamableHTTPOptions{
			Logger:                       opts.Logger,
			Stateless:                    true,
			MaxRequestBodyBytes:          opts.MaxRequestBodyBytes,
			PropagateRequestCancellation: true,
		},
	)

	protection := http.NewCrossOriginProtection()

	transport.handler = protection.Handler(handler)

	return transport
}

// Handler returns the HTTP handler of the transport.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// SDKServer returns the SDK server under the transport, so the composition
// root and the tests can reach it.
func (s *Server) SDKServer() *sdk.Server {
	return s.server
}
