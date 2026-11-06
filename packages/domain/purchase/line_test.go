package purchase

import (
	"errors"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// TestTheLineCarriesTheNameTheOwnerSaidIt proves that the name of a line
// is the name the owner said, without the spaces around it.
func TestTheLineCarriesTheNameTheOwnerSaidIt(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(25, "kg")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	line, err := NewLine("  Harina  ", measure, Price{Total: 12000})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	if line.Name != "Harina" {
		t.Errorf("the name is %q, want %q", line.Name, "Harina")
	}
	if line.Measure.Quantity != 25 {
		t.Errorf("the quantity is %v, want 25", line.Measure.Quantity)
	}
	if line.Measure.Unit != "kg" {
		t.Errorf("the unit is %q, want %q", line.Measure.Unit, "kg")
	}
	if line.Price.Total != 12000 {
		t.Errorf("the money is %d, want 12000", line.Price.Total)
	}
}

// TestTheLineCarriesTheIdentityOfTheIngredientItNames proves that a line
// carries the identity of the ingredient of the catalog it names, and that
// the name is kept as the owner said it: a line the catalog does not know
// carries no identity.
func TestTheLineCarriesTheIdentityOfTheIngredientItNames(t *testing.T) {
	t.Parallel()

	line := theLine(t, "Harina", 25, "kg", 12000)
	if line.IngredientID != ingredient.ID(0) {
		t.Errorf("the identity of the ingredient is %d, want none", line.IngredientID)
	}

	named := line.WithIngredient(ingredient.ID(7))
	if named.IngredientID != ingredient.ID(7) {
		t.Errorf("the identity of the ingredient is %d, want 7", named.IngredientID)
	}
	if named.Name != "Harina" {
		t.Errorf("the name is %q, want %q", named.Name, "Harina")
	}
	if named.Measure.Quantity != 25 {
		t.Errorf("the quantity is %v, want 25", named.Measure.Quantity)
	}
	if named.Price.Total != 12000 {
		t.Errorf("the money is %d, want 12000", named.Price.Total)
	}
}

// TestTheLineWithoutANameIsRefused proves that a line without a name is
// refused: the name is what the person reads in her book.
func TestTheLineWithoutANameIsRefused(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(1, "kg")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	_, err = NewLine("   ", measure, Price{Total: 1000})
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidName)
	}
}

// TestTheMeasureWithoutAUnitIsRefused proves the rule of the domain: the
// quantity and its unit travel together, and a quantity alone is a number
// nobody can buy.
func TestTheMeasureWithoutAUnitIsRefused(t *testing.T) {
	t.Parallel()

	_, err := NewMeasure(25, "   ")
	if !errors.Is(err, ErrMissingUnit) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingUnit)
	}
}

// TestTheMeasureCarriesTheUnitTheBookKnows proves the rule of the domain:
// the unit of a measurement the book stores is the unit the book knows. The
// owner may say the unit the way she says it, and the book keeps the unit
// it knows.
func TestTheMeasureCarriesTheUnitTheBookKnows(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spoken string
		want   string
	}{
		"the kilos are the kilograms":   {spoken: "kilos", want: "kg"},
		"the kilos in singular":         {spoken: "kilo", want: "kg"},
		"the gramos are the grams":      {spoken: "gramos", want: "g"},
		"the grams":                     {spoken: "gr", want: "g"},
		"the paquetes are the packages": {spoken: "paquetes", want: "paquete"},
		"the piezas are the pieces":     {spoken: "piezas", want: "unidad"},
		"the unidades":                  {spoken: "unidades", want: "unidad"},
	}

	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			measure, err := NewMeasure(25, one.spoken)
			if err != nil {
				t.Fatalf("the measure was refused: %v", err)
			}

			if measure.Unit != one.want {
				t.Errorf("the unit is %q, want %q", measure.Unit, one.want)
			}
		})
	}
}

// TestTheMeasureWithAUnitTheBookDoesNotKnowIsRefused proves that a unit the
// book does not know is never guessed and never dropped: the measure is
// refused with the error of the domain, and the message names the units of
// the book.
func TestTheMeasureWithAUnitTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	_, err := NewMeasure(1, "libra")
	if !errors.Is(err, ingredient.ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrUnknownUnit)
	}

	for _, unit := range ingredient.Units() {
		if !strings.Contains(err.Error(), unit.String()) {
			t.Errorf("the message %q does not name the unit %q of the book", err.Error(), unit.String())
		}
	}
}

