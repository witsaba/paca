package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theReferenceOfTheMarketOfTheMargin is what the market was seen at: four
// hundred eighty pesos the kilogram, read in a bulletin, on the first of
// March.
func theReferenceOfTheMarketOfTheMargin() purchase.Reference {
	return purchase.Reference{
		ID:           purchase.ReferenceID(3),
		BusinessID:   business.ID(77),
		IngredientID: ingredient.ID(10),
		Price:        ledger.Amount(480),
		Unit:         ingredient.UnitKilogram.String(),
		Market:       "Plaza de Mercado",
		Source:       "Boletín de precios al por mayor, DANE",
		SeenOn:       ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)),
	}
}

// theServiceOfTheMargin is the use case of the margin, with the recipe,
// the layers, the catalog, the references and the clock of the tests.
func theServiceOfTheMargin(t *testing.T, references *fakeReferences) *application.MarginService {
	t.Helper()

	costs := application.NewBatchCostService(
		&fakeRecipes{held: []recipe.Recipe{theRecipeOfTheCost(t)}},
		&fakeLayers{layers: theLayersOfTheCost()},
		theCatalogOfTheCost(t),
		theLaborOfTheBusiness(t),
		theOverheadOfTheBusiness(t),
	)

	return application.NewMarginService(costs, references, theCatalogOfTheCost(t), theClock())
}

// TestTheMarginCarriesTheProfitAndTheMarket proves that the use case reads
// the cost of the batch, the price the owner declared and the references
// of the market, and answers the margin with the profit and with what the
// market said.
func TestTheMarginCarriesTheProfitAndTheMarket(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{held: []purchase.Reference{theReferenceOfTheMarketOfTheMargin()}}
	service := theServiceOfTheMargin(t, references)

	margins, err := service.Read(t.Context(), ports.ReadMargin{
		BusinessID: business.ID(77),
		Price:      8000,
		PriceBasis: "unidad",
	})
	if err != nil {
		t.Fatalf("the margin was refused: %v", err)
	}

	if len(margins) != 1 {
		t.Fatalf("the answer carries %d margins, want 1", len(margins))
	}

	margin := margins[0]

	if margin.Price.Money != ledger.Amount(8000) {
		t.Errorf("the price of the owner is %d, want 8000", margin.Price.Money)
	}
	if margin.Direct.Money != ledger.Amount(240) {
		t.Errorf("the direct cost is %d, want 240", margin.Direct.Money)
	}
	if margin.Profit.Unknown {
		t.Fatalf("the profit is not known: %s", margin.Profit.Reason)
	}
	if margin.Profit.PerUnit != 8000-24 {
		t.Errorf("the profit per unit is %v, want %v", margin.Profit.PerUnit, float64(8000-24))
	}
	if margin.Market.Unknown {
		t.Fatalf("the money of the market is not known: %s", margin.Market.Reason)
	}
	if margin.Market.Money != ledger.Amount(240) {
		t.Errorf("the market says %d, want 240", margin.Market.Money)
	}
	if margin.Market.Paid != ledger.Amount(240) {
		t.Errorf("what the owner paid is %d, want 240", margin.Market.Paid)
	}
}

// TestTheMarginShowsTheAgeOfTheReferences proves that the age of a
// reference is counted against the day of the reading, and that a
// reference older than the owner expects is shown with its age.
func TestTheMarginShowsTheAgeOfTheReferences(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{held: []purchase.Reference{theReferenceOfTheMarketOfTheMargin()}}
	service := theServiceOfTheMargin(t, references)

	margins, err := service.Read(t.Context(), ports.ReadMargin{
		BusinessID: business.ID(77),
		Price:      8000,
		PriceBasis: "unidad",
		MaxAgeDays: 7,
	})
	if err != nil {
		t.Fatalf("the margin was refused: %v", err)
	}

	if len(margins) != 1 {
		t.Fatalf("the answer carries %d margins, want 1", len(margins))
	}

	found := margins[0].Market.Lines[0].References
	if len(found) != 1 {
		t.Fatalf("the line carries %d references, want 1", len(found))
	}
	if found[0].AgeDays != 14 {
		t.Errorf("the age of the reference is %d days, want 14", found[0].AgeDays)
	}
	if !found[0].Old {
		t.Error("the reference of fourteen days is not old, want it old: the owner expects seven days")
	}
	if found[0].Source != "Boletín de precios al por mayor, DANE" {
		t.Errorf("the source is %q, want %q", found[0].Source, "Boletín de precios al por mayor, DANE")
	}
}

// TestTheMarginOfAProductNobodyChargedIsNotKnown proves that the money of
// what the owner sells is never inferred: the margin of a price nobody
// declared says that it is not known, and the cost and the market still
// answer.
func TestTheMarginOfAProductNobodyChargedIsNotKnown(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{held: []purchase.Reference{theReferenceOfTheMarketOfTheMargin()}}
	service := theServiceOfTheMargin(t, references)

	margins, err := service.Read(t.Context(), ports.ReadMargin{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the margin was refused: %v", err)
	}

	if len(margins) != 1 {
		t.Fatalf("the answer carries %d margins, want 1", len(margins))
	}

	if !margins[0].Profit.Unknown {
		t.Fatal("the profit is known, want the news that it is not")
	}
	if margins[0].Direct.Money != ledger.Amount(240) {
		t.Errorf("the direct cost is %d, want 240: the price of the owner does not move it",
			margins[0].Direct.Money)
	}
	if margins[0].Market.Money != ledger.Amount(240) {
		t.Errorf("the market says %d, want 240", margins[0].Market.Money)
	}
}

// TestTheMarginRefusesAWayOfChargingTheBookDoesNotKnow proves that the
// way the owner sells is her decision and never a guess: a way of charging
// the book does not know is refused.
func TestTheMarginRefusesAWayOfChargingTheBookDoesNotKnow(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{held: []purchase.Reference{theReferenceOfTheMarketOfTheMargin()}}
	service := theServiceOfTheMargin(t, references)

	_, err := service.Read(t.Context(), ports.ReadMargin{
		BusinessID: business.ID(77),
		Price:      8000,
		PriceBasis: "durzimo",
	})
	if !errors.Is(err, purchase.ErrUnknownPriceBasis) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrUnknownPriceBasis)
	}
}

// TestTheMarginOfABusinessThatIsNotThereIsRefused proves that the margin
// belongs to a business: a request that names no business is refused.
func TestTheMarginOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := theServiceOfTheMargin(t, references)

	_, err := service.Read(t.Context(), ports.ReadMargin{BusinessID: business.ID(0)})
	if !errors.Is(err, purchase.ErrMissingBusiness) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrMissingBusiness)
	}
}

// TestTheMarginWithoutADatabaseSaysSo proves that a service without a
// database says so, and it says it in the only capability it has.
func TestTheMarginWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewMarginService(nil, nil, nil, nil)

	_, err := service.Read(t.Context(), ports.ReadMargin{BusinessID: business.ID(77)})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
