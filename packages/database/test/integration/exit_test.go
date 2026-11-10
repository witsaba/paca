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

// exitRepository returns the repository of the exits, bound to the database
// of the suite.
func exitRepository(t *testing.T) *postgres.ExitRepository {
	t.Helper()

	repo, err := postgres.NewExitRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the exits could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheExit opens the business that owns the exits of the test.
func theBakeryOfTheExit(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Exit")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería Del Exit"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theBookOfTheExit writes the stock, the recipe and the production the
// exits of the test take units from.
func theBookOfTheExit(
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

// theExitOfTheTest derives the exit the tests of the table write: three
// units of the product the owner consumed.
func theExitOfTheTest(
	t *testing.T,
	bakery business.Business,
	baked recipe.Recipe,
	production recipe.Production,
	units []recipe.ExpiringUnit,
	kind recipe.ExitKind,
) recipe.Exit {
	t.Helper()

	exit, err := recipe.ExitOf(bakery.ID, []recipe.Production{production}, units, recipe.ExitDefinition{
		RecipeID:      baked.ID,
		RecipeName:    baked.Name,
		Kind:          kind,
		Quantity:      3,
		Unit:          "unidad",
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)),
		Utterance:     "me comí tres panes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the exit of the test was refused: %v", err)
	}

	return exit
}

// TestTheExitIsKeptWithItsProductAndItsCost proves the rules of the table
// against a real database: the exit is written with the kind, the product,
// the quantity, the unit, the cost it released, the tax treatment it waits
// for, the day it happened and the word of the owner, and it is read back
// as it was written.
func TestTheExitIsKeptWithItsProductAndItsCost(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := exitRepository(t)
	bakery := theBakeryOfTheExit(t, ctx)
	baked, production, units := theBookOfTheExit(t, ctx, bakery)

	written, err := repo.Record(ctx, theExitOfTheTest(t, bakery, baked, production, units, recipe.ExitConsumption))
	if err != nil {
		t.Fatalf("the exit could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the exit has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the exit is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.Kind != recipe.ExitConsumption {
		t.Errorf("the kind of the exit is %q, want %q", written.Kind, recipe.ExitConsumption)
	}
	if written.RecipeID != baked.ID {
		t.Errorf("the product of the exit is %d, want %d", written.RecipeID, baked.ID)
	}
	if written.RecipeName != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", written.RecipeName, "Pan de molde")
	}
	if written.Quantity != 3 {
		t.Errorf("the quantity of the exit is %v, want 3", written.Quantity)
	}
	if written.Unit != "unidad" {
		t.Errorf("the unit of the exit is %q, want %q", written.Unit, "unidad")
	}
	if written.Cost <= 0 {
		t.Errorf("the cost of the exit is %d, want the cost it releases", written.Cost)
	}
	if written.TaxTreatment != recipe.TaxPendingTheProfessional {
		t.Errorf("the tax treatment of the exit is %q, want %q",
			written.TaxTreatment, recipe.TaxPendingTheProfessional)
	}
	if written.TaxQuestion == "" {
		t.Error("the exit names no question about the taxes")
	}
	if written.Day.String() != "2026-03-14" {
		t.Errorf("the day of the exit is %q, want %q", written.Day.String(), "2026-03-14")
	}
	if written.Utterance != "me comí tres panes" {
		t.Errorf("the sentence of the exit is %q, want the one of the owner", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if len(written.Units) != 3 {
		t.Errorf("the exit took %d units, want 3", len(written.Units))
	}
}

// TestTheDonationIsKeptWithItsOwnKind proves the rules of the table
// against a real database: the donation is a fact of its own kind, and the
// tax treatment it waits for is kept with it.
func TestTheDonationIsKeptWithItsOwnKind(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := exitRepository(t)
	bakery := theBakeryOfTheExit(t, ctx)
	baked, production, units := theBookOfTheExit(t, ctx, bakery)

	written, err := repo.Record(ctx, theExitOfTheTest(t, bakery, baked, production, units, recipe.ExitDonation))
	if err != nil {
		t.Fatalf("the donation could not be written: %v", err)
	}

	if written.Kind != recipe.ExitDonation {
		t.Errorf("the kind of the exit is %q, want %q", written.Kind, recipe.ExitDonation)
	}

	if written.TaxTreatment != recipe.TaxPendingTheProfessional {
		t.Errorf("the tax treatment of the exit is %q, want %q",
			written.TaxTreatment, recipe.TaxPendingTheProfessional)
	}
}

// TestTheUnitsThatLeftTheBookAreNotInItAnyMore proves the rule of the
// table against a real database: a unit that left the book is never taken
// again.
func TestTheUnitsThatLeftTheBookAreNotInItAnyMore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := exitRepository(t)
	bakery := theBakeryOfTheExit(t, ctx)
	baked, production, units := theBookOfTheExit(t, ctx, bakery)

	before, err := unitRepository(t).ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(before) != 10 {
		t.Fatalf("the book holds %d units, want 10", len(before))
	}

	if _, err := repo.Record(ctx, theExitOfTheTest(t, bakery, baked, production, units, recipe.ExitConsumption)); err != nil {
		t.Fatalf("the exit could not be written: %v", err)
	}

	after, err := unitRepository(t).ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(after) != 7 {
		t.Fatalf("the book holds %d units, want the 7 that did not leave", len(after))
	}
}

// TestTheUnitThatWasConsumedIsRefused proves the rule of the table against
// a real database: a unit leaves the book only once, and the second exit
// of it is refused.
func TestTheUnitThatWasConsumedIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := exitRepository(t)
	bakery := theBakeryOfTheExit(t, ctx)
	baked, production, units := theBookOfTheExit(t, ctx, bakery)

	exit := theExitOfTheTest(t, bakery, baked, production, units, recipe.ExitConsumption)

	if _, err := repo.Record(ctx, exit); err != nil {
		t.Fatalf("the exit could not be written: %v", err)
	}

	_, err := repo.Record(ctx, exit)
	if !errors.Is(err, recipe.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidValue)
	}
}

