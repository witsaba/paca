-- The first table of PACA: the person that uses it.
--
-- The primary key is the national document ID of the person, and the caller
-- supplies it. Nothing in the database generates it: the number is not
-- sequential because it is not a counter, it is the number the person
-- already carries.
--
-- The status is checked by the database, so no writer can store a state that
-- does not exist. The four states are the states of the domain, and the
-- domain refuses a fifth one too.
--
-- The optional columns are NULL when the value is not known. An empty
-- string is a different fact, and the domain never writes one.
--
-- The timestamps carry the time of the system. The database fills them on
-- insert; whoever updates a row is responsible for updated_at after that.

-- +goose Up
CREATE TABLE users (
    id                  BIGINT      NOT NULL,
    full_name           TEXT        NULL,
    status              TEXT        NOT NULL DEFAULT 'onboarding',
    status_observations TEXT        NULL,
    phone               TEXT        NULL,
    email               TEXT        NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_status_check CHECK (status IN ('onboarding', 'active', 'inactive', 'blocked'))
);

-- +goose Down
DROP TABLE users;
