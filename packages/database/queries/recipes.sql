-- The queries of the recipes.
--
-- The ingredients of a recipe and its steps are rows of their own tables,
-- and the position of a row is the order the owner said it in.
--
-- Every query writes one row, and every one of them is a single statement:
-- the library does not read a row to find out whether it may write it.

-- name: CreateRecipe :one
INSERT INTO recipes (
    business_id, name, yield_units, yield_grams, shelf_life_days, utterance, confirmed_with
)
VALUES (
    sqlc.arg(business_id), sqlc.arg(name), sqlc.arg(yield_units), sqlc.arg(yield_grams),
    sqlc.arg(shelf_life_days), sqlc.arg(utterance), sqlc.arg(confirmed_with)
)
RETURNING *;

-- name: CreateRecipeIngredient :one
INSERT INTO recipe_ingredients (recipe_id, position, ingredient_id, grams)
VALUES (sqlc.arg(recipe_id), sqlc.arg(position), sqlc.arg(ingredient_id), sqlc.arg(grams))
RETURNING *;

-- name: CreateRecipeStep :one
INSERT INTO recipe_steps (recipe_id, position, instruction, cue, minutes, temperature, labor_kind)
VALUES (
    sqlc.arg(recipe_id), sqlc.arg(position), sqlc.arg(instruction), sqlc.arg(cue),
    sqlc.arg(minutes), sqlc.arg(temperature), sqlc.arg(labor_kind)
)
RETURNING *;

-- name: GetRecipeByID :one
SELECT id, business_id, name, yield_units, yield_grams, shelf_life_days, utterance, confirmed_with, created_at
FROM recipes
WHERE id = sqlc.arg(id);

-- name: ListRecipesByBusiness :many
--
-- The recipes of a business and of no other one, in the order of their
-- identity: the answer is the same every time it is read.
SELECT id, business_id, name, yield_units, yield_grams, shelf_life_days, utterance, confirmed_with, created_at
FROM recipes
WHERE business_id = sqlc.arg(business_id)
ORDER BY id;

-- name: ListRecipeIngredientsByRecipe :many
--
-- The ingredients of the recipe, in the order the owner said them.
SELECT id, recipe_id, position, ingredient_id, grams
FROM recipe_ingredients
WHERE recipe_id = sqlc.arg(recipe_id)
ORDER BY position;

-- name: ListRecipeStepsByRecipe :many
--
-- The steps of the recipe, in the order the owner said them.
SELECT id, recipe_id, position, instruction, cue, minutes, temperature, labor_kind
FROM recipe_steps
WHERE recipe_id = sqlc.arg(recipe_id)
ORDER BY position;
