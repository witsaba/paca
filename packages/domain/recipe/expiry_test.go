package recipe_test

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theDayOfTheExpiry is the day the production of the test was made.
func theDayOfTheExpiry() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 14, 8, 0, 0, 0, time.UTC))
}

// theRecipeOfTheExpiry is the recipe the units of the test were made of: it
// declares three days of shelf life.
func theRecipeOfTheExpiry(t *testing.T) recipe.Recipe {
	t.Helper()

	built := theRecipeOfTheProduction(t)
	built.ShelfLifeDays = 3

	return built
}

// theCorrection is a correction of the shelf life of the recipe of the test:
// five days instead of the three the recipe declares.
func theCorrection() recipe.ShelfLifeCorrection {
	return recipe.ShelfLifeCorrection{
		ID:            recipe.CorrectionID(1),
		BusinessID:    business.ID(77),
		RecipeID:      recipe.ID(9),
		RecipeName:    "Pan de molde",
		ShelfLifeDays: 5,
		Day:           theDayOfTheExpiry(),
		ConfirmedWith: "sí",
	}
}

// theUnitOfTheExpiry is a unit that came out of a production: it weighs
// ninety grams and it expires the day the shelf life says.
func theUnitOfTheExpiry(id recipe.UnitID, expiresOn ledger.Date) recipe.ExpiringUnit {
	return recipe.ExpiringUnit{
		ProductionUnit: recipe.ProductionUnit{
			ID:        id,
			ExpiresOn: expiresOn,
			Grams:     90,
		},
		RecipeID:   recipe.ID(9),
		RecipeName: "Pan de molde",
		ProducedOn: theDayOfTheExpiry(),
	}
}

func TestTheCorrectionOfTheShelfLifeIsKeptWithTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	correction, err := recipe.CorrectShelfLife(business.ID(77), theRecipeOfTheExpiry(t), recipe.ShelfLifeDefinition{
		RecipeID:      recipe.ID(9),
		ShelfLifeDays: 5,
		Day:           theDayOfTheExpiry(),
		Utterance:     "el pan de molde dura cinco dias",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the correction of the shelf life was refused: %v", err)
	}

	if correction.RecipeID != recipe.ID(9) {
		t.Errorf("the correction names the recipe %d, want %d", correction.RecipeID, recipe.ID(9))
	}

	if correction.ShelfLifeDays != 5 {
		t.Errorf("the correction declares %d days, want 5", correction.ShelfLifeDays)
	}

	if correction.Day != theDayOfTheExpiry() {
		t.Errorf("the correction carries the day %q, want %q", correction.Day.String(), theDayOfTheExpiry().String())
	}

	if correction.ConfirmedWith != "sí" {
		t.Errorf("the correction carries the word %q, want %q", correction.ConfirmedWith, "sí")
	}

	if correction.Utterance != "el pan de molde dura cinco dias" {
		t.Errorf("the correction carries the sentence %q, want the sentence of the owner", correction.Utterance)
	}
}

