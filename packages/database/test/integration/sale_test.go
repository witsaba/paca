//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/user"
)

// saleRepository returns the repository of the sales, bound to the database
// of the suite.
func saleRepository(t *testing.T) *postgres.SaleRepository {
	t.Helper()

	repo, err := postgres.NewSaleRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the sales could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheSale opens the business that owns the sales of the test.
func theBakeryOfTheSale(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Sale")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería Del Sale"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theBookOfTheSale writes the stock, the recipe and the production the sale
// of the test takes units from.
func theBookOfTheSale(
	t *testing.T,
	ctx context.Context,
	bakery business.Business,
) (recipe.Recipe, recipe.Production, []recipe.ExpiringUnit) {
	t.Helper()

	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	production, err := productionsRepository(t).Record(ctx, theProductionOfTheTest(t, bakery, baked, layers, flour))
	if err != nil {
		t.Fatalf("the production could not be written: %v", err)
	}

	units, err := unitRepository(t).ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	return baked, production, units
}

// theSaleOfTheTest derives the sale the tests of the table write: three
// units of the product for five thousand pesos.
func theSaleOfTheTest(
	t *testing.T,
	bakery business.Business,
	baked recipe.Recipe,
	production recipe.Production,
	units []recipe.ExpiringUnit,
) recipe.Sale {
	t.Helper()

	sale, err := recipe.SaleOf(bakery.ID, []recipe.Production{production}, units, recipe.SaleDefinition{
		RecipeID:      baked.ID,
		RecipeName:    baked.Name,
		Quantity:      3,
		Unit:          "unidad",
		Money:         5000,
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)),
		Utterance:     "vendí tres panes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the sale of the test was refused: %v", err)
	}

	return sale
}

// TestTheSaleIsKeptWithItsProductAndItsMoney proves the rules of the table
// against a real database: the sale is written with the product, the
// quantity, the unit, the money the client paid, the cost the sale
// released, the day it happened and the word of the owner, and it is read
// back as it was written.
func TestTheSaleIsKeptWithItsProductAndItsMoney(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := saleRepository(t)
	bakery := theBakeryOfTheSale(t, ctx)
	baked, production, units := theBookOfTheSale(t, ctx, bakery)

	written, err := repo.Record(ctx, theSaleOfTheTest(t, bakery, baked, production, units))
	if err != nil {
		t.Fatalf("the sale could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the sale has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the sale is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.RecipeID != baked.ID {
		t.Errorf("the product of the sale is %d, want %d", written.RecipeID, baked.ID)
	}
	if written.RecipeName != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", written.RecipeName, "Pan de molde")
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
	if written.Cost <= 0 {
		t.Errorf("the cost of the sale is %d, want the cost it releases", written.Cost)
	}
	if written.Day.String() != "2026-03-14" {
		t.Errorf("the day of the sale is %q, want %q", written.Day.String(), "2026-03-14")
	}
	if written.Utterance != "vendí tres panes" {
		t.Errorf("the sentence of the sale is %q, want the one of the owner", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if len(written.Units) != 3 {
		t.Errorf("the sale took %d units, want 3", len(written.Units))
	}
}

// TestTheUnitsThatWereSoldAreNotInTheBookAnyMore proves the rule of the
// table against a real database: a unit that left the book is never taken
// again.
func TestTheUnitsThatWereSoldAreNotInTheBookAnyMore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := saleRepository(t)
	bakery := theBakeryOfTheSale(t, ctx)
	baked, production, units := theBookOfTheSale(t, ctx, bakery)

	before, err := unitRepository(t).ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(before) != 10 {
		t.Fatalf("the book holds %d units, want 10", len(before))
	}

	if _, err := repo.Record(ctx, theSaleOfTheTest(t, bakery, baked, production, units)); err != nil {
		t.Fatalf("the sale could not be written: %v", err)
	}

	after, err := unitRepository(t).ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(after) != 7 {
		t.Fatalf("the book holds %d units, want the 7 that did not leave", len(after))
	}
}

// TestTheUnitThatWasSoldIsRefused proves the rule of the table against a
// real database: a unit leaves the book only once, and the second sale of
// it is refused.
func TestTheUnitThatWasSoldIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := saleRepository(t)
	bakery := theBakeryOfTheSale(t, ctx)
	baked, production, units := theBookOfTheSale(t, ctx, bakery)

	sale := theSaleOfTheTest(t, bakery, baked, production, units)

	if _, err := repo.Record(ctx, sale); err != nil {
		t.Fatalf("the sale could not be written: %v", err)
	}

	_, err := repo.Record(ctx, sale)
	if !errors.Is(err, recipe.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidValue)
	}
}

