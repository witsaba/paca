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
)

// aLayerOfTheTest is a layer of the stock the tests of the use case use.
func aLayerOfTheTest(id ingredient.ID, name string, quantity float64, unit string, cost ledger.Amount) purchase.Layer {
	return purchase.Layer{
		ID:           purchase.LayerID(1),
		BusinessID:   business.ID(77),
		IngredientID: id,
		Name:         name,
		PurchaseID:   purchase.ID(9),
		Day:          ledger.NewDate(theDayOfTheStock()),
		Bought:       purchase.Measure{Quantity: quantity, Unit: unit},
		Left:         purchase.Measure{Quantity: quantity, Unit: unit},
		Cost:         cost,
	}
}

// theDayOfTheStock is the day the layers of the test were bought.
func theDayOfTheStock() time.Time {
	return time.Date(2026, time.March, 1, 10, 0, 0, 0, time.UTC)
}

// TestTheStockAnswersWhatTheBusinessHolds proves that the use case reads
// the layers of the business and answers with what it holds and what it
// cost.
func TestTheStockAnswersWhatTheBusinessHolds(t *testing.T) {
	t.Parallel()

	layers := &fakeLayers{layers: []purchase.Layer{
		aLayerOfTheTest(ingredient.ID(10), "harina", 25, "kg", 12000),
	}}
	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera", "harina"),
	}}

	service := application.NewStockService(layers, ingredients)

	stock, err := service.Read(t.Context(), ports.ReadStock{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the stock was refused: %v", err)
	}

	if len(stock.Holdings) != 1 {
		t.Fatalf("the stock holds %d holdings, want 1", len(stock.Holdings))
	}
	if stock.Holdings[0].Name != "Harina panadera" {
		t.Errorf("the name of the holding is %q, want %q", stock.Holdings[0].Name, "Harina panadera")
	}
	if stock.Holdings[0].Quantity != 25 {
		t.Errorf("the quantity is %v, want 25", stock.Holdings[0].Quantity)
	}
	if stock.Holdings[0].Unit != "kg" {
		t.Errorf("the unit is %q, want %q", stock.Holdings[0].Unit, "kg")
	}
	if stock.Cost != ledger.Amount(12000) {
		t.Errorf("the money of the stock is %d, want 12000", stock.Cost)
	}
}

// TestTheStockOfOneIngredientIsTheStockOfThatIngredient proves that the
// answer of the use case holds the layers of the ingredient the agent
// names, and of no other one.
func TestTheStockOfOneIngredientIsTheStockOfThatIngredient(t *testing.T) {
	t.Parallel()

	layers := &fakeLayers{layers: []purchase.Layer{
		aLayerOfTheTest(ingredient.ID(10), "harina", 25, "kg", 12000),
	}}
	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera", "harina"),
	}}

	service := application.NewStockService(layers, ingredients)

	stock, err := service.Read(t.Context(), ports.ReadStock{
		BusinessID:   business.ID(77),
		IngredientID: 10,
	})
	if err != nil {
		t.Fatalf("the stock was refused: %v", err)
	}

	if len(stock.Holdings) != 1 {
		t.Fatalf("the stock holds %d holdings, want 1", len(stock.Holdings))
	}
	if stock.Holdings[0].IngredientID != ingredient.ID(10) {
		t.Errorf("the holding is of the ingredient %d, want 10", stock.Holdings[0].IngredientID)
	}
}

// TestTheStockOfAnIngredientTheCatalogDoesNotHoldIsRefused proves that the
// use case refuses an ingredient the catalog of the business does not hold.
func TestTheStockOfAnIngredientTheCatalogDoesNotHoldIsRefused(t *testing.T) {
	t.Parallel()

	layers := &fakeLayers{}
	ingredients := &fakeIngredients{}

	service := application.NewStockService(layers, ingredients)

	_, err := service.Read(t.Context(), ports.ReadStock{
		BusinessID:   business.ID(77),
		IngredientID: 10,
	})
	if !errors.Is(err, ingredient.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrNotFound)
	}
}

// TestTheStockWithoutADatabaseSaysSo proves that a service without a
// database says so, and it says it in the only capability it has.
func TestTheStockWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewStockService(nil, nil)

	_, err := service.Read(t.Context(), ports.ReadStock{BusinessID: business.ID(77)})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
