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

// wasteRepository returns the repository of the wastes, bound to the
// database of the suite.
func wasteRepository(t *testing.T) *postgres.WasteRepository {
	t.Helper()

	repo, err := postgres.NewWasteRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the wastes could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheWaste opens the business that owns the wastes of the test.
func theBakeryOfTheWaste(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Waste")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería Del Waste"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theWasteOfTheTest derives the waste the tests of the table write: three
// units of the product that were burnt.
func theWasteOfTheTest(
	t *testing.T,
	bakery business.Business,
	baked recipe.Recipe,
	production recipe.Production,
	units []recipe.ExpiringUnit,
) recipe.Waste {
	t.Helper()

	waste, err := recipe.WasteOf(bakery.ID, []recipe.Production{production}, units, recipe.WasteDefinition{
		RecipeID:      baked.ID,
		RecipeName:    baked.Name,
		Quantity:      3,
		Reason:        recipe.WasteBurnt,
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)),
		Utterance:     "se me quemaron tres panes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the waste of the test was refused: %v", err)
	}

	return waste
}

// theBookOfTheWaste writes the stock, the recipe and the production the
// waste of the test consumes.
func theBookOfTheWaste(
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

// TestTheWasteIsKeptWithItsProductAndItsReason proves the rules of the
// table against a real database: the waste is written with the product, the
// quantity, the unit, the reason, the day it happened and the word of the
// owner, and it is read back as it was written.
func TestTheWasteIsKeptWithItsProductAndItsReason(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := wasteRepository(t)
	bakery := theBakeryOfTheWaste(t, ctx)
	baked, production, units := theBookOfTheWaste(t, ctx, bakery)

	written, err := repo.Record(ctx, theWasteOfTheTest(t, bakery, baked, production, units))
	if err != nil {
		t.Fatalf("the waste could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the waste has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the waste is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.RecipeID != baked.ID {
		t.Errorf("the product of the waste is %d, want %d", written.RecipeID, baked.ID)
	}
	if written.RecipeName != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", written.RecipeName, "Pan de molde")
	}
	if written.Quantity != 3 {
		t.Errorf("the quantity of the waste is %v, want 3", written.Quantity)
	}
	if written.Unit != "unidad" {
		t.Errorf("the unit of the waste is %q, want %q", written.Unit, "unidad")
	}
	if written.Reason != recipe.WasteBurnt {
		t.Errorf("the reason of the waste is %q, want %q", written.Reason, recipe.WasteBurnt)
	}
	if written.Day.String() != "2026-03-14" {
		t.Errorf("the day of the waste is %q, want %q", written.Day.String(), "2026-03-14")
	}
	if written.Utterance != "se me quemaron tres panes" {
		t.Errorf("the sentence of the waste is %q, want the one of the owner", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if written.Money <= 0 {
		t.Errorf("the money of the waste is %d, want the cost it releases", written.Money)
	}
	if len(written.Units) != 3 {
		t.Errorf("the waste took %d units, want 3", len(written.Units))
	}
}

// TestTheUnitsThatLeftTheBookAreNotInTheBookAnyMore proves the rule of the
// table against a real database: a unit that left the book is never taken
// again.
func TestTheUnitsThatLeftTheBookAreNotInTheBookAnyMore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := wasteRepository(t)
	bakery := theBakeryOfTheWaste(t, ctx)
	baked, production, units := theBookOfTheWaste(t, ctx, bakery)

	before, err := unitRepository(t).ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(before) != 10 {
		t.Fatalf("the book holds %d units, want 10", len(before))
	}

	if _, err := repo.Record(ctx, theWasteOfTheTest(t, bakery, baked, production, units)); err != nil {
		t.Fatalf("the waste could not be written: %v", err)
	}

	after, err := unitRepository(t).ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(after) != 7 {
		t.Fatalf("the book holds %d units, want the 7 that did not leave", len(after))
	}
}

// TestTheUnitThatLeftTheBookIsRefused proves the rule of the table against a
// real database: a unit leaves the book only once, and the second waste of
// it is refused.
func TestTheUnitThatLeftTheBookIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := wasteRepository(t)
	bakery := theBakeryOfTheWaste(t, ctx)
	baked, production, units := theBookOfTheWaste(t, ctx, bakery)

	waste := theWasteOfTheTest(t, bakery, baked, production, units)

	if _, err := repo.Record(ctx, waste); err != nil {
		t.Fatalf("the waste could not be written: %v", err)
	}

	_, err := repo.Record(ctx, waste)
	if !errors.Is(err, recipe.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidValue)
	}
}

