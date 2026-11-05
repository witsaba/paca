// The health library of the PACA monorepository.
//
// The module holds the health of a service: the use case that reports it and
// the HTTP endpoint that answers it. It is a library: it publishes no port
// and runs no container.
//
// The dependency rule holds: the model of the health lives in another
// module, packages/domain, and this one depends on it. The domain imports
// nothing but the standard library, so it cannot reach the HTTP endpoint of
// this module.
//
// The Go version is pinned to the latest security release of the 1.27 line.
// The toolchain downloads it automatically when the local Go is older.
module github.com/witsaba/paca/packages/health

go 1.27.3

require github.com/witsaba/paca/packages/domain v0.0.0

// The domain is a module of its own, next to this one: the workspace of the
// repository resolves it, and the replace keeps `go mod tidy` working
// inside the module alone.
replace github.com/witsaba/paca/packages/domain => ../domain
