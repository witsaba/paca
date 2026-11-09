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

// The fake of the storage of the units that are still in the book, for the
// rotation of the test.
//
// It holds the units of the business, and it remembers the business the
// caller asked about.
type fakeRotationUnits struct {
	held   []recipe.ExpiringUnit
	listed business.ID
	err    error
}

// ListInTheBook answers the units of the business that are still in it.
func (f *fakeRotationUnits) ListInTheBook(_ context.Context, own business.ID) ([]recipe.ExpiringUnit, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	return f.held, nil
}

// A unit of the book of the rotation of the test.
//
// It was made on the fourteenth of March, and it expires on the day the
// test names.
func aUnitOfTheRotation(
	id int64,
	production int64,
	position int,
	of recipe.ID,
	expires time.Time,
) recipe.ExpiringUnit {
	return recipe.ExpiringUnit{
		ProductionUnit: recipe.ProductionUnit{
			ID:           recipe.UnitID(id),
			ProductionID: recipe.ProductionID(production),
			Position:     position,
			Grams:        90,
			ExpiresOn:    ledger.NewDate(expires),
		},
		RecipeID:   of,
		RecipeName: "Pan de molde",
		ProducedOn: ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
	}
}

// theRotationService builds the use case of the rotation with the fakes of
// the test.
func theRotationService(t *testing.T, units *fakeRotationUnits, recipes *fakeRecipes) *application.RotationService {
	t.Helper()

	return application.NewRotationService(units, recipes, theClock())
}

// TestTheRotationIsReadFromTheUnitsThatAreStillInTheBook proves that the
// use case reads the units of the book and hands the rotation to the
// adapter: the unit that expires first is the first one that leaves.
func TestTheRotationIsReadFromTheUnitsThatAreStillInTheBook(t *testing.T) {
	t.Parallel()

	late := aUnitOfTheRotation(2, 3, 2, recipe.ID(9), time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC))
	early := aUnitOfTheRotation(1, 3, 1, recipe.ID(9), time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC))

	units := &fakeRotationUnits{held: []recipe.ExpiringUnit{late, early}}
	service := theRotationService(t, units, &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheProduction(t)}})

	answer, err := service.Read(t.Context(), ports.ReadRotation{
		BusinessID: business.ID(77),
	})
	if err != nil {
		t.Fatalf("the rotation was refused: %v", err)
	}

	if units.listed != business.ID(77) {
		t.Errorf("the units of the business %d were read, want the %d", units.listed, business.ID(77))
	}

	if answer.Day.String() != "2026-03-15" {
		t.Errorf("the answer carries the day %q, want the day of today", answer.Day.String())
	}

	if len(answer.Units) != 2 {
		t.Fatalf("the answer carries %d units, want 2", len(answer.Units))
	}

	if answer.Units[0].ExpiresOn.String() != "2026-03-17" {
		t.Errorf("the first unit expires on %q, want the one that expires first",
			answer.Units[0].ExpiresOn.String())
	}

	if answer.Units[0].DaysLeft != 2 {
		t.Errorf("the first unit has %d days left, want 2", answer.Units[0].DaysLeft)
	}
}

// TestTheRotationOfOneProductIsTheRotationOfThatProduct proves that the
// use case answers the rotation of the product the owner asked about: the
// agent names the product, and the answer is the rotation of that product.
func TestTheRotationOfOneProductIsTheRotationOfThatProduct(t *testing.T) {
	t.Parallel()

	bread := aUnitOfTheRotation(1, 3, 1, recipe.ID(9), time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC))
	cake := aUnitOfTheRotation(2, 4, 1, recipe.ID(10), time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC))

	other := theRecipeOfTheProduction(t)
	other.ID = recipe.ID(10)
	other.Name = "Pastel"

	units := &fakeRotationUnits{held: []recipe.ExpiringUnit{bread, cake}}
	service := theRotationService(t, units, &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheProduction(t), other}})

	answer, err := service.Read(t.Context(), ports.ReadRotation{
		BusinessID: business.ID(77),
		RecipeName: "Pastel",
	})
	if err != nil {
		t.Fatalf("the rotation was refused: %v", err)
	}

	if answer.RecipeID != recipe.ID(10) {
		t.Errorf("the product of the rotation is %d, want %d", answer.RecipeID, recipe.ID(10))
	}

	if len(answer.Units) != 1 {
		t.Fatalf("the answer carries %d units, want the one of the product", len(answer.Units))
	}

	if answer.Units[0].RecipeID != recipe.ID(10) {
		t.Errorf("the unit of the rotation is of the product %d, want %d",
			answer.Units[0].RecipeID, recipe.ID(10))
	}
}

// TestTheRotationIsRefusedWhenTheNameCarriesMoreThanOneProduct proves the
// rule of the use case: the book never guesses which product the owner
// asked about.
func TestTheRotationIsRefusedWhenTheNameCarriesMoreThanOneProduct(t *testing.T) {
	t.Parallel()

	other := theRecipeOfTheProduction(t)
	other.ID = recipe.ID(10)

	units := &fakeRotationUnits{}
	service := theRotationService(t, units, &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheProduction(t), other}})

	_, err := service.Read(t.Context(), ports.ReadRotation{
		BusinessID: business.ID(77),
		RecipeName: "Pan de molde",
	})
	if !errors.Is(err, recipe.ErrAmbiguousRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrAmbiguousRecipe)
	}
}

// TestTheRotationIsRefusedWhenTheProductIsNotThere proves the rule of the
// use case: the rotation of a product the business does not hold is
// refused, and the book never guesses it.
func TestTheRotationIsRefusedWhenTheProductIsNotThere(t *testing.T) {
	t.Parallel()

	units := &fakeRotationUnits{}
	service := theRotationService(t, units, &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheProduction(t)}})

	_, err := service.Read(t.Context(), ports.ReadRotation{
		BusinessID: business.ID(77),
		RecipeName: "Pastel",
	})
	if !errors.Is(err, recipe.ErrUnknownRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownRecipe)
	}
}

// TestTheRotationIsRefusedWithoutTheBusiness proves the rule of the use
// case: the rotation belongs to a business, and the use case refuses the
// business that is not there.
func TestTheRotationIsRefusedWithoutTheBusiness(t *testing.T) {
	t.Parallel()

	units := &fakeRotationUnits{}
	service := theRotationService(t, units, &fakeRecipes{held: []recipe.Recipe{theRecipeOfTheProduction(t)}})

	_, err := service.Read(t.Context(), ports.ReadRotation{
		BusinessID: business.ID(0),
	})
	if !errors.Is(err, recipe.ErrMissingBusiness) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingBusiness)
	}
}

// TestTheRotationWithoutADatabaseSaysSo proves that the use case with no
// storage behind it says that the database is not there, and it never
// answers a rotation of nobody.
func TestTheRotationWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewRotationService(nil, nil, nil)

	_, err := service.Read(t.Context(), ports.ReadRotation{
		BusinessID: business.ID(77),
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
