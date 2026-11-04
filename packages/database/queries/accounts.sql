-- The queries of the accounts.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may write
-- it. The write tells the truth by itself -- the unique index of the name
-- inside the business -- and the library maps that truth to its own errors.
--
-- The optional columns are NULL when the value is not set. The queries use
-- sqlc.narg for them, so the library passes NULL and the column keeps the rule
-- of the domain: an empty value is an absent value.

-- name: CreateAccount :one
INSERT INTO accounts (business_id, kind, name, counterparty, due_date)
VALUES (sqlc.arg(business_id), sqlc.arg(kind), sqlc.arg(name),
        sqlc.narg(counterparty)::text, sqlc.narg(due_date)::date)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = sqlc.arg(id);

-- name: ListAccountsByBusiness :many
SELECT * FROM accounts WHERE business_id = sqlc.arg(business_id) ORDER BY id;
