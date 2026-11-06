package purchase

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// theReferenceOfTheTests is the definition of a reference the tests of the
// domain use.
func theReferenceOfTheTests() ReferenceDefinition {
	return ReferenceDefinition{
		IngredientID:  ingredient.ID(7),
		Price:         ledger.Amount(480),
		Unit:          "kg",
		Market:        "Plaza de Mercado",
		Source:        "Boletín de precios al por mayor, DANE",
		SeenOn:        ledger.NewDate(theDay(2026, time.March, 1).Time()),
		Utterance:     "La harina está a 480 el kilo en la plaza",
		ConfirmedWith: "sí",
	}
}

// TestTheReferenceCarriesTheIngredientThePriceTheUnitTheMarketTheSourceAndTheDay
// proves that the reference holds everything a later analysis reads: the
// ingredient it is about, the price that was seen, the unit of that price,
// the market where it was seen, the place it was read and the day it was
// read.
func TestTheReferenceCarriesTheIngredientThePriceTheUnitTheMarketTheSourceAndTheDay(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()

	reference, err := NewReference(business.ID(77), definition)
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	if reference.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", reference.BusinessID)
	}
	if reference.IngredientID != ingredient.ID(7) {
		t.Errorf("the ingredient is %d, want 7", reference.IngredientID)
	}
	if reference.Price != ledger.Amount(480) {
		t.Errorf("the price is %d, want 480", reference.Price)
	}
	if reference.Unit != "kg" {
		t.Errorf("the unit is %q, want %q", reference.Unit, "kg")
	}
	if reference.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", reference.Market, "Plaza de Mercado")
	}
	if reference.Source != "Boletín de precios al por mayor, DANE" {
		t.Errorf("the source is %q, want the place the price was read", reference.Source)
	}
	if reference.SeenOn != definition.SeenOn {
		t.Errorf("the day it was seen is %v, want %v", reference.SeenOn, definition.SeenOn)
	}
	if reference.Utterance != definition.Utterance {
		t.Errorf("the sentence is %q, want %q", reference.Utterance, definition.Utterance)
	}
	if reference.ConfirmedWith != "sí" {
		t.Errorf("the word of the owner is %q, want %q", reference.ConfirmedWith, "sí")
	}
}

// TestTheReferenceIsLabelledAsAReference proves the rule of the domain: a
// reference is a reference, and the name the person reads in her book says
// so. It is never the price of the ingredient: the price the owner pays
// comes from her own purchases.
func TestTheReferenceIsLabelledAsAReference(t *testing.T) {
	t.Parallel()

	reference, err := NewReference(business.ID(77), theReferenceOfTheTests())
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	if reference.Label() != ReferenceKind {
		t.Errorf("the label is %q, want %q", reference.Label(), ReferenceKind)
	}
	if ReferenceKind != "reference" {
		t.Errorf("the kind of the reference is %q, want %q", ReferenceKind, "reference")
	}
}

// TestTheReferenceWithoutTheIngredientOfTheCatalogIsRefused proves that a
// reference names an ingredient of the catalog: a reference of nothing is a
// price of nobody.
func TestTheReferenceWithoutTheIngredientOfTheCatalogIsRefused(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()
	definition.IngredientID = 0

	_, err := NewReference(business.ID(77), definition)
	if !errors.Is(err, ErrMissingIngredient) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingIngredient)
	}
}

// TestTheReferenceWithoutAPriceIsRefused proves that the price of a
// reference is a price: a reference without one carries nothing.
func TestTheReferenceWithoutAPriceIsRefused(t *testing.T) {
	t.Parallel()

	for _, price := range []ledger.Amount{0, -480} {
		definition := theReferenceOfTheTests()
		definition.Price = price

		_, err := NewReference(business.ID(77), definition)
		if !errors.Is(err, ErrInvalidReferencePrice) {
			t.Errorf("the error is %v, want %v", err, ErrInvalidReferencePrice)
		}
	}
}

// TestTheReferenceWithoutItsUnitIsRefused proves the rule of the domain: a
// price without its unit is a number nobody can compare with anything.
func TestTheReferenceWithoutItsUnitIsRefused(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()
	definition.Unit = "   "

	_, err := NewReference(business.ID(77), definition)
	if !errors.Is(err, ErrMissingReferenceUnit) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingReferenceUnit)
	}
}

// TestTheReferenceCarriesTheUnitTheBookKnows proves the rule of the domain:
// the unit of a reference is the unit the book knows. The research may name
// the unit the way a person says it, and the book keeps the unit it knows.
func TestTheReferenceCarriesTheUnitTheBookKnows(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()
	definition.Unit = "kilos"

	reference, err := NewReference(business.ID(77), definition)
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	if reference.Unit != "kg" {
		t.Errorf("the unit is %q, want %q", reference.Unit, "kg")
	}
	if reference.Utterance != definition.Utterance {
		t.Errorf("the sentence is %q, want %q", reference.Utterance, definition.Utterance)
	}
}

