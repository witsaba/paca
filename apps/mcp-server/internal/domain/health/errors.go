package health

import "errors"

// ErrDegraded is what a checker returns when the component it watches still
// works, but not perfectly. It is the only error that does not make the
// service unhealthy.
var ErrDegraded = errors.New("degraded")
