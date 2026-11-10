//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theRotationRepository returns the repository of the units of the
// productions, bound to the database of the suite.
//
// The rotation reads the units that are still in the book, and the
// repository is the reader of them: there is no query of the rotation, and
// the book never reads a unit that left it.
func theRotationRepository(t *testing.T) *postgres.UnitRepository {
	t.Helper()

	repo, err := postgres.NewUnitRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the units could not be created: %v", err)
	}

	return repo
}

// theRecipeOfTheRotation writes the recipe of the rotation of the test.
//
// The shelf life is the one the test names: the units of one recipe expire
// before the units of the other one, and the rotation is the order they
// leave the book.
func theRecipeOfTheRotation(
	t *testing.T,
	ctx context.Context,
	bakery business.Business,
	flour ingredient.ID,
	name string,
	shelfLifeDays int,
) recipe.Recipe {
	t.Helper()

	built, err := recipe.New(bakery.ID, recipe.Definition{
		Name: name,
		Ingredients: []recipe.Ingredient{
			{IngredientID: flour, Grams: 500},
		},
		Steps: []recipe.Step{
			{Instruction: "Hornear."},
		},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: shelfLifeDays,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the recipe of the rotation was refused: %v", err)
	}

	written, err := recipesRepository(t).Record(ctx, built)
	if err != nil {
		t.Fatalf("the recipe of the rotation could not be written: %v", err)
	}

	return written
}

// theWasteOfTheRotation derives the waste that takes three units of the
// production: the units that leave the book are never offered again.
func theWasteOfTheRotation(
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
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the waste of the rotation was refused: %v", err)
	}

	return waste
}

// TestTheRotationOfARealBookHoldsTheUnitsThatAreStillInIt proves the rules
// of the book against a real database: the rotation holds the units the
// bakery still has, the unit that expires first is the first one that
// leaves, and the unit that left the book is never offered again.
func TestTheRotationOfARealBookHoldsTheUnitsThatAreStillInIt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := theRotationRepository(t)
	bakery := theBakeryOfTheExpiry(t, ctx)
	flour, layers := theStockOfTheProductions(t, ctx, bakery)

	bread := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)
	whole := theRecipeOfTheRotation(t, ctx, bakery, flour.ID, "Pan integral", 5)

	first, err := productionsRepository(t).Record(ctx, theProductionOfTheTest(t, bakery, bread, layers, flour))
	if err != nil {
		t.Fatalf("the production could not be written: %v", err)
	}

	second, err := productionsRepository(t).Record(ctx, theProductionOfTheTest(t, bakery, whole, layers, flour))
	if err != nil {
		t.Fatalf("the production could not be written: %v", err)
	}

	held, err := repo.ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	if len(held) != 20 {
		t.Fatalf("the book holds %d units, want 20", len(held))
	}

	// The three units that expire first leave the book: they were burnt.
	if _, err := wasteRepository(t).Record(ctx, theWasteOfTheRotation(t, bakery, bread, first, held)); err != nil {
		t.Fatalf("the waste could not be written: %v", err)
	}

	inTheBook, err := repo.ListInTheBook(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the units could not be read: %v", err)
	}

	rotation := recipe.RotationOf(inTheBook, 0, "", ledger.NewDate(time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)))

	if len(rotation.Units) != 17 {
		t.Fatalf("the rotation holds %d units, want the 17 that are still in the book", len(rotation.Units))
	}

	// The units that expire first are the first ones of the rotation.
	for index, one := range rotation.Units {
		want := "2026-03-17"

		if index >= 7 {
			want = "2026-03-19"
		}

		if one.ExpiresOn.String() != want {
			t.Errorf("the unit %d of the rotation expires on %q, want %q", index+1, one.ExpiresOn.String(), want)
		}
	}

	// The units that left the book are not in the rotation.
	for _, one := range rotation.Units {
		if one.ProductionID != first.ID && one.ProductionID != second.ID {
			t.Errorf("the rotation holds the unit %d of the production %d, want a unit of the bakery",
				one.UnitID, one.ProductionID)
		}
	}

	// The rotation of one product holds the units of that product alone.
	ofTheBread := recipe.RotationOf(inTheBook, bread.ID, bread.Name, ledger.NewDate(time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)))

	if len(ofTheBread.Units) != 7 {
		t.Fatalf("the rotation of the product holds %d units, want the 7 that are still in the book",
			len(ofTheBread.Units))
	}

	for _, one := range ofTheBread.Units {
		if one.RecipeID != bread.ID {
			t.Errorf("the rotation of the product holds the unit %d of the product %d, want %d",
				one.UnitID, one.RecipeID, bread.ID)
		}
	}
}
