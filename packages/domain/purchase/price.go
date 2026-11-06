package purchase

import (
	"fmt"
	"strings"

	"github.com/witsaba/paca/packages/domain/ledger"
)

// PriceBasis is the way the owner charges for what the business sells.
//
// The bakery sells in two ways: by the unit she makes — and then the grams
// are not her worry — or by the weight of it. The book knows the three
// ways the owner may name it: the unit, the gram and the kilogram.
//
// The basis belongs to the domain: the price of the owner travels with it,
// and the price of the unit and the price of the gram are two different
// truths. Only one of them is hers.
type PriceBasis string

const (
	// PricePerUnit is the price of one unit of what the recipe makes.
	PricePerUnit PriceBasis = "unit"

	// PricePerGram is the price of one gram of what the recipe makes.
	PricePerGram PriceBasis = "gram"

	// PricePerKilogram is the price of one kilogram of what the recipe
	// makes.
	PricePerKilogram PriceBasis = "kilogram"
)

// theSpokenPriceBases are the names the owner and the agent say for the
// ways of charging.
//
// The values of the bases are the ones the storage and the protocol carry:
// the names a person says are read the way a person reads them. The list is
// the vocabulary of the domain, and it holds the plain ways a person says
// the three ways of charging — and nothing else: a way the book does not
// know stays unknown.
var theSpokenPriceBases = map[string]PriceBasis{
	"unit":       PricePerUnit,
	"units":      PricePerUnit,
	"unidad":     PricePerUnit,
	"unidades":   PricePerUnit,
	"piece":      PricePerUnit,
	"pieces":     PricePerUnit,
	"pieza":      PricePerUnit,
	"piezas":     PricePerUnit,
	"gram":       PricePerGram,
	"grams":      PricePerGram,
	"gramo":      PricePerGram,
	"gramos":     PricePerGram,
	"g":          PricePerGram,
	"kg":         PricePerKilogram,
	"kilo":       PricePerKilogram,
	"kilos":      PricePerKilogram,
	"kilogram":   PricePerKilogram,
	"kilograms":  PricePerKilogram,
	"kilogramo":  PricePerKilogram,
	"kilogramos": PricePerKilogram,
}

// PriceBases returns the ways the owner may charge, in the order she reads
// them.
//
// The vocabulary belongs to the domain: the description of the tools reads
// it, and the agent sends the way she charges the way the book knows it.
func PriceBases() []PriceBasis {
	return []PriceBasis{PricePerUnit, PricePerGram, PricePerKilogram}
}

// String returns the name of the way of charging.
//
// It is the name the storage and the protocol carry, with no translation in
// between.
func (b PriceBasis) String() string {
	return string(b)
}

// ParsePriceBasis answers the way the owner charges.
//
// The name is read the way a person reads it: trimmed, and without case.
// The answer is the way the book knows, never the way it was said: the
// storage and the protocol carry the three names of the ways of charging.
//
// The error is ErrUnknownPriceBasis when the name is not one of them: the
// way the owner sells is her decision, and a way the book does not know is
// never guessed.
func ParsePriceBasis(spoken string) (PriceBasis, error) {
	key := strings.ToLower(strings.TrimSpace(spoken))
	if key == "" {
		return "", fmt.Errorf("%w: the way the owner charges is not empty", ErrUnknownPriceBasis)
	}

	basis, known := theSpokenPriceBases[key]
	if !known {
		return "", fmt.Errorf("%w: %q", ErrUnknownPriceBasis, spoken)
	}

	return basis, nil
}

// OwnerPrice is the price the owner charges for what the business sells.
//
// The price is hers: she declares it, and the book never moves it. It
// travels with the way she charges, because the price of the unit and the
// price of the gram are two different truths and only one of them is hers.
//
// The money of the book counts whole pesos: the price of the owner is a
// price of the book, and it is counted like every other one.
type OwnerPrice struct {
	// Money is the money of the price, in whole pesos: what she charges.
	Money ledger.Amount

	// Basis is the way she charges: per unit, per gram or per kilogram.
	Basis PriceBasis
}

// NewOwnerPrice builds the price of the owner.
//
// The rules of the domain hold here:
//
//   - the price is a price: it is positive, and the money of what the
//     business sells is never a number PACA invents;
//   - the price carries the way she charges: a price without its basis is a
//     number nobody can compare with anything, and the book never guesses
//     which of the two ways of selling she uses.
//
// The error is ErrInvalidOwnerPrice when the money is not positive, and
// ErrUnknownPriceBasis when the way she charges is not one of the three
// the book knows.
func NewOwnerPrice(money ledger.Amount, basis PriceBasis) (OwnerPrice, error) {
	if money <= 0 {
		return OwnerPrice{}, fmt.Errorf("%w: the price of the owner is %d", ErrInvalidOwnerPrice, money)
	}

	switch basis {
	case PricePerUnit, PricePerGram, PricePerKilogram:
	default:
		return OwnerPrice{}, fmt.Errorf("%w: %q", ErrUnknownPriceBasis, basis)
	}

	return OwnerPrice{Money: money, Basis: basis}, nil
}