// TestTheReferenceWithAUnitTheBookDoesNotKnowIsRefused proves that a unit
// the book does not know is refused with the error of the domain, and that
// no reference is written: the unit of the book is a rule.
func TestTheReferenceWithAUnitTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()
	definition.Unit = "libra"

	_, err := NewReference(business.ID(77), definition)
	if !errors.Is(err, ingredient.ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrUnknownUnit)
	}
}

// TestTheReferenceWithoutItsSourceIsRefused proves the rule of the domain: a
// price without a source is a price nobody can check.
func TestTheReferenceWithoutItsSourceIsRefused(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()
	definition.Source = "   "

	_, err := NewReference(business.ID(77), definition)
	if !errors.Is(err, ErrMissingSource) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingSource)
	}
}

// TestTheReferenceWithoutItsDayIsRefused proves the rule of the domain: a
// reference without the day it was seen is a price that never was.
func TestTheReferenceWithoutItsDayIsRefused(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()
	definition.SeenOn = ledger.Date{}

	_, err := NewReference(business.ID(77), definition)
	if !errors.Is(err, ErrMissingDay) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingDay)
	}
}

// TestTheReferenceWithoutTheWordOfTheOwnerIsRefused proves the rule of the
// domain: the reference is written with the word of the owner, the way the
// purchase is written.
func TestTheReferenceWithoutTheWordOfTheOwnerIsRefused(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()
	definition.ConfirmedWith = "  "

	_, err := NewReference(business.ID(77), definition)
	if !errors.Is(err, ErrMissingConfirmation) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingConfirmation)
	}
}

// TestTheMarketOfAReferenceIsOptional proves that the market is there when
// the research named one, and absent when it named none.
func TestTheMarketOfAReferenceIsOptional(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()
	definition.Market = ""

	reference, err := NewReference(business.ID(77), definition)
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	if reference.Market != "" {
		t.Errorf("the market is %q, want it absent", reference.Market)
	}
}

// TestTheMoneyOfTheBookCountsWholePesos proves that the money of the book
// holds the reference too: the price of a reference is whole pesos, like
// every other price of the book.
func TestTheMoneyOfAReferenceCountsWholePesos(t *testing.T) {
	t.Parallel()

	definition := theReferenceOfTheTests()
	definition.Price = ledger.Amount(480)

	reference, err := NewReference(business.ID(77), definition)
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	if reference.Price != ledger.Amount(480) {
		t.Errorf("the price is %d, want 480", reference.Price)
	}
}

// TestTheAgeOfAReferenceIsCountedInDaysOfTheBook proves that the age of a
// reference is a rule of the domain: it is counted in days of the book, and
// against the day of the reading.
func TestTheAgeOfAReferenceIsCountedInDaysOfTheBook(t *testing.T) {
	t.Parallel()

	reference, err := NewReference(business.ID(77), theReferenceOfTheTests())
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	age := reference.AgeOn(ledger.NewDate(theDay(2026, time.March, 11).Time()))

	if age != 10 {
		t.Errorf("the age of the reference is %d days, want 10", age)
	}
}

// TestTheAgeOfAReferenceOfTheDayOfTheReadingIsZero proves that a
// reference seen the same day is not old: the age of a reference is never
// negative.
func TestTheAgeOfAReferenceOfTheDayOfTheReadingIsZero(t *testing.T) {
	t.Parallel()

	reference, err := NewReference(business.ID(77), theReferenceOfTheTests())
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	age := reference.AgeOn(ledger.NewDate(theDay(2026, time.March, 1).Time()))

	if age != 0 {
		t.Errorf("the age of the reference is %d days, want 0", age)
	}
}

// TestAReferenceOlderThanTheOwnerExpectsIsOld proves that a reference
// older than the owner expects is old, and that a reference inside the
// expectation is not.
func TestAReferenceOlderThanTheOwnerExpectsIsOld(t *testing.T) {
	t.Parallel()

	reference, err := NewReference(business.ID(77), theReferenceOfTheTests())
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	day := ledger.NewDate(theDay(2026, time.March, 11).Time())

	if !reference.IsOlderThan(day, 7) {
		t.Error("the reference of ten days is not old, want it old: the owner expects seven days")
	}
	if reference.IsOlderThan(day, 30) {
		t.Error("the reference of ten days is old, want it fresh: the owner expects thirty days")
	}
	if reference.IsOlderThan(day, 0) {
		t.Error("the reference is old, want it fresh: the owner expects no age at all")
	}
}
