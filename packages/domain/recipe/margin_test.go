package recipe_test

import (
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// almost asserts that two rates are the same number.
//
// The rates are not money: they are what the money weighs against, and two
// ways of dividing the same money do not always land on the same bit. The
// tolerance is the one of a rate that nobody trades with.
func almost(t *testing.T, name string, got, want float64) {
	t.Helper()

	const epsilon = 1e-9

	if difference := got - want; difference > epsilon || difference < -epsilon {
		t.Errorf("%s is %v, want %v", name, got, want)
	}
}

// theCostOfTheMargin is the cost of a batch of the margin: ten units of
// ninety grams, and one ingredient of five hundred grams.
//
// The direct cost is the money of the layers of the stock: two thousand
// four hundred pesos for the batch.
func theCostOfTheMargin() recipe.Cost {
	return recipe.Cost{
		RecipeID: recipe.ID(9),
		Name:     "Pan de molde",
		Lines: []recipe.CostLine{
			{
				IngredientID: ingredient.ID(10),
				Name:         "Harina panadera",
				Grams:        500,
				Money:        ledger.Amount(2400),
			},
		},
		Direct:      ledger.Amount(2400),
		Yield:       recipe.Yield{Units: 10, Grams: 90},
		PerUnit:     240,
		PerGram:     2400.0 / 900.0,
		PerKilogram: 2400.0 / 900.0 * 1000,
		Complete:    true,
		Missing:     []string{},
	}
}

// thePriceOfTheOwnerOfTheMargin is the price the owner charges for the
// product: eight thousand pesos for each one of the pieces.
func thePriceOfTheOwnerOfTheMargin() purchase.OwnerPrice {
	return purchase.OwnerPrice{Money: ledger.Amount(8000), Basis: purchase.PricePerUnit}
}

// theCatalogOfTheMargin is the catalog of the names the business buys.
func theCatalogOfTheMargin(t *testing.T) ingredient.Catalog {
	t.Helper()

	catalog, err := ingredient.NewCatalog(business.ID(77), []ingredient.Ingredient{
		{ID: ingredient.ID(10), BusinessID: business.ID(77), Name: "Harina panadera", Unit: ingredient.UnitKilogram},
	})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	return catalog
}

// theReferenceOfTheMarket is what the market was seen at: five thousand
// pesos the kilogram, read in a bulletin, on the first of March.
func theReferenceOfTheMarket() purchase.Reference {
	return purchase.Reference{
		ID:           purchase.ReferenceID(3),
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		Price:        ledger.Amount(5000),
		Unit:         ingredient.UnitKilogram.String(),
		Market:       "Plaza de Mercado",
		Source:       "Boletín de precios al por mayor, DANE",
		SeenOn:       theDayOfTheFirstOfMarch(),
	}
}

// theDayOfTheFirstOfMarch is the day the reference of the market was seen.
func theDayOfTheFirstOfMarch() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
}

// theDayOfTheReading is the day the owner reads the margin.
func theDayOfTheReading() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 11, 0, 0, 0, 0, time.UTC))
}

// TestTheMarginComparesThePriceOfTheOwnerWithTheDirectCost proves that
// the margin compares the price of the owner with the direct cost of the
// batch: the profit is what the price leaves above the cost.
func TestTheMarginComparesThePriceOfTheOwnerWithTheDirectCost(t *testing.T) {
	t.Parallel()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if margin.Price.Money != ledger.Amount(8000) {
		t.Errorf("the price of the owner is %d, want 8000", margin.Price.Money)
	}
	if margin.Direct.Money != ledger.Amount(2400) {
		t.Errorf("the direct cost is %d, want 2400", margin.Direct.Money)
	}
	if margin.Profit.Unknown {
		t.Fatalf("the profit is not known: %s", margin.Profit.Reason)
	}
	if margin.Profit.PerUnit != 8000-240 {
		t.Errorf("the profit per unit is %v, want %v", margin.Profit.PerUnit, float64(8000-240))
	}
	almost(t, "the profit per gram", margin.Profit.PerGram, 8000.0/90.0-2400.0/900.0)
	almost(t, "the profit per kilogram", margin.Profit.PerKilogram, 8000.0/90.0*1000-2400.0/900.0*1000)
}