// TestTheLineCarriesTheUnitOfTheBookAndTheNameOfTheOwner proves the rule of
// the domain: the unit of a line is the unit the book knows, and the name
// of the ingredient is the name the owner said.
func TestTheLineCarriesTheUnitOfTheBookAndTheNameOfTheOwner(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(25, "kilos")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	line, err := NewLine("Harina", measure, Price{Total: 12000})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	if line.Measure.Unit != "kg" {
		t.Errorf("the unit of the line is %q, want %q", line.Measure.Unit, "kg")
	}
	if line.Name != "Harina" {
		t.Errorf("the name of the line is %q, want %q", line.Name, "Harina")
	}
}

// TestTheLineWithAUnitTheBookDoesNotKnowIsRefused proves that a line whose
// unit the book does not know is refused, and that nothing of it is kept:
// the unit of the book is a rule of the domain.
func TestTheLineWithAUnitTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	_, err := NewLine("Harina", Measure{Quantity: 1, Unit: "libra"}, Price{Total: 1000})
	if !errors.Is(err, ingredient.ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrUnknownUnit)
	}
}

// TestTheMeasureWithAQuantityThatIsNotPositiveIsRefused proves that a
// measure carries a quantity of something.
func TestTheMeasureWithAQuantityThatIsNotPositiveIsRefused(t *testing.T) {
	t.Parallel()

	for _, quantity := range []float64{0, -3} {
		if _, err := NewMeasure(quantity, "kg"); !errors.Is(err, ErrInvalidQuantity) {
			t.Errorf("the error is %v, want %v", err, ErrInvalidQuantity)
		}
	}
}

// TestThePriceThatWasInferredIsRefused proves the rule of the domain: the
// money is never inferred. The owner did not say it, so the draft asks for
// it.
func TestThePriceThatWasInferredIsRefused(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(2, "kg")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	_, err = NewLine("Azúcar", measure, Price{Total: 5000, Inferred: true})
	if !errors.Is(err, ErrInferredPrice) {
		t.Fatalf("the error is %v, want %v", err, ErrInferredPrice)
	}
}

// TestTheTwoPricesThatDisagreeAreRefused proves the rule of the domain:
// when the owner named both the money that was paid and the price of the
// unit, the two must agree.
func TestTheTwoPricesThatDisagreeAreRefused(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(25, "kg")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	_, err = NewLine("Harina", measure, Price{Total: 12000, Unit: 500})
	if !errors.Is(err, ErrPriceMismatch) {
		t.Fatalf("the error is %v, want %v", err, ErrPriceMismatch)
	}
}

// TestTheTwoPricesThatAgreeAreAccepted proves that the pair that agrees is
// the one the domain accepts: 25 kilos at 480 pesos are 12000 pesos.
func TestTheTwoPricesThatAgreeAreAccepted(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(25, "kg")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	line, err := NewLine("Harina", measure, Price{Total: 12000, Unit: 480})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	if line.Price.Total != 12000 {
		t.Errorf("the money paid is %d, want 12000", line.Price.Total)
	}
	if line.Price.Unit != 480 {
		t.Errorf("the price of the unit is %d, want 480", line.Price.Unit)
	}
}

// TestTheLineWithoutMoneyIsBuilt proves that a line the owner did not
// price is a line the draft still needs: the money is missing, and the
// draft says so.
func TestTheLineWithoutMoneyIsBuilt(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(3, "unidad")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	line, err := NewLine("Pan", measure, Price{})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	if line.Price.Present() {
		t.Errorf("the line carries money: %+v", line.Price)
	}
}

// TestThePriceOfANegativeMoneyIsRefused proves that the money of a line is
// never a negative number.
func TestThePriceOfANegativeMoneyIsRefused(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(3, "unidad")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	_, err = NewLine("Pan", measure, Price{Total: -100})
	if !errors.Is(err, ErrInvalidPrice) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidPrice)
	}
}

// TestTheMoneyOfTheBookIsWholePesos proves that the money of a line is the
// money of the book: pesos, never a fraction of one.
func TestTheMoneyOfTheBookIsWholePesos(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(2, "kg")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	line, err := NewLine("Harina", measure, Price{Total: ledger.Amount(12000)})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	if line.Price.Total != ledger.Amount(12000) {
		t.Errorf("the money is %d, want 12000", line.Price.Total)
	}
}
