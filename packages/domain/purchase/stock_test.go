package purchase

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// aLineOfTheCatalog is a line that names an ingredient of the catalog.
//
// The name the owner said is the name of the line, and the identity of the
// ingredient is the one the catalog answered.
func aLineOfTheCatalog(
	t *testing.T,
	name string,
	quantity float64,
	unit string,
	total ledger.Amount,
	id ingredient.ID,
) Line {
	t.Helper()

	return theLine(t, name, quantity, unit, total).WithIngredient(id)
}

// thePurchaseOfTheTest builds the purchase the tests of the stock use.
func thePurchaseOfTheTest(t *testing.T, day ledger.Date, lines ...Line) Purchase {
	t.Helper()

	total := ledger.Amount(0)
	for _, one := range lines {
		total += one.Price.Amount(one.Measure.Quantity)
	}

	return Purchase{
		ID:         ID(9),
		BusinessID: business.ID(77),
		Day:        day,
		Lines:      lines,
		Total:      total,
	}
}

// theCatalogOfTheTest builds the catalog of the business of the test.
func theCatalogOfTheTest(t *testing.T) ingredient.Catalog {
	t.Helper()

	flour, err := ingredient.New(business.ID(77), "Harina panadera", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}
	flour.ID = ingredient.ID(10)

	salt, err := ingredient.New(business.ID(77), "Sal", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}
	salt.ID = ingredient.ID(11)

	catalog, err := ingredient.NewCatalog(business.ID(77), []ingredient.Ingredient{flour, salt})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	return catalog
}

// TestTheLayerIsDerivedFromThePurchase proves the rule of the domain: the
// purchase is the fact and the layer is derived from it. The layer carries
// the ingredient, the quantity with its unit, the money it cost, the day
// it was bought and the purchase it comes from.
func TestTheLayerIsDerivedFromThePurchase(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		aLineOfTheCatalog(t, "Harina", 25, "kg", 12000, ingredient.ID(10)))

	layer, err := newLayer(bought, bought.Lines[0], 1)
	if err != nil {
		t.Fatalf("the layer was refused: %v", err)
	}

	if layer.IngredientID != ingredient.ID(10) {
		t.Errorf("the ingredient is %d, want 10", layer.IngredientID)
	}
	if layer.Bought.Quantity != 25 || layer.Bought.Unit != "kg" {
		t.Errorf("the quantity is %v %s, want 25 kg", layer.Bought.Quantity, layer.Bought.Unit)
	}
	if layer.Left.Quantity != 25 || layer.Left.Unit != "kg" {
		t.Errorf("what is left is %v %s, want 25 kg", layer.Left.Quantity, layer.Left.Unit)
	}
	if layer.Cost != ledger.Amount(12000) {
		t.Errorf("the money of the layer is %d, want 12000", layer.Cost)
	}
	if layer.Day != theDay(2026, time.March, 1) {
		t.Errorf("the day of the layer is %v, want the day of the purchase", layer.Day)
	}
	if layer.PurchaseID != bought.ID {
		t.Errorf("the purchase of the layer is %d, want %d", layer.PurchaseID, bought.ID)
	}
	if layer.BusinessID != business.ID(77) {
		t.Errorf("the business of the layer is %d, want 77", layer.BusinessID)
	}
}

// aWholeLayer is a layer that is whole: every rule of Checked holds for
// it, and the tests break one rule at a time.
func aWholeLayer() Layer {
	return Layer{
		ID:                 LayerID(3),
		BusinessID:         business.ID(77),
		PurchaseBusinessID: business.ID(77),
		IngredientID:       ingredient.ID(10),
		Name:               "Harina panadera",
		PurchaseID:         ID(9),
		Position:           1,
		Day:                theDay(2026, time.March, 1),
		Bought:             Measure{Quantity: 25, Unit: "kg"},
		Left:               Measure{Quantity: 25, Unit: "kg"},
		Cost:               ledger.Amount(12000),
	}
}

