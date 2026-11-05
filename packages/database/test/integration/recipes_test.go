//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/user"
)

// recipesRepository returns the repository of the recipes, bound to the
// database of the suite.
func recipesRepository(t *testing.T) *postgres.RecipeRepository {
	t.Helper()

	repo, err := postgres.NewRecipeRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the recipes could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheRecipes opens the business that owns the recipes of the
// test.
func theBakeryOfTheRecipes(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Recipes")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theFlourOfTheRecipes writes the ingredient of the catalog that the recipe
// of the test names.
func theFlourOfTheRecipes(t *testing.T, ctx context.Context, bakery business.Business) ingredient.Ingredient {
	t.Helper()

	flour, err := ingredient.New(bakery.ID, "Harina panadera", []string{"harina de trigo"})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	written, err := ingredientsRepository(t).Create(ctx, flour)
	if err != nil {
		t.Fatalf("the ingredient could not be written: %v", err)
	}

	return written
}

// theRecipeOfTheTest builds the recipe the tests of the table write.
func theRecipeOfTheTest(bakery business.Business, flour ingredient.Ingredient) recipe.Recipe {
	built, err := recipe.New(bakery.ID, recipe.Definition{
		Name: "Pan de molde",
		Ingredients: []recipe.Ingredient{
			{IngredientID: flour.ID, Grams: 500},
		},
		Steps: []recipe.Step{
			{Instruction: "Mezclar la harina con el agua y amasar.", Cue: "la masa que holds la forma", Minutes: 20, Temperature: 24},
			{Instruction: "Hornear.", Minutes: 45, Temperature: 220},
		},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		Utterance:     "El pan de molde lleva 500 gramos de harina",
		ConfirmedWith: "sí",
	})
	if err != nil {
		panic("the recipe of the test could not be built: " + err.Error())
	}

	return built
}

// TestTheRecipeIsKeptWithItsIngredientsAndItsSteps proves the rules of the
// table against a real database: the recipe is written with its name, its
// yield, its shelf life, its sentence and the word of the owner, and it is
// read back as it was written.
func TestTheRecipeIsKeptWithItsIngredientsAndItsSteps(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := recipesRepository(t)
	bakery := theBakeryOfTheRecipes(t, ctx)
	flour := theFlourOfTheRecipes(t, ctx, bakery)

	written, err := repo.Record(ctx, theRecipeOfTheTest(bakery, flour))
	if err != nil {
		t.Fatalf("the recipe could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the recipe has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the recipe is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.Name != "Pan de molde" {
		t.Errorf("the name is %q, want %q", written.Name, "Pan de molde")
	}
	if written.Yield.Units != 10 || written.Yield.Grams != 90 {
		t.Errorf("the yield is %+v, want 10 units of 90 grams", written.Yield)
	}
	if written.ShelfLifeDays != 3 {
		t.Errorf("the shelf life is %d days, want 3", written.ShelfLifeDays)
	}
	if written.Utterance != "El pan de molde lleva 500 gramos de harina" {
		t.Errorf("the sentence is %q, want the one of the owner", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if written.CreatedAt.IsZero() {
		t.Error("the recipe carries no moment of creation, want the one the database filled")
	}
}

// TestTheIngredientsOfTheRecipeAreKeptWithTheirGrams proves the rule of
// the table against a real database: the ingredient of a recipe is the
// ingredient of the catalog, with the grams of it, and it is never a second
// copy of the flour.
func TestTheIngredientsOfTheRecipeAreKeptWithTheirGrams(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := recipesRepository(t)
	bakery := theBakeryOfTheRecipes(t, ctx)
	flour := theFlourOfTheRecipes(t, ctx, bakery)

	written, err := repo.Record(ctx, theRecipeOfTheTest(bakery, flour))
	if err != nil {
		t.Fatalf("the recipe could not be written: %v", err)
	}

	rows, err := theSuite.db.QueryContext(ctx,
		`SELECT ingredient_id, grams FROM recipe_ingredients WHERE recipe_id = $1 ORDER BY position`, written.ID)
	if err != nil {
		t.Fatalf("the ingredients of the recipe could not be read: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var ingredientID int64
		var grams float64
		if err := rows.Scan(&ingredientID, &grams); err != nil {
			t.Fatalf("the ingredient of the recipe could not be read: %v", err)
		}

		count++
		if ingredientID != int64(flour.ID) {
			t.Errorf("the ingredient is %d, want %d", ingredientID, flour.ID)
		}
		if grams != 500 {
			t.Errorf("the grams of the ingredient are %v, want 500", grams)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("the ingredients of the recipe could not be read: %v", err)
	}
	if count != 1 {
		t.Errorf("the recipe holds %d ingredients, want 1", count)
	}
}

// TestTheStepsOfTheRecipeKeepTheirOrder proves the rule of the table
// against a real database: the steps of a recipe keep the order the owner
// said them in, with their times, their temperatures and their cues.
func TestTheStepsOfTheRecipeKeepTheirOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := recipesRepository(t)
	bakery := theBakeryOfTheRecipes(t, ctx)
	flour := theFlourOfTheRecipes(t, ctx, bakery)

	written, err := repo.Record(ctx, theRecipeOfTheTest(bakery, flour))
	if err != nil {
		t.Fatalf("the recipe could not be written: %v", err)
	}

	rows, err := theSuite.db.QueryContext(ctx,
		`SELECT instruction, cue, minutes, temperature FROM recipe_steps WHERE recipe_id = $1 ORDER BY position`,
		written.ID)
	if err != nil {
		t.Fatalf("the steps of the recipe could not be read: %v", err)
	}
	defer rows.Close()

	type step struct {
		instruction string
		cue         string
		minutes     int
		temperature float64
	}

	steps := make([]step, 0, 2)
	for rows.Next() {
		var one step
		if err := rows.Scan(&one.instruction, &one.cue, &one.minutes, &one.temperature); err != nil {
			t.Fatalf("the step of the recipe could not be read: %v", err)
		}

		steps = append(steps, one)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("the steps of the recipe could not be read: %v", err)
	}

	if len(steps) != 2 {
		t.Fatalf("the recipe holds %d steps, want 2", len(steps))
	}
	if steps[0].instruction != "Mezclar la harina con el agua y amasar." {
		t.Errorf("the first step is %q, want the one of the owner", steps[0].instruction)
	}
	if steps[0].cue != "la masa que holds la forma" {
		t.Errorf("the cue of the first step is %q, want the one of the owner", steps[0].cue)
	}
	if steps[0].minutes != 20 {
		t.Errorf("the time of the first step is %d minutes, want 20", steps[0].minutes)
	}
	if steps[0].temperature != 24 {
		t.Errorf("the temperature of the first step is %v, want 24", steps[0].temperature)
	}
	if steps[1].instruction != "Hornear." {
		t.Errorf("the second step is %q, want %q", steps[1].instruction, "Hornear.")
	}
	if steps[1].cue != "" {
		t.Errorf("the cue of the second step is %q, want none", steps[1].cue)
	}
	if steps[1].minutes != 45 || steps[1].temperature != 220 {
		t.Errorf("the second step is %+v, want 45 minutes at 220 degrees", steps[1])
	}
}

// TestTheTableRefusesTheRecipeWithoutAName proves the rule of the domain
// against a real database: the name of a recipe is not empty.
func TestTheTableRefusesTheRecipeWithoutAName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheRecipes(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO recipes (business_id, name, yield_units, yield_grams, shelf_life_days, confirmed_with)
		 VALUES ($1, $2, 10, 90, 3, 'sí')`,
		int64(bakery.ID), "",
	)
	if err == nil {
		t.Fatal("the database stored a recipe without a name, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "recipes_name_not_empty_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "recipes_name_not_empty_check")
	}
}

// TestTheTableRefusesTheRecipeWithoutAYield proves the rule of the domain
// against a real database: the yield is a declared constant, and it is
// positive.
func TestTheTableRefusesTheRecipeWithoutAYield(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheRecipes(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO recipes (business_id, name, yield_units, yield_grams, shelf_life_days, confirmed_with)
		 VALUES ($1, $2, 0, 90, 3, 'sí')`,
		int64(bakery.ID), "Pan de molde",
	)
	if err == nil {
		t.Fatal("the database stored a recipe without a yield, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "recipes_yield_units_positive_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "recipes_yield_units_positive_check")
	}
}

// TestTheTableRefusesTheIngredientThatIsNotInTheCatalog proves the rule of
// the domain against a real database: the ingredient of a recipe is the
// ingredient of the catalog, and it is never a second copy of the flour.
func TestTheTableRefusesTheIngredientThatIsNotInTheCatalog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := recipesRepository(t)
	bakery := theBakeryOfTheRecipes(t, ctx)
	flour := theFlourOfTheRecipes(t, ctx, bakery)

	written, err := repo.Record(ctx, theRecipeOfTheTest(bakery, flour))
	if err != nil {
		t.Fatalf("the recipe could not be written: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO recipe_ingredients (recipe_id, position, ingredient_id, grams) VALUES ($1, 2, $2, 500)`,
		written.ID, int64(flour.ID)+1000,
	)
	if err == nil {
		t.Fatal("the database stored an ingredient that is not in the catalog, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != foreignKeyViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, foreignKeyViolation)
	}
	if pgErr.ConstraintName != "recipe_ingredients_ingredient_id_fkey" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "recipe_ingredients_ingredient_id_fkey")
	}
}

// TestTheTableRefusesTwoIngredientsInTheSamePlace proves the rule of the
// domain against a real database: the ingredients of a recipe keep their
// order, and two of them never share a place.
func TestTheTableRefusesTwoIngredientsInTheSamePlace(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := recipesRepository(t)
	bakery := theBakeryOfTheRecipes(t, ctx)
	flour := theFlourOfTheRecipes(t, ctx, bakery)

	written, err := repo.Record(ctx, theRecipeOfTheTest(bakery, flour))
	if err != nil {
		t.Fatalf("the recipe could not be written: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO recipe_ingredients (recipe_id, position, ingredient_id, grams) VALUES ($1, 1, $2, 100)`,
		written.ID, int64(flour.ID),
	)
	if err == nil {
		t.Fatal("the database stored two ingredients in the same place, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != uniqueViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, uniqueViolation)
	}
	if pgErr.ConstraintName != "recipe_ingredients_one_position" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "recipe_ingredients_one_position")
	}
}

// TestTheRecipeThatIsRefusedLeavesNothingBehind proves the rule of the
// domain against a real database: the recipe, its ingredients and its steps
// are written whole or not at all, and a refused recipe leaves no row
// behind.
func TestTheRecipeThatIsRefusedLeavesNothingBehind(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := recipesRepository(t)
	bakery := theBakeryOfTheRecipes(t, ctx)

	unknown := theRecipeOfTheTest(bakery, ingredient.Ingredient{ID: ingredient.ID(bakery.ID) + 1000})

	_, err := repo.Record(ctx, unknown)
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}

	var recipes int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM recipes WHERE business_id = $1`, int64(bakery.ID)).Scan(&recipes); err != nil {
		t.Fatalf("the recipes of the bakery could not be counted: %v", err)
	}
	if recipes != 0 {
		t.Errorf("the database stored %d recipes of the refused one, want none", recipes)
	}
}
