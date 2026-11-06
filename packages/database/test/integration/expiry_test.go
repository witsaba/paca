//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/user"
)

// shelfLifeRepository returns the repository of the corrections of the shelf
// life, bound to the database of the suite.
func shelfLifeRepository(t *testing.T) *postgres.ShelfLifeRepository {
	t.Helper()

	repo, err := postgres.NewShelfLifeRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the corrections could not be created: %v", err)
	}

	return repo
}

// unitRepository returns the repository of the units of the productions,
// bound to the database of the suite.
func unitRepository(t *testing.T) *postgres.UnitRepository {
	t.Helper()

	repo, err := postgres.NewUnitRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the units could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheExpiry opens the business that owns the corrections of the
// test.
func theBakeryOfTheExpiry(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Expiry")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería Del Expiry"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theCorrectionOfTheTest is the correction the owner declared: five days
// instead of the three the recipe declares.
func theCorrectionOfTheTest(bakery business.Business, baked recipe.Recipe) recipe.ShelfLifeCorrection {
	correction, err := recipe.CorrectShelfLife(bakery.ID, baked, recipe.ShelfLifeDefinition{
		RecipeID:      baked.ID,
		ShelfLifeDays: 5,
		Day:           ledger.NewDate(time.Date(2026, time.March, 16, 8, 0, 0, 0, time.UTC)),
		Utterance:     "el pan de molde dura cinco dias",
		ConfirmedWith: "sí",
	})
	if err != nil {
		panic("the correction of the test was refused: " + err.Error())
	}

	return correction
}

// TestTheCorrectionOfTheShelfLifeIsKeptWithItsRecipeAndItsDay proves the
// rules of the table against a real database: the correction is written with
// the recipe it corrects, the number of days, the day it was declared and
// the word of the owner, and it is read back as it was written.
func TestTheCorrectionOfTheShelfLifeIsKeptWithItsRecipeAndItsDay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := shelfLifeRepository(t)
	bakery := theBakeryOfTheExpiry(t, ctx)
	flour, _ := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	written, err := repo.Record(ctx, theCorrectionOfTheTest(bakery, baked))
	if err != nil {
		t.Fatalf("the correction could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the correction has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the correction is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.RecipeID != baked.ID {
		t.Errorf("the recipe of the correction is %d, want %d", written.RecipeID, baked.ID)
	}
	if written.RecipeName != "Pan de molde" {
		t.Errorf("the name of the recipe is %q, want %q", written.RecipeName, "Pan de molde")
	}
	if written.ShelfLifeDays != 5 {
		t.Errorf("the number of days is %d, want 5", written.ShelfLifeDays)
	}
	if written.Day.String() != "2026-03-16" {
		t.Errorf("the day of the correction is %q, want %q", written.Day.String(), "2026-03-16")
	}
	if written.Utterance != "el pan de molde dura cinco dias" {
		t.Errorf("the sentence of the correction is %q, want the one of the owner", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}

	read, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the corrections could not be read: %v", err)
	}

	if len(read) != 1 {
		t.Fatalf("the business holds %d corrections, want 1", len(read))
	}

	if read[0].ShelfLifeDays != 5 {
		t.Errorf("the number of days of the correction is %d, want 5", read[0].ShelfLifeDays)
	}
	if read[0].Day.String() != "2026-03-16" {
		t.Errorf("the day of the correction is %q, want %q", read[0].Day.String(), "2026-03-16")
	}
}

// TestTheCorrectionsAreReadInTheOrderOfTheDayTheyWereDeclared proves the
// rule of the table against a real database: the corrections are read in the
// order of the day they were declared, so that the last one the owner
// declared is the one that counts.
func TestTheCorrectionsAreReadInTheOrderOfTheDayTheyWereDeclared(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := shelfLifeRepository(t)
	bakery := theBakeryOfTheExpiry(t, ctx)
	flour, _ := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	older := theCorrectionOfTheTest(bakery, baked)
	older.ShelfLifeDays = 4
	older.Day = ledger.NewDate(time.Date(2026, time.March, 10, 8, 0, 0, 0, time.UTC))

	newer := theCorrectionOfTheTest(bakery, baked)
	newer.ShelfLifeDays = 6
	newer.Day = ledger.NewDate(time.Date(2026, time.March, 18, 8, 0, 0, 0, time.UTC))

	if _, err := repo.Record(ctx, newer); err != nil {
		t.Fatalf("the correction could not be written: %v", err)
	}

	if _, err := repo.Record(ctx, older); err != nil {
		t.Fatalf("the correction could not be written: %v", err)
	}

	read, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the corrections could not be read: %v", err)
	}

	if len(read) != 2 {
		t.Fatalf("the business holds %d corrections, want 2", len(read))
	}

	if read[0].Day.String() != "2026-03-10" {
		t.Errorf("the first correction is the one of %q, want the one of 2026-03-10", read[0].Day.String())
	}

	if read[1].Day.String() != "2026-03-18" {
		t.Errorf("the second correction is the one of %q, want the one of 2026-03-18", read[1].Day.String())
	}
}

// TestTheCorrectionOfARecipeThatIsNotThereIsRefused proves the rule of the
// table against a real database: the correction names a recipe of the
// business, and a recipe that is not there is refused.
func TestTheCorrectionOfARecipeThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := shelfLifeRepository(t)
	bakery := theBakeryOfTheExpiry(t, ctx)

	correction, err := recipe.CorrectShelfLife(bakery.ID, recipe.Recipe{
		ID:         recipe.ID(9999),
		BusinessID: bakery.ID,
		Name:       "Pan de molde",
	}, recipe.ShelfLifeDefinition{
		RecipeID:      9999,
		ShelfLifeDays: 5,
		Day:           ledger.NewDate(time.Date(2026, time.March, 16, 8, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the correction of the test was refused: %v", err)
	}

	_, err = repo.Record(ctx, correction)
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}
}

// TestTheUnitsAreReadWithTheProductionAndTheRecipe proves the rules of the
// table against a real database: the units of the productions are read with
// the recipe they were made of and the day they were made, and they are
// read in the order they expire.
func TestTheUnitsAreReadWithTheProductionAndTheRecipe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := unitRepository(t)
	bakery := theBakeryOfTheExpiry(t, ctx)
	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	written, err := productionsRepository(t).Record(ctx, theProductionOfTheTest(t, bakery, baked, layers, flour))
	if err != nil {
		t.Fatalf("the production could not be written: %v", err)
	}

	units, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(units) != 10 {
		t.Fatalf("the business holds %d units, want 10", len(units))
	}

	for index, one := range units {
		if one.RecipeID != baked.ID {
			t.Errorf("the unit %d is of the recipe %d, want %d", index+1, one.RecipeID, baked.ID)
		}
		if one.RecipeName != "Pan de molde" {
			t.Errorf("the name of the recipe of the unit %d is %q, want %q", index+1, one.RecipeName, "Pan de molde")
		}
		if one.ProducedOn.String() != "2026-03-14" {
			t.Errorf("the unit %d was made on %q, want %q", index+1, one.ProducedOn.String(), "2026-03-14")
		}
		if one.ExpiresOn.String() != "2026-03-17" {
			t.Errorf("the unit %d expires on %q, want %q", index+1, one.ExpiresOn.String(), "2026-03-17")
		}
		if one.ProductionID != written.ID {
			t.Errorf("the unit %d comes from the production %d, want %d", index+1, one.ProductionID, written.ID)
		}
	}
}

// TestTheDayOfTheExpiryOfAUnitIsMovedByTheCorrection proves the rules of the
// table against a real database: the day of the expiry is written, and the
// unit that changed under the correction is refused.
func TestTheDayOfTheExpiryOfAUnitIsMovedByTheCorrection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	units := unitRepository(t)
	bakery := theBakeryOfTheExpiry(t, ctx)
	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	if _, err := productionsRepository(t).Record(ctx, theProductionOfTheTest(t, bakery, baked, layers, flour)); err != nil {
		t.Fatalf("the production could not be written: %v", err)
	}

	read, err := units.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(read) == 0 {
		t.Fatal("the business holds no unit, want the units of the production")
	}

	moved := []recipe.CorrectedUnit{
		{
			UnitID: read[0].ID,
			Was:    read[0].ExpiresOn,
			Now:    ledger.NewDate(read[0].ExpiresOn.Time().AddDate(0, 0, 2)),
		},
	}

	if err := units.WriteExpiry(ctx, moved); err != nil {
		t.Fatalf("the day of the expiry could not be written: %v", err)
	}

	again, err := units.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	found := false

	for _, one := range again {
		if one.ID != moved[0].UnitID {
			continue
		}

		found = true

		if one.ExpiresOn.String() != "2026-03-19" {
			t.Errorf("the unit expires on %q, want %q", one.ExpiresOn.String(), "2026-03-19")
		}
	}

	if !found {
		t.Error("the unit that was moved is not in the book of the business")
	}

	// The unit that changed under the correction is refused: the day is
	// written only when the unit still expires on the day the correction
	// read.
	err = units.WriteExpiry(ctx, moved)
	if !errors.Is(err, recipe.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidValue)
	}
}
