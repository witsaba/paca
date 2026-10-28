// Package config assembles the connection string of the database.
//
// The library reads no environment variable: a service maps its own
// environment onto the credentials of the database, so the same library
// serves every service of the monorepository with its own settings.
//
// The package is pure. It reads no variable, no file and no clock: it takes
// the credentials and returns the connection string. The resolution of
// which variable wins, the defaults and the validation of the values stay
// in the service, because every service states them in its own words.
package config

import (
	"net"
	"net/url"
)

// Credentials holds the credentials of a PostgreSQL database.
//
// The fields are named, because two services assembled the same connection
// string from the same values in a different order, and a mistake in the
// order is a mistake the compiler does not see.
type Credentials struct {
	// User is the role that owns the database.
	User string
	// Password is the password of the role.
	Password string
	// Database is the name of the database.
	Database string
	// Host is the host the database answers on.
	Host string
	// Port is the port the database answers on.
	Port string
}

// ConnectionString assembles the connection string of the database from its
// credentials.
//
// The user and the password are escaped the way a URL carries them, so a
// password with an `@`, a `/` or a `:` in it does not change the meaning of
// the connection string. The password is never trimmed: a password may hold
// a space, and a password that is trimmed is a password that is wrong.
//
// The port is a string, and the service that reads it owns its validation:
// the library does not decide what a port may be.
func ConnectionString(c Credentials) string {
	connection := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     net.JoinHostPort(c.Host, c.Port),
		Path:     "/" + c.Database,
		RawQuery: "sslmode=disable",
	}

	return connection.String()
}
