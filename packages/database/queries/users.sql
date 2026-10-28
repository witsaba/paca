-- The queries of the users.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may write
-- it. The write tells the truth by itself -- a primary key that is taken, a
-- unique index that refuses a value -- and the library maps that truth to its
-- own errors.
--
-- The optional columns are NULL when the value is not set. The queries use
-- sqlc.narg for them, so the library passes NULL and the column keeps the rule
-- of the domain: an empty value is an absent value.

-- name: CreateUser :one
INSERT INTO users (id, full_name, phone, email, status, status_observations)
VALUES (sqlc.arg(id), sqlc.narg(full_name)::text, sqlc.narg(phone)::text, sqlc.narg(email)::text, 'onboarding', NULL)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = sqlc.arg(id);

-- name: UpdateUserDetails :one
UPDATE users
   SET full_name  = COALESCE(sqlc.narg(full_name)::text, full_name),
       phone      = COALESCE(sqlc.narg(phone)::text, phone),
       email      = COALESCE(sqlc.narg(email)::text, email),
       updated_at = now()
 WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ChangeUserStatus :one
UPDATE users
   SET status              = sqlc.arg(status),
       status_observations = sqlc.narg(status_observations)::text,
       updated_at          = now()
 WHERE id = sqlc.arg(id)
   AND status = sqlc.arg(current_status)
RETURNING *;
