package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The fake of the storage of the sales.
//
// It remembers the sale it wrote.
type fakeSales struct {
	written []recipe.Sale
	err     error
}

// Record writes the sale and remembers it.
func (f *fakeSales) Record(_ context.Context, written recipe.Sale) (recipe.Sale, error) {
	if f.err != nil {
		return recipe.Sale{}, f.err
	}

	written.ID = recipe.SaleID(5)
	f.written = append(f.written, written)

	return written, nil
}

// The fake of the transaction of the storage of the sale.
//
// It keeps what the work wrote, and it gives it back when the work fails:
// a transaction that rolls back leaves the storage as it was.
type fakeSaleWork struct {
	sales *fakeSales
	book  *fakeBook
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakeSaleWork) Do(ctx context.Context, work func(context.Context) error) error {
	sales := len(f.sales.written)
	movements := len(f.book.recorded)

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.sales.written = f.sales.written[:sales]
	f.book.recorded = f.book.recorded[:movements]

	return err
}

// theProductionOfTheSale is the production the owner made: ten units of
// bread that cost ten thousand pesos.
func theProductionOfTheSale() recipe.Production {
	units := make([]recipe.ProductionUnit, 0, 10)

	for position := 1; position <= 10; position++ {
		units = append(units, recipe.ProductionUnit{
			ID:        recipe.UnitID(int64(position)),
			Position:  position,
			Grams:     90,
			ExpiresOn: ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)),
		})
	}

	return recipe.Production{
		ID:         recipe.ProductionID(3),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(9),
		RecipeName: "Pan de molde",
		Day:        ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
		GramsIn:    900,
		Money:      10000,
		Units:      units,
	}
}

// theUnitsOfTheSale are the units of the production that are still in the
// book.
func theUnitsOfTheSale() []recipe.ExpiringUnit {
	units := make([]recipe.ExpiringUnit, 0, 10)

	for position := 1; position <= 10; position++ {
		units = append(units, recipe.ExpiringUnit{
			ProductionUnit: recipe.ProductionUnit{
				ID:           recipe.UnitID(int64(position)),
				ProductionID: recipe.ProductionID(3),
				Position:     position,
				Grams:        90,
				ExpiresOn:    ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)),
			},
			RecipeID:   recipe.ID(9),
			RecipeName: "Pan de molde",
			ProducedOn: ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
		})
	}

	return units
}

// theAccountsOfTheSale are the accounts of the book of the bakery: the
// cash, the inventory, the sales and the cost of the sales.
func theAccountsOfTheSale() []account.Account {
	return []account.Account{
		{ID: account.ID(1), BusinessID: business.ID(77), Kind: account.KindAsset, Name: "Efectivo"},
		{ID: account.ID(3), BusinessID: business.ID(77), Kind: account.KindAsset, Name: "Inventario"}, //nolint:misspell // "Inventario" is the name the owner reads in their book.
		{ID: account.ID(4), BusinessID: business.ID(77), Kind: account.KindIncome, Name: "Ventas"},
		{ID: account.ID(5), BusinessID: business.ID(77), Kind: account.KindExpense, Name: "Costo de ventas"},
	}
}

// theSaleService builds the use case of the sale with the fakes of the
// test.
func theSaleService(
	t *testing.T,
	productions *fakeProductionsRead,
	units *fakeUnitsInTheBook,
	sales *fakeSales,
	book *fakeBook,
) *application.SaleService {
	t.Helper()

	return application.NewSaleService(
		productions,
		units,
		&fakeAccounts{existing: theAccountsOfTheSale()},
		book,
		sales,
		&fakeSaleWork{sales: sales, book: book},
		theClock(),
	)
}

// TestTheSaleIsWrittenWithTheProductTheQuantityAndTheMoney proves that
// the use case writes the sale the owner confirmed: the product, the
// quantity, the unit, the money the client paid and her word.
func TestTheSaleIsWrittenWithTheProductTheQuantityAndTheMoney(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheSale()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheSale()}
	sales := &fakeSales{}
	book := &fakeBook{}
	service := theSaleService(t, productions, units, sales, book)

	sale, err := service.Record(t.Context(), ports.RecordSale{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Unit:          "unidad",
		Money:         5000,
		Utterance:     "vendí tres panes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the sale was refused: %v", err)
	}

	if len(sales.written) != 1 {
		t.Fatalf("the use case wrote %d sales, want 1", len(sales.written))
	}

	written := sales.written[0]
	if written.RecipeID != recipe.ID(9) {
		t.Errorf("the product of the sale is %d, want %d", written.RecipeID, recipe.ID(9))
	}

	if written.Quantity != 3 {
		t.Errorf("the quantity of the sale is %v, want 3", written.Quantity)
	}

	if written.Unit != "unidad" {
		t.Errorf("the unit of the sale is %q, want %q", written.Unit, "unidad")
	}

	if written.Money != 5000 {
		t.Errorf("the money of the sale is %d, want 5000", written.Money)
	}

	if written.Cost != 3000 {
		t.Errorf("the cost of the sale is %d, want 3000", written.Cost)
	}

	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day of the sale is %q, want the day of today", written.Day.String())
	}

	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the owner is %q, want %q", written.ConfirmedWith, "sí")
	}

	if sale.ID != recipe.SaleID(5) {
		t.Errorf("the identity of the sale is %d, want 5", sale.ID)
	}
}

