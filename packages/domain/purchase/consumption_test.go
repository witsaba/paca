package purchase

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// aLayerOfTheTest is a layer of the stock the tests of the consumption
// use.
//
// The layer bought what it holds, and it holds all of it: the tests of the
// consumption break it when they need a layer that is partly consumed.
func aLayerOfTheTest(
	id ingredient.ID,
	name string,
	quantity float64,
	unit string,
	cost ledger.Amount,
	day ledger.Date,
	purchaseID ID,
) Layer {
	return Layer{
		ID:                 LayerID(int(purchaseID)),
		BusinessID:         business.ID(77),
		PurchaseBusinessID: business.ID(77),
		IngredientID:       id,
		Name:               name,
		PurchaseID:         purchaseID,
		Position:           1,
		Day:                day,
		Bought:             Measure{Quantity: quantity, Unit: unit},
		Left:               Measure{Quantity: quantity, Unit: unit},
		Cost:               cost,
	}
}

// theFlourOfTheTest is the ingredient of the catalog the tests of the
// consumption use.
func theFlourOfTheTest() ingredient.Ingredient {
	flour, err := ingredient.New(business.ID(77), "Harina panadera", nil)
	if err != nil {
		panic(err)
	}
	flour.ID = ingredient.ID(10)

	return flour
}

// TestTheMoneyOfWhatIsLeftOfALayerIsTheMoneyOfWhatItBought proves the
// rule of the domain: the money of what is left of a layer is the money of
// what it bought, pro rata the quantity that is left.
//
// The rule was open (F4 of the verification of T11): a layer that holds a
// fifth of what it bought still carried the whole money.
func TestTheMoneyOfWhatIsLeftOfALayerIsTheMoneyOfWhatItBought(t *testing.T) {
	t.Parallel()

	layer := aLayerOfTheTest(ingredient.ID(10), "Harina panadera", 25, "kg", 12000,
		theDay(2026, time.March, 1), ID(9))
	layer.Left = Measure{Quantity: 5, Unit: "kg"}

	if layer.MoneyLeft() != ledger.Amount(2400) {
		t.Errorf("the money of what is left is %d, want 2400", layer.MoneyLeft())
	}
}

// TestTheMoneyOfWhatIsLeftOfALayerIsWhole proves that the money of the
// book counts whole pesos: the money of what is left of a layer is rounded
// to the nearest peso.
func TestTheMoneyOfWhatIsLeftOfALayerIsWhole(t *testing.T) {
	t.Parallel()

	layer := aLayerOfTheTest(ingredient.ID(10), "Harina panadera", 3, "kg", 100,
		theDay(2026, time.March, 1), ID(9))
	layer.Left = Measure{Quantity: 1, Unit: "kg"}

	if layer.MoneyLeft() != ledger.Amount(33) {
		t.Errorf("the money of what is left is %d, want 33", layer.MoneyLeft())
	}
}

// TestTheMoneyOfALayerThatIsWholeIsTheMoneyItCost proves that a layer that
// holds everything it bought carries the whole money.
func TestTheMoneyOfALayerThatIsWholeIsTheMoneyItCost(t *testing.T) {
	t.Parallel()

	layer := aLayerOfTheTest(ingredient.ID(10), "Harina panadera", 25, "kg", 12000,
		theDay(2026, time.March, 1), ID(9))

	if layer.MoneyLeft() != ledger.Amount(12000) {
		t.Errorf("the money of what is left is %d, want 12000", layer.MoneyLeft())
	}
}

// TestTheBatchTakesTheLayersInTheOrderOfTheFIFO proves the rule of the
// domain: the recipe consumes the layers in the order of the FIFO, and the
// oldest layer is the first one that leaves.
func TestTheBatchTakesTheLayersInTheOrderOfTheFIFO(t *testing.T) {
	t.Parallel()

	older := aLayerOfTheTest(ingredient.ID(10), "Harina panadera", 1, "kg", 1000,
		theDay(2026, time.March, 1), ID(9))
	newer := aLayerOfTheTest(ingredient.ID(10), "Harina panadera", 1, "kg", 2000,
		theDay(2026, time.March, 5), ID(10))

	consumption, err := Consume([]Layer{newer, older}, ingredient.ID(10), 1500, theFlourOfTheTest())
	if err != nil {
		t.Fatalf("the consumption was refused: %v", err)
	}

	if len(consumption.Takes) != 2 {
		t.Fatalf("the batch took from %d layers, want 2", len(consumption.Takes))
	}
	if consumption.Takes[0].PurchaseID != ID(9) {
		t.Errorf("the batch took first from the purchase %d, want the oldest one (9)", consumption.Takes[0].PurchaseID)
	}
	if consumption.Takes[0].Grams != 1000 {
		t.Errorf("the batch took %v grams of the oldest layer, want 1000", consumption.Takes[0].Grams)
	}
	if consumption.Takes[1].PurchaseID != ID(10) {
		t.Errorf("the batch took then from the purchase %d, want the newest one (10)", consumption.Takes[1].PurchaseID)
	}
	if consumption.Takes[1].Grams != 500 {
		t.Errorf("the batch took %v grams of the newest layer, want 500", consumption.Takes[1].Grams)
	}
}