// TestTheMarginAnswersTheProfitInPercent proves that the margin says what
// the profit is of the direct cost, and that the percent of nothing is not
// a number.
func TestTheMarginAnswersTheProfitInPercent(t *testing.T) {
	t.Parallel()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	almost(t, "the profit in percent of the direct cost", margin.Percent, (8000-240)/240.0*100)
}

// TestTheMarginComparesThePriceOfTheOwnerWithTheReferenceOfTheMarket
// proves that the margin compares the price of the owner with the
// reference of the market: what the owner pays and what the market says
// travel side by side, and the profit over the market is a number of its
// own.
func TestTheMarginComparesThePriceOfTheOwnerWithTheReferenceOfTheMarket(t *testing.T) {
	t.Parallel()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if margin.Market.Unknown {
		t.Fatalf("the money of the market is not known: %s", margin.Market.Reason)
	}
	if margin.Market.Money != ledger.Amount(2500) {
		t.Errorf("the market says %d for the ingredients of the batch, want 2500", margin.Market.Money)
	}
	if margin.Market.Paid != ledger.Amount(2400) {
		t.Errorf("the owner paid %d, want 2400", margin.Market.Paid)
	}
	if margin.Market.Difference != ledger.Amount(-100) {
		t.Errorf("the difference is %d, want -100", margin.Market.Difference)
	}
	if margin.MarketProfit.Unknown {
		t.Fatalf("the profit over the market is not known: %s", margin.MarketProfit.Reason)
	}
	if margin.MarketProfit.PerUnit != 8000-250 {
		t.Errorf("the profit over the market per unit is %v, want %v",
			margin.MarketProfit.PerUnit, float64(8000-250))
	}
}

// TestTheMarginNeverMovesTheMoneyOfTheBookWithAReference proves that a
// reference never moves the money of the book: the direct cost comes from
// the layers of the stock, and the reference is read beside it.
func TestTheMarginNeverMovesTheMoneyOfTheBookWithAReference(t *testing.T) {
	t.Parallel()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if margin.Direct.Money != ledger.Amount(2400) {
		t.Errorf("the direct cost is %d, want 2400: the reference never moves the money of the book",
			margin.Direct.Money)
	}
	if margin.Price.Money != ledger.Amount(8000) {
		t.Errorf("the price of the owner is %d, want 8000: the book never moves her price",
			margin.Price.Money)
	}
	if margin.Market.Lines[0].Paid != ledger.Amount(2400) {
		t.Errorf("what the owner paid is %d, want 2400", margin.Market.Lines[0].Paid)
	}
}

// TestTheMarginShowsEveryReferenceWithItsSourceAndItsDay proves that
// every reference carries its source and its day, and that the margin
// shows them.
func TestTheMarginShowsEveryReferenceWithItsSourceAndItsDay(t *testing.T) {
	t.Parallel()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	lines := margin.Market.Lines
	if len(lines) != 1 {
		t.Fatalf("the market carries %d lines, want 1", len(lines))
	}

	found := lines[0].References
	if len(found) != 1 {
		t.Fatalf("the line carries %d references, want 1", len(found))
	}
	if found[0].Source != "Boletín de precios al por mayor, DANE" {
		t.Errorf("the source is %q, want %q", found[0].Source, "Boletín de precios al por mayor, DANE")
	}
	if found[0].SeenOn.String() != "2026-03-01" {
		t.Errorf("the day it was seen is %q, want %q", found[0].SeenOn.String(), "2026-03-01")
	}
	if found[0].Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", found[0].Market, "Plaza de Mercado")
	}
}

// TestTheMarginShowsTheAgeOfAReferenceOlderThanTheOwnerExpects proves that
// a reference older than the owner expects is shown with its age, and that
// the age is counted in days of the book.
func TestTheMarginShowsTheAgeOfAReferenceOlderThanTheOwnerExpects(t *testing.T) {
	t.Parallel()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{MaxAgeDays: 7},
		theDayOfTheReading(),
	)

	reference := margin.Market.Lines[0].References[0]

	if reference.AgeDays != 10 {
		t.Errorf("the age of the reference is %d days, want 10", reference.AgeDays)
	}
	if !reference.Old {
		t.Error("the reference of ten days is not old, want it old: the owner expects seven days")
	}
}

