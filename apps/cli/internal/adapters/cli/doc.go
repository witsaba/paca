// Package cli is the driving adapter of PACA: it turns command line input into
// calls on the application layer, and turns the results back into output.
//
// This package owns the command tree, the flags, the help text, and the
// presentation of errors. It is the only place in the application that knows
// that the delivery mechanism is a terminal.
//
// Dependency rule: this package imports the application and ports packages, and
// the domain package when it needs to name a domain type. It never imports the
// database, and it never implements a driven port — a repository or a clock
// belongs to a different adapter.
//
// Future commands (accounts, movements, loans, reports) are added as subcommands
// of the root command built here, one file per command, each with its own test.
package cli
