// Package clock provides the clock of the service.
//
// The clock is an outbound port of the application, so the use cases can be
// tested with a clock the test controls.
package clock

import "time"

// System is the clock of the host.
type System struct{}

// Now returns the current time.
func (System) Now() time.Time {
	return time.Now()
}
