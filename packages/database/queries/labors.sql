-- The queries of the labor of the owner.
--
-- The labor is measured in hours, in the three kinds of the labor, and the
-- value of the hour is the one the owner declared. The money of the labor
-- is not stored: it is derived from the hours and the value of the hour.
--
-- The list is read in the order of the day the labor was worked and of its
-- identity, so that the value of the hour the business declared is always
-- the last one it declared.

-- name: CreateLabor :one
INSERT INTO labors (
    business_id, day, hands_on_hours, oven_hours, waiting_hours, hour_value, utterance, confirmed_with
)
VALUES (
    sqlc.arg(business_id), sqlc.arg(day), sqlc.arg(hands_on_hours), sqlc.arg(oven_hours),
    sqlc.arg(waiting_hours), sqlc.arg(hour_value), sqlc.arg(utterance), sqlc.arg(confirmed_with)
)
RETURNING *;

-- name: ListLaborsByBusiness :many
--
-- The labor of a business and of no other one, in the order of the day it
-- was worked: the answer is the same every time it is read.
SELECT * FROM labors
WHERE business_id = sqlc.arg(business_id)
ORDER BY day, id;