// TestTheWasteOfTwoProductionsOfTheSameProduct proves the rule of the
// table against a real database: a waste may take the units of more than
// one production of the same product, and the position of a unit is the
// position it had in its own production.
func TestTheWasteOfTwoProductionsOfTheSameProduct(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := wasteRepository(t)
	bakery := theBakeryOfTheWaste(t, ctx)
	baked, production, units := theBookOfTheWaste(t, ctx, bakery)

	second, err := recipe.NewProduction(bakery.ID, recipe.ProductionFact{
		RecipeID:      baked.ID,
		RecipeName:    baked.Name,
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 8, 0, 0, 0, time.UTC)),
		GramsIn:       500,
		Money:         240,
		Units:         theUnitsOfTheSecondProduction(),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the second production was refused: %v", err)
	}

	secondWritten, err := productionsRepository(t).Record(ctx, second)
	if err != nil {
		t.Fatalf("the second production could not be written: %v", err)
	}

	all := append(units, unitsOfTheSecondProduction(t, secondWritten)...)

	waste, err := recipe.WasteOf(bakery.ID, []recipe.Production{production, secondWritten}, all, recipe.WasteDefinition{
		RecipeID:      baked.ID,
		RecipeName:    baked.Name,
		Quantity:      12,
		Reason:        recipe.WasteSpoiled,
		Day:           ledger.NewDate(time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the waste was refused: %v", err)
	}

	if len(waste.Units) != 12 {
		t.Fatalf("the waste took %d units, want 12", len(waste.Units))
	}

	recorded, err := repo.Record(ctx, waste)
	if err != nil {
		t.Fatalf("the waste could not be written: %v", err)
	}

	if len(recorded.Units) != 12 {
		t.Errorf("the waste holds %d units, want 12", len(recorded.Units))
	}
}

// theUnitsOfTheSecondProduction are the units of the second production of
// the test: ten units of ninety grams that expire three days later.
func theUnitsOfTheSecondProduction() []recipe.ProductionUnit {
	units := make([]recipe.ProductionUnit, 0, 10)

	for position := 1; position <= 10; position++ {
		units = append(units, recipe.ProductionUnit{
			Position:  position,
			Grams:     90,
			ExpiresOn: ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)),
		})
	}

	return units
}

// unitsOfTheSecondProduction answers the units of the production of the
// test that are still in the book.
func unitsOfTheSecondProduction(t *testing.T, production recipe.Production) []recipe.ExpiringUnit {
	t.Helper()

	units := make([]recipe.ExpiringUnit, 0, len(production.Units))

	for _, one := range production.Units {
		units = append(units, recipe.ExpiringUnit{
			ProductionUnit: one,
			RecipeID:       production.RecipeID,
			RecipeName:     production.RecipeName,
			ProducedOn:     production.Day,
		})
	}

	return units
}

// TestTheWasteAndItsUnitsAndItsMovementAreWrittenWhole proves the rule of
// the transaction against a real database: the waste, the units that left
// the book and the movement of its money are written whole or not at all,
// and the storage is as it was when the work failed.
func TestTheWasteAndItsUnitsAndItsMovementAreWrittenWhole(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wastes := wasteRepository(t)
	book := ledgerRepository(t)
	bakery := theBakeryOfTheWaste(t, ctx)
	baked, production, units := theBookOfTheWaste(t, ctx, bakery)

	waste := theWasteOfTheTest(t, bakery, baked, production, units)

	accounts := theAccountsOfTheBook(t, ctx, bakery, account.RoleWaste, account.RoleInventory)

	movement, err := waste.Movement(accounts[account.RoleWaste].ID, accounts[account.RoleInventory].ID)
	if err != nil {
		t.Fatalf("the movement of the money of the waste was refused: %v", err)
	}

	err = unitOfWork(t).Do(ctx, func(ctx context.Context) error {
		if _, err := wastes.Record(ctx, waste); err != nil {
			return err
		}

		if _, err := book.Record(ctx, movement); err != nil {
			return err
		}

		// The work fails after the waste and its money were written: the
		// work goes back with them.
		return errors.New("the money of the waste did not move")
	})
	if err == nil {
		t.Fatal("the work was committed, want the failure of the work")
	}

	var lost int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM wastes WHERE business_id = $1`, int64(bakery.ID)).Scan(&lost); err != nil {
		t.Fatalf("the wastes could not be counted: %v", err)
	}

	if lost != 0 {
		t.Errorf("the waste was left behind: %d wastes of the business", lost)
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

// TestTheWasteOfAProductThatIsNotThereIsRefused proves the rule of the
// table against a real database: the waste names a product of the business,
// and a product that is not there is refused.
func TestTheWasteOfAProductThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := wasteRepository(t)
	bakery := theBakeryOfTheWaste(t, ctx)

	waste, err := recipe.NewWaste(bakery.ID, recipe.WasteFact{
		RecipeID:   recipe.ID(9999),
		RecipeName: "Pan de molde",
		Day:        ledger.NewDate(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)),
		Quantity:   3,
		Unit:       "unidad",
		Reason:     recipe.WasteBurnt,
		Money:      72,
		Units: []recipe.WastedUnit{
			{UnitID: recipe.UnitID(1), ProductionID: recipe.ProductionID(1), Position: 1, Grams: 90},
		},
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the waste of the test was refused: %v", err)
	}

	_, err = repo.Record(ctx, waste)
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}
}
