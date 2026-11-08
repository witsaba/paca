-- The queries of the exits of the book.
--
-- A unit leaves the book once, whichever exit it takes: the waste or the
-- sale. The table of the exits owns that rule, and the query of the units
-- that are still in the book reads it: a unit that left is never offered
-- again, whatever exit took it.
--
-- The exit of a unit carries the kind of the exit and the identity of the
-- row of that exit: the waste or the sale that took it.

-- name: CreateUnitExit :one
INSERT INTO unit_exits (unit_id, exit_kind, exit_id, production_id, position, grams, expires_on)
VALUES (
    sqlc.arg(unit_id), sqlc.arg(exit_kind), sqlc.arg(exit_id), sqlc.arg(production_id),
    sqlc.arg(position), sqlc.arg(grams), sqlc.arg(expires_on)
)
RETURNING unit_id, exit_kind, exit_id, production_id, position, grams, expires_on;

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
  AND NOT EXISTS (SELECT 1 FROM unit_exits exits WHERE exits.unit_id = units.id)
ORDER BY units.expires_on, units.id;