// TestTheMoneyOfWhatTheBatchTakesIsTheMoneyOfTheLayer proves the rule of
// the domain: the money of what a batch takes is the money of the layer,
// pro rata the quantity that was taken.
func TestTheMoneyOfWhatTheBatchTakesIsTheMoneyOfTheLayer(t *testing.T) {
	t.Parallel()

	layer := aLayerOfTheTest(ingredient.ID(10), "Harina panadera", 25, "kg", 12000,
		theDay(2026, time.March, 1), ID(9))

	consumption, err := Consume([]Layer{layer}, ingredient.ID(10), 500, theFlourOfTheTest())
	if err != nil {
		t.Fatalf("the consumption was refused: %v", err)
	}

	if consumption.Money != ledger.Amount(240) {
		t.Errorf("the money of the batch is %d, want 240", consumption.Money)
	}
	if consumption.Taken != 500 {
		t.Errorf("the batch took %v grams, want 500", consumption.Taken)
	}
	if consumption.Missing != 0 {
		t.Errorf("the batch is missing %v grams, want none", consumption.Missing)
	}
}

// TestTheBatchTakesWhatTheStockHoldsAndSaysWhatIsMissing proves that the
// consumption never invents what the stock does not hold: the answer says
// what is missing.
func TestTheBatchTakesWhatTheStockHoldsAndSaysWhatIsMissing(t *testing.T) {
	t.Parallel()

	layer := aLayerOfTheTest(ingredient.ID(10), "Harina panadera", 1, "kg", 1200,
		theDay(2026, time.March, 1), ID(9))

	consumption, err := Consume([]Layer{layer}, ingredient.ID(10), 2500, theFlourOfTheTest())
	if err != nil {
		t.Fatalf("the consumption was refused: %v", err)
	}

	if consumption.Taken != 1000 {
		t.Errorf("the batch took %v grams, want the 1000 the stock holds", consumption.Taken)
	}
	if consumption.Missing != 1500 {
		t.Errorf("the batch is missing %v grams, want 1500", consumption.Missing)
	}
	if consumption.Money != ledger.Amount(1200) {
		t.Errorf("the money of the batch is %d, want 1200", consumption.Money)
	}
}

// TestTheMoneyOfAnIngredientHeldInPiecesIsNotDerivable proves that the
// book never assumes the weight of a piece: the money of an ingredient the
// stock holds in pieces is not derived, and the answer says why.
func TestTheMoneyOfAnIngredientHeldInPiecesIsNotDerivable(t *testing.T) {
	t.Parallel()

	layer := aLayerOfTheTest(ingredient.ID(10), "Huevos", 12, "unidad", 9000,
		theDay(2026, time.March, 1), ID(9))

	consumption, err := Consume([]Layer{layer}, ingredient.ID(10), 120, theFlourOfTheTest())
	if err != nil {
		t.Fatalf("the consumption was refused: %v", err)
	}

	if !consumption.Unknown {
		t.Fatal("the money of the ingredient was derived, want the news that it is not derivable")
	}
	if consumption.Reason == "" {
		t.Error("the answer does not say why the money is not derivable")
	}
}

// TestTheMoneyOfAPackageWithoutItsDeclaredWeightIsNotDerivable proves that
// the weight of a package is never assumed: the money of an ingredient the
// stock holds in packages is not derived when the owner declared no
// weight.
func TestTheMoneyOfAPackageWithoutItsDeclaredWeightIsNotDerivable(t *testing.T) {
	t.Parallel()

	layer := aLayerOfTheTest(ingredient.ID(10), "Harina panadera", 2, "paquete", 12000,
		theDay(2026, time.March, 1), ID(9))

	consumption, err := Consume([]Layer{layer}, ingredient.ID(10), 1000, theFlourOfTheTest())
	if err != nil {
		t.Fatalf("the consumption was refused: %v", err)
	}

	if !consumption.Unknown {
		t.Fatal("the money of the ingredient was derived, want the news that it is not derivable")
	}
}

