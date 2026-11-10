-- The queries of the wastes.
--
-- The waste is a fact of the business: the product it lost, the quantity it
-- lost, the unit, the reason, the day it happened and the word of the
-- owner. The reason is what a later analysis reads, and it is a column of
-- its own: it is not buried inside a sentence.
--
-- The units that left the book are rows of the table of the exits: the
-- trace of the rotation of the output by the expiry. The table of the
-- exits owns the rule of the book: a unit leaves the book only once,
-- whichever exit takes it.

-- name: CreateWaste :one
WITH written AS (
    INSERT INTO wastes (
        business_id, recipe_id, day, quantity, unit, reason, money, utterance, confirmed_with
    )
    VALUES (
        sqlc.arg(business_id), sqlc.arg(recipe_id), sqlc.arg(day), sqlc.arg(quantity),
        sqlc.arg(unit), sqlc.arg(reason), sqlc.arg(money), sqlc.arg(utterance),
        sqlc.arg(confirmed_with)
    )
    RETURNING id, business_id, recipe_id, day, quantity, unit, reason, money, utterance,
              confirmed_with, created_at
)
SELECT written.id, written.business_id, written.recipe_id, written.day, written.quantity,
       written.unit, written.reason, written.money, written.utterance, written.confirmed_with,
       written.created_at,
       recipes.name AS recipe_name
FROM written
JOIN recipes ON recipes.id = written.recipe_id;



