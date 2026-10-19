// Package domain holds the core business model of PACA: entities, value objects,
// invariants, and the domain services that enforce them.
//
// This is the innermost layer of the hexagonal architecture. It is the one place
// where the rules of personal finance management live, and it is the layer that
// must survive every delivery mechanism: the same rules apply whether PACA is
// driven by the CLI, by a desktop client, or by an API.
//
// Dependency rule: this package imports nothing from the project except the
// standard library. It never imports an adapter, an application package, or any
// I/O library, and it never touches the database, the clock, the filesystem, or
// the network. Anything the domain needs from the outside world arrives as a
// value that is already in memory, or through a port declared in
// [github.com/witsaba/paca/packages/core/ports].
//
// The data currently stored by PACA is described in the repository README, which
// documents the SQLite schema under packages/db. That schema is the persistence
// view of this model, not the model itself: the domain owns the meaning, the
// schema owns the storage.
package domain
