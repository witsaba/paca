// Package core is the shared persistence library of the PACA monorepository.
//
// The services of the monorepository import it: the MCP server today, the
// worker and the REST backend tomorrow. It holds what they would otherwise
// write twice — the models of the domain, the contracts of the repositories
// and the migrations of the schema.
//
// The dependency rule of the hexagonal architecture holds here too:
//
//	domain/    the models and the rules of the domain. They know nothing
//	           about the database.
//	postgres/  the adapter that talks to PostgreSQL. It knows the models
//	           and it knows nothing about the services.
//	queries/   the SQL of the queries, read by the generator.
//	migrations/ the history of the schema, applied by a single writer.
//
// The library publishes no port and runs no container. A service maps its
// own environment onto the configuration of the core, opens the pool and
// uses the models.
//
// The primary key of the tables is the national document ID of the person,
// which the caller supplies. Nothing in the core generates an identifier.
package core
