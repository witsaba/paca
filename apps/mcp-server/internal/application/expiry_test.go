package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The fake of the storage of the corrections of the shelf life.
//
// It remembers the correction it wrote.
type fakeShelfLifeCorrections struct {
	written []recipe.ShelfLifeCorrection
	err     error
}

// Record writes the correction and remembers it.
func (f *fakeShelfLifeCorrections) Record(_ context.Context, written recipe.ShelfLifeCorrection) (recipe.ShelfLifeCorrection, error) {
	if f.err != nil {
		return recipe.ShelfLifeCorrection{}, f.err
	}

	written.ID = recipe.CorrectionID(1)
	f.written = append(f.written, written)

	return written, nil
}

// The fake of the storage of the units of the productions.
//
// It holds the units of the business, and it remembers the days the
// correction moved.
type fakeUnits struct {
	held      []recipe.ExpiringUnit
	corrected []recipe.CorrectedUnit
	err       error
	writeErr  error
}

// List answers the units of the business.
func (f *fakeUnits) List(_ context.Context, _ business.ID) ([]recipe.ExpiringUnit, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.held, nil
}

// WriteExpiry remembers the days the correction moved.
//
// The failure of the write is its own: the list of the units may answer
// while the write of the days fails.
func (f *fakeUnits) WriteExpiry(_ context.Context, units []recipe.CorrectedUnit) error {
	if f.writeErr != nil {
		return f.writeErr
	}

	if f.err != nil {
		return f.err
	}

	f.corrected = append(f.corrected, units...)

	return nil
}

// The fake of the transaction of the storage of the expiry.
//
// It keeps what the work wrote, and it gives it back when the work fails:
// a transaction that rolls back leaves the storage as it was.
type fakeExpiryWork struct {
	corrections *fakeShelfLifeCorrections
	units       *fakeUnits
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakeExpiryWork) Do(ctx context.Context, work func(context.Context) error) error {
	corrections := len(f.corrections.written)
	units := len(f.units.corrected)

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.corrections.written = f.corrections.written[:corrections]
	f.units.corrected = f.units.corrected[:units]

	return err
}

// theUnitOfTheExpiry is a unit of a production of the recipe of the test:
// it was made on the fourteenth of March and it expires three days later.
func theUnitOfTheExpiry() recipe.ExpiringUnit {
	return recipe.ExpiringUnit{
		ProductionUnit: recipe.ProductionUnit{
			ID:           recipe.UnitID(1),
			ProductionID: recipe.ProductionID(3),
			Position:     1,
			Grams:        90,
			ExpiresOn:    ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)),
		},
		RecipeID:   recipe.ID(9),
		RecipeName: "Pan de molde",
		ProducedOn: ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
	}
}

// theExpiryService builds the use case of the expiry with the fakes of the
// test.
func theExpiryService(
	t *testing.T,
	corrections *fakeShelfLifeCorrections,
	units *fakeUnits,
) *application.ExpiryService {
	t.Helper()

	return application.NewExpiryService(
		&fakeRecipes{held: []recipe.Recipe{theRecipeOfTheProduction(t)}},
		corrections,
		units,
		&fakeExpiryWork{corrections: corrections, units: units},
		theClock(),
	)
}

