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
INSERT INTO recipe_steps (recipe_id, position, instruction, cue, minutes, temperature)
VALUES (
    sqlc.arg(recipe_id), sqlc.arg(position), sqlc.arg(instruction), sqlc.arg(cue),
    sqlc.arg(minutes), sqlc.arg(temperature)
)
RETURNING *;
