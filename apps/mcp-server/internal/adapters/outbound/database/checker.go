// Package database provides the checker that reports whether the database of
// the service answers.
//
// It is an outbound adapter: the application asks it about a component of
// the service, and the answer comes from the database itself. The pool and
// the driver belong to the core, so the adapter never names them.
package database

import (
	"context"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
)

// Checker reports whether the database answers.
type Checker struct {
	pinger ports.Pinger
}

// NewChecker builds the checker of the database.
//
// The pinger is the port of the capability: the pool of the core answers the
// question, and the adapter never names the handle that answers it.
func NewChecker(pinger ports.Pinger) *Checker {
	return &Checker{pinger: pinger}
}

// Name identifies the component the checker watches.
func (c *Checker) Name() string {
	return "database"
}

// Check asks the database whether it answers.
//
// The question goes through the pool of the core, on the same connection
// and with the same driver as everything else the service does with the
// database.
func (c *Checker) Check(ctx context.Context) error {
	return c.pinger.Ping(ctx)
}
