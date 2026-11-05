-- The queries of the catalog of the names the business buys.
--
-- The aliases of an ingredient travel as JSONB: they are the names the
-- owner uses for the thing she buys, and the adapter encodes them. The
-- allergens travel the same way.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.

-- name: ListIngredientsByBusiness :many
SELECT * FROM ingredients WHERE business_id = sqlc.arg(business_id) ORDER BY id;

-- name: CreateIngredient :one
INSERT INTO ingredients (business_id, name, aliases)
VALUES (sqlc.arg(business_id), sqlc.arg(name), sqlc.arg(aliases))
RETURNING *;

-- name: UpdateIngredient :one
UPDATE ingredients
SET unit = sqlc.arg(unit),
    package_grams = sqlc.arg(package_grams),
    allergens = sqlc.arg(allergens),
    certification_name = sqlc.arg(certification_name),
    certification_evidence = sqlc.arg(certification_evidence),
    conversion = sqlc.arg(conversion),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;
