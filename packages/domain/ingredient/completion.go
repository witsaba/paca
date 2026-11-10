package ingredient

import (
	"fmt"
	"strings"
)

// Certification is a claim with the document that proves it.
//
// "Organic" is a claim, and a claim carries its evidence: the certification
// of an ingredient is the name of the claim and the document that proves
// it. The evidence is never empty, because a claim nobody can check is a
// wish.
type Certification struct {
	// Name is the claim itself: "orgánico".
	Name string

	// Evidence is the document that proves the claim.
	Evidence string
}

// NewCertification builds the certification of an ingredient.
//
// The name of the claim is required, and the evidence is required too: the
// errors are ErrInvalidCertification and ErrMissingEvidence.
func NewCertification(name, evidence string) (Certification, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Certification{}, fmt.Errorf("%w: the certification carries no name", ErrInvalidCertification)
	}

	trimmedEvidence := strings.TrimSpace(evidence)
	if trimmedEvidence == "" {
		return Certification{}, fmt.Errorf("%w: the certification %q", ErrMissingEvidence, trimmedName)
	}

	return Certification{Name: trimmedName, Evidence: trimmedEvidence}, nil
}

// Declared reports whether the certification carries a claim.
//
// A certification that was never declared is not a certification without
// evidence: it is no certification at all.
func (c Certification) Declared() bool {
	return c.Name != ""
}

// Conversion is the conversion of an ingredient: the grams of the raw that
// one gram of the prepared ingredient takes.
//
// The conversion is declared, never assumed: the owner decides the factor,
// and the book carries it. Nothing in the tree invents one.
type Conversion struct {
	// RawGrams is the grams of the raw that one gram of the prepared
	// ingredient takes.
	RawGrams float64
}

// RawConversion is the conversion of an ingredient whose owner declared
// none.
//
// The owner decided that, for now, the grams of a recipe are the raw ones:
// the conversion is one to one. It is a declared constant of the book, and
// the mechanism carries any other factor the owner declares later.
const RawConversion = 1.0

// NewConversion builds the conversion of an ingredient.
//
// The factor is positive: the conversion of an ingredient is a number of
// grams, and it is never zero nor negative. The error is
// ErrInvalidConversion.
func NewConversion(rawGrams float64) (Conversion, error) {
	if rawGrams <= 0 {
		return Conversion{}, fmt.Errorf("%w: %v", ErrInvalidConversion, rawGrams)
	}

	return Conversion{RawGrams: rawGrams}, nil
}

// Declared reports whether the conversion carries a factor.
//
// The factor of a conversion nobody declared is zero: the ingredient keeps
// the conversion it had. A factor that is not zero is a factor the owner
// declared, and the domain refuses it when it is not positive.
func (c Conversion) Declared() bool {
	return c.RawGrams != 0
}

// Declaration is what the owner declares about an ingredient.
//
// The zero value of a field means that the owner declared nothing about
// it, and the ingredient keeps what it had. A declaration never empties a
// field of the ingredient: the owner declares what she knows, and the rest
// stays as it is.
type Declaration struct {
	// Unit is the unit the ingredient is bought in.
	Unit Unit

	// PackageGrams is the weight of a package of the ingredient, in grams.
	// It is zero when the owner declared none.
	PackageGrams float64

	// Allergens are the allergens the ingredient carries. It is nil when
	// the owner declared none.
	Allergens []string

	// Certification is the claim the ingredient carries and the evidence
	// of it. It is empty when the owner declared none.
	Certification Certification

	// Aliases are the names the owner uses for the ingredient. It is nil
	// when she declared none.
	Aliases []string

	// Conversion is the conversion of the raw into the prepared. It is
	// empty when the owner declared none.
	Conversion Conversion

	// ConversionDeclared says that the owner declared the conversion,
	// whatever the factor she declared is.
	//
	// The factor of a conversion nobody declared is zero, and the
	// ingredient keeps the conversion it had. The factor of a conversion
	// the owner did declare may be zero too, and then the declaration is
	// refused: the conversion is a number of grams, and the table is not
	// the only one that says so.
	ConversionDeclared bool
}

