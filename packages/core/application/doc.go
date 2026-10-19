// Package application holds the use cases of PACA: the operations that a user
// or an external system can ask the application to perform.
//
// A use case orchestrates. It loads the state it needs through a port, applies
// the rules owned by the domain package, and stores the result back through a
// port. It contains no I/O of its own: an application use case never opens a
// database connection, never reads a file, and never writes to stdout.
//
// Dependency rule: this package imports the domain package and the ports
// package, and nothing else from the project. It never imports an adapter, so
// the same use cases can be driven by the CLI, by a desktop client, or by an
// API server without duplication.
package application