// TestTheMarginDoesNotCallAFreshReferenceOld proves that the age of the
// reference is the one the owner expects: a reference inside the
// expectation is not old.
func TestTheMarginDoesNotCallAFreshReferenceOld(t *testing.T) {
	t.Parallel()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{MaxAgeDays: 30},
		theDayOfTheReading(),
	)

	reference := margin.Market.Lines[0].References[0]

	if reference.AgeDays != 10 {
		t.Errorf("the age of the reference is %d days, want 10", reference.AgeDays)
	}
	if reference.Old {
		t.Error("the reference of ten days is old, want it fresh: the owner expects thirty days")
	}
}

// TestTheMarginComparesWithTheNewestReferenceAndKeepsTheHistory proves
// that the market of today is the last one that was seen, and that the
// older references travel with it as the history they are.
func TestTheMarginComparesWithTheNewestReferenceAndKeepsTheHistory(t *testing.T) {
	t.Parallel()

	older := theReferenceOfTheMarket()
	older.ID = purchase.ReferenceID(2)
	older.Price = ledger.Amount(4000)
	older.SeenOn = ledger.NewDate(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))

	newer := theReferenceOfTheMarket()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{older, newer},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	line := margin.Market.Lines[0]

	if len(line.References) != 2 {
		t.Fatalf("the line carries %d references, want 2", len(line.References))
	}
	if line.Reference.ID != newer.ID {
		t.Errorf("the reference that compares is %d, want %d", line.Reference.ID, newer.ID)
	}
	if margin.Market.Money != ledger.Amount(2500) {
		t.Errorf("the market says %d, want 2500: the newest reference is the one that compares",
			margin.Market.Money)
	}
	if line.References[1].ID != older.ID {
		t.Errorf("the oldest reference is %d, want %d", line.References[1].ID, older.ID)
	}
}

// TestTheMarginOfAProductNobodyChargedIsNotKnown proves that the money of
// what the owner sells is never inferred: the margin of a price nobody
// declared says that it is not known, and the cost and the market still
// answer.
func TestTheMarginOfAProductNobodyChargedIsNotKnown(t *testing.T) {
	t.Parallel()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		purchase.OwnerPrice{},
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if !margin.Profit.Unknown {
		t.Fatal("the profit is known, want the news that it is not")
	}
	if margin.Profit.Reason == "" {
		t.Error("the margin does not say why the profit is not known")
	}
	if margin.Direct.Money != ledger.Amount(2400) {
		t.Errorf("the direct cost is %d, want 2400: the price of the owner does not move it",
			margin.Direct.Money)
	}
	if margin.Market.Money != ledger.Amount(2500) {
		t.Errorf("the market says %d, want 2500", margin.Market.Money)
	}
}