// TestTheLayerIsWholeProvesThatAWholeLayerIsAccepted proves that the
// layer that breaks no rule is accepted: the check is not a ceremony.
func TestTheLayerIsWholeProvesThatAWholeLayerIsAccepted(t *testing.T) {
	t.Parallel()

	whole := aWholeLayer()

	if _, err := whole.Checked(); err != nil {
		t.Fatalf("the layer was refused: %v", err)
	}
}

// TestTheLayerCheckedRefusesEveryBrokenRow proves every rule of the
// domain of the layer, one at a time: the layer names the ingredient of
// the catalog and it carries the name of that ingredient, it belongs to
// the business of its purchase, the quantity it bought is positive, the
// position of its line is positive, the unit is one of the units of the
// book, what is left of it is never more than what it bought, what is
// left of it is in the unit it bought, and the money is not negative.
//
// The rules of a domain are proved without a database.
func TestTheLayerCheckedRefusesEveryBrokenRow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		breaks func(l *Layer)
		wants  error
	}{
		{
			name:   "the layer names nothing of the catalog",
			breaks: func(l *Layer) { l.IngredientID = 0 },
			wants:  ErrLayerWithoutIngredient,
		},
		{
			name:   "the layer carries no name",
			breaks: func(l *Layer) { l.Name = "" },
			wants:  ErrLayerWithoutName,
		},
		{
			name:   "the layer claims another business than its purchase",
			breaks: func(l *Layer) { l.BusinessID = business.ID(78) },
			wants:  ErrLayerOfAnotherBusiness,
		},
		{
			name:   "the layer bought no quantity",
			breaks: func(l *Layer) { l.Bought.Quantity = 0 },
			wants:  ErrInvalidQuantity,
		},
		{
			name:   "the layer bought a negative quantity",
			breaks: func(l *Layer) { l.Bought.Quantity = -1 },
			wants:  ErrInvalidQuantity,
		},
		{
			name:   "the layer comes from no line of its purchase",
			breaks: func(l *Layer) { l.Position = 0 },
			wants:  ErrInvalidPosition,
		},
		{
			name:   "the layer carries a unit the book does not know",
			breaks: func(l *Layer) { l.Bought.Unit = "libra" },
			wants:  ingredient.ErrUnknownUnit,
		},
		{
			name:   "the layer carries a unit that is not the one of the book",
			breaks: func(l *Layer) { l.Bought.Unit = "kilos" },
			wants:  ErrInvalidUnit,
		},
		{
			name:   "the layer holds more than it bought",
			breaks: func(l *Layer) { l.Left.Quantity = 26 },
			wants:  ErrInvalidLayer,
		},
		{
			name:   "the layer holds a negative quantity",
			breaks: func(l *Layer) { l.Left.Quantity = -1 },
			wants:  ErrInvalidLayer,
		},
		{
			name:   "the layer holds another unit than the one it bought",
			breaks: func(l *Layer) { l.Left.Unit = "g" },
			wants:  ErrInvalidLayer,
		},
		{
			name:   "the money of the layer is negative",
			breaks: func(l *Layer) { l.Cost = -1 },
			wants:  ErrInvalidPrice,
		},
	}

	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			t.Parallel()

			broken := aWholeLayer()
			one.breaks(&broken)

			if _, err := broken.Checked(); !errors.Is(err, one.wants) {
				t.Fatalf("the error is %v, want %v", err, one.wants)
			}
		})
	}
}

