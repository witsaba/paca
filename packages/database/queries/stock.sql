-- The queries of the layers of the stock.
--
-- A layer is derived from a purchase: it is written with the purchase it
-- comes from, and it is read back to answer what the business holds.
--
-- The position of the layer is the position of the line of the purchase it
-- comes from, and the unique index of the table holds the rule that a
-- layer is derived from one line and from no other.
--
-- The layers of a business are read in the order they leave the stock:
-- the oldest layer is the first. The order of the output by expiry is a
-- different rule, and it does not live here.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.

-- name: CreateStockLayer :one
INSERT INTO stock_layers (
    business_id, ingredient_id, purchase_id, position, name, day, quantity, unit, remaining, cost
)
VALUES (
    sqlc.arg(business_id), sqlc.arg(ingredient_id), sqlc.arg(purchase_id), sqlc.arg(position),
    sqlc.arg(name), sqlc.arg(day), sqlc.arg(quantity), sqlc.arg(unit), sqlc.arg(remaining),
    sqlc.arg(cost)
)
RETURNING *;

-- name: ListStockLayersByBusiness :many
SELECT * FROM stock_layers
WHERE business_id = sqlc.arg(business_id)
ORDER BY day, purchase_id, id;
