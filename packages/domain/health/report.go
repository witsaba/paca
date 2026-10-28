package health

import "time"

// Identity identifies the service in a report.
type Identity struct {
	// Name is the name of the service, as the MCP protocol sees it.
	Name string
	// Version is the version of the running service.
	Version string
}

// Check is the outcome of a single probe.
type Check struct {
	// Name identifies the component the check is about.
	Name string
	// Status is the state of the component.
	Status Status
	// Latency is how long the probe took.
	Latency time.Duration
	// Detail explains the status. It is empty when everything is fine.
	Detail string
}

// Report is the health of the service at a point in time.
type Report struct {
	// Identity identifies the service the report is about.
	Identity Identity
	// Status is the aggregate of every check.
	Status Status
	// StartedAt is when the service started.
	StartedAt time.Time
	// CheckedAt is when the report was taken.
	CheckedAt time.Time
	// Uptime is how long the service has been running.
	Uptime time.Duration
	// Checks holds the outcome of every probe, in the order they ran.
	Checks []Check
}