// TestTheMarginOfABatchThatCannotBeBakedIsNotKnown proves that the
// margin of a batch the stock does not cover is not known: the sum of a
// batch that cannot be baked is a number nobody can use.
func TestTheMarginOfABatchThatCannotBeBakedIsNotKnown(t *testing.T) {
	t.Parallel()

	cost := theCostOfTheMargin()
	cost.Complete = false
	cost.Missing = []string{"the stock does not hold 100 grams of Harina panadera"}

	margin := recipe.MarginOf(
		cost,
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if !margin.Profit.Unknown {
		t.Fatal("the profit is known, want the news that it is not")
	}
	if margin.Profit.Reason == "" {
		t.Error("the margin does not say why the profit is not known")
	}
	if len(margin.Missing) == 0 {
		t.Error("the margin does not say what the stock does not hold")
	}
}

// TestTheMarginOfAnIngredientTheMarketDoesNotKnowIsNotKnown proves that a
// sum of the market that hides a missing part is a lie: the money of the
// market is not known when one of the ingredients has no reference, and
// the answer says which one.
func TestTheMarginOfAnIngredientTheMarketDoesNotKnowIsNotKnown(t *testing.T) {
	t.Parallel()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		nil,
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if !margin.Market.Unknown {
		t.Fatal("the money of the market is known, want the news that it is not")
	}
	if margin.Market.Money != ledger.Amount(0) {
		t.Errorf("the market says %d, want none", margin.Market.Money)
	}
	if margin.Market.Reason == "" {
		t.Error("the margin does not say why the money of the market is not known")
	}
	if !margin.MarketProfit.Unknown {
		t.Error("the profit over the market is known, want the news that it is not")
	}
}

// TestTheMarginOfAReferenceInAPieceIsNotKnown proves that the weight of a
// piece is never assumed: a reference of a piece cannot be compared with
// the grams of the recipe.
func TestTheMarginOfAReferenceInAPieceIsNotKnown(t *testing.T) {
	t.Parallel()

	reference := theReferenceOfTheMarket()
	reference.Unit = ingredient.UnitPiece.String()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{reference},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if !margin.Market.Unknown {
		t.Fatal("the money of the market is known, want the news that it is not")
	}
	if margin.Market.Lines[0].Reason == "" {
		t.Error("the line does not say why the money of the market is not known")
	}
}

// TestTheMarginOfAReferenceInAPackageWithoutTheDeclaredWeightIsNotKnown
// proves that the weight of a package is declared and never assumed: the
// money of the market is not known until the owner says how much the
// package weighs.
func TestTheMarginOfAReferenceInAPackageWithoutTheDeclaredWeightIsNotKnown(t *testing.T) {
	t.Parallel()

	reference := theReferenceOfTheMarket()
	reference.Unit = ingredient.UnitPackage.String()

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{reference},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if !margin.Market.Unknown {
		t.Fatal("the money of the market is known, want the news that it is not")
	}
	if margin.Market.Lines[0].Reason == "" {
		t.Error("the line does not say why the money of the market is not known")
	}
}

// TestTheMarginOfAReferenceInAPackageWithTheDeclaredWeightIsKnown proves
// that the weight of a package the owner declared is used: the money of
// the market is derived from it.
func TestTheMarginOfAReferenceInAPackageWithTheDeclaredWeightIsKnown(t *testing.T) {
	t.Parallel()

	catalog, err := ingredient.NewCatalog(business.ID(77), []ingredient.Ingredient{
		{
			ID:           ingredient.ID(10),
			BusinessID:   business.ID(77),
			Name:         "Harina panadera",
			Unit:         ingredient.UnitPackage,
			PackageGrams: 1000,
		},
	})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	reference := theReferenceOfTheMarket()
	reference.Unit = ingredient.UnitPackage.String()
	reference.Price = ledger.Amount(5000)

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{reference},
		catalog,
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if margin.Market.Unknown {
		t.Fatalf("the money of the market is not known: %s", margin.Market.Reason)
	}
	if margin.Market.Money != ledger.Amount(2500) {
		t.Errorf("the market says %d, want 2500", margin.Market.Money)
	}
}

// TestTheMarginOfAProductSoldByWeight proves that the margin of a
// product the owner sells by the weight answers the three ways of
// selling: the price of the gram is hers, and the price of the unit is
// derived from the weight the recipe declared.
func TestTheMarginOfAProductSoldByWeight(t *testing.T) {
	t.Parallel()

	price := purchase.OwnerPrice{Money: ledger.Amount(100), Basis: purchase.PricePerGram}

	margin := recipe.MarginOf(
		theCostOfTheMargin(),
		price,
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if margin.PriceBasis != purchase.PricePerGram {
		t.Errorf("the way she charges is %q, want %q", margin.PriceBasis, purchase.PricePerGram)
	}
	if margin.Price.PerGram != 100 {
		t.Errorf("the price per gram is %v, want 100", margin.Price.PerGram)
	}
	if margin.Price.PerUnit != 9000 {
		t.Errorf("the price per unit is %v, want 9000", margin.Price.PerUnit)
	}
	if margin.Profit.PerUnit != 9000-240 {
		t.Errorf("the profit per unit is %v, want %v", margin.Profit.PerUnit, float64(9000-240))
	}
}

// TestTheMarginOfAnIngredientWhoseMoneyIsNotDerivableIsNotKnown proves
// that the difference between what the owner paid and what the market says
// needs both of them: a line whose money is not derivable leaves the money
// of the market unknown.
func TestTheMarginOfAnIngredientWhoseMoneyIsNotDerivableIsNotKnown(t *testing.T) {
	t.Parallel()

	cost := theCostOfTheMargin()
	cost.Lines[0].Unknown = true
	cost.Complete = false

	margin := recipe.MarginOf(
		cost,
		thePriceOfTheOwnerOfTheMargin(),
		[]purchase.Reference{theReferenceOfTheMarket()},
		theCatalogOfTheMargin(t),
		recipe.ReferenceExpectation{},
		theDayOfTheReading(),
	)

	if !margin.Market.Unknown {
		t.Fatal("the money of the market is known, want the news that it is not")
	}
	if margin.Market.Lines[0].Reason == "" {
		t.Error("the line does not say why the money of the market is not known")
	}
}
