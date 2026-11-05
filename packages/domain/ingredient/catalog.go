package ingredient

import (
	"fmt"
	"sort"
	"strings"

	"github.com/witsaba/paca/packages/domain/business"
)

// Status is the answer of the catalog about a name the owner said.
type Status string

const (
	// StatusExact is the answer of a name that one ingredient answers.
	StatusExact Status = "exact"

	// StatusAmbiguous is the answer of a name that several ingredients
	// answer: the options are offered, so the owner chooses with one word.
	StatusAmbiguous Status = "ambiguous"

	// StatusUnknown is the answer of a name that no ingredient answers: the
	// catalog does not know it, and the name is new.
	StatusUnknown Status = "unknown"
)

// String returns the name of the status.
//
// It is the name the storage and the protocol carry, with no translation in
// between.
func (s Status) String() string {
	return string(s)
}

// Option is one of the candidates of an ambiguous answer.
type Option struct {
	// ID is the identity of the ingredient.
	ID ID

	// Name is the name the owner reads.
	Name string
}

// Match is the answer of the catalog about a name the owner said.
type Match struct {
	// Name is the name as she said it.
	Name string

	// Status is the answer of the catalog.
	Status Status

	// Ingredient is the ingredient of an exact answer. It is empty when the
	// answer is not exact.
	Ingredient Ingredient

	// Options are the candidates of an ambiguous answer. They are in the
	// order of the identity of the ingredients.
	Options []Option

	// New says that the catalog does not know the name. A name the catalog
	// does not know is not an error: it is recorded as she said it.
	New bool
}

// Resolution is what the catalog answers about the name of a line.
type Resolution struct {
	// Ingredient is the ingredient of the line. It is empty when the
	// catalog cannot answer.
	Ingredient Ingredient

	// New says that the catalog does not know the name: the catalog grows
	// with it, and the name is recorded as she said it.
	New bool
}

// Catalog is the set of the ingredients of a business.
//
// The catalog begins empty and grows with what the owner buys. The order of
// the ingredients is the order of their identity, so the answer of the
// catalog is the same every time.
type Catalog struct {
	// own is the business the catalog belongs to.
	own business.ID

	// items are the ingredients of the catalog, in the order of their
	// identity.
	items []Ingredient
}

// NewCatalog builds the catalog of a business.
//
// The rule of the domain: two ingredients of the same business never carry
// the same name, and the name is compared the way a person reads it:
// trimmed, and without case. The error is ErrDuplicateName.
//
// An ingredient of another business is not in this catalog: the error is
// ErrWrongBusiness.
func NewCatalog(own business.ID, items []Ingredient) (Catalog, error) {
	kept := make([]Ingredient, 0, len(items))
	seen := map[string]bool{}

	for _, one := range items {
		if one.BusinessID != own {
			return Catalog{}, fmt.Errorf("%w: the ingredient %q is of the business %d",
				ErrWrongBusiness, one.Name, one.BusinessID)
		}

		key := keyOf(one.Name)
		if seen[key] {
			return Catalog{}, fmt.Errorf("%w: the business holds two ingredients named %q",
				ErrDuplicateName, one.Name)
		}

		seen[key] = true
		kept = append(kept, one)
	}

	sort.Slice(kept, func(first, second int) bool {
		return kept[first].ID < kept[second].ID
	})

	return Catalog{own: own, items: kept}, nil
}

// Match answers the name the owner said.
//
// The rules of the domain: the answer is exact when one ingredient answers
// the name, ambiguous when several answer it, and unknown when none does.
// The options of an ambiguous answer are in the order of the identity of
// the ingredients, so the answer is the same every time.
//
// A name the catalog does not know is not an error: the answer says that it
// is new.
//
// The error is ErrInvalidName when the name is empty: the owner said
// something, and the catalog never answers about nothing.
func (c Catalog) Match(spoken string) (Match, error) {
	name := strings.TrimSpace(spoken)
	if name == "" {
		return Match{}, fmt.Errorf("%w: the name the owner said is not empty", ErrInvalidName)
	}

	found := make([]Ingredient, 0, len(c.items))
	for _, one := range c.items {
		if one.answers(name) {
			found = append(found, one)
		}
	}

	switch len(found) {
	case 0:
		return Match{Name: name, Status: StatusUnknown, New: true}, nil
	case 1:
		return Match{Name: name, Status: StatusExact, Ingredient: found[0]}, nil
	default:
		options := make([]Option, 0, len(found))
		for _, one := range found {
			options = append(options, Option{ID: one.ID, Name: one.Name})
		}

		return Match{Name: name, Status: StatusAmbiguous, Options: options}, nil
	}
}

// Resolve answers what the catalog knows about the name of a line.
//
// The rules of the domain:
//
//   - a line that carries the identity of an ingredient of the catalog
//     keeps it;
//   - a name the catalog knows exactly is the ingredient of the line;
//   - a name the catalog does not know is a new ingredient: the catalog
//     grows with what the owner buys, and the name is recorded as she said
//     it;
//   - a name the catalog cannot answer is not resolved: the line keeps the
//     name as she said it, and the catalog does not grow either.
//
// The error is ErrInvalidName when the name is empty.
func (c Catalog) Resolve(id ID, spoken string) (Resolution, error) {
	if id != 0 {
		for _, one := range c.items {
			if one.ID == id {
				return Resolution{Ingredient: one}, nil
			}
		}
	}

	matched, err := c.Match(spoken)
	if err != nil {
		return Resolution{}, err
	}

	switch matched.Status {
	case StatusExact:
		return Resolution{Ingredient: matched.Ingredient}, nil
	case StatusUnknown:
		return Resolution{New: true}, nil
	default:
		return Resolution{}, nil
	}
}

// Holds answers whether the catalog holds the ingredient of that identity.
//
// The rule of the domain: an ingredient of another business is not in this
// catalog, and an ingredient that was never bought is not in it either. A
// recipe names the ingredients of the catalog, and this is the question the
// catalog answers about them.
func (c Catalog) Holds(id ID) bool {
	for _, one := range c.items {
		if one.ID == id {
			return true
		}
	}

	return false
}

// With answers the catalog with one more ingredient.
//
// The rule of the domain: two ingredients of the same business never carry
// the same name. The error is ErrDuplicateName.
func (c Catalog) With(item Ingredient) (Catalog, error) {
	grown := make([]Ingredient, 0, len(c.items)+1)
	grown = append(grown, c.items...)
	grown = append(grown, item)

	return NewCatalog(c.own, grown)
}
