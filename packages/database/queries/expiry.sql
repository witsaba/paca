-- The queries of the expiry.
--
-- The correction of the shelf life is a fact of the business: the recipe it
-- corrects, the number of days what it makes keeps, the day it was declared
-- and the word of the owner. Every correction is a row of its own, and the
-- list is read in the order of the day it was declared and of its identity,
-- so that the correction that counts is always the last one the owner
-- declared.
--
-- The units are the rows of the productions: the query reads them with the
-- production they came out of and the recipe they were made of, because the
-- expiry is read for one business, and the day of the expiry is a day of
-- that business.
--
-- The day of the expiry of a unit is written only when the unit still
-- expires on the day the correction read: a unit that changed under the
-- correction is refused.

-- name: CreateShelfLifeCorrection :one
WITH written AS (
    INSERT INTO recipe_shelf_life_corrections (
        business_id, recipe_id, shelf_life_days, day, utterance, confirmed_with
    )
    VALUES (
        sqlc.arg(business_id), sqlc.arg(recipe_id), sqlc.arg(shelf_life_days), sqlc.arg(day),
        sqlc.arg(utterance), sqlc.arg(confirmed_with)
    )
    RETURNING id, business_id, recipe_id, shelf_life_days, day, utterance, confirmed_with, created_at
)
SELECT written.id, written.business_id, written.recipe_id, written.shelf_life_days, written.day,
       written.utterance, written.confirmed_with, written.created_at,
       recipes.name AS recipe_name
FROM written
JOIN recipes ON recipes.id = written.recipe_id;

-- name: ListShelfLifeCorrectionsByBusiness :many
--
-- The corrections of a business and of no other one, in the order of the day
-- they were declared: the answer is the same every time it is read.
SELECT corrections.id, corrections.business_id, corrections.recipe_id, corrections.shelf_life_days,
       corrections.day, corrections.utterance, corrections.confirmed_with, corrections.created_at,
       recipes.name AS recipe_name
FROM recipe_shelf_life_corrections corrections
JOIN recipes ON recipes.id = corrections.recipe_id
WHERE corrections.business_id = sqlc.arg(business_id)
ORDER BY corrections.day, corrections.id;

-- name: ListProductionUnitsByBusiness :many
--
-- The units of the productions of the business, in the order they expire:
-- the unit that expires first is the first one.
SELECT units.id, units.production_id, units.position, units.grams, units.expires_on,
       productions.recipe_id, productions.day AS produced_on, recipes.name AS recipe_name
FROM production_units units
JOIN productions ON productions.id = units.production_id
JOIN recipes ON recipes.id = productions.recipe_id
WHERE productions.business_id = sqlc.arg(business_id)
ORDER BY units.expires_on, units.id;

-- name: UpdateProductionUnitExpiry :execrows
--
-- The day of the expiry of a unit is written only when the unit still
-- expires on the day the correction read: a unit that changed under the
-- correction is refused.
UPDATE production_units
SET expires_on = sqlc.arg(expires_on)
WHERE id = sqlc.arg(id)
  AND expires_on = sqlc.arg(previous_expires_on);
