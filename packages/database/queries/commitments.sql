-- The queries of the commitments of the bakery.
--
-- The commitment is one row of its own table: the promise of what is due and
-- the day it is due, and the day it was met when it was met.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.
--
-- A promise is NEVER DELETED, and there is no query here that deletes one:
-- the promise that was kept stays in the book with the day it was kept.

-- name: CreateCommitment :one
--
-- The promise is written OPEN: the day of meeting is not a column of this
-- statement, and the book never writes one at the creation.
INSERT INTO commitments (business_id, what, due_on)
VALUES (sqlc.arg(business_id), sqlc.arg(what), sqlc.arg(due_on))
RETURNING *;

-- name: ListCommitmentsByBusiness :many
--
-- The promises of the business and of no other one, in the order of their
-- identity: the answer is the same every time it is read. The list holds the
-- promises that were kept too.
SELECT * FROM commitments WHERE business_id = sqlc.arg(business_id) ORDER BY id;

-- name: MeetCommitment :one
--
-- The meeting of a promise: the day it was met, written over the row of the
-- promise and read back as it is now.
--
-- The meeting moves a promise that is OPEN, and only one: the day of meeting
-- is written where no day of meeting is held, so a promise that was already
-- met is never met again and the day the book holds is never hidden.
--
-- The meeting also carries its day: a meeting with no day matches no row,
-- because the promise that was kept stays in the book WITH the day it was
-- kept. The rule is the one of the domain (`commitment.ErrMissingMetDay`),
-- and the table holds it as its defense in depth.
UPDATE commitments
SET met_on = sqlc.arg(met_on)::date, updated_at = now()
WHERE id = sqlc.arg(id) AND business_id = sqlc.arg(business_id) AND met_on IS NULL
  AND sqlc.arg(met_on)::date IS NOT NULL
RETURNING *;

-- name: TheMeetingOfTheCommitment :one
--
-- The day the promise was met, as the book holds it now. It is read when the
-- meeting did not find the promise it may meet, so the adapter can tell the
-- promise that is not there from the promise that was met already and from
-- the open promise the meeting carried no day for.
SELECT met_on FROM commitments WHERE id = sqlc.arg(id) AND business_id = sqlc.arg(business_id);
