-- The sale of the bakery: what she sold, and what it cost her.
--
-- The sale is a movement of the book, and it is recorded as one from the
-- start: the product, the quantity, the unit, the money the client paid
-- and the cost the sale released are what the owner said and what the book
-- derived.
--
-- The rules of the domain are that the business and the product are there,
-- that the quantity is positive and whole, that the unit is the unit of a
-- piece, that the money is positive, that the cost is not negative and
-- that the word of the owner is not empty. The table holds the same rules
-- as a defense in depth.
--
-- The exits of the book live in one table: `unit_exits`. A unit leaves
-- the book once, whichever exit it takes - the waste or the sale - and
-- the primary key over the unit holds that rule for both of them. The
-- rows of the waste that already left the book move here with the
-- migration, so the rule of the book holds across the exits from now on.

-- +goose Up
CREATE TABLE unit_exits (
    unit_id       BIGINT             PRIMARY KEY REFERENCES production_units (id),
    exit_kind     TEXT               NOT NULL,
    exit_id       BIGINT             NOT NULL,
    production_id BIGINT             NOT NULL REFERENCES productions (id),
    position      INTEGER            NOT NULL,
    grams         DOUBLE PRECISION   NOT NULL,
    expires_on    DATE               NOT NULL,
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT unit_exits_kind_check CHECK (exit_kind IN ('waste', 'sale')),
    CONSTRAINT unit_exits_position_positive_check CHECK (position > 0),
    CONSTRAINT unit_exits_grams_positive_check CHECK (grams > 0)
);

CREATE UNIQUE INDEX unit_exits_one_position ON unit_exits (exit_kind, exit_id, production_id, position);

INSERT INTO unit_exits (unit_id, exit_kind, exit_id, production_id, position, grams, expires_on)
SELECT unit_id, 'waste', waste_id, production_id, position, grams, expires_on
FROM waste_units;

DROP TABLE waste_units;

CREATE TABLE sales (
    id             BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id    BIGINT             NOT NULL REFERENCES businesses (id),
    recipe_id      BIGINT             NOT NULL REFERENCES recipes (id),
    day            DATE               NOT NULL,
    quantity       DOUBLE PRECISION   NOT NULL,
    unit           TEXT               NOT NULL,
    money          BIGINT             NOT NULL,
    cost           BIGINT             NOT NULL,
    utterance      TEXT               NOT NULL DEFAULT '',
    confirmed_with TEXT               NOT NULL,
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT sales_quantity_positive_check CHECK (quantity > 0),
    CONSTRAINT sales_quantity_whole_check CHECK (quantity = trunc(quantity)),
    CONSTRAINT sales_unit_check CHECK (unit = 'unidad'),
    CONSTRAINT sales_money_positive_check CHECK (money > 0),
    CONSTRAINT sales_cost_not_negative_check CHECK (cost >= 0),
    CONSTRAINT sales_confirmed_with_not_empty_check CHECK (confirmed_with <> '')
);

-- +goose Down
DROP TABLE sales;
DROP INDEX unit_exits_one_position;

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

INSERT INTO waste_units (waste_id, unit_id, production_id, position, grams, expires_on)
SELECT exit_id, unit_id, production_id, position, grams, expires_on
FROM unit_exits
WHERE exit_kind = 'waste';

DROP TABLE unit_exits;