// TestTheCorrectionIsWrittenWithTheDaysTheDayAndTheWord proves that the
// use case writes the correction the owner confirmed: the recipe she
// corrected, the number of days, the day it was declared and the word she
// used.
func TestTheCorrectionIsWrittenWithTheDaysTheDayAndTheWord(t *testing.T) {
	t.Parallel()

	corrections := &fakeShelfLifeCorrections{}
	units := &fakeUnits{held: []recipe.ExpiringUnit{theUnitOfTheExpiry()}}
	service := theExpiryService(t, corrections, units)

	answer, err := service.Correct(t.Context(), ports.CorrectShelfLife{
		BusinessID:    business.ID(77),
		RecipeID:      9,
		ShelfLifeDays: 5,
		Day:           "2026-03-16",
		Utterance:     "el pan de molde dura cinco dias",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the correction of the shelf life was refused: %v", err)
	}

	if len(corrections.written) != 1 {
		t.Fatalf("the use case wrote %d corrections, want 1", len(corrections.written))
	}

	written := corrections.written[0]
	if written.ShelfLifeDays != 5 {
		t.Errorf("the correction declares %d days, want 5", written.ShelfLifeDays)
	}

	if written.Day.String() != "2026-03-16" {
		t.Errorf("the correction carries the day %q, want %q", written.Day.String(), "2026-03-16")
	}

	if written.ConfirmedWith != "sí" {
		t.Errorf("the correction carries the word %q, want %q", written.ConfirmedWith, "sí")
	}

	if answer.Correction.ShelfLifeDays != 5 {
		t.Errorf("the answer carries %d days, want 5", answer.Correction.ShelfLifeDays)
	}
}

// TestTheCorrectionMovesTheExpiryOfTheUnitsOfTheRecipe proves that the use
// case moves the day of the units that were already made: the unit expires
// again, the day it was made plus the days the owner corrected.
func TestTheCorrectionMovesTheExpiryOfTheUnitsOfTheRecipe(t *testing.T) {
	t.Parallel()

	corrections := &fakeShelfLifeCorrections{}
	units := &fakeUnits{held: []recipe.ExpiringUnit{theUnitOfTheExpiry()}}
	service := theExpiryService(t, corrections, units)

	answer, err := service.Correct(t.Context(), ports.CorrectShelfLife{
		BusinessID:    business.ID(77),
		RecipeID:      9,
		ShelfLifeDays: 5,
		Day:           "2026-03-16",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the correction of the shelf life was refused: %v", err)
	}

	if len(units.corrected) != 1 {
		t.Fatalf("the use case moved %d units, want 1", len(units.corrected))
	}

	if units.corrected[0].Now.String() != "2026-03-19" {
		t.Errorf("the unit expires on %q, want %q", units.corrected[0].Now.String(), "2026-03-19")
	}

	if units.corrected[0].Was.String() != "2026-03-17" {
		t.Errorf("the unit expired on %q before, want %q", units.corrected[0].Was.String(), "2026-03-17")
	}

	if len(answer.Units) != 1 {
		t.Fatalf("the answer carries %d units, want 1", len(answer.Units))
	}
}

// TestTheCorrectionIsRefusedWithoutTheWordOfTheOwner proves that the use
// case writes nothing when the correction carries no word: the correction is
// written with the confirmation of the owner.
func TestTheCorrectionIsRefusedWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	corrections := &fakeShelfLifeCorrections{}
	units := &fakeUnits{held: []recipe.ExpiringUnit{theUnitOfTheExpiry()}}
	service := theExpiryService(t, corrections, units)

	_, err := service.Correct(t.Context(), ports.CorrectShelfLife{
		BusinessID:    business.ID(77),
		RecipeID:      9,
		ShelfLifeDays: 5,
		Day:           "2026-03-16",
	})
	if !errors.Is(err, recipe.ErrMissingConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingConfirmation)
	}

	if len(corrections.written) != 0 {
		t.Errorf("the use case wrote %d corrections, want none", len(corrections.written))
	}

	if len(units.corrected) != 0 {
		t.Errorf("the use case moved %d units, want none", len(units.corrected))
	}
}

// TestTheCorrectionIsRefusedWhenTheNameCarriesMoreThanOneRecipe proves the
// rule of the use case: the book never guesses which recipe the owner
// corrected.
func TestTheCorrectionIsRefusedWhenTheNameCarriesMoreThanOneRecipe(t *testing.T) {
	t.Parallel()

	other := theRecipeOfTheProduction(t)
	other.ID = recipe.ID(10)

	corrections := &fakeShelfLifeCorrections{}
	units := &fakeUnits{}
	service := application.NewExpiryService(
		&fakeRecipes{held: []recipe.Recipe{theRecipeOfTheProduction(t), other}},
		corrections,
		units,
		&fakeExpiryWork{corrections: corrections, units: units},
		theClock(),
	)

	_, err := service.Correct(t.Context(), ports.CorrectShelfLife{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		ShelfLifeDays: 5,
		Day:           "2026-03-16",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrAmbiguousRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrAmbiguousRecipe)
	}

	if len(corrections.written) != 0 {
		t.Errorf("the use case wrote %d corrections, want none", len(corrections.written))
	}
}

