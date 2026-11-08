-- The waste of the bakery: what she lost, and what it was worth.
--
-- The waste is a movement of the book, and it is recorded as one from the
-- start: the product, the quantity, the unit and the reason are what the
-- owner said, and the reason is what a later analysis reads.
--
-- The units that left the book are rows of their own table: they are the
-- trace of the rotation of the output by the expiry, and the unit that
-- expired first is the first one that left. The position of a row is the
-- position the unit had in the production it came out of, and it is unique
-- inside the waste and inside that production. A unit leaves the book only
-- once: the unique index over the unit holds that rule, and a unit that
-- already left is never taken again.
--
-- The rules of the domain are that the business and the product are there,
-- that the quantity is positive and whole, that the unit is the unit of a
-- piece, that the reason is one of the four of the book, that the money is
-- not negative and that the word of the owner is not empty. The table holds
-- the same rules as a defense in depth.

-- +goose Up
CREATE TABLE wastes (
    id             BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id    BIGINT             NOT NULL REFERENCES businesses (id),
    recipe_id      BIGINT             NOT NULL REFERENCES recipes (id),
    day            DATE               NOT NULL,
    quantity       DOUBLE PRECISION   NOT NULL,
    unit           TEXT               NOT NULL,
    reason         TEXT               NOT NULL,
    money          BIGINT             NOT NULL,
    utterance      TEXT               NOT NULL DEFAULT '',
    confirmed_with TEXT               NOT NULL,
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT wastes_quantity_positive_check CHECK (quantity > 0),
    CONSTRAINT wastes_quantity_whole_check CHECK (quantity = trunc(quantity)),
    CONSTRAINT wastes_unit_check CHECK (unit = 'unidad'),
    CONSTRAINT wastes_reason_check CHECK (reason IN ('burnt', 'spoiled', 'dropped', 'returned')),
    CONSTRAINT wastes_money_not_negative_check CHECK (money >= 0),
    CONSTRAINT wastes_confirmed_with_not_empty_check CHECK (confirmed_with <> '')
);

CREATE TABLE waste_units (
    id            BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    waste_id      BIGINT             NOT NULL REFERENCES wastes (id) ON DELETE CASCADE,
    unit_id       BIGINT             NOT NULL REFERENCES production_units (id),
    production_id BIGINT             NOT NULL REFERENCES productions (id),
    position      INTEGER            NOT NULL,
    grams         DOUBLE PRECISION   NOT NULL,
    expires_on    DATE               NOT NULL,

    CONSTRAINT waste_units_position_positive_check CHECK (position > 0),
    CONSTRAINT waste_units_grams_positive_check CHECK (grams > 0)
);

CREATE UNIQUE INDEX waste_units_one_position ON waste_units (waste_id, production_id, position);

CREATE UNIQUE INDEX waste_units_one_unit ON waste_units (unit_id);

-- +goose Down
DROP INDEX waste_units_one_unit;
DROP INDEX waste_units_one_position;
DROP TABLE waste_units;
DROP TABLE wastes;
