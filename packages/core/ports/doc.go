// Package ports declares the boundaries between the inside of PACA and the
// outside world.
//
// A port is an interface owned by the application, not by the adapter that
// implements it. There are two kinds:
//
//   - Driving ports (the input side) are what a delivery mechanism calls: a use
//     case such as "list the accounts" is exposed to the CLI as a driving port
//     and implemented by the application layer.
//   - Driven ports (the output side) are what the application needs from the
//     outside: a repository, a clock, a logger, an id generator. Each one is an
//     interface declared here and implemented by an adapter.
//
// Dependency rule: this package imports only the standard library and the
// domain package. It must never import an adapter, so that the concrete
// technology behind a port — SQLite, the system clock, a file — can be replaced
// without touching the application layer.
package ports
