package purchase

import (
	"fmt"
	"math"
	"strings"

	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// Measure is the quantity of a line with its unit.
//
// The quantity and its unit travel together: a quantity alone is a number
// nobody can buy, and a unit alone is a word nobody can count.
//
// The unit of a measure is the unit the book knows: the measure is what the
// book stores, and the book stores the gram, the kilogram, the package and
// the piece.
type Measure struct {
	// Quantity is how much of the thing the owner named.
	Quantity float64

	// Unit is the unit of that quantity: the gram, the kilogram, the
	// package, the piece.
	Unit string
}

// NewMeasure builds the measure of a line.
//
// The quantity is positive, and the unit is not empty: the two travel
// together.
//
// The unit is the unit the book knows: the way the owner said it ("kilos",
// "gramos") is read and normalised, and the measure carries the unit of
// the book ("kg", "g"). A unit the book does not know is refused with the
// error of the domain (ingredient.ErrUnknownUnit): it is never guessed and
// never dropped.
func NewMeasure(quantity float64, unit string) (Measure, error) {
	if quantity <= 0 {
		return Measure{}, fmt.Errorf("%w: %v", ErrInvalidQuantity, quantity)
	}

	trimmed := strings.TrimSpace(unit)
	if trimmed == "" {
		return Measure{}, fmt.Errorf("%w: the measure of a line carries its unit", ErrMissingUnit)
	}

	known, err := ingredient.ParseUnit(trimmed)
	if err != nil {
		return Measure{}, fmt.Errorf("%w: the measure of a line carries the unit %q", err, trimmed)
	}

	return Measure{Quantity: quantity, Unit: known.String()}, nil
}

// Price is the money of a line: the money that was paid for it, or the
// price of one unit of it.
//
// Never two truths that disagree: when the owner named both, they must
// agree, and the domain refuses the pair that does not.
//
// The money is never inferred. A price the agent filled is refused, and the
// draft asks the owner for it.
type Price struct {
	// Total is the money that was paid for the whole line, in pesos.
	Total ledger.Amount

	// Unit is the price of one unit of the line, in pesos.
	Unit ledger.Amount

	// Inferred says that the price was filled by the agent and not said by
	// the owner.
	Inferred bool
}

// Present reports whether the line carries money.
//
// A line the owner did not price carries none: the money is what the draft
// still needs.
func (p Price) Present() bool {
	return p.Total > 0 || p.Unit > 0
}

// Amount answers the money of the line in the money of the book: whole
// pesos.
//
// The money that was paid is the money of the line. When the owner named
// only the price of the unit, the money of the line is that price times the
// quantity, rounded to the nearest peso: the book counts whole pesos, and
// the sentence of the owner keeps the exact wording.
func (p Price) Amount(quantity float64) ledger.Amount {
	if p.Total > 0 {
		return p.Total
	}

	if p.Unit <= 0 {
		return 0
	}

	return ledger.Amount(math.Round(float64(p.Unit) * quantity))
}

// AgreesWith reports whether the money that was paid and the price of the
// unit are the same truth.
//
// The rule of the domain: when the owner named both, they must agree. The
// money that was paid is the price of the unit times the quantity, rounded
// to the nearest peso.
func (p Price) AgreesWith(quantity float64) bool {
	if p.Total <= 0 || p.Unit <= 0 {
		return true
	}

	return p.Total == ledger.Amount(math.Round(float64(p.Unit)*quantity))
}

// Line is a line of a purchase: the name as the owner said it, the
// measure and the money.
//
// A line of a draft is what the agent read, and a line of a purchase is
// what was bought: the money of a purchase is whole pesos, and it is the
// money that was paid for the line.
type Line struct {
	// Name is the name as the owner said it.
	Name string

	// Measure is the quantity with its unit.
	Measure Measure

	// Price is the money of the line.
	Price Price

	// IngredientID is the identity of the ingredient of the catalog the
	// name belongs to.
	//
	// It is zero when the catalog does not know the name: the line keeps
	// the name as the owner said it, and the catalog grows with it when
	// the owner buys it.
	IngredientID ingredient.ID
}

// WithIngredient answers the line with the identity of the ingredient the
// name belongs to.
//
// The identity is zero when the catalog does not know the name: the line
// keeps the name as the owner said it, and nothing else of the line moves.
func (l Line) WithIngredient(id ingredient.ID) Line {
	l.IngredientID = id

	return l
}

// NewLine builds a line of a draft.
//
// The name is the name the owner said, and it may not be empty.
//
// The measure carries the quantity and its unit: the two travel together,
// and a line without a unit is refused.
//
// The unit of the measure is the unit the book knows: the line stores the
// unit of the book, and the way the owner said it is kept in the name of
// the ingredient and in the sentence of her utterance. A unit the book
// does not know is refused with the error of the domain.
//
// The money may be absent: a line the owner did not price is a line the
// draft still needs, and the draft says so. The money is never inferred: a
// price the agent filled is refused.
//
// When the owner named both the money that was paid and the price of the
// unit, the two must agree: the domain refuses the pair that does not.
func NewLine(name string, measure Measure, price Price) (Line, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Line{}, fmt.Errorf("%w: the name of a line is not empty", ErrInvalidName)
	}

	canonical, err := NewMeasure(measure.Quantity, measure.Unit)
	if err != nil {
		return Line{}, err
	}

	if price.Inferred {
		return Line{}, fmt.Errorf("%w: the money of the line %q is never inferred", ErrInferredPrice, trimmed)
	}

	if price.Total < 0 || price.Unit < 0 {
		return Line{}, fmt.Errorf("%w: the money of the line %q is not negative", ErrInvalidPrice, trimmed)
	}

	if !price.AgreesWith(canonical.Quantity) {
		return Line{}, fmt.Errorf("%w: the line %q carries a total of %d and a unit of %d",
			ErrPriceMismatch, trimmed, price.Total, price.Unit)
	}

	return Line{Name: trimmed, Measure: canonical, Price: price}, nil
}