// TestTheLayerCarriesTheCanonicalUnitOfTheBook proves that the layer
// carries the unit the book knows: the layer is derived, and the purchase
// it comes from carries the unit of the book too.
//
// The rule that changed: the purchase keeps the unit of the book as well,
// and not only the layer. The way the owner said the unit ("kilos") is
// normalised where the book stores the measurement, and the sentence of the
// owner keeps the exact wording.
func TestTheLayerCarriesTheCanonicalUnitOfTheBook(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		aLineOfTheCatalog(t, "Harina", 25, "kilos", 12000, ingredient.ID(10)))

	layer, err := newLayer(bought, bought.Lines[0], 1)
	if err != nil {
		t.Fatalf("the layer was refused: %v", err)
	}

	if layer.Bought.Unit != "kg" {
		t.Errorf("the unit of what it bought is %q, want %q", layer.Bought.Unit, "kg")
	}
	if layer.Left.Unit != "kg" {
		t.Errorf("the unit of what is left is %q, want %q", layer.Left.Unit, "kg")
	}
	if bought.Lines[0].Measure.Unit != "kg" {
		t.Errorf("the unit of the line of the purchase is %q, want %q: the book stores the unit of the book",
			bought.Lines[0].Measure.Unit, "kg")
	}
}

// TestTheLayerOfALineWithAUnitTheBookDoesNotKnowIsRefused proves that a
// line whose unit is not one of the units of the book has no layer: the
// quantity and its unit travel together, and the unit of the book is a
// rule of the domain.
//
// The line is built the way a row another writer broke it: the measure
// carries the unit, and the domain is the one that refuses it.
func TestTheLayerOfALineWithAUnitTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		Line{
			Name:    "Harina",
			Measure: Measure{Quantity: 25, Unit: "libra"},
			Price:   Price{Total: 12000},
		}.WithIngredient(ingredient.ID(10)))

	_, err := newLayer(bought, bought.Lines[0], 1)
	if !errors.Is(err, ingredient.ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrUnknownUnit)
	}
}

// TestTheLayerOfALineTheCatalogDoesNotKnowIsRefused proves that a line
// that names nothing of the catalog has no layer: a layer without its
// ingredient is not a layer.
func TestTheLayerOfALineTheCatalogDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		theLine(t, "Harina", 25, "kg", 12000))

	_, err := newLayer(bought, bought.Lines[0], 1)
	if !errors.Is(err, ErrLayerWithoutIngredient) {
		t.Fatalf("the error is %v, want %v", err, ErrLayerWithoutIngredient)
	}
}

// TestTheLayerOfALineWithoutAPositionIsRefused proves that a layer comes
// from one line of its purchase and from no other one: the position of the
// line is positive.
func TestTheLayerOfALineWithoutAPositionIsRefused(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		aLineOfTheCatalog(t, "Harina", 25, "kg", 12000, ingredient.ID(10)))

	_, err := newLayer(bought, bought.Lines[0], 0)
	if !errors.Is(err, ErrInvalidPosition) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidPosition)
	}
}

// TestTheLayersOfThePurchaseAreDerivedFromItsLines proves that every line
// of the purchase that names an ingredient of the catalog is a layer of the
// stock.
func TestTheLayersOfThePurchaseAreDerivedFromItsLines(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		aLineOfTheCatalog(t, "Harina", 25, "kg", 12000, ingredient.ID(10)),
		aLineOfTheCatalog(t, "Sal", 500, "g", 900, ingredient.ID(11)))

	layers, err := LayersOf(bought)
	if err != nil {
		t.Fatalf("the layers were refused: %v", err)
	}

	if len(layers) != 2 {
		t.Fatalf("the purchase derived %d layers, want 2", len(layers))
	}
	if layers[0].IngredientID != ingredient.ID(10) || layers[1].IngredientID != ingredient.ID(11) {
		t.Errorf("the layers are %v, want the lines of the purchase", layers)
	}
}

