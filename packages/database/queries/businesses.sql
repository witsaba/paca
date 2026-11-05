-- The queries of the businesses.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may write
-- it. The write tells the truth by itself -- the identity of the business,
-- the partial unique index of the personal business -- and the library maps
-- that truth to its own errors.
--
-- The personal business of a person is read by its owner: the kind is part
-- of the query, and the row it refuses to read is the row of another kind.

-- name: CreateBusiness :one
INSERT INTO businesses (owner_id, kind, name, currency)
VALUES (sqlc.arg(owner_id), sqlc.arg(kind), sqlc.arg(name), sqlc.arg(currency))
RETURNING *;

-- name: GetBusinessByID :one
SELECT * FROM businesses WHERE id = sqlc.arg(id);

-- name: GetPersonalBusinessByOwner :one
SELECT * FROM businesses WHERE owner_id = sqlc.arg(owner_id) AND kind = 'personal';

-- name: ListBusinessesByOwner :many
SELECT * FROM businesses WHERE owner_id = sqlc.arg(owner_id) ORDER BY id;
