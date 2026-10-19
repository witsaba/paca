// Package sqlite is the SQLite adapter of PACA: the concrete implementation
// of the repository ports of packages/core/ports, over the schema published by
// packages/db.
//
// Dependency rule: this package imports the domain and the ports of the shared
// core, and the driver. It never imports a delivery mechanism: the CLI and
// the MCP server both reach the same adapter through the same ports, and
// neither of them knows that the storage is SQLite.
//
// The error contract of this package is the C7 taxonomy of the domain: a
// failure that the driver reports becomes a typed domain error, and the
// message of the driver never reaches the detail of that error.
package sqlite
