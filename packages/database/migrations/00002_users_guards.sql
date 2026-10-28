-- The guards of the table of the users.
--
-- The first migration created the table. This one holds the rules that the
-- rows themselves have to obey, so that no writer -- not the library, not a
-- future service, not a session of psql -- can break them.
--
-- The email and the phone are unique when they are set. A person may have no
-- email and no phone, and two people without an email do not collide: the
-- indexes are partial, they only cover the rows that carry a value.
--
-- The email is compared without case. Two addresses that differ only in the
-- case of the name are the same mailbox, and the mailbox belongs to one
-- person.
--
-- The formats are checked too: a phone holds digits only, an address holds
-- the shape of an address, and no optional column holds an empty string. An
-- empty string is a different fact from NULL, and the domain never writes
-- one.
--
-- The observation belongs to the status: the two closed states carry the
-- reason why the person is not using the product, and the two open states
-- carry none.

-- +goose Up
CREATE UNIQUE INDEX users_email_key ON users (lower(email)) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX users_phone_key ON users (phone) WHERE phone IS NOT NULL;

ALTER TABLE users ADD CONSTRAINT users_phone_format_check
    CHECK (phone IS NULL OR phone ~ '^[0-9]{11,15}$');

ALTER TABLE users ADD CONSTRAINT users_email_format_check
    CHECK (email IS NULL OR email ~ '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$');

ALTER TABLE users ADD CONSTRAINT users_optional_not_empty_check
    CHECK ((full_name IS NULL OR full_name <> '')
       AND (phone    IS NULL OR phone    <> '')
       AND (email    IS NULL OR email    <> ''));

ALTER TABLE users ADD CONSTRAINT users_status_observation_check
    CHECK ((status IN ('inactive', 'blocked')
            AND status_observations IS NOT NULL
            AND status_observations <> '')
        OR (status IN ('onboarding', 'active')
            AND status_observations IS NULL));

-- +goose Down
DROP INDEX users_email_key;
DROP INDEX users_phone_key;

ALTER TABLE users DROP CONSTRAINT users_status_observation_check;
ALTER TABLE users DROP CONSTRAINT users_optional_not_empty_check;
ALTER TABLE users DROP CONSTRAINT users_email_format_check;
ALTER TABLE users DROP CONSTRAINT users_phone_format_check;
