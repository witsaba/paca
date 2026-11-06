-- The production of the bakery: what came out of the oven on one day.
--
-- The production is a fact of the business: the recipe it was made of, the
-- day it was made, the grams that went in and the units that came out. It
-- is written with the word of the owner, the way the recipe and the
-- purchase are written, and the sentence of the owner is kept with it.
--
-- The layers the production consumed are rows of their own table: they are
-- the trace of the FIFO, and the money of each of them is the money the
-- layer gave. The position of a row is the order the layers left the stock
-- in, and it is unique inside the production.
--
-- The units the production made are rows of their own table: every unit
-- carries the day it expires, which is the day of the production plus the
-- shelf life of the recipe. The shelf life is approximate until the owner
-- corrects it.
--
-- The rules of the domain are that the business and the recipe are there,
-- that the day is there, that the grams that went in are positive, that the
-- money is not negative and that the word of the owner is not empty. The
-- table holds the same rules as a defense in depth.

-- +goose Up
CREATE TABLE productions (
    id             BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id    BIGINT             NOT NULL REFERENCES businesses (id),
    recipe_id      BIGINT             NOT NULL REFERENCES recipes (id),
    day            DATE               NOT NULL,
    grams_in       DOUBLE PRECISION   NOT NULL,
    money          BIGINT             NOT NULL,
    utterance      TEXT               NOT NULL DEFAULT '',
    confirmed_with TEXT               NOT NULL,
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT productions_grams_in_positive_check CHECK (grams_in > 0),
    CONSTRAINT productions_money_not_negative_check CHECK (money >= 0),
    CONSTRAINT productions_confirmed_with_not_empty_check CHECK (confirmed_with <> '')
);

CREATE TABLE production_layers (
    id            BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    production_id BIGINT             NOT NULL REFERENCES productions (id) ON DELETE CASCADE,
    position      INTEGER            NOT NULL,
    layer_id      BIGINT             NOT NULL REFERENCES stock_layers (id),
    purchase_id   BIGINT             NOT NULL REFERENCES purchases (id),
    quantity      DOUBLE PRECISION   NOT NULL,
    unit          TEXT               NOT NULL,
    grams         DOUBLE PRECISION   NOT NULL,
    money         BIGINT             NOT NULL,

    CONSTRAINT production_layers_position_positive_check CHECK (position > 0),
    CONSTRAINT production_layers_quantity_positive_check CHECK (quantity > 0),
    CONSTRAINT production_layers_grams_positive_check CHECK (grams > 0),
    CONSTRAINT production_layers_money_not_negative_check CHECK (money >= 0),
    CONSTRAINT production_layers_unit_check CHECK (unit IN ('g', 'kg', 'paquete', 'unidad'))
);

CREATE UNIQUE INDEX production_layers_one_position ON production_layers (production_id, position);

CREATE TABLE production_units (
    id            BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    production_id BIGINT             NOT NULL REFERENCES productions (id) ON DELETE CASCADE,
    position      INTEGER            NOT NULL,
    grams         DOUBLE PRECISION   NOT NULL,
    expires_on    DATE               NOT NULL,

    CONSTRAINT production_units_position_positive_check CHECK (position > 0),
    CONSTRAINT production_units_grams_positive_check CHECK (grams > 0)
);

CREATE UNIQUE INDEX production_units_one_position ON production_units (production_id, position);

-- +goose Down
DROP INDEX production_units_one_position;
DROP TABLE production_units;

DROP INDEX production_layers_one_position;
DROP TABLE production_layers;

DROP TABLE productions;