// TestTheLayersOfAPurchaseWithALineTheCatalogDoesNotKnowAreTheOnesOfThe
// LinesThatNameAnIngredient proves that a line the catalog does not know
// has no layer: the layer names the ingredient of the catalog, and the
// stock is the stock of the ingredients the business buys.
func TestTheLayersOfAPurchaseWithALineTheCatalogDoesNotKnowAreTheOnesOfTheLinesThatNameAnIngredient(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		theLine(t, "Harina", 25, "kg", 12000),
		aLineOfTheCatalog(t, "Sal", 500, "g", 900, ingredient.ID(11)))

	layers, err := LayersOf(bought)
	if err != nil {
		t.Fatalf("the layers were refused: %v", err)
	}

	if len(layers) != 1 {
		t.Fatalf("the purchase derived %d layers, want the one of the salt", len(layers))
	}
	if layers[0].IngredientID != ingredient.ID(11) {
		t.Errorf("the layer is of the ingredient %d, want 11", layers[0].IngredientID)
	}
	if layers[0].Position != 2 {
		t.Errorf("the position of the layer is %d, want 2", layers[0].Position)
	}
}

// TestTheStockIsTheSumOfTheLayersThatAreStillOpen proves the rule of the
// domain: the stock of an ingredient is the sum of the layers that are
// still open. A layer that is closed is not in the answer.
func TestTheStockIsTheSumOfTheLayersThatAreStillOpen(t *testing.T) {
	t.Parallel()

	open := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(9),
		Day:          theDay(2026, time.March, 1),
		Bought:       Measure{Quantity: 25, Unit: "kg"},
		Left:         Measure{Quantity: 25, Unit: "kg"},
		Cost:         ledger.Amount(12000),
	}
	closed := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(10),
		Day:          theDay(2026, time.February, 28),
		Bought:       Measure{Quantity: 5, Unit: "kg"},
		Left:         Measure{Quantity: 0, Unit: "kg"},
		Cost:         ledger.Amount(2000),
	}

	stock := StockOf([]Layer{open, closed}, theCatalogOfTheTest(t))

	if len(stock.Holdings) != 1 {
		t.Fatalf("the stock holds %d holdings, want 1", len(stock.Holdings))
	}
	holding := stock.Holdings[0]
	if holding.Quantity != 25 {
		t.Errorf("the quantity is %v, want 25", holding.Quantity)
	}
	if holding.Cost != ledger.Amount(12000) {
		t.Errorf("the money of the holding is %d, want 12000", holding.Cost)
	}
	if len(holding.Layers) != 1 {
		t.Fatalf("the holding carries %d layers, want the one that is open", len(holding.Layers))
	}
	if stock.Cost != ledger.Amount(12000) {
		t.Errorf("the money of the stock is %d, want 12000", stock.Cost)
	}
}

// TestTheOldestLayerLeavesFirst proves the rule of the domain: the
// valuation is FIFO, so the oldest layer is the first one that leaves.
func TestTheOldestLayerLeavesFirst(t *testing.T) {
	t.Parallel()

	newest := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(11),
		Day:          theDay(2026, time.March, 1),
		Bought:       Measure{Quantity: 10, Unit: "kg"},
		Left:         Measure{Quantity: 10, Unit: "kg"},
		Cost:         ledger.Amount(6000),
	}
	oldest := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(9),
		Day:          theDay(2026, time.February, 28),
		Bought:       Measure{Quantity: 25, Unit: "kg"},
		Left:         Measure{Quantity: 25, Unit: "kg"},
		Cost:         ledger.Amount(12000),
	}

	stock := StockOf([]Layer{newest, oldest}, theCatalogOfTheTest(t))

	holding := stock.Holdings[0]
	if len(holding.Layers) != 2 {
		t.Fatalf("the holding carries %d layers, want 2", len(holding.Layers))
	}
	if holding.Layers[0].Day != oldest.Day {
		t.Errorf("the first layer is the one of %v, want the oldest one", holding.Layers[0].Day)
	}
	if holding.Layers[1].Day != newest.Day {
		t.Errorf("the second layer is the one of %v, want the newest one", holding.Layers[1].Day)
	}
}

