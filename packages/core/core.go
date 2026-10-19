// Package core holds the identity of PACA and the entry point to the layers
// that both delivery mechanisms share.
//
// It is the root package of the shared core module. The inner layers live in
// the subpackages of this module:
//
//   - domain — the business model: entities, value objects, invariants.
//   - ports — the boundaries between the inside of PACA and the outside world.
//   - application — the use cases, which orchestrate the domain through ports.
//
// Dependency rule: this package imports nothing from the project, and the
// layers it documents import nothing but each other and the standard library.
// The shared core never imports a delivery mechanism, so the same rules apply
// whether PACA is driven by the CLI, by an MCP server, or by a desktop client.
package core

const (
	// ProductName is the name of the product every delivery mechanism
	// presents: the CLI, the MCP server, and any future client.
	ProductName = "PACA"

	// ProductDescription is the one-line description of the product.
	ProductDescription = "a personal finance manager for the Colombian market"
)