func TestTheCorrectionOfTheShelfLifeIsRefusedWithoutDays(t *testing.T) {
	t.Parallel()

	_, err := recipe.CorrectShelfLife(business.ID(77), theRecipeOfTheExpiry(t), recipe.ShelfLifeDefinition{
		RecipeID:      recipe.ID(9),
		ShelfLifeDays: 0,
		Day:           theDayOfTheExpiry(),
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingShelfLife) {
		t.Fatalf("the correction without days was refused with %v, want %v", err, recipe.ErrMissingShelfLife)
	}
}

func TestTheCorrectionOfTheShelfLifeIsRefusedWithoutTheDay(t *testing.T) {
	t.Parallel()

	_, err := recipe.CorrectShelfLife(business.ID(77), theRecipeOfTheExpiry(t), recipe.ShelfLifeDefinition{
		RecipeID:      recipe.ID(9),
		ShelfLifeDays: 5,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingDay) {
		t.Fatalf("the correction without a day was refused with %v, want %v", err, recipe.ErrMissingDay)
	}
}

func TestTheCorrectionOfTheShelfLifeIsRefusedWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	_, err := recipe.CorrectShelfLife(business.ID(77), theRecipeOfTheExpiry(t), recipe.ShelfLifeDefinition{
		RecipeID:      recipe.ID(9),
		ShelfLifeDays: 5,
		Day:           theDayOfTheExpiry(),
	})
	if !errors.Is(err, recipe.ErrMissingConfirmation) {
		t.Fatalf("the correction without a word was refused with %v, want %v", err, recipe.ErrMissingConfirmation)
	}
}

func TestTheCorrectionOfTheShelfLifeIsRefusedForAnotherBusiness(t *testing.T) {
	t.Parallel()

	_, err := recipe.CorrectShelfLife(business.ID(88), theRecipeOfTheExpiry(t), recipe.ShelfLifeDefinition{
		RecipeID:      recipe.ID(9),
		ShelfLifeDays: 5,
		Day:           theDayOfTheExpiry(),
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrWrongBusiness) {
		t.Fatalf("the correction for another business was refused with %v, want %v", err, recipe.ErrWrongBusiness)
	}
}

func TestTheShelfLifeThatCountsIsTheOneTheOwnerCorrected(t *testing.T) {
	t.Parallel()

	if got := recipe.ShelfLifeOf(theRecipeOfTheExpiry(t), nil); got != 3 {
		t.Errorf("the shelf life of the recipe is %d days, want the 3 it declares", got)
	}
}

func TestTheLastCorrectionOfTheOwnerIsTheOneThatCounts(t *testing.T) {
	t.Parallel()

	older := theCorrection()
	older.ID = recipe.CorrectionID(1)
	older.ShelfLifeDays = 5
	older.Day = ledger.NewDate(time.Date(2026, time.March, 10, 8, 0, 0, 0, time.UTC))

	newer := theCorrection()
	newer.ID = recipe.CorrectionID(2)
	newer.ShelfLifeDays = 7
	newer.Day = ledger.NewDate(time.Date(2026, time.March, 12, 8, 0, 0, 0, time.UTC))

	if got := recipe.ShelfLifeOf(theRecipeOfTheExpiry(t), []recipe.ShelfLifeCorrection{newer, older}); got != 7 {
		t.Errorf("the shelf life is %d days, want the 7 of the last correction", got)
	}
}

func TestTheCorrectionOfAnotherRecipeDoesNotCount(t *testing.T) {
	t.Parallel()

	other := theCorrection()
	other.ID = recipe.CorrectionID(3)
	other.RecipeID = recipe.ID(10)
	other.ShelfLifeDays = 30

	if got := recipe.ShelfLifeOf(theRecipeOfTheExpiry(t), []recipe.ShelfLifeCorrection{other}); got != 3 {
		t.Errorf("the shelf life is %d days, want the 3 of the recipe: the correction of another recipe does not count", got)
	}
}

func TestTheCorrectionMovesTheExpiryOfTheUnitsOfTheRecipe(t *testing.T) {
	t.Parallel()

	units := []recipe.ExpiringUnit{
		theUnitOfTheExpiry(1, ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC))),
	}

	other := theUnitOfTheExpiry(2, ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)))
	other.RecipeID = recipe.ID(10)
	units = append(units, other)

	corrected, err := recipe.CorrectedUnits(units, recipe.ID(9), 5)
	if err != nil {
		t.Fatalf("the correction of the units was refused: %v", err)
	}

	if len(corrected) != 1 {
		t.Fatalf("the correction moved %d units, want the one of the recipe", len(corrected))
	}

	want := ledger.NewDate(time.Date(2026, time.March, 19, 0, 0, 0, 0, time.UTC))
	if corrected[0].Now != want {
		t.Errorf("the unit expires on %q, want %q", corrected[0].Now.String(), want.String())
	}

	if corrected[0].Was != units[0].ExpiresOn {
		t.Errorf("the unit expired on %q before, want %q", corrected[0].Was.String(), units[0].ExpiresOn.String())
	}
}

