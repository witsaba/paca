// Package env holds the conventions of the environment of the monorepository:
// the names every service reads the same way, and the way a value is read.
//
// The package reads nothing by itself: a service reads its own environment
// and hands the values over. What the package holds is what two services
// would otherwise write twice.
package env

import (
	"fmt"
	"strings"
	"time"
)

// ServiceVersion is the name of the variable that carries the version of the
// service.
const ServiceVersion = "PACA_SERVICE_VERSION"

// Duration reads a duration from the raw value of a variable.
//
// The error names the variable, so an operator knows which one to fix.
func Duration(name, raw string) (time.Duration, error) {
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a duration", name, raw)
	}
	return value, nil
}