// TestTheCorrectionIsNotWrittenWhenTheExpiryFails proves that a failure
// inside the unit of work leaves no correction behind: the correction and
// the days of the units are written whole or not at all, and the error of
// the failure is the error of the use case.
func TestTheCorrectionIsNotWrittenWhenTheExpiryFails(t *testing.T) {
	t.Parallel()

	failure := errors.New("the storage refused the days of the units")

	corrections := &fakeShelfLifeCorrections{}
	units := &fakeUnits{
		held:     []recipe.ExpiringUnit{theUnitOfTheExpiry()},
		writeErr: failure,
	}
	service := theExpiryService(t, corrections, units)

	_, err := service.Correct(t.Context(), ports.CorrectShelfLife{
		BusinessID:    business.ID(77),
		RecipeID:      9,
		ShelfLifeDays: 5,
		Day:           "2026-03-16",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, failure) {
		t.Fatalf("the error is %v, want %v", err, failure)
	}

	if len(corrections.written) != 0 {
		t.Errorf("the use case wrote %d corrections, want none: the correction and the days of the "+
			"units are written whole or not at all", len(corrections.written))
	}

	if len(units.corrected) != 0 {
		t.Errorf("the use case moved %d units, want none", len(units.corrected))
	}
}

// TestTheCorrectionThatMovedNoUnitIsWritten proves that the correction the
// owner declared is kept even when it moved no unit: the correction is the
// fact she declared, and the last one she declared is the shelf life that
// counts.
func TestTheCorrectionThatMovedNoUnitIsWritten(t *testing.T) {
	t.Parallel()

	corrections := &fakeShelfLifeCorrections{}
	units := &fakeUnits{}
	service := theExpiryService(t, corrections, units)

	answer, err := service.Correct(t.Context(), ports.CorrectShelfLife{
		BusinessID:    business.ID(77),
		RecipeID:      9,
		ShelfLifeDays: 5,
		Day:           "2026-03-16",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the correction of the shelf life was refused: %v", err)
	}

	if len(corrections.written) != 1 {
		t.Fatalf("the use case wrote %d corrections, want 1: the correction is kept", len(corrections.written))
	}

	if len(answer.Units) != 0 {
		t.Errorf("the answer carries %d units, want none", len(answer.Units))
	}
}

// TestTheReadAnswersWhatExpiresAndWhen proves that the use case answers the
// units of the business with the day they expire, the first one the one that
// expires first.
func TestTheReadAnswersWhatExpiresAndWhen(t *testing.T) {
	t.Parallel()

	late := theUnitOfTheExpiry()
	late.ID = recipe.UnitID(2)
	late.ExpiresOn = ledger.NewDate(time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC))

	corrections := &fakeShelfLifeCorrections{}
	units := &fakeUnits{held: []recipe.ExpiringUnit{late, theUnitOfTheExpiry()}}
	service := theExpiryService(t, corrections, units)

	answer, err := service.Read(t.Context(), ports.ReadExpiring{
		BusinessID: business.ID(77),
	})
	if err != nil {
		t.Fatalf("the read of what expires was refused: %v", err)
	}

	if answer.Day.String() != "2026-03-15" {
		t.Errorf("the answer carries the day %q, want the day of today", answer.Day.String())
	}

	if len(answer.Units) != 2 {
		t.Fatalf("the answer carries %d units, want 2", len(answer.Units))
	}

	if answer.Units[0].ExpiresOn.String() != "2026-03-17" {
		t.Errorf("the first unit expires on %q, want the one that expires first", answer.Units[0].ExpiresOn.String())
	}
}
