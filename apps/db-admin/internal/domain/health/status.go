// Package health holds the health model of the database admin service: the
// state of the service, the outcome of a single check, and the report that
// aggregates them.
//
// The model is pure. It knows nothing about HTTP, about PostgreSQL, or about
// the process that hosts the service, so the rules of the health use case
// can be tested without a server.
package health

// Status is the state of the service, or of one of its components.
type Status string

const (
	// StatusOK means that everything works as expected.
	StatusOK Status = "ok"
	// StatusDegraded means that the service still works, but something is
	// wrong with one of its components.
	StatusDegraded Status = "degraded"
	// StatusUnhealthy means that the service cannot serve.
	StatusUnhealthy Status = "unhealthy"
)

// rank orders the statuses from the healthiest to the sickest, so a set of
// checks collapses into a single status.
func (s Status) rank() int {
	switch s {
	case StatusOK:
		return 0
	case StatusDegraded:
		return 1
	case StatusUnhealthy:
		return 2
	default:
		// An unknown status is never trusted: it is treated as unhealthy.
		return 2
	}
}

// Worst returns the sickest of the two statuses. It is how the report
// aggregates the checks: one failing component makes the whole service
// unhealthy.
func (s Status) Worst(other Status) Status {
	if other.rank() > s.rank() {
		return other.canonical()
	}
	return s.canonical()
}

// canonical maps an unknown status to the sickest known one: a status the
// model does not know is never trusted.
func (s Status) canonical() Status {
	switch s {
	case StatusOK, StatusDegraded, StatusUnhealthy:
		return s
	default:
		return StatusUnhealthy
	}
}

// String returns the status as the text the API exposes.
func (s Status) String() string {
	return string(s)
}

// IsHealthy reports whether the service is completely healthy.
func (s Status) IsHealthy() bool {
	return s == StatusOK
}

// IsServing reports whether the service is able to serve traffic. A degraded
// service still serves; an unhealthy one does not.
func (s Status) IsServing() bool {
	return s != StatusUnhealthy
}
