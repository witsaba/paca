package recipe_test

import (
	"testing"

	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theCostOfTheReport is the cost of a batch of a recipe: ten units of
// ninety grams, the direct cost of the layers of the stock, the labour of
// the hours of its steps and the overhead of the hours of the oven.
//
// The three layers travel apart, and so do they here.
func theCostOfTheReport() recipe.Cost {
	return recipe.Cost{
		RecipeID: recipe.ID(9),
		Name:     "Pan de molde",
		Direct:   ledger.Amount(240),
		Labor: recipe.LaborCost{
			Hours: recipe.Hours{HandsOn: 0.5, Oven: 0.75, Waiting: 1},
			Value: ledger.Amount(20000),
			Money: ledger.Amount(15000),
		},
		Overhead: recipe.OverheadCost{
			OvenHours: 0.75,
			Rate:      1500,
			Money:     ledger.Amount(1125),
		},
		Yield:    recipe.Yield{Units: 10, Grams: 90},
		PerUnit:  24,
		Complete: true,
		Missing:  []string{},
	}
}

// thePriceOfTheOwner is the price the owner charges for the product: eight
// thousand pesos for each one of the pieces.
func thePriceOfTheOwner() purchase.OwnerPrice {
	return purchase.OwnerPrice{Money: ledger.Amount(8000), Basis: purchase.PricePerUnit}
}

// TestTheReportOfTheRealPriceShowsTheThreeLayersApart proves that the
// report shows the direct cost, the labour and the overhead apart: no
// layer of the cost hides inside another one.
func TestTheReportOfTheRealPriceShowsTheThreeLayersApart(t *testing.T) {
	t.Parallel()

	report := recipe.RealPriceOf(theCostOfTheReport(), thePriceOfTheOwner())

	if report.Direct.Money != ledger.Amount(240) {
		t.Errorf("the direct cost is %d, want 240", report.Direct.Money)
	}
	if report.Labor.Money != ledger.Amount(15000) {
		t.Errorf("the labour is %d, want 15000", report.Labor.Money)
	}
	if report.Overhead.Money != ledger.Amount(1125) {
		t.Errorf("the overhead is %d, want 1125", report.Overhead.Money)
	}
	if report.Direct.Money == report.Labor.Money {
		t.Error("the direct cost and the labour carry the same money: a layer hides inside another one")
	}
	if report.Direct.Money+report.Labor.Money == report.Direct.Money {
		t.Error("the layers of the cost were added into the direct cost")
	}
}

// TestTheReportOfTheRealPriceNeverHidesALayerInsideAnotherOne proves that
// the report never hides a layer inside another one: the sum of the three
// layers travels apart, beside them, and the layers keep their own money.
func TestTheReportOfTheRealPriceNeverHidesALayerInsideAnotherOne(t *testing.T) {
	t.Parallel()

	report := recipe.RealPriceOf(theCostOfTheReport(), thePriceOfTheOwner())

	if report.Total.Money != ledger.Amount(240+15000+1125) {
		t.Errorf("the total of the three layers is %d, want %d",
			report.Total.Money, ledger.Amount(240+15000+1125))
	}
	if report.Total.Money == report.Direct.Money {
		t.Error("the total carries the money of the direct cost: a layer hides inside another one")
	}
	if report.Total.PerUnit == report.Direct.PerUnit {
		t.Error("the total carries the rate of the direct cost: a layer hides inside another one")
	}
}

// TestTheReportOfTheRealPriceAnswersPerUnitPerGramAndPerKilogram proves
// that every layer of the report answers per unit, per gram and per
// kilogram: the comparison between two products is a lie when only one of
// the three is known.
func TestTheReportOfTheRealPriceAnswersPerUnitPerGramAndPerKilogram(t *testing.T) {
	t.Parallel()

	report := recipe.RealPriceOf(theCostOfTheReport(), thePriceOfTheOwner())

	if report.Direct.PerUnit != 24 {
		t.Errorf("the direct cost per unit is %v, want 24", report.Direct.PerUnit)
	}
	if report.Direct.PerGram != 240.0/900.0 {
		t.Errorf("the direct cost per gram is %v, want %v", report.Direct.PerGram, 240.0/900.0)
	}
	if report.Direct.PerKilogram != 240.0/900.0*1000 {
		t.Errorf("the direct cost per kilogram is %v, want %v",
			report.Direct.PerKilogram, 240.0/900.0*1000)
	}
	if report.Labor.PerUnit != 1500 {
		t.Errorf("the labour per unit is %v, want 1500", report.Labor.PerUnit)
	}
	if report.Overhead.PerUnit != 112.5 {
		t.Errorf("the overhead per unit is %v, want 112.5", report.Overhead.PerUnit)
	}
}

// TestTheReportOfTheRealPriceCarriesThePriceOfTheOwnerBesideTheLayers
// proves that the price of the owner travels beside the layers of the
// cost, and that the book never moves it: the money she declared is the
// money of the report.
func TestTheReportOfTheRealPriceCarriesThePriceOfTheOwnerBesideTheLayers(t *testing.T) {
	t.Parallel()

	report := recipe.RealPriceOf(theCostOfTheReport(), thePriceOfTheOwner())

	if report.Price.Money != ledger.Amount(8000) {
		t.Errorf("the price of the owner is %d, want 8000", report.Price.Money)
	}
	if report.PriceBasis != purchase.PricePerUnit {
		t.Errorf("the way she charges is %q, want %q", report.PriceBasis, purchase.PricePerUnit)
	}
	if report.Price.PerUnit != 8000 {
		t.Errorf("the price per unit is %v, want 8000", report.Price.PerUnit)
	}
	if report.Price.PerGram != 8000.0/90.0 {
		t.Errorf("the price per gram is %v, want %v", report.Price.PerGram, 8000.0/90.0)
	}
	if report.Price.PerKilogram != 8000.0/90.0*1000 {
		t.Errorf("the price per kilogram is %v, want %v", report.Price.PerKilogram, 8000.0/90.0*1000)
	}
}

// TestTheReportOfTheRealPriceOfAProductSheSellsByWeight proves that the
// report answers the price of the owner in the three ways of selling: the
// price of the gram is hers, and the price of the unit is derived from
// the weight the recipe declared.
func TestTheReportOfTheRealPriceOfAProductSheSellsByWeight(t *testing.T) {
	t.Parallel()

	price := purchase.OwnerPrice{Money: ledger.Amount(100), Basis: purchase.PricePerGram}

	report := recipe.RealPriceOf(theCostOfTheReport(), price)

	if report.Price.PerGram != 100 {
		t.Errorf("the price per gram is %v, want 100", report.Price.PerGram)
	}
	if report.Price.PerUnit != 9000 {
		t.Errorf("the price per unit is %v, want 9000", report.Price.PerUnit)
	}
	if report.Price.PerKilogram != 100000 {
		t.Errorf("the price per kilogram is %v, want 100000", report.Price.PerKilogram)
	}
}

// TestTheReportOfTheRealPriceSaysThatThePriceOfTheOwnerIsNotKnown proves
// that the money of what the owner sells is never inferred: a report of a
// price nobody declared says that it does not know it, and the layers of
// the cost still answer.
func TestTheReportOfTheRealPriceSaysThatThePriceOfTheOwnerIsNotKnown(t *testing.T) {
	t.Parallel()

	report := recipe.RealPriceOf(theCostOfTheReport(), purchase.OwnerPrice{})

	if !report.Price.Unknown {
		t.Fatal("the price of the owner is known, want the news that it is not")
	}
	if report.Price.Money != ledger.Amount(0) {
		t.Errorf("the price of the owner is %d, want none", report.Price.Money)
	}
	if report.Price.Reason == "" {
		t.Error("the answer does not say why the price of the owner is not known")
	}

	found := false

	for _, one := range report.Missing {
		if one == report.Price.Reason {
			found = true
		}
	}

	if !found {
		t.Errorf("the report does not say what it still needs: %v", report.Missing)
	}

	if report.Direct.Money != ledger.Amount(240) {
		t.Errorf("the direct cost is %d, want 240: the price of the owner does not move it",
			report.Direct.Money)
	}
}

// TestTheReportOfTheRealPriceOfABatchThatCannotBeBakedHasNoTotal proves
// that the sum of the three layers is the sum of the three layers of a
// batch that can be baked: a cost the stock cannot cover has no total, and
// the answer says what the stock does not hold.
func TestTheReportOfTheRealPriceOfABatchThatCannotBeBakedHasNoTotal(t *testing.T) {
	t.Parallel()

	cost := theCostOfTheReport()
	cost.Complete = false
	cost.Missing = []string{"the stock does not hold 100 grams of Harina panadera"}

	report := recipe.RealPriceOf(cost, thePriceOfTheOwner())

	if !report.Total.Unknown {
		t.Error("the total of a batch that cannot be baked is known, want the news that it is not")
	}
	if report.Total.Money != ledger.Amount(0) {
		t.Errorf("the total is %d, want none", report.Total.Money)
	}
	if len(report.Missing) == 0 {
		t.Error("the report does not say what the stock does not hold")
	}
}

// TestTheReportOfTheRealPriceOfABatchWithNoLabourHasNoTotal proves that
// the total of the three layers is unknown when one of them is: the money
// of the labour is never inferred.
func TestTheReportOfTheRealPriceOfABatchWithNoLabourHasNoTotal(t *testing.T) {
	t.Parallel()

	cost := theCostOfTheReport()
	cost.Labor = recipe.LaborCost{
		Unknown: true,
		Reason:  "the owner has not declared the value of the hour",
	}

	report := recipe.RealPriceOf(cost, thePriceOfTheOwner())

	if !report.Labor.Unknown {
		t.Error("the labour is known, want the news that it is not")
	}
	if !report.Total.Unknown {
		t.Error("the total is known, want the news that one of the layers is not")
	}
}
