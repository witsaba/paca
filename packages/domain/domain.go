// Package domain holds the models of the PACA monorepository.
//
// The models are pure: they know nothing about SQL, about a table, or
// about a driver. The adapter that reads a table maps its rows onto these
// models, and the models carry the rules that a row cannot.
//
// The module imports nothing but the standard library, and the compiler
// says so: a model that reached for a driver would break the build of
// every service of the monorepository.
//
//	domain/user/   the model of the person that PACA knows: the identifier,
//	               the states and the port of the storage
//	domain/schema/ the state of the schema of the database: the version, the
//	               migrations and what is pending
package domain
