-- The queries of the drafts of the purchases.
--
-- The lines of a draft, what it still needs, what it was not sure about
-- and what PACA filled by herself travel as JSONB: they are the reading of
-- a sentence and not the rows of a fact, and the adapter encodes them.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.

-- name: CreatePurchaseDraft :one
INSERT INTO purchase_drafts (
    business_id, state, revision, utterance, lines, missing, ambiguous, inferred,
    day, market, invoice, proposed_on, expires_on
)
VALUES (
    sqlc.arg(business_id), sqlc.arg(state), sqlc.arg(revision), sqlc.arg(utterance),
    sqlc.arg(lines), sqlc.arg(missing), sqlc.arg(ambiguous), sqlc.arg(inferred),
    sqlc.arg(day), sqlc.arg(market), sqlc.arg(invoice), sqlc.arg(proposed_on), sqlc.arg(expires_on)
)
RETURNING *;

-- name: UpdatePurchaseDraft :one
UPDATE purchase_drafts
SET state = sqlc.arg(state),
    revision = sqlc.arg(revision),
    utterance = sqlc.arg(utterance),
    lines = sqlc.arg(lines),
    missing = sqlc.arg(missing),
    ambiguous = sqlc.arg(ambiguous),
    inferred = sqlc.arg(inferred),
    day = sqlc.arg(day),
    market = sqlc.arg(market),
    invoice = sqlc.arg(invoice),
    proposed_on = sqlc.arg(proposed_on),
    expires_on = sqlc.arg(expires_on),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetPurchaseDraftByID :one
SELECT * FROM purchase_drafts WHERE id = sqlc.arg(id);

-- name: CreatePurchase :one
INSERT INTO purchases (
    business_id, draft_id, day, market, invoice, utterance, confirmed_with, total
)
VALUES (
    sqlc.arg(business_id), sqlc.arg(draft_id), sqlc.arg(day), sqlc.arg(market),
    sqlc.arg(invoice), sqlc.arg(utterance), sqlc.arg(confirmed_with), sqlc.arg(total)
)
RETURNING *;

-- name: CreatePurchaseLine :one
INSERT INTO purchase_lines (
    purchase_id, position, name, quantity, unit, total, unit_price, ingredient_id
)
VALUES (
    sqlc.arg(purchase_id), sqlc.arg(position), sqlc.arg(name), sqlc.arg(quantity),
    sqlc.arg(unit), sqlc.arg(total), sqlc.arg(unit_price), sqlc.arg(ingredient_id)
)
RETURNING *;

-- name: ConfirmPurchaseDraft :one
--
-- The row must be the one the owner saw: the revision is the one she saw,
-- and the state is the one of a draft that waits for the yes of its owner.
-- The vocabulary of the states belongs to the domain, and the caller sends
-- both of them.
UPDATE purchase_drafts
SET state = sqlc.arg(state), updated_at = now()
WHERE id = sqlc.arg(id) AND revision = sqlc.arg(revision) AND state = sqlc.arg(expected_state)
RETURNING *;

-- name: CancelPurchaseDraft :one
--
-- The row must be one that waits: the state is the one of a draft that
-- waits for the yes of its owner. A draft that is confirmed or that is
-- cancelled already does not wait any more, and the row is not touched.
-- The vocabulary of the states belongs to the domain, and the caller sends
-- both of them.
UPDATE purchase_drafts
SET state = sqlc.arg(state), updated_at = now()
WHERE id = sqlc.arg(id) AND state = sqlc.arg(expected_state)
RETURNING *;

-- name: ListPurchaseDraftsByBusiness :many
--
-- The drafts of a business and of no other one, in the order of their
-- identity. The state is the one of a draft that waits: the domain decides
-- which of them are still pending, because the expiry is computed when the
-- draft is read.
SELECT * FROM purchase_drafts
WHERE business_id = sqlc.arg(business_id) AND state = sqlc.arg(state)
ORDER BY id;
