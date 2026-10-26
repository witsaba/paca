-- The ping asks the database whether it answers.
--
-- It touches no table: it is not a query of the domain and it reads
-- nothing. It exists because a service that checks the health of its
-- dependencies needs a real question to ask the database, and because the
-- generator needs one query to write the code of the connection.

-- name: Ping :one
SELECT 1 AS alive;