// TestTheUnitThatWasSoldIsNotConsumed proves the rule of the table against
// a real database: the exits of the book share one table, and a unit that
// left as a sale never leaves again as a consumption.
func TestTheUnitThatWasSoldIsNotConsumed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheExit(t, ctx)
	baked, production, units := theBookOfTheExit(t, ctx, bakery)

	sale, err := recipe.SaleOf(bakery.ID, []recipe.Production{production}, units, recipe.SaleDefinition{
		RecipeID:      baked.ID,
		RecipeName:    baked.Name,
		Quantity:      3,
		Unit:          "unidad",
		Money:         5000,
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the sale of the test was refused: %v", err)
	}

	if _, err := saleRepository(t).Record(ctx, sale); err != nil {
		t.Fatalf("the sale could not be written: %v", err)
	}

	_, err = exitRepository(t).Record(ctx, theExitOfTheTest(t, bakery, baked, production, units, recipe.ExitConsumption))
	if !errors.Is(err, recipe.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidValue)
	}
}

// TestTheUnitThatWasWastedIsNotDonated proves the rule of the table against
// a real database: the exits of the book share one table, and a unit that
// left as a waste never leaves again as a donation.
func TestTheUnitThatWasWastedIsNotDonated(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheExit(t, ctx)
	baked, production, units := theBookOfTheExit(t, ctx, bakery)

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

	_, err = exitRepository(t).Record(ctx, theExitOfTheTest(t, bakery, baked, production, units, recipe.ExitDonation))
	if !errors.Is(err, recipe.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidValue)
	}
}

// TestTheExitAndItsUnitsAndItsMovementsAreWrittenWhole proves the rule of
// the transaction against a real database: the exit, the units that left the
// book and the movement of its cost are written whole or not at all, and
// the storage is as it was when the work failed.
func TestTheExitAndItsUnitsAndItsMovementsAreWrittenWhole(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	exits := exitRepository(t)
	book := ledgerRepository(t)
	bakery := theBakeryOfTheExit(t, ctx)
	baked, production, units := theBookOfTheExit(t, ctx, bakery)

	exit := theExitOfTheTest(t, bakery, baked, production, units, recipe.ExitConsumption)

	accounts := theAccountsOfTheBook(t, ctx, bakery,
		account.RoleCostOfSales, account.RoleInventory)

	cost, err := exit.CostMovement(accounts[account.RoleCostOfSales].ID, accounts[account.RoleInventory].ID)
	if err != nil {
		t.Fatalf("the movement of the cost of the exit was refused: %v", err)
	}

	err = unitOfWork(t).Do(ctx, func(ctx context.Context) error {
		if _, err := exits.Record(ctx, exit); err != nil {
			return err
		}

		if _, err := book.Record(ctx, cost); err != nil {
			return err
		}

		// The work fails after the exit and its cost were written: the
		// work goes back with them.
		return errors.New("the cost of the exit did not move")
	})
	if err == nil {
		t.Fatal("the work was committed, want the failure of the work")
	}

	var written int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM exits WHERE business_id = $1`, int64(bakery.ID)).Scan(&written); err != nil {
		t.Fatalf("the exits could not be counted: %v", err)
	}

	if written != 0 {
		t.Errorf("the exit was left behind: %d exits of the business", written)
	}

	var left int
	if err := theSuite.db.QueryRowContext(ctx, `
		SELECT count(*)
		  FROM unit_exits e
		  JOIN productions p ON p.id = e.production_id
		 WHERE p.business_id = $1`, int64(bakery.ID)).Scan(&left); err != nil {
		t.Fatalf("the exits of the units could not be counted: %v", err)
	}

	if left != 0 {
		t.Errorf("the units were left behind: %d exits of the business", left)
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

// TestTheExitOfAProductThatIsNotThereIsRefused proves the rule of the
// table against a real database: the exit names a product of the business,
// and a product that is not there is refused.
func TestTheExitOfAProductThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := exitRepository(t)
	bakery := theBakeryOfTheExit(t, ctx)

	exit, err := recipe.NewExit(bakery.ID, recipe.ExitFact{
		RecipeID:      recipe.ID(9999),
		RecipeName:    "Pan de molde",
		Kind:          recipe.ExitConsumption,
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)),
		Quantity:      3,
		Unit:          "unidad",
		Cost:          3000,
		TaxTreatment:  recipe.TaxPendingTheProfessional,
		TaxQuestion:   "la pregunta de los impuestos",
		ConfirmedWith: "sí",
		Units: []recipe.ExitedUnit{
			{UnitID: recipe.UnitID(1), ProductionID: recipe.ProductionID(1), Position: 1, Grams: 90},
		},
	})
	if err != nil {
		t.Fatalf("the exit of the test was refused: %v", err)
	}

	_, err = repo.Record(ctx, exit)
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}
}
