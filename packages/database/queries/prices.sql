-- The queries of the price list of the bakery.
--
-- The product and the price are two tables of their own, and the queries
-- of them are here: the product is opened the first time it is named, and
-- the price is a row that is kept forever.
--
-- The optional columns are NULL when the value is not set. The queries use
-- sqlc.narg for them, so the library passes NULL and the column keeps the
-- rule of the domain: an empty value is an absent value.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.

-- name: ListProductsByBusiness :many
SELECT * FROM products WHERE business_id = sqlc.arg(business_id) ORDER BY id;

-- name: CreateProduct :one
INSERT INTO products (business_id, recipe_id, name, tax_treatment)
VALUES (sqlc.arg(business_id), sqlc.arg(recipe_id), sqlc.arg(name), sqlc.narg(tax_treatment)::text)
RETURNING *;

-- name: StateProduct :one
UPDATE products
SET tax_treatment = sqlc.narg(tax_treatment)::text,
    updated_at = now()
WHERE id = sqlc.arg(id) AND business_id = sqlc.arg(business_id)
RETURNING *;

-- name: ListPricesByBusiness :many
SELECT * FROM prices WHERE business_id = sqlc.arg(business_id) ORDER BY id;

-- name: CreatePrice :one
INSERT INTO prices (business_id, product_id, channel, amount, basis, day)
VALUES (sqlc.arg(business_id), sqlc.arg(product_id), sqlc.arg(channel), sqlc.arg(amount), sqlc.arg(basis), sqlc.arg(day))
RETURNING *;
