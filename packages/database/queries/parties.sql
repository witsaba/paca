-- The queries of the parties of the business.
--
-- The roles of a party travel as JSONB: they are the roles the party holds
-- in the business, and the adapter encodes them. The legal human that
-- represents a store or a farm travels in the party it belongs to: it is a
-- fact of the party, and it is never a row of the people of PACA.
--
-- The optional columns are NULL when the value is not set. The queries use
-- sqlc.narg for them, so the library passes NULL and the column keeps the
-- rule of the domain: an empty value is an absent value.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.

-- name: ListPartiesByBusiness :many
SELECT * FROM parties WHERE business_id = sqlc.arg(business_id) ORDER BY id;

-- name: CreateParty :one
INSERT INTO parties (business_id, kind, name, roles, phone, email, address, legal_human_name, legal_figure)
VALUES (sqlc.arg(business_id), sqlc.arg(kind), sqlc.arg(name), sqlc.arg(roles),
        sqlc.narg(phone)::text, sqlc.narg(email)::text, sqlc.narg(address)::text,
        sqlc.narg(legal_human_name)::text, sqlc.narg(legal_figure)::text)
RETURNING *;