// TestTheUnitThatWasWastedIsNotSold proves the rule of the table against a
// real database: the exits of the book share one table, and a unit that
// left as a waste is never offered for sale.
func TestTheUnitThatWasWastedIsNotSold(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheSale(t, ctx)
	baked, production, units := theBookOfTheSale(t, ctx, bakery)

	waste, err := recipe.WasteOf(bakery.ID, []recipe.Production{production}, units, recipe.WasteDefinition{
		RecipeID:      baked.ID,
		RecipeName:    baked.Name,
		Quantity:      3,
		Reason:        recipe.WasteBurnt,
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the waste of the test was refused: %v", err)
	}

	if _, err := wasteRepository(t).Record(ctx, waste); err != nil {
		t.Fatalf("the waste could not be written: %v", err)
	}

	held, err := unitRepository(t).ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	for _, one := range held {
		if one.ID == waste.Units[0].UnitID {
			t.Fatal("the unit that left the book as a waste is still in the book")
		}
	}

	// The sale takes the units that are still in the book: the ones that
	// left as a waste are not among them.
	sale, err := recipe.SaleOf(bakery.ID, []recipe.Production{production}, held, recipe.SaleDefinition{
		RecipeID:      baked.ID,
		RecipeName:    baked.Name,
		Quantity:      3,
		Unit:          "unidad",
		Money:         5000,
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the sale was refused: %v", err)
	}

	for _, one := range sale.Units {
		if one.UnitID == waste.Units[0].UnitID {
			t.Fatal("the sale took a unit that left the book as a waste")
		}
	}
}

// TestTheUnitThatWasSoldIsNotWasted proves the rule of the table against a
// real database: the exits of the book share one table, and a unit that
// left as a sale cannot leave again as a waste.
func TestTheUnitThatWasSoldIsNotWasted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := saleRepository(t)
	bakery := theBakeryOfTheSale(t, ctx)
	baked, production, units := theBookOfTheSale(t, ctx, bakery)

	sale := theSaleOfTheTest(t, bakery, baked, production, units)

	if _, err := repo.Record(ctx, sale); err != nil {
		t.Fatalf("the sale could not be written: %v", err)
	}

	waste, err := recipe.WasteOf(bakery.ID, []recipe.Production{production}, units, recipe.WasteDefinition{
		RecipeID:      baked.ID,
		RecipeName:    baked.Name,
		Quantity:      3,
		Reason:        recipe.WasteBurnt,
		Day:           ledger.NewDate(time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the waste of the test was refused: %v", err)
	}

	_, err = wasteRepository(t).Record(ctx, waste)
	if !errors.Is(err, recipe.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidValue)
	}
}

// TestTheSaleAndItsUnitsAndItsMovementsAreWrittenWhole proves the rule of
// the transaction against a real database: the sale, the units that left the
// book and the two movements of its money are written whole or not at all,
// and the storage is as it was when the work failed.
func TestTheSaleAndItsUnitsAndItsMovementsAreWrittenWhole(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sales := saleRepository(t)
	book := ledgerRepository(t)
	bakery := theBakeryOfTheSale(t, ctx)
	baked, production, units := theBookOfTheSale(t, ctx, bakery)

	sale := theSaleOfTheTest(t, bakery, baked, production, units)

	accounts := theAccountsOfTheBook(t, ctx, bakery,
		account.RoleCash, account.RoleSales, account.RoleCostOfSales, account.RoleInventory)

	money, err := sale.MoneyMovement(accounts[account.RoleCash].ID, accounts[account.RoleSales].ID)
	if err != nil {
		t.Fatalf("the movement of the money of the sale was refused: %v", err)
	}

	cost, err := sale.CostMovement(accounts[account.RoleCostOfSales].ID, accounts[account.RoleInventory].ID)
	if err != nil {
		t.Fatalf("the movement of the cost of the sale was refused: %v", err)
	}

	err = unitOfWork(t).Do(ctx, func(ctx context.Context) error {
		if _, err := sales.Record(ctx, sale); err != nil {
			return err
		}

		if _, err := book.Record(ctx, money); err != nil {
			return err
		}

		if _, err := book.Record(ctx, cost); err != nil {
			return err
		}

		// The work fails after the sale and its money were written: the
		// work goes back with them.
		return errors.New("the money of the sale did not move")
	})
	if err == nil {
		t.Fatal("the work was committed, want the failure of the work")
	}

	var sold int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM sales WHERE business_id = $1`, int64(bakery.ID)).Scan(&sold); err != nil {
		t.Fatalf("the sales could not be counted: %v", err)
	}

	if sold != 0 {
		t.Errorf("the sale was left behind: %d sales of the business", sold)
	}

	var exits int
	if err := theSuite.db.QueryRowContext(ctx, `
		SELECT count(*)
		  FROM unit_exits e
		  JOIN productions p ON p.id = e.production_id
		 WHERE p.business_id = $1`, int64(bakery.ID)).Scan(&exits); err != nil {
		t.Fatalf("the exits could not be counted: %v", err)
	}

	if exits != 0 {
		t.Errorf("the units were left behind: %d exits of the business", exits)
	}

	var movements int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM ledger_entries WHERE business_id = $1`, int64(bakery.ID)).Scan(&movements); err != nil {
		t.Fatalf("the movements could not be counted: %v", err)
	}

	if movements != 0 {
		t.Errorf("the money was left behind: %d movements of the business", movements)
	}
}

// TestTheSaleOfAProductThatIsNotThereIsRefused proves the rule of the
// table against a real database: the sale names a product of the business,
// and a product that is not there is refused.
func TestTheSaleOfAProductThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := saleRepository(t)
	bakery := theBakeryOfTheSale(t, ctx)

	sale, err := recipe.NewSale(bakery.ID, recipe.SaleFact{
		RecipeID:   recipe.ID(9999),
		RecipeName: "Pan de molde",
		Day:        ledger.NewDate(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)),
		Quantity:   3,
		Unit:       "unidad",
		Money:      5000,
		Cost:       3000,
		Units: []recipe.SoldUnit{
			{UnitID: recipe.UnitID(1), ProductionID: recipe.ProductionID(1), Position: 1, Grams: 90},
		},
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the sale of the test was refused: %v", err)
	}

	_, err = repo.Record(ctx, sale)
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}
}
