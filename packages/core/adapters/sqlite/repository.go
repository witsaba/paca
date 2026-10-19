package sqlite

import (
	"github.com/witsaba/paca/packages/core/domain"
)

// The conventions the repositories of this package share.
const (
	// defaultCurrency is the currency of an account when the caller names
	// none. MVP1 is Colombian pesos only.
	defaultCurrency = "COP"
)

// optionalText returns the value of an optional text, or nil when the value
// is empty.
//
// The schema stores the absence of a value as NULL, and not as an empty
// string: the two are different things, and a report that groups by a
// category has to tell them apart.
func optionalText(value string) any {
	if value == "" {
		return nil
	}

	return value
}

// notFound rewrites the detail of a not-found failure with one that names
// the entity the caller was looking for.
//
// A failure that is not a not-found failure is returned untouched: a
// repository never turns another failure into one it understands better.
func notFound(err error, detail string) error {
	if domain.CodeOf(err) != domain.CodeNotFound {
		return err
	}

	return domain.Wrap(domain.CodeNotFound, detail, err)
}
