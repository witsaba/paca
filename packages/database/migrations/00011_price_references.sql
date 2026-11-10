-- The reference prices: what the market was seen at, where it was read and
-- the day it was read.
--
-- A reference is NEVER the price of the ingredient: the price the owner
-- pays exists only in the market of her town and only in her mouth on the
-- day. The research of Paca returns public references with their source and
-- their day, and this table keeps them as what they are: references.
--
-- The reference belongs to the business that researched it, and it names
-- the ingredient of the catalog it is about: the reference never carries a
-- second copy of the flour.
--
-- The rules of the domain are that the price is positive, that the price
-- carries its unit, that the source is not empty and that the word of the
-- owner is not empty. The table holds the same rules as a defense in
-- depth.
--
-- The money of the book counts whole pesos: the price of a reference is
-- whole pesos, like every other price of the book.

-- +goose Up
CREATE TABLE price_references (
    id             BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id    BIGINT      NOT NULL REFERENCES businesses (id),
    ingredient_id  BIGINT      NOT NULL REFERENCES ingredients (id),
    price          BIGINT      NOT NULL,
    unit           TEXT        NOT NULL,
    market         TEXT        NULL,
    source         TEXT        NOT NULL,
    seen_on        DATE        NOT NULL,
    utterance      TEXT        NOT NULL DEFAULT '',
    confirmed_with TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT price_references_price_positive_check CHECK (price > 0),
    CONSTRAINT price_references_unit_not_empty_check CHECK (unit <> ''),
    CONSTRAINT price_references_source_not_empty_check CHECK (source <> ''),
    CONSTRAINT price_references_confirmed_with_not_empty_check CHECK (confirmed_with <> '')
);

-- +goose Down
DROP TABLE price_references;