// TestTheMoneyOfAPackageIsDerivedWhenTheOwnerDeclaredItsWeight proves that
// the weight of a package is the one the owner declared: the money of the
// ingredient is derived from it.
func TestTheMoneyOfAPackageIsDerivedWhenTheOwnerDeclaredItsWeight(t *testing.T) {
	t.Parallel()

	layer := aLayerOfTheTest(ingredient.ID(10), "Harina panadera", 2, "paquete", 12000,
		theDay(2026, time.March, 1), ID(9))

	flour := theFlourOfTheTest()
	flour.PackageGrams = 1000

	consumption, err := Consume([]Layer{layer}, ingredient.ID(10), 1000, flour)
	if err != nil {
		t.Fatalf("the consumption was refused: %v", err)
	}

	if consumption.Unknown {
		t.Fatalf("the money of the ingredient was not derived: %s", consumption.Reason)
	}
	if consumption.Taken != 1000 {
		t.Errorf("the batch took %v grams, want 1000", consumption.Taken)
	}
	if consumption.Money != ledger.Amount(6000) {
		t.Errorf("the money of the batch is %d, want 6000", consumption.Money)
	}
}

// TestTheConsumptionOfNoGramsIsRefused proves that a batch that needs no
// grams is not a consumption.
func TestTheConsumptionOfNoGramsIsRefused(t *testing.T) {
	t.Parallel()

	_, err := Consume(nil, ingredient.ID(10), 0, theFlourOfTheTest())
	if !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidQuantity)
	}
}

// TestTheLayerIsRefusedWhenItsMoneyIsNotTheMoneyOfItsLine proves the rule
// of the domain: the money of the layer is the money of the line of the
// purchase it comes from.
//
// The rule was open (F5 of the verification of T11): a row another writer
// broke with another positive money passed the table and the domain.
func TestTheLayerIsRefusedWhenItsMoneyIsNotTheMoneyOfItsLine(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		aLineOfTheCatalog(t, "Harina", 25, "kg", 12000, ingredient.ID(10)))

	layer, err := newLayer(bought, bought.Lines[0], 1)
	if err != nil {
		t.Fatalf("the layer was refused: %v", err)
	}

	if err := layer.CheckedAgainstTheLine(bought, bought.Lines[0], 1); err != nil {
		t.Fatalf("the layer was refused: %v", err)
	}

	broken := layer
	broken.Cost = layer.Cost + 1

	err = broken.CheckedAgainstTheLine(bought, bought.Lines[0], 1)
	if !errors.Is(err, ErrLayerNotTied) {
		t.Fatalf("the error is %v, want %v", err, ErrLayerNotTied)
	}
}

// TestTheLayerIsRefusedWhenItsQuantityIsNotTheQuantityOfItsLine proves
// that the tie is not only about the money: a layer that claims another
// quantity than the line it comes from is refused.
func TestTheLayerIsRefusedWhenItsQuantityIsNotTheQuantityOfItsLine(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		aLineOfTheCatalog(t, "Harina", 25, "kg", 12000, ingredient.ID(10)))

	layer, err := newLayer(bought, bought.Lines[0], 1)
	if err != nil {
		t.Fatalf("the layer was refused: %v", err)
	}

	broken := layer
	broken.Bought = Measure{Quantity: 24, Unit: "kg"}
	broken.Left = Measure{Quantity: 24, Unit: "kg"}

	err = broken.CheckedAgainstTheLine(bought, bought.Lines[0], 1)
	if !errors.Is(err, ErrLayerNotTied) {
		t.Fatalf("the error is %v, want %v", err, ErrLayerNotTied)
	}
}

// TestTheLayerIsRefusedWhenItIsOfAnotherIngredientThanItsLine proves that
// the tie holds the ingredient too: a layer of another ingredient than the
// line it comes from is refused.
func TestTheLayerIsRefusedWhenItIsOfAnotherIngredientThanItsLine(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		aLineOfTheCatalog(t, "Harina", 25, "kg", 12000, ingredient.ID(10)))

	layer, err := newLayer(bought, bought.Lines[0], 1)
	if err != nil {
		t.Fatalf("the layer was refused: %v", err)
	}

	broken := layer
	broken.IngredientID = ingredient.ID(11)

	err = broken.CheckedAgainstTheLine(bought, bought.Lines[0], 1)
	if !errors.Is(err, ErrLayerNotTied) {
		t.Fatalf("the error is %v, want %v", err, ErrLayerNotTied)
	}
}
