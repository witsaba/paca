// Package serving provides the checker that reports whether the MCP
// transport of the service is serving.
//
// It is an outbound adapter: the application asks it about a component of
// the service, and the answer comes from the transport itself.
package serving

import (
	"context"
	"errors"
	"sync"
)

// ErrNotServing is what the checker reports while the transport has not
// started serving yet.
var ErrNotServing = errors.New("the MCP transport is not serving yet")

// Checker reports whether the MCP transport is serving.
//
// The state is safe for concurrent use: the composition root sets it while
// the server starts, and the health use case reads it from the requests that
// arrive at the same time.
type Checker struct {
	mutex   sync.RWMutex
	serving bool
}

// NewChecker builds the checker of the MCP transport.
func NewChecker() *Checker {
	return &Checker{}
}

// Name identifies the component the checker watches.
func (c *Checker) Name() string {
	return "mcp-transport"
}

// Check reports whether the transport is serving.
func (c *Checker) Check(_ context.Context) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.serving {
		return ErrNotServing
	}
	return nil
}

// SetServing records whether the transport is serving.
func (c *Checker) SetServing(serving bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.serving = serving
}