// TestTheLayersOfTheSameDayAreReadInTheOrderOfTheirPurchase proves that
// the answer of the stock is the same every time it is read.
func TestTheLayersOfTheSameDayAreReadInTheOrderOfTheirPurchase(t *testing.T) {
	t.Parallel()

	first := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(9),
		Day:          theDay(2026, time.March, 1),
		Bought:       Measure{Quantity: 5, Unit: "kg"},
		Left:         Measure{Quantity: 5, Unit: "kg"},
		Cost:         ledger.Amount(2000),
	}
	second := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(10),
		Day:          theDay(2026, time.March, 1),
		Bought:       Measure{Quantity: 5, Unit: "kg"},
		Left:         Measure{Quantity: 5, Unit: "kg"},
		Cost:         ledger.Amount(2500),
	}

	stock := StockOf([]Layer{second, first}, theCatalogOfTheTest(t))

	holding := stock.Holdings[0]
	if holding.Layers[0].PurchaseID != first.PurchaseID {
		t.Errorf("the first layer is the one of the purchase %d, want %d", holding.Layers[0].PurchaseID, first.PurchaseID)
	}
}

// TestTheLayersOfDifferentUnitsAreHeldApart proves the rule of the tree:
// the quantity and its unit travel together, so the book never adds a
// gram to a kilogram.
func TestTheLayersOfDifferentUnitsAreHeldApart(t *testing.T) {
	t.Parallel()

	kilograms := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(9),
		Day:          theDay(2026, time.March, 1),
		Bought:       Measure{Quantity: 25, Unit: "kg"},
		Left:         Measure{Quantity: 25, Unit: "kg"},
		Cost:         ledger.Amount(12000),
	}
	grams := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(10),
		Day:          theDay(2026, time.March, 2),
		Bought:       Measure{Quantity: 500, Unit: "g"},
		Left:         Measure{Quantity: 500, Unit: "g"},
		Cost:         ledger.Amount(900),
	}

	stock := StockOf([]Layer{kilograms, grams}, theCatalogOfTheTest(t))

	if len(stock.Holdings) != 2 {
		t.Fatalf("the stock holds %d holdings, want 2: the quantity and its unit travel together", len(stock.Holdings))
	}
	if stock.Holdings[0].Unit != "kg" || stock.Holdings[0].Quantity != 25 {
		t.Errorf("the first holding is %v %s, want 25 kg", stock.Holdings[0].Quantity, stock.Holdings[0].Unit)
	}
	if stock.Holdings[1].Unit != "g" || stock.Holdings[1].Quantity != 500 {
		t.Errorf("the second holding is %v %s, want 500 g", stock.Holdings[1].Quantity, stock.Holdings[1].Unit)
	}
	if stock.Cost != ledger.Amount(12900) {
		t.Errorf("the money of the stock is %d, want 12900", stock.Cost)
	}
}

// TestTheStockCarriesTheNameTheOwnerReads proves that the name of the
// holding is the name the owner reads of the ingredient, and not the name
// she said when she bought it.
func TestTheStockCarriesTheNameTheOwnerReads(t *testing.T) {
	t.Parallel()

	layer := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(9),
		Day:          theDay(2026, time.March, 1),
		Name:         "harina",
		Bought:       Measure{Quantity: 25, Unit: "kg"},
		Left:         Measure{Quantity: 25, Unit: "kg"},
		Cost:         ledger.Amount(12000),
	}

	stock := StockOf([]Layer{layer}, theCatalogOfTheTest(t))

	if stock.Holdings[0].Name != "Harina panadera" {
		t.Errorf("the name of the holding is %q, want %q", stock.Holdings[0].Name, "Harina panadera")
	}
}

