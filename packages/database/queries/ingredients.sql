-- The queries of the catalog of the names the business buys.
--
-- The aliases of an ingredient travel as JSONB: they are the names the
-- owner uses for the thing she buys, and the adapter encodes them.
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