// TestTheSaleMovesTheMoneyInAndReleasesTheCost proves that the use case
// writes the two movements of the sale: the money lands in the account
// that receives it and it comes from the sales, and the cost leaves the
// inventory and lands in the account of the cost of the sales.
func TestTheSaleMovesTheMoneyInAndReleasesTheCost(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheSale()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheSale()}
	sales := &fakeSales{}
	book := &fakeBook{}
	service := theSaleService(t, productions, units, sales, book)

	_, err := service.Record(t.Context(), ports.RecordSale{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Money:         5000,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the sale was refused: %v", err)
	}

	if len(book.recorded) != 2 {
		t.Fatalf("the use case wrote %d movements, want 2", len(book.recorded))
	}

	money := book.recorded[0]
	if money.DebitAccountID != account.ID(1) {
		t.Errorf("the money lands in the account %d, want the cash", money.DebitAccountID)
	}

	if money.CreditAccountID != account.ID(4) {
		t.Errorf("the money comes from the account %d, want the sales", money.CreditAccountID)
	}

	if money.Amount != 5000 {
		t.Errorf("the movement carries %d pesos, want 5000", money.Amount)
	}

	cost := book.recorded[1]
	if cost.DebitAccountID != account.ID(5) {
		t.Errorf("the cost lands in the account %d, want the cost of the sales", cost.DebitAccountID)
	}

	if cost.CreditAccountID != account.ID(3) {
		t.Errorf("the cost comes from the account %d, want the inventory", cost.CreditAccountID)
	}

	if cost.Amount != 3000 {
		t.Errorf("the movement carries %d pesos, want 3000", cost.Amount)
	}
}

// TestTheSaleMovesTheMoneyToTheAccountTheOwnerNames proves that the use
// case writes the money of the sale in the account the owner names, and in
// the cash when she names none.
func TestTheSaleMovesTheMoneyToTheAccountTheOwnerNames(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheSale()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheSale()}
	sales := &fakeSales{}
	book := &fakeBook{}
	service := theSaleService(t, productions, units, sales, book)

	_, err := service.Record(t.Context(), ports.RecordSale{
		BusinessID:         business.ID(77),
		RecipeName:         "Pan de molde",
		Quantity:           3,
		Money:              5000,
		ReceivingAccountID: account.ID(3),
		ConfirmedWith:      "sí",
	})
	if err != nil {
		t.Fatalf("the sale was refused: %v", err)
	}

	if len(book.recorded) != 2 {
		t.Fatalf("the use case wrote %d movements, want 2", len(book.recorded))
	}

	if book.recorded[0].DebitAccountID != account.ID(3) {
		t.Errorf("the money lands in the account %d, want the one the owner named",
			book.recorded[0].DebitAccountID)
	}
}

// TestTheSaleAndItsMovementsAreWrittenInOneTransaction proves that a
// failure inside the transaction leaves nothing behind: the sale and the
// two movements of its money land together or not at all.
func TestTheSaleAndItsMovementsAreWrittenInOneTransaction(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheSale()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheSale()}
	sales := &fakeSales{}
	book := &fakeBook{err: errors.New("the book of the business is closed")}
	service := theSaleService(t, productions, units, sales, book)

	_, err := service.Record(t.Context(), ports.RecordSale{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Money:         5000,
		ConfirmedWith: "sí",
	})
	if err == nil {
		t.Fatal("the sale was written, want the failure of the book")
	}

	if len(sales.written) != 0 {
		t.Errorf("the use case wrote %d sales, want none", len(sales.written))
	}

	if len(book.recorded) != 0 {
		t.Errorf("the use case wrote %d movements, want none", len(book.recorded))
	}
}

// TestTheSaleIsRefusedWithoutTheWordOfTheOwner proves that the use case
// writes nothing when the sale carries no word: the sale is written with
// the confirmation of the owner.
func TestTheSaleIsRefusedWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheSale()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheSale()}
	sales := &fakeSales{}
	book := &fakeBook{}
	service := theSaleService(t, productions, units, sales, book)

	_, err := service.Record(t.Context(), ports.RecordSale{
		BusinessID: business.ID(77),
		RecipeName: "Pan de molde",
		Quantity:   3,
		Money:      5000,
	})
	if !errors.Is(err, recipe.ErrMissingSaleConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingSaleConfirmation)
	}

	if len(sales.written) != 0 {
		t.Errorf("the use case wrote %d sales, want none", len(sales.written))
	}
}

// TestTheSaleOfAnExpiredUnitIsRefused proves that the use case writes
// nothing when one of the units it takes expired: a unit that expired is
// never sold.
func TestTheSaleOfAnExpiredUnitIsRefused(t *testing.T) {
	t.Parallel()

	held := theUnitsOfTheSale()

	expired := held[0]
	expired.ExpiresOn = ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC))
	held[0] = expired

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheSale()}}
	units := &fakeUnitsInTheBook{held: held}
	sales := &fakeSales{}
	book := &fakeBook{}
	service := theSaleService(t, productions, units, sales, book)

	_, err := service.Record(t.Context(), ports.RecordSale{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Money:         5000,
		Day:           "2026-03-15",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrExpiredUnit) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrExpiredUnit)
	}

	if len(sales.written) != 0 {
		t.Errorf("the use case wrote %d sales, want none", len(sales.written))
	}

	if len(book.recorded) != 0 {
		t.Errorf("the use case wrote %d movements, want none", len(book.recorded))
	}
}

// TestTheSaleWithoutADatabaseSaysSo proves that the use case refuses to
// write anything when it has no collaborators: a service with no database
// is not a service that writes.
func TestTheSaleWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewSaleService(nil, nil, nil, nil, nil, nil, nil)

	_, err := service.Record(t.Context(), ports.RecordSale{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Money:         5000,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
