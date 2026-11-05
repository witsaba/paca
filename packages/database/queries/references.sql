-- The queries of the reference prices.
--
-- A reference is what the market was seen at: the ingredient it is about,
-- the price, the unit of that price, the market, the source and the day it
-- was read.
--
-- The list is read for one ingredient or for every one of them: the two
-- queries are the two answers, and the adapter chooses the one the caller
-- asked for.

-- name: CreatePriceReference :one
INSERT INTO price_references (
    business_id, ingredient_id, price, unit, market, source, seen_on, utterance, confirmed_with
)
VALUES (
    sqlc.arg(business_id), sqlc.arg(ingredient_id), sqlc.arg(price), sqlc.arg(unit),
    sqlc.arg(market), sqlc.arg(source), sqlc.arg(seen_on), sqlc.arg(utterance),
    sqlc.arg(confirmed_with)
)
RETURNING *;

-- name: ListPriceReferencesByBusiness :many
SELECT * FROM price_references WHERE business_id = sqlc.arg(business_id) ORDER BY id;

-- name: ListPriceReferencesOfIngredient :many
SELECT * FROM price_references
WHERE business_id = sqlc.arg(business_id) AND ingredient_id = sqlc.arg(ingredient_id)
ORDER BY id;
