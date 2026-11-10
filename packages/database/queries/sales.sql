-- The queries of the sales.
--
-- The sale is a fact of the business: the product it sold, the quantity it
-- sold, the unit, the money the client paid, the cost the sale released,
-- the day it happened and the word of the owner.
--
-- The units that left the book are rows of the table of the exits, and
-- they are written with the sale: a sale without its units is not a sale,
-- and a unit that left the book never leaves it again.

-- name: CreateSale :one
WITH written AS (
    INSERT INTO sales (
        business_id, recipe_id, day, quantity, unit, money, cost, utterance, confirmed_with
    )
    VALUES (
        sqlc.arg(business_id), sqlc.arg(recipe_id), sqlc.arg(day), sqlc.arg(quantity),
        sqlc.arg(unit), sqlc.arg(money), sqlc.arg(cost), sqlc.arg(utterance),
        sqlc.arg(confirmed_with)
    )
    RETURNING id, business_id, recipe_id, day, quantity, unit, money, cost, utterance,
              confirmed_with, created_at
)
SELECT written.id, written.business_id, written.recipe_id, written.day, written.quantity,
       written.unit, written.money, written.cost, written.utterance, written.confirmed_with,
       written.created_at,
       recipes.name AS recipe_name
FROM written
JOIN recipes ON recipes.id = written.recipe_id;