func TestTheCorrectionOfTheUnitsIsRefusedWithoutDays(t *testing.T) {
	t.Parallel()

	_, err := recipe.CorrectedUnits([]recipe.ExpiringUnit{theUnitOfTheExpiry(1, theDayOfTheExpiry())}, recipe.ID(9), 0)
	if !errors.Is(err, recipe.ErrMissingShelfLife) {
		t.Fatalf("the correction without days was refused with %v, want %v", err, recipe.ErrMissingShelfLife)
	}
}

func TestAUnitExpiresOnTheFirstDayItIsNoLongerGood(t *testing.T) {
	t.Parallel()

	expires := ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC))
	unit := theUnitOfTheExpiry(1, expires)

	before := ledger.NewDate(time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC))
	if unit.Expired(before) {
		t.Errorf("the unit expired on %q, want it to be good until the day it expires", before.String())
	}

	if !unit.Expired(expires) {
		t.Errorf("the unit did not expire on %q, want it to be expired on the day it expires", expires.String())
	}

	after := ledger.NewDate(time.Date(2026, time.March, 18, 0, 0, 0, 0, time.UTC))
	if !unit.Expired(after) {
		t.Errorf("the unit did not expire on %q, want it to be expired", after.String())
	}
}

func TestTheDaysLeftOfTheUnitAreTheDaysUntilItExpires(t *testing.T) {
	t.Parallel()

	expires := ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC))
	unit := theUnitOfTheExpiry(1, expires)

	produced := ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC))
	if got := unit.DaysLeft(produced); got != 3 {
		t.Errorf("the unit has %d days left, want 3", got)
	}

	if got := unit.DaysLeft(expires); got != 0 {
		t.Errorf("the unit has %d days left, want 0 the day it expires", got)
	}

	after := ledger.NewDate(time.Date(2026, time.March, 18, 0, 0, 0, 0, time.UTC))
	if got := unit.DaysLeft(after); got != 0 {
		t.Errorf("the unit has %d days left, want 0: it expired already", got)
	}
}

func TestAUnitThatExpiredIsNeverSold(t *testing.T) {
	t.Parallel()

	expires := ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC))
	unit := theUnitOfTheExpiry(1, expires)

	good := ledger.NewDate(time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC))
	if err := unit.CheckSellable(good); err != nil {
		t.Fatalf("the unit that did not expire was refused: %v", err)
	}

	err := unit.CheckSellable(expires)
	if !errors.Is(err, recipe.ErrExpiredUnit) {
		t.Fatalf("the sale of the expired unit was refused with %v, want %v", err, recipe.ErrExpiredUnit)
	}
}

func TestTheSaleOfNoDayRefusesTheUnit(t *testing.T) {
	t.Parallel()

	unit := theUnitOfTheExpiry(1, theDayOfTheExpiry())

	err := unit.CheckSellable(ledger.Date{})
	if !errors.Is(err, recipe.ErrMissingSaleDay) {
		t.Fatalf("the sale of no day was refused with %v, want %v", err, recipe.ErrMissingSaleDay)
	}
}

func TestTheUnitsAreOrderedByTheDayTheyExpire(t *testing.T) {
	t.Parallel()

	late := theUnitOfTheExpiry(2, ledger.NewDate(time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC)))
	early := theUnitOfTheExpiry(1, ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)))

	ordered := recipe.ExpiringUnits([]recipe.ExpiringUnit{late, early})
	if len(ordered) != 2 {
		t.Fatalf("the answer carries %d units, want 2", len(ordered))
	}

	if ordered[0].ID != early.ID {
		t.Errorf("the first unit is the %d, want the %d that expires first", ordered[0].ID, early.ID)
	}

	if ordered[1].ID != late.ID {
		t.Errorf("the second unit is the %d, want the %d", ordered[1].ID, late.ID)
	}
}
