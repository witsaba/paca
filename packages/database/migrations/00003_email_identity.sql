-- The identity of the email address.
--
-- The table already refused two addresses that differ only in the case of the
-- name, and it does so with an index on lower(email). The domain now owns
-- that rule: it stores the canonical form of the address, in lower case, so
-- the two addresses are one value of the domain and never two.
--
-- This migration brings the rows that exist to the canonical form, and it
-- holds the rule as a constraint of the table: the table guards against the
-- writer that does not know the domain, the same way the index and the
-- checks of the second migration do.
--
-- The update refuses a mailbox that two people carry: the unique index
-- refuses it and the migration fails with it. Which of the two rows keeps
-- the address is a fact about the people, and only a human decides it.

-- +goose Up
UPDATE users SET email = lower(email) WHERE email IS NOT NULL AND email <> lower(email);

ALTER TABLE users ADD CONSTRAINT users_email_canonical_check
    CHECK (email IS NULL OR email = lower(email));

-- +goose Down
-- The case that the addresses had before is not a fact the database keeps:
-- the canonical form is the only form the domain writes, and the migration
-- down does not invent one.
ALTER TABLE users DROP CONSTRAINT users_email_canonical_check;
