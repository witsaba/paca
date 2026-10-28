// The domain of the PACA monorepository.
//
// The module holds the models of the domain and the ports of the storage:
// what a service needs to know about a person, and what it expects from
// the place that keeps the people.
//
// It is a library. It publishes no port of its own on the network, it
// starts no process, and it reads no environment variable.
//
// The dependency rule of the hexagonal architecture holds here, and the
// toolchain enforces it: the module imports nothing but the standard
// library. A model that reached for a driver, for a table or for a
// connection string would break the build of every service of the
// monorepository.
//
// The Go version is pinned to the latest security release of the 1.27 line.
// The toolchain downloads it automatically when the local Go is older.
module github.com/witsaba/paca/packages/domain

go 1.27.3
