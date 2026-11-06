package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names the database gives to the constraints of the tables of the
// recipes.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	recipeIngredientsOnePositionIndex = "recipe_ingredients_one_position"
	recipeStepsOnePositionIndex       = "recipe_steps_one_position"
)

// RecipeRepository is the adapter of the storage of the recipes.
//
// The ingredients of a recipe and its steps are rows of their own tables,
// and the adapter writes them with it: a recipe without its ingredients and
// its steps is not a recipe.
//
// The row never decides what a recipe is: the model is built by the domain,
// through its own constructors.
type RecipeRepository struct {
	pool *Pool
}

// NewRecipeRepository returns the repository of the recipes, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewRecipeRepository(pool *Pool) (*RecipeRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &RecipeRepository{pool: pool}, nil
}

// Record writes the recipe with its ingredients and its steps, and answers
// it with the identity that the storage generated.
//
// The recipe and its rows are written in one transaction: a recipe whose
// ingredient is not in the catalog leaves nothing behind, because a guide
// without its ingredients teaches nothing.
//
// The error is recipe.ErrInvalidValue when the row breaks a rule of the
// table, recipe.ErrNotFound when the business or the ingredient of the
// recipe is not there, and recipe.ErrNoStorage when the storage is not
// open.
func (r *RecipeRepository) Record(ctx context.Context, written recipe.Recipe) (recipe.Recipe, error) {
	if r == nil || r.pool == nil {
		return recipe.Recipe{}, recipe.ErrNoStorage
	}

	// The set joins the transaction of the caller when the caller opened
	// one, and opens its own when it did not: the recipe, its ingredients
	// and its steps are written whole or not at all.
	if _, ok := transactionIn(ctx); ok {
		return r.write(ctx, r.pool.queriesFor(ctx), written)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return recipe.Recipe{}, fmt.Errorf("the transaction of the recipes could not be opened: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	recorded, err := r.write(ctx, r.pool.queriesFor(ctx).WithTx(tx), written)
	if err != nil {
		return recipe.Recipe{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return recipe.Recipe{}, fmt.Errorf("the transaction of the recipes could not be committed: %w", err)
	}

	return recorded, nil
}

// write writes the recipe, its ingredients and its steps, and answers the
// recipe that was written.
func (r *RecipeRepository) write(ctx context.Context, queries *db.Queries, written recipe.Recipe) (recipe.Recipe, error) {
	yieldUnits, err := countOf(written.Yield.Units, "the yield of the recipe")
	if err != nil {
		return recipe.Recipe{}, err
	}

	shelfLife, err := countOf(written.ShelfLifeDays, "the shelf life of the recipe")
	if err != nil {
		return recipe.Recipe{}, err
	}

	row, err := queries.CreateRecipe(ctx, db.CreateRecipeParams{
		BusinessID:    int64(written.BusinessID),
		Name:          written.Name,
		YieldUnits:    yieldUnits,
		YieldGrams:    written.Yield.Grams,
		ShelfLifeDays: shelfLife,
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
	})
	if err != nil {
		return recipe.Recipe{}, writeRecipeError(err)
	}

	ingredients := make([]recipe.Ingredient, 0, len(written.Ingredients))
	for position, one := range written.Ingredients {
		line, err := queries.CreateRecipeIngredient(ctx, db.CreateRecipeIngredientParams{
			RecipeID:     row.ID,
			Position:     int32(position + 1),
			IngredientID: int64(one.IngredientID),
			Grams:        one.Grams,
		})
		if err != nil {
			return recipe.Recipe{}, writeRecipeError(err)
		}

		ingredients = append(ingredients, recipe.Ingredient{
			IngredientID: ingredient.ID(line.IngredientID),
			Grams:        line.Grams,
		})
	}

	steps := make([]recipe.Step, 0, len(written.Steps))
	for position, one := range written.Steps {
		minutes, err := countOf(one.Minutes, "the time of a step of the recipe")
		if err != nil {
			return recipe.Recipe{}, err
		}

		step, err := queries.CreateRecipeStep(ctx, db.CreateRecipeStepParams{
			RecipeID:    row.ID,
			Position:    int32(position + 1),
			Instruction: one.Instruction,
			Cue:         one.Cue,
			Minutes:     minutes,
			Temperature: one.Temperature,
		})
		if err != nil {
			return recipe.Recipe{}, writeRecipeError(err)
		}

		steps = append(steps, recipe.Step{
			Instruction: step.Instruction,
			Cue:         step.Cue,
			Minutes:     int(step.Minutes),
			Temperature: step.Temperature,
		})
	}

	// The recipe is built by the domain: the row never decides what a
	// recipe is, and the model refuses a row that breaks a rule of it.
	recorded, err := recipe.New(business.ID(row.BusinessID), recipe.Definition{
		Name:          row.Name,
		Ingredients:   ingredients,
		Steps:         steps,
		Yield:         recipe.Yield{Units: int(row.YieldUnits), Grams: row.YieldGrams},
		ShelfLifeDays: int(row.ShelfLifeDays),
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
	})
	if err != nil {
		return recipe.Recipe{}, err
	}

	recorded.ID = recipe.ID(row.ID)
	recorded.CreatedAt = timestamp(row.CreatedAt)

	return recorded, nil
}

// Find returns the recipe that carries that identity, with its ingredients
// and its steps.
//
// The recipe is built by the domain: the row never decides what a recipe
// is, and a row that another writer broke is refused.
//
// The error is recipe.ErrNotFound when no recipe carries that identity, and
// recipe.ErrNoStorage when the storage is not open.
func (r *RecipeRepository) Find(ctx context.Context, id recipe.ID) (recipe.Recipe, error) {
	if r == nil || r.pool == nil {
		return recipe.Recipe{}, recipe.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).GetRecipeByID(ctx, int64(id))
	if err != nil {
		return recipe.Recipe{}, writeRecipeError(err)
	}

	return r.recipeOf(ctx, rowOfARecipe{
		ID:            row.ID,
		BusinessID:    row.BusinessID,
		Name:          row.Name,
		YieldUnits:    row.YieldUnits,
		YieldGrams:    row.YieldGrams,
		ShelfLifeDays: row.ShelfLifeDays,
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
		CreatedAt:     row.CreatedAt,
	})
}

// List returns the recipes of the business, with their ingredients and
// their steps.
//
// The recipes are read in the order of their identity, so the answer is the
// same every time it is read.
//
// The error is recipe.ErrNoStorage when the storage is not open.
func (r *RecipeRepository) List(ctx context.Context, own business.ID) ([]recipe.Recipe, error) {
	if r == nil || r.pool == nil {
		return nil, recipe.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListRecipesByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeRecipeError(err)
	}

	recipes := make([]recipe.Recipe, 0, len(rows))

	for _, row := range rows {
		built, err := r.recipeOf(ctx, rowOfARecipe{
			ID:            row.ID,
			BusinessID:    row.BusinessID,
			Name:          row.Name,
			YieldUnits:    row.YieldUnits,
			YieldGrams:    row.YieldGrams,
			ShelfLifeDays: row.ShelfLifeDays,
			Utterance:     row.Utterance,
			ConfirmedWith: row.ConfirmedWith,
			CreatedAt:     row.CreatedAt,
		})
		if err != nil {
			return nil, err
		}

		recipes = append(recipes, built)
	}

	return recipes, nil
}

// rowOfARecipe is what a row of the recipes carries, however it was read.
type rowOfARecipe struct {
	ID            int64
	BusinessID    int64
	Name          string
	YieldUnits    int32
	YieldGrams    float64
	ShelfLifeDays int32
	Utterance     string
	ConfirmedWith string
	CreatedAt     pgtype.Timestamptz
}

// recipeOf builds the recipe of a row with its ingredients and its steps,
// and asks the domain whether it is whole.
//
// A recipe without its ingredients and its steps is not a recipe, and the
// domain refuses it.
func (r *RecipeRepository) recipeOf(ctx context.Context, row rowOfARecipe) (recipe.Recipe, error) {
	ingredientRows, err := r.pool.queriesFor(ctx).ListRecipeIngredientsByRecipe(ctx, row.ID)
	if err != nil {
		return recipe.Recipe{}, writeRecipeError(err)
	}

	ingredients := make([]recipe.Ingredient, 0, len(ingredientRows))
	for _, one := range ingredientRows {
		ingredients = append(ingredients, recipe.Ingredient{
			IngredientID: ingredient.ID(one.IngredientID),
			Grams:        one.Grams,
		})
	}

	stepRows, err := r.pool.queriesFor(ctx).ListRecipeStepsByRecipe(ctx, row.ID)
	if err != nil {
		return recipe.Recipe{}, writeRecipeError(err)
	}

	steps := make([]recipe.Step, 0, len(stepRows))
	for _, one := range stepRows {
		steps = append(steps, recipe.Step{
			Instruction: one.Instruction,
			Cue:         one.Cue,
			Minutes:     int(one.Minutes),
			Temperature: one.Temperature,
		})
	}

	built, err := recipe.New(business.ID(row.BusinessID), recipe.Definition{
		Name:          row.Name,
		Ingredients:   ingredients,
		Steps:         steps,
		Yield:         recipe.Yield{Units: int(row.YieldUnits), Grams: row.YieldGrams},
		ShelfLifeDays: int(row.ShelfLifeDays),
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
	})
	if err != nil {
		return recipe.Recipe{}, err
	}

	built.ID = recipe.ID(row.ID)
	built.CreatedAt = timestamp(row.CreatedAt)

	return built, nil
}

// countOf returns the count as the column of the table holds it.
//
// The column is an integer of thirty-two bits, and the counts of a recipe
// are the units it makes, the days it keeps and the minutes of a step: a
// count that does not fit the column is a count the table cannot hold.
func countOf(value int, what string) (int32, error) {
	if value < 0 || value > math.MaxInt32 {
		return 0, fmt.Errorf("%w: %s is %d", recipe.ErrInvalidValue, what, value)
	}

	return int32(value), nil
}

// writeRecipeError translates the truth of the database into the errors of
// the port of the recipes.
//
// The constraints of the tables hold the rules of the domain as a defense in
// depth, and the business or the ingredient that is not there is the refusal
// of the foreign key.
func writeRecipeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return recipe.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		if pgErr.ConstraintName == recipeIngredientsOnePositionIndex ||
			pgErr.ConstraintName == recipeStepsOnePositionIndex {
			return fmt.Errorf("%w: the constraint %q refused the row", recipe.ErrInvalidValue, pgErr.ConstraintName)
		}
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business or the ingredient of the recipe is not there", recipe.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", recipe.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
