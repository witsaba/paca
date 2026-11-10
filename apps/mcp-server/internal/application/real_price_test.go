package application_test

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theRecipeOfTheRealPrice is the recipe of the report: the same recipe of
// the cost, with the hours of the oven in its steps.
func theRecipeOfTheRealPrice(t *testing.T) recipe.Recipe {
	t.Helper()

	baked := theRecipeOfTheCost(t)
	baked.Steps = []recipe.Step{
		{Instruction: "Amasar.", Minutes: 30, Kind: recipe.LaborHandsOn},
		{Instruction: "Hornear.", Minutes: 45, Kind: recipe.LaborOven},
	}

	return baked
}

// TestTheReportOfTheRealPriceCarriesTheThreeLayersAndThePrice proves that
// the use case reads the cost of the batch and the price the owner
// declared, and answers the report with the three layers of the cost apart
// and the price of the owner beside them.
func TestTheReportOfTheRealPriceCarriesTheThreeLayersAndThePrice(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheRealPrice(t)}}
	layers := &fakeLayers{layers: theLayersOfTheCost()}

	service := application.NewRealPriceService(recipes, layers, theCatalogOfTheCost(t),
		theLaborOfTheBusiness(t), theOverheadOfTheBusiness(t))

	reports, err := service.Read(t.Context(), ports.ReadRealPrice{
		BusinessID: business.ID(77),
		Price:      8000,
		PriceBasis: "unidad",
	})
	if err != nil {
		t.Fatalf("the report was refused: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("the answer carries %d reports, want 1", len(reports))
	}

	report := reports[0]

	if report.Direct.Money != ledger.Amount(240) {
		t.Errorf("the direct cost is %d, want 240", report.Direct.Money)
	}
	if report.Labor.Money != ledger.Amount(25000) {
		t.Errorf("the labor is %d, want 25000", report.Labor.Money)
	}
	if report.Overhead.Money != ledger.Amount(1125) {
		t.Errorf("the overhead is %d, want 1125", report.Overhead.Money)
	}
	if report.Total.Money != ledger.Amount(240+25000+1125) {
		t.Errorf("the total of the three layers is %d, want %d",
			report.Total.Money, ledger.Amount(240+25000+1125))
	}
	if report.Price.Money != ledger.Amount(8000) {
		t.Errorf("the price of the owner is %d, want 8000", report.Price.Money)
	}
	if report.PriceBasis != purchase.PricePerUnit {
		t.Errorf("the way she charges is %q, want %q", report.PriceBasis, purchase.PricePerUnit)
	}
}

// TestTheReportOfTheRealPriceSaysThatThePriceOfTheOwnerIsNotKnown proves
// that the money of what the owner sells is never inferred: a report of a
// price nobody declared says that it is not known, and the layers of the
// cost still answer.
func TestTheReportOfTheRealPriceSaysThatThePriceOfTheOwnerIsNotKnown(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheRealPrice(t)}}
	layers := &fakeLayers{layers: theLayersOfTheCost()}

	service := application.NewRealPriceService(recipes, layers, theCatalogOfTheCost(t),
		theLaborOfTheBusiness(t), theOverheadOfTheBusiness(t))

	reports, err := service.Read(t.Context(), ports.ReadRealPrice{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the report was refused: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("the answer carries %d reports, want 1", len(reports))
	}

	if !reports[0].Price.Unknown {
		t.Fatal("the price of the owner is known, want the news that it is not")
	}
	if reports[0].Price.Money != ledger.Amount(0) {
		t.Errorf("the price of the owner is %d, want none", reports[0].Price.Money)
	}
	if len(reports[0].Missing) == 0 {
		t.Error("the report does not say what it still needs")
	}
	if reports[0].Direct.Money != ledger.Amount(240) {
		t.Errorf("the direct cost is %d, want 240: the price of the owner does not move it",
			reports[0].Direct.Money)
	}
}

// TestTheReportOfTheRealPriceRefusesAWayOfChargingTheBookDoesNotKnow
// proves that the way the owner sells is her decision and never a guess:
// a way of charging the book does not know is refused.
func TestTheReportOfTheRealPriceRefusesAWayOfChargingTheBookDoesNotKnow(t *testing.T) {
	t.Parallel()

	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheRealPrice(t)}}
	layers := &fakeLayers{layers: theLayersOfTheCost()}

	service := application.NewRealPriceService(recipes, layers, theCatalogOfTheCost(t),
		theLaborOfTheBusiness(t), theOverheadOfTheBusiness(t))

	_, err := service.Read(t.Context(), ports.ReadRealPrice{
		BusinessID: business.ID(77),
		Price:      8000,
		PriceBasis: "durzimo",
	})
	if !errors.Is(err, purchase.ErrUnknownPriceBasis) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrUnknownPriceBasis)
	}
}

// TestTheReportOfTheRealPriceWithoutADatabaseSaysSo proves that a service
// without a database says so, and it says it in the only capability it
// has.
func TestTheReportOfTheRealPriceWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewRealPriceService(nil, nil, nil, nil, nil)

	_, err := service.Read(t.Context(), ports.ReadRealPrice{BusinessID: business.ID(77)})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
