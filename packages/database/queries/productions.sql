-- The queries of the productions.
--
-- A production is a fact of the business: the recipe it was made of, the
-- day it was made, the grams that went in and the units that came out. The
-- rows of the layers it consumed are the trace of the FIFO, and the rows of
-- the units it made carry the day each of them expires.
--
-- The query of the production reads the name of the recipe with it: the
-- recipe is the owner of its name, and the production never becomes a
-- second truth about it.
--
-- Every query writes one row, and every one of them is a single statement:
-- the library does not read a row to find out whether it may write it.

-- name: CreateProduction :one
WITH written AS (
    INSERT INTO productions (
        business_id, recipe_id, day, grams_in, money, utterance, confirmed_with
    )
    VALUES (
        sqlc.arg(business_id), sqlc.arg(recipe_id), sqlc.arg(day), sqlc.arg(grams_in),
        sqlc.arg(money), sqlc.arg(utterance), sqlc.arg(confirmed_with)
    )
    RETURNING id, business_id, recipe_id, day, grams_in, money, utterance, confirmed_with, created_at
)
SELECT written.id, written.business_id, written.recipe_id, written.day, written.grams_in,
       written.money, written.utterance, written.confirmed_with, written.created_at,
       recipes.name AS recipe_name
FROM written
JOIN recipes ON recipes.id = written.recipe_id;

-- name: CreateProductionLayer :one
INSERT INTO production_layers (
    production_id, position, layer_id, purchase_id, quantity, unit, grams, money
)
VALUES (
    sqlc.arg(production_id), sqlc.arg(position), sqlc.arg(layer_id), sqlc.arg(purchase_id),
    sqlc.arg(quantity), sqlc.arg(unit), sqlc.arg(grams), sqlc.arg(money)
)
RETURNING id, production_id, position, layer_id, purchase_id, quantity, unit, grams, money;

-- name: CreateProductionUnit :one
INSERT INTO production_units (production_id, position, grams, expires_on)
VALUES (sqlc.arg(production_id), sqlc.arg(position), sqlc.arg(grams), sqlc.arg(expires_on))
RETURNING id, production_id, position, grams, expires_on;
