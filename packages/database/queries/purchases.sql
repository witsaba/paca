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
