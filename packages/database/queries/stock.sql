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
-- Every query reads the purchase of the layer with it: the layer belongs
-- to the business of its purchase, and the query never answers a layer
-- whose purchase is of another business. The domain holds the same rule,
-- and the query holds it as a defense in depth.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.

-- name: CreateStockLayer :one
WITH written AS (
    INSERT INTO stock_layers (
        business_id, ingredient_id, purchase_id, position, name, day, quantity, unit, remaining, cost
    )
    VALUES (
        sqlc.arg(business_id), sqlc.arg(ingredient_id), sqlc.arg(purchase_id), sqlc.arg(position),
        sqlc.arg(name), sqlc.arg(day), sqlc.arg(quantity), sqlc.arg(unit), sqlc.arg(remaining),
        sqlc.arg(cost)
    )
    RETURNING id, business_id, ingredient_id, purchase_id, position, name, day, quantity, unit, remaining, cost, created_at
)
SELECT written.id, written.business_id, written.ingredient_id, written.purchase_id, written.position,
       written.name, written.day, written.quantity, written.unit, written.remaining, written.cost,
       written.created_at, purchases.business_id AS purchase_business_id
FROM written
JOIN purchases ON purchases.id = written.purchase_id;

-- name: ListStockLayersByBusiness :many
SELECT layers.id, layers.business_id, layers.ingredient_id, layers.purchase_id, layers.position,
       layers.name, layers.day, layers.quantity, layers.unit, layers.remaining, layers.cost,
       layers.created_at, purchases.business_id AS purchase_business_id
FROM stock_layers layers
JOIN purchases ON purchases.id = layers.purchase_id
WHERE layers.business_id = sqlc.arg(business_id) AND purchases.business_id = sqlc.arg(business_id)
ORDER BY layers.day, layers.purchase_id, layers.id;