// TestTheLayersOfOneIngredientAreTheOnesOfThatIngredient proves that the
// stock of one ingredient is the sum of its own layers.
func TestTheLayersOfOneIngredientAreTheOnesOfThatIngredient(t *testing.T) {
	t.Parallel()

	flour := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		PurchaseID:   ID(9),
		Day:          theDay(2026, time.March, 1),
		Bought:       Measure{Quantity: 25, Unit: "kg"},
		Left:         Measure{Quantity: 25, Unit: "kg"},
		Cost:         ledger.Amount(12000),
	}
	salt := Layer{
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(11),
		PurchaseID:   ID(10),
		Day:          theDay(2026, time.March, 1),
		Bought:       Measure{Quantity: 500, Unit: "g"},
		Left:         Measure{Quantity: 500, Unit: "g"},
		Cost:         ledger.Amount(900),
	}

	held := OfIngredient([]Layer{flour, salt}, ingredient.ID(10))

	if len(held) != 1 {
		t.Fatalf("the answer holds %d layers, want the one of the flour", len(held))
	}
	if held[0].IngredientID != ingredient.ID(10) {
		t.Errorf("the layer is of the ingredient %d, want 10", held[0].IngredientID)
	}
}

// TestTheLayerBelongsToTheBusinessOfItsPurchase proves the rule of the
// domain: a layer is derived from its purchase, and it belongs to the
// business of that purchase.
func TestTheLayerBelongsToTheBusinessOfItsPurchase(t *testing.T) {
	t.Parallel()

	bought := thePurchaseOfTheTest(t, theDay(2026, time.March, 1),
		aLineOfTheCatalog(t, "Harina", 25, "kg", 12000, ingredient.ID(10)))

	layer, err := newLayer(bought, bought.Lines[0], 1)
	if err != nil {
		t.Fatalf("the layer was refused: %v", err)
	}

	if layer.PurchaseBusinessID != bought.BusinessID {
		t.Errorf("the business of the purchase of the layer is %d, want %d",
			layer.PurchaseBusinessID, bought.BusinessID)
	}
}

// TestTheStockNeedsTheBusinessItIsReadFor proves the rule of the domain:
// the stock belongs to a business, and a request that names no business is
// refused: there is no stock of nobody.
func TestTheStockNeedsTheBusinessItIsReadFor(t *testing.T) {
	t.Parallel()

	if _, err := TheBusinessOfTheStock(0); !errors.Is(err, ErrMissingBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingBusiness)
	}

	own, err := TheBusinessOfTheStock(business.ID(77))
	if err != nil {
		t.Fatalf("the business was refused: %v", err)
	}
	if own != business.ID(77) {
		t.Errorf("the business is %d, want 77", own)
	}
}

// TestTheStockCarriesTheMoneyOfWhatIsLeftOfItsLayers proves the rule of the
// domain: the money of what is left of a layer is the money of what it
// bought, pro rata the quantity that is left, and the stock carries the
// money of what it still holds.
//
// The rule was open (F4 of the verification of T11): a layer that holds a
// fifth of what it bought still carried the whole money.
func TestTheStockCarriesTheMoneyOfWhatIsLeftOfItsLayers(t *testing.T) {
	t.Parallel()

	layer := Layer{
		ID:                 LayerID(3),
		BusinessID:         business.ID(77),
		PurchaseBusinessID: business.ID(77),
		IngredientID:       ingredient.ID(10),
		Name:               "Harina panadera",
		PurchaseID:         ID(9),
		Position:           1,
		Day:                theDay(2026, time.March, 1),
		Bought:             Measure{Quantity: 25, Unit: "kg"},
		Left:               Measure{Quantity: 5, Unit: "kg"},
		Cost:               ledger.Amount(12000),
	}

	stock := StockOf([]Layer{layer}, theCatalogOfTheTest(t))

	if len(stock.Holdings) != 1 {
		t.Fatalf("the stock holds %d holdings, want 1", len(stock.Holdings))
	}
	if stock.Holdings[0].Cost != ledger.Amount(2400) {
		t.Errorf("the money of the holding is %d, want 2400", stock.Holdings[0].Cost)
	}
	if stock.Cost != ledger.Amount(2400) {
		t.Errorf("the money of the stock is %d, want 2400", stock.Cost)
	}
}
