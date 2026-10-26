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

require (
	github.com/jackc/pgx/v5 v5.11.0
	github.com/pressly/goose/v3 v3.29.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.3 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/sethvargo/go-retry v0.5.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)
