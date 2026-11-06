package purchase_test

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// TestThePriceOfTheOwnerIsAPrice proves that the price of the owner is a
// price: the money of what the business sells is never a number PACA
// invents, and a price of nothing is not a price.
func TestThePriceOfTheOwnerIsAPrice(t *testing.T) {
	t.Parallel()

	_, err := purchase.NewOwnerPrice(ledger.Amount(0), purchase.PricePerUnit)
	if !errors.Is(err, purchase.ErrInvalidOwnerPrice) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrInvalidOwnerPrice)
	}
}

// TestThePriceOfTheOwnerCarriesTheWaySheCharges proves that the price of
// the owner carries the way she charges: the price of the unit and the
// price of the gram are two different truths, and the book never guesses
// which one is hers.
func TestThePriceOfTheOwnerCarriesTheWaySheCharges(t *testing.T) {
	t.Parallel()

	_, err := purchase.NewOwnerPrice(ledger.Amount(8000), purchase.PriceBasis("durzimo"))
	if !errors.Is(err, purchase.ErrUnknownPriceBasis) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrUnknownPriceBasis)
	}
}

// TestThePriceOfTheOwnerIsBuiltWithItsWay proves that the price of the
// owner is built with the money she declared and the way she charges.
func TestThePriceOfTheOwnerIsBuiltWithItsWay(t *testing.T) {
	t.Parallel()

	price, err := purchase.NewOwnerPrice(ledger.Amount(8000), purchase.PricePerUnit)
	if err != nil {
		t.Fatalf("the price of the owner was refused: %v", err)
	}

	if price.Money != ledger.Amount(8000) {
		t.Errorf("the money of the price is %d, want 8000", price.Money)
	}
	if price.Basis != purchase.PricePerUnit {
		t.Errorf("the way she charges is %q, want %q", price.Basis, purchase.PricePerUnit)
	}
}

// TestTheWayTheOwnerChargesIsReadTheWaySheSaysIt proves that the way the
// owner charges is read the way a person reads it: trimmed, and without
// case.
func TestTheWayTheOwnerChargesIsReadTheWaySheSaysIt(t *testing.T) {
	t.Parallel()

	wants := []struct {
		spoken string
		basis  purchase.PriceBasis
	}{
		{spoken: "unit", basis: purchase.PricePerUnit},
		{spoken: "unidad", basis: purchase.PricePerUnit},
		{spoken: " UNIDAD ", basis: purchase.PricePerUnit},
		{spoken: "pieza", basis: purchase.PricePerUnit},
		{spoken: "gram", basis: purchase.PricePerGram},
		{spoken: "gramos", basis: purchase.PricePerGram},
		{spoken: "kilo", basis: purchase.PricePerKilogram},
		{spoken: "kilogramos", basis: purchase.PricePerKilogram},
	}

	for _, one := range wants {
		got, err := purchase.ParsePriceBasis(one.spoken)
		if err != nil {
			t.Fatalf("the way she charges %q was refused: %v", one.spoken, err)
		}
		if got != one.basis {
			t.Errorf("the way she charges %q is %q, want %q", one.spoken, got, one.basis)
		}
	}
}

// TestTheWayTheOwnerChargesThatTheBookDoesNotKnowIsRefused proves that a
// way of charging the book does not know is refused: the basis of the
// price of the owner is a rule of the book, and it is never guessed.
func TestTheWayTheOwnerChargesThatTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	_, err := purchase.ParsePriceBasis("durzimo")
	if !errors.Is(err, purchase.ErrUnknownPriceBasis) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrUnknownPriceBasis)
	}
}

// TestTheWaysTheOwnerMayChargeAreTheThreeTheBookKnows proves that the
// book knows three ways of charging: the unit, the gram and the kilogram,
// and the description of the tool reads them from here.
func TestTheWaysTheOwnerMayChargeAreTheThreeTheBookKnows(t *testing.T) {
	t.Parallel()

	bases := purchase.PriceBases()

	if len(bases) != 3 {
		t.Fatalf("the book knows %d ways of charging, want 3", len(bases))
	}

	wants := []purchase.PriceBasis{
		purchase.PricePerUnit,
		purchase.PricePerGram,
		purchase.PricePerKilogram,
	}

	for index, want := range wants {
		if bases[index] != want {
			t.Errorf("the way of charging %d is %q, want %q", index, bases[index], want)
		}
	}
}
