package ingredient

import (
	"fmt"
	"sort"
	"strings"
)

// Unit is the unit an ingredient is bought in.
//
// The units are the four the book of the bakery knows: the gram, the
// kilogram, the package and the piece. The conversion of a unit into the
// gram is declared, never assumed: the gram is one, the kilogram is a
// thousand, the package is the weight the owner declares, and the piece is
// one unit of the ingredient, whose weight the book never assumes.
type Unit string

const (
	// UnitGram is the gram: the weight a person reads on the scale.
	UnitGram Unit = "g"

	// UnitKilogram is the kilogram: a thousand grams, and nothing else.
	UnitKilogram Unit = "kg"

	// UnitPackage is the package: the weight of it is what the owner
	// declares, and it is never assumed.
	UnitPackage Unit = "paquete"

	// UnitPiece is the piece: one unit of the ingredient.
	UnitPiece Unit = "unidad"
)

// GramsPerKilogram is the conversion of the kilogram into the gram.
//
// The number is a rule of the book, and it is written here: a thousand
// grams are a thousand grams, and nobody has to say it twice.
const GramsPerKilogram = 1000.0

// theSpokenUnits are the names the owner and the agent say for the units.
//
// The values of the units are the ones the storage and the protocol carry:
// the names a person says are read the way a person reads them. The list
// is the vocabulary of the domain, and it holds the plain ways a person
// says the four units of the book - and nothing else: a unit the book does
// not know stays unknown.
var theSpokenUnits = map[string]Unit{
	"g":          UnitGram,
	"gr":         UnitGram,
	"gram":       UnitGram,
	"gramo":      UnitGram,
	"gramos":     UnitGram,
	"grs":        UnitGram,
	"kg":         UnitKilogram,
	"kgs":        UnitKilogram,
	"kilo":       UnitKilogram,
	"kilos":      UnitKilogram,
	"kilogramo":  UnitKilogram,
	"kilogramos": UnitKilogram,
	"paquete":    UnitPackage,
	"paquetes":   UnitPackage,
	"unidad":     UnitPiece,
	"unidades":   UnitPiece,
	"pieza":      UnitPiece,
	"piezas":     UnitPiece,
}

// Units returns the units of the book, in the order the owner reads them.
//
// The vocabulary belongs to the domain: the description of the tools reads
// it, and the check of the table holds the same rule as a defense in
// depth.
func Units() []Unit {
	return []Unit{UnitGram, UnitKilogram, UnitPackage, UnitPiece}
}

// SpokenUnits returns the ways the owner and the agent say the units of the
// book, in the order a person reads them.
//
// The list is the declared vocabulary of the domain: the description of the
// tools reads it, so the agent sends the unit the way the owner said it.
// The list holds no unit the book does not know.
func SpokenUnits() []string {
	spoken := make([]string, 0, len(theSpokenUnits))
	for name := range theSpokenUnits {
		spoken = append(spoken, name)
	}

	sort.Strings(spoken)

	return spoken
}

// String returns the name of the unit.
//
// It is the name the storage and the protocol carry, with no translation in
// between.
func (u Unit) String() string {
	return string(u)
}

// ParseUnit answers the unit the owner named.
//
// The name is read the way a person reads it: trimmed, and without case.
// The answer is the unit the book knows, never the way it was said: the
// storage and the protocol carry the four names of the units.
//
// The error is ErrUnknownUnit when the name is not one of them: the unit is
// a rule of the book, and a unit the book does not know is never guessed.
func ParseUnit(spoken string) (Unit, error) {
	key := strings.ToLower(strings.TrimSpace(spoken))
	if key == "" {
		return "", fmt.Errorf("%w: the unit is not empty", ErrUnknownUnit)
	}

	unit, known := theSpokenUnits[key]
	if !known {
		return "", fmt.Errorf("%w: %q", ErrUnknownUnit, spoken)
	}

	return unit, nil
}

// namesOfTheUnits renders the units of the book in a sentence the caller
// reads.
//
// The vocabulary belongs to the domain: the message of a refusal names the
// units the book knows, and it never writes them a second time.
func namesOfTheUnits() string {
	names := make([]string, 0, len(Units()))

	for _, one := range Units() {
		names = append(names, one.String())
	}

	return strings.Join(names, ", ")
}
