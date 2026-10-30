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

// The names of the environment of the database.
//
// They are the names of the core, and every service of the monorepository
// reads them the same way: the names live here, and the configuration of a
// service names them through this package instead of writing them twice.
const (
	// DatabaseURL is the whole connection string of the database. When it
	// carries a value, it wins, and nothing else of the database is read.
	DatabaseURL = "PACA_DATABASE_URL"
	// DatabaseHost is the host the database answers on. It is only read
	// when the connection string is assembled from the credentials of the
	// PostgreSQL stack.
	DatabaseHost = "PACA_DATABASE_HOST"
	// DatabasePort is the port the database answers on. It is only read
	// when the connection string is assembled from the credentials of the
	// PostgreSQL stack.
	DatabasePort = "PACA_DATABASE_PORT"
	// PostgresUser is the role that owns the application database.
	PostgresUser = "POSTGRES_USER"
	// PostgresPassword is the password of the role.
	PostgresPassword = "POSTGRES_PASSWORD" //nolint:gosec // the name of the environment variable, not a credential.
	// PostgresDB is the application database.
	PostgresDB = "POSTGRES_DB"
)

// The defaults of the environment of the database.
//
// They are the container and the port of the PostgreSQL stack of the
// monorepository: the database of the monorepository answers there.
const (
	// DefaultDatabaseHost is the name of the container of the database of
	// the monorepository.
	DefaultDatabaseHost = "postgres"
	// DefaultDatabasePort is the port the database of the monorepository
	// answers on.
	DefaultDatabasePort = "5432"
)

// Lookup is how a service reads one variable of its environment.
//
// The platform reads no environment variable: a service hands it the
// reader, and the reader is the one that knows where the variables live.
type Lookup func(name string) (string, bool)

// Value reads a variable of the environment and reports whether it carries
// a value.
//
// The value is trimmed: a variable that carries spaces carries nothing but
// the spaces, and a host with a space in it is a host nobody answers on.
//
// An empty value is no value. The installer of the environment of the
// monorepository never writes an empty value, so a variable that is set to
// nothing is a variable that is not set, and the rule lives here so that
// every service of the monorepository reads the environment the same way.
func Value(lookup Lookup, name string) (string, bool) {
	raw, ok := lookup(name)
	if !ok {
		return "", false
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}

	return value, true
}

// Secret reads a secret of the environment and reports whether it carries
// one.
//
// The secret is never trimmed: a password may hold a space, and a password
// that is trimmed is a password that is wrong.
//
// An empty secret is no secret, for the same reason an empty value is no
// value.
func Secret(lookup Lookup, name string) (string, bool) {
	raw, ok := lookup(name)
	if !ok {
		return "", false
	}

	if raw == "" {
		return "", false
	}

	return raw, true
}

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
