-- The queries of the wastes.
--
-- The waste is a fact of the business: the product it lost, the quantity it
-- lost, the unit, the reason, the day it happened and the word of the
-- owner. The reason is what a later analysis reads, and it is a column of
-- its own: it is not buried inside a sentence.
--
-- The units that left the book are rows of their own table: they are the
-- trace of the rotation of the output by the expiry. The unique index over
-- the unit holds the rule of the book: a unit leaves the book only once.
--
-- The units that are still in the book are the ones that did not leave it:
-- the waste takes them, and a unit that already left is never taken again.

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

-- name: CreateWasteUnit :one
INSERT INTO waste_units (waste_id, unit_id, production_id, position, grams, expires_on)
VALUES (
    sqlc.arg(waste_id), sqlc.arg(unit_id), sqlc.arg(production_id), sqlc.arg(position),
    sqlc.arg(grams), sqlc.arg(expires_on)
)
RETURNING id, waste_id, unit_id, production_id, position, grams, expires_on;

-- name: ListProductionUnitsInTheBook :many
--
-- The units of the productions of the business that are still in the book,
-- in the order they expire: the one that expires first is the first one.
SELECT units.id, units.production_id, units.position, units.grams, units.expires_on,
       productions.recipe_id, productions.day AS produced_on, recipes.name AS recipe_name
FROM production_units units
JOIN productions ON productions.id = units.production_id
JOIN recipes ON recipes.id = productions.recipe_id
WHERE productions.business_id = sqlc.arg(business_id)
  AND NOT EXISTS (SELECT 1 FROM waste_units wasted WHERE wasted.unit_id = units.id)
ORDER BY units.expires_on, units.id;
