-- The queries of the overhead of the house.
--
-- The overhead carries the electricity the owner paid, the hours the oven
-- ran and the allocation she declared. The money of the overhead is not
-- stored: it is derived from the electricity and the allocation.
--
-- The list is read in the order of the day the overhead was declared and
-- of its identity, so that the rate of the house the business declared is
-- always the last one it declared.

-- name: CreateOverhead :one
INSERT INTO overheads (
    business_id, day, electricity, oven_hours, allocation, utterance, confirmed_with
)
VALUES (
    sqlc.arg(business_id), sqlc.arg(day), sqlc.arg(electricity), sqlc.arg(oven_hours),
    sqlc.arg(allocation), sqlc.arg(utterance), sqlc.arg(confirmed_with)
)
RETURNING *;

-- name: ListOverheadsByBusiness :many
--
-- The overhead of a business and of no other one, in the order of the day
-- it was declared: the answer is the same every time it is read.
SELECT * FROM overheads
WHERE business_id = sqlc.arg(business_id)
ORDER BY day, id;
