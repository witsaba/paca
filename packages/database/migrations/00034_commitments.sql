-- The commitments of the bakery: the promises of the business, what is due
-- and the day it is due.
--
-- A commitment is a PROMISE of the business and not a sale and not an
-- order: the order already owns the client delivery — the party, the
-- products, the money and the round — and the promise never copies it. The
-- table carries NO money and NO party of its own: the "what" is the promise
-- in the words of the owner, and it may name the party in her own words.
--
-- The table carries NO state column. The day of meeting IS the fact: a row
-- with no `met_on` is an open promise and a row that carries it is a promise
-- that was kept, and a state column would be a second truth for the same
-- fact.
--
-- A promise is NEVER DELETED: the promise that was kept stays in the book
-- with the day it was kept, and there is no query of this table that deletes
-- a row.
--
-- The ALARM is the work of the agent. The table holds the fact of what is
-- due and the day it is due, and the agent is the one that rings: it reads
-- the calendar of the day and it says what is due and what is still owed.
--
-- The `what` is never empty: the check `commitments_what_check` holds the
-- rule as a defense in depth, and the domain refuses it first with its own
-- error (`commitment.ErrMissingWhat`). A promise that says nothing about
-- what is due is not a promise.

-- +goose Up
CREATE TABLE commitments (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT      NOT NULL REFERENCES businesses (id),
    what        TEXT        NOT NULL,
    due_on      DATE        NOT NULL,
    met_on      DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT commitments_what_check CHECK (btrim(what) <> '')
);

CREATE INDEX commitments_of_their_business ON commitments (business_id, id);

-- +goose Down
DROP INDEX commitments_of_their_business;

DROP TABLE commitments;
