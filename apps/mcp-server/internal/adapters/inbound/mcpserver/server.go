// Package mcpserver adapts the service to the Model Context Protocol.
//
// The transport is the official Go SDK of the protocol. It registers no tool
// yet: this is the initialization of the server, not its first feature.
package mcpserver

import (
	"log/slog"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
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
}

// Server is the MCP transport of the service.
type Server struct {
	server  *sdk.Server
	handler http.Handler
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

	return &Server{
		server:  server,
		handler: protection.Handler(handler),
	}
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
