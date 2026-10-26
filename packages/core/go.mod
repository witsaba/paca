// The core of the PACA monorepository.
//
// The module holds what the services of the monorepository share: the models
// of the domain, the contracts of the repositories and the migrations of the
// schema. It is a library: it publishes no port and runs no container.
//
// The dependency rule holds: the domain knows nothing about the database, the
// adapters know the contracts, and the services know the core.
//
// The Go version is pinned to the latest security release of the 1.27 line.
// The toolchain downloads it automatically when the local Go is older.
module github.com/witsaba/paca/packages/core

go 1.27.3
