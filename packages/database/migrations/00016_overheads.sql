-- The overhead of the house of the bakery: the electricity the owner paid,
-- the hours the oven ran and the allocation she declared.
--
-- The overhead is the electricity of the house and the hours of the oven:
-- the owner declares the share of the electricity that belongs to the oven,
-- and the money of the overhead is derived from the electricity and the
-- allocation. The allocation is never inferred: the money of the house is
-- not a number PACA may invent.
--
-- The money of the overhead is not stored: it is derived from the
-- electricity and the allocation, the way the money of the labor is derived
-- from the hours and the value of the hour. A second truth about the same
-- money is a truth that drifts.
--
-- The overhead is a fact of the business: it moves no money of the ledger
-- yet, and the account of the overhead exists in the seed of the business.
--
-- The rules of the domain are that the electricity is positive, that the
-- hours of the oven are positive, that the allocation is a share of the
-- electricity and that the word of the owner is not empty. The table holds
-- the same rules as a defense in depth.

-- +goose Up
CREATE TABLE overheads (
    id             BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id    BIGINT      NOT NULL REFERENCES businesses (id),
    day            DATE        NOT NULL,
    electricity    BIGINT      NOT NULL,
    oven_hours     DOUBLE PRECISION NOT NULL,
    allocation     DOUBLE PRECISION NOT NULL,
    utterance      TEXT        NOT NULL DEFAULT '',
    confirmed_with TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT overheads_electricity_positive_check CHECK (electricity > 0),
    CONSTRAINT overheads_oven_hours_positive_check CHECK (oven_hours > 0),
    CONSTRAINT overheads_allocation_share_check CHECK (allocation > 0 AND allocation <= 1),
    CONSTRAINT overheads_confirmed_with_not_empty_check CHECK (confirmed_with <> '')
);

-- +goose Down
DROP TABLE overheads;