// Complete applies the declaration of the owner to the ingredient of the
// catalog.
//
// The rules of the domain:
//
//   - the ingredient must be there: it carries the identity the storage
//     gave it (ErrNotFound);
//   - the unit is one of the units the book knows (ErrUnknownUnit);
//   - the weight of a package is positive (ErrInvalidPackageWeight);
//   - a certification carries the document that proves it
//     (ErrMissingEvidence);
//   - the conversion is positive (ErrInvalidConversion).
//
// A declaration that carries a conversion carries a factor, and a factor
// that is not positive is refused: the ingredient keeps the conversion it
// had, and the table is not the only one that refuses the row.
//
// Nothing of the ingredient is lost: the declaration fills what the owner
// declared and keeps the rest. The name of the ingredient is never an
// alias of it, and the aliases the owner declares are added to the ones it
// already had.
func Complete(current Ingredient, declaration Declaration) (Ingredient, error) {
	if current.ID == 0 {
		return Ingredient{}, fmt.Errorf("%w: the ingredient has no identity", ErrNotFound)
	}

	completed := current

	if declaration.Unit != "" {
		unit, err := ParseUnit(string(declaration.Unit))
		if err != nil {
			return Ingredient{}, err
		}

		completed.Unit = unit
	}

	if declaration.PackageGrams < 0 {
		return Ingredient{}, fmt.Errorf("%w: %v", ErrInvalidPackageWeight, declaration.PackageGrams)
	}

	if declaration.PackageGrams > 0 {
		completed.PackageGrams = declaration.PackageGrams
	}

	if declaration.Allergens != nil {
		completed.Allergens = allergensOf(declaration.Allergens)
	}

	if declaration.Certification.Declared() {
		certification, err := NewCertification(declaration.Certification.Name, declaration.Certification.Evidence)
		if err != nil {
			return Ingredient{}, err
		}

		completed.Certification = certification
	}

	if declaration.Aliases != nil {
		completed.Aliases = aliasesOf(completed.Name, append(append([]string{}, completed.Aliases...), declaration.Aliases...))
	}

	if declaration.ConversionDeclared || declaration.Conversion.Declared() {
		conversion, err := NewConversion(declaration.Conversion.RawGrams)
		if err != nil {
			return Ingredient{}, err
		}

		completed.Conversion = conversion
	}

	return completed, nil
}

// Missing answers what the ingredient still needs from its owner.
//
// The rule of the domain: the unit the ingredient is bought in is what the
// owner declares, and the weight of a package is what she declares too.
// The rest of the ingredient is whole: the conversion is the one to one the
// owner decided, and a certification always carries its evidence.
func Missing(item Ingredient) []string {
	missing := []string{}

	if item.Unit == "" {
		missing = append(missing, "the unit the ingredient is bought in")
	}

	if item.Unit == UnitPackage && item.PackageGrams <= 0 {
		missing = append(missing, "the weight of a package of the ingredient")
	}

	return missing
}

// Grams answers how many grams a quantity of the ingredient weighs.
//
// The conversion of a unit into the gram is declared, never assumed: the
// gram is one, the kilogram is a thousand, the package is the weight the
// owner declares, and the piece is one unit of the ingredient, whose
// weight the book never assumes.
//
// The errors are ErrInvalidQuantity when the quantity is not positive,
// ErrMissingPackageWeight when the package has no declared weight, and
// ErrUnknownConversion when the unit is never converted into grams.
func (i Ingredient) Grams(quantity float64) (float64, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("%w: %v", ErrInvalidQuantity, quantity)
	}

	switch i.Unit {
	case UnitGram:
		return quantity, nil
	case UnitKilogram:
		return quantity * GramsPerKilogram, nil
	case UnitPackage:
		if i.PackageGrams <= 0 {
			return 0, fmt.Errorf("%w: the ingredient %q", ErrMissingPackageWeight, i.Name)
		}

		return quantity * i.PackageGrams, nil
	case UnitPiece:
		return 0, fmt.Errorf("%w: the piece is one unit of the ingredient %q",
			ErrUnknownConversion, i.Name)
	default:
		return 0, fmt.Errorf("%w: the ingredient %q", ErrUnknownUnit, i.Name)
	}
}

// allergensOf answers the allergens the owner declared.
//
// The list loses the spaces around every name, an empty one is not an
// allergen, and the allergen that is said twice is one allergen: the name
// is compared the way a person reads it, trimmed and without case.
func allergensOf(declared []string) []string {
	kept := make([]string, 0, len(declared))
	seen := map[string]bool{}

	for _, one := range declared {
		trimmed := strings.TrimSpace(one)
		if trimmed == "" {
			continue
		}

		key := keyOf(trimmed)
		if seen[key] {
			continue
		}

		seen[key] = true
		kept = append(kept, trimmed)
	}

	return kept
}
