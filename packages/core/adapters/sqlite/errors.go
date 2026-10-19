package sqlite

import (
	"database/sql"
	"errors"

	sqliteDriver "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/witsaba/paca/packages/core/domain"
)

// The result codes of SQLite that the adapter tells apart. The extended codes
// carry the primary code in their low byte, so masking a code is what tells a
// constraint violation from any other failure.
const (
	codeConstraint  = sqlite3.SQLITE_CONSTRAINT
	codeForeignKey  = sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY
	codeNotNull     = sqlite3.SQLITE_CONSTRAINT_NOTNULL
	codeUnique      = sqlite3.SQLITE_CONSTRAINT_UNIQUE
	codePrimaryKey  = sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
	codeCheck       = sqlite3.SQLITE_CONSTRAINT_CHECK
	codeTrigger     = sqlite3.SQLITE_CONSTRAINT_TRIGGER
	maskPrimaryCode = 0xff
)

// The details of the failures. They say what went wrong in the language of
// the caller, and never repeat the message of the driver.
const (
	detailDuplicate    = "a record with the same identity already exists"
	detailRequired     = "a required field is missing"
	detailCheck        = "the value breaks a rule of the schema"
	detailForeignKey   = "the request names a record that does not exist"
	detailConstraint   = "the request breaks a rule of the schema"
	detailNotFound     = "the requested record does not exist"
	detailUnclassified = ""
)

// translate turns a failure of the driver into a failure of the domain.
//
// The message of the driver never reaches the detail: the detail travels to
// an agent, and the internals of the storage do not. The cause stays
// reachable through errors.Is and errors.As, so a caller can log it.
func translate(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return domain.Wrap(domain.CodeNotFound, detailNotFound, err)
	}

	var driverErr *sqliteDriver.Error
	if errors.As(err, &driverErr) {
		if code := driverErr.Code() & maskPrimaryCode; code == codeConstraint {
			return domain.Wrap(domain.CodeValidationFailed, constraintDetail(driverErr.Code()), err)
		}
	}

	return domain.Wrap(domain.CodeInternal, detailUnclassified, err)
}

// constraintDetail returns the detail of a constraint violation, from its
// extended result code.
func constraintDetail(code int) string {
	switch code {
	case codeUnique, codePrimaryKey:
		return detailDuplicate
	case codeNotNull:
		return detailRequired
	case codeCheck:
		return detailCheck
	case codeForeignKey:
		return detailForeignKey
	case codeTrigger:
		return detailConstraint
	default:
		return detailConstraint
	}
}
