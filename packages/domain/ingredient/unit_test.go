package ingredient

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
)

// anIngredientWithTheUnit is an ingredient of the catalog that the owner
// completed with the unit she declared.
func anIngredientWithTheUnit(t *testing.T, unit Unit) Ingredient {
	t.Helper()

	completed, err := Complete(anIngredient(t, 10, business.ID(77), "Harina"), Declaration{Unit: unit})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	return completed
}

// TestTheUnitsOfTheBookAreTheFourTheOwnerReads proves that the units of
// the book are the gram, the kilogram, the package and the piece, and that
// the vocabulary belongs to the domain.
func TestTheUnitsOfTheBookAreTheFourTheOwnerReads(t *testing.T) {
	t.Parallel()

	want := []Unit{UnitGram, UnitKilogram, UnitPackage, UnitPiece}

	all := Units()
	if len(all) != len(want) {
		t.Fatalf("the book knows %d units, want %d", len(all), len(want))
	}

	for index, one := range want {
		if all[index] != one {
			t.Errorf("the unit %d is %q, want %q", index, all[index], one)
		}
	}
}

// TestTheUnitIsTheNameTheStorageAndTheProtocolCarry proves that the unit
// carries the name the storage and the protocol hold.
func TestTheUnitIsTheNameTheStorageAndTheProtocolCarry(t *testing.T) {
	t.Parallel()

	if UnitGram.String() != "g" {
		t.Errorf("the gram is %q, want %q", UnitGram.String(), "g")
	}
	if UnitKilogram.String() != "kg" {
		t.Errorf("the kilogram is %q, want %q", UnitKilogram.String(), "kg")
	}
	if UnitPackage.String() != "paquete" {
		t.Errorf("the package is %q, want %q", UnitPackage.String(), "paquete")
	}
	if UnitPiece.String() != "unidad" {
		t.Errorf("the piece is %q, want %q", UnitPiece.String(), "unidad")
	}
}

// TestTheUnitTheOwnerNamedIsReadTheWayAPersonReadsIt proves that the name
// of the unit is read the way a person reads it: trimmed, and without
// case, and the way the owner says it.
func TestTheUnitTheOwnerNamedIsReadTheWayAPersonReadsIt(t *testing.T) {
	t.Parallel()

	cases := map[string]Unit{
		"g":         UnitGram,
		" GRAMO ":   UnitGram,
		"kg":        UnitKilogram,
		"Kilos":     UnitKilogram,
		"kilogramo": UnitKilogram,
		" PAQUETE ": UnitPackage,
		"unidad":    UnitPiece,
		" Piezas  ": UnitPiece,
	}

	for spoken, want := range cases {
		unit, err := ParseUnit(spoken)
		if err != nil {
			t.Fatalf("the unit %q was refused: %v", spoken, err)
		}
		if unit != want {
			t.Errorf("the unit %q is %q, want %q", spoken, unit, want)
		}
	}
}

// TestTheUnitTheBookDoesNotKnowIsRefused proves that a unit the book does
// not know is never guessed: the units of the book are a rule.
func TestTheUnitTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	_, err := ParseUnit("libra")
	if !errors.Is(err, ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownUnit)
	}
}

// TestTheUnitThatIsNotThereIsRefused proves that a unit is required: the
// owner said something, and the book never answers about nothing.
func TestTheUnitThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	_, err := ParseUnit("   ")
	if !errors.Is(err, ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownUnit)
	}
}

// TestTheGramIsOneAndTheKilogramIsAThousand proves that the conversion of
// the units into the gram is declared: the gram is one, and the kilogram
// is a thousand.
func TestTheGramIsOneAndTheKilogramIsAThousand(t *testing.T) {
	t.Parallel()

	grams, err := anIngredientWithTheUnit(t, UnitGram).Grams(1)
	if err != nil {
		t.Fatalf("the grams were refused: %v", err)
	}
	if grams != 1 {
		t.Errorf("the gram is %v, want 1", grams)
	}

	kilograms, err := anIngredientWithTheUnit(t, UnitKilogram).Grams(2.5)
	if err != nil {
		t.Fatalf("the grams were refused: %v", err)
	}
	if kilograms != 2500 {
		t.Errorf("the kilograms are %v grams, want 2500", kilograms)
	}
}

// TestThePackageIsTheWeightTheOwnerDeclares proves that the package is
// converted into the gram with the weight the owner declared, and that the
// weight of a package is never assumed.
func TestThePackageIsTheWeightTheOwnerDeclares(t *testing.T) {
	t.Parallel()

	declared, err := Complete(anIngredient(t, 10, business.ID(77), "Harina"), Declaration{
		Unit:         UnitPackage,
		PackageGrams: 50000,
	})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	grams, err := declared.Grams(2)
	if err != nil {
		t.Fatalf("the grams were refused: %v", err)
	}
	if grams != 100000 {
		t.Errorf("the packages are %v grams, want 100000", grams)
	}

	undeclared := anIngredientWithTheUnit(t, UnitPackage)

	if _, err := undeclared.Grams(1); !errors.Is(err, ErrMissingPackageWeight) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingPackageWeight)
	}
}

// TestThePieceIsNeverConvertedIntoGrams proves that the piece is one unit
// of the ingredient, and that its weight is never assumed: the book never
// converts it into grams.
func TestThePieceIsNeverConvertedIntoGrams(t *testing.T) {
	t.Parallel()

	piece := anIngredientWithTheUnit(t, UnitPiece)

	if _, err := piece.Grams(3); !errors.Is(err, ErrUnknownConversion) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownConversion)
	}
}

// TestTheQuantityThatIsNotPositiveIsRefused proves that a quantity of the
// book is a positive number.
func TestTheQuantityThatIsNotPositiveIsRefused(t *testing.T) {
	t.Parallel()

	grams := anIngredientWithTheUnit(t, UnitGram)

	if _, err := grams.Grams(0); !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidQuantity)
	}
}
