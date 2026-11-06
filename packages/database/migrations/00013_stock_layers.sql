-- The layers of the stock: what a purchase brought, and what is left of it.
--
-- The purchase is the fact of what the business bought. The layer is
-- derived from it: it is the part of that purchase that the business still
-- holds. Nobody types a layer, and the layer never becomes a second truth
-- about what was bought.
--
-- The layer carries the ingredient it is of, the quantity that was bought
-- with its unit, the money it cost, the day it was bought and the purchase
-- it comes from. The position of the layer is the position of the line of
-- the purchase it comes from, and it is unique inside the purchase: a
-- layer is derived from one line and from no other.
--
-- The quantity and its unit travel together: the unit is one of the four
-- the book knows, and the quantity that is left of the layer is never more
-- than the quantity that was bought.
--
-- The money is whole pesos: the book counts the pesos the person has, and
-- the money of a layer is the money that was paid for it.
--
-- The rules of the domain are that the name is not empty, that the unit is
-- one of the units of the book, that the quantity is positive, that what
-- is left of the layer is not more than what was bought and that the money
-- is not negative. The table holds the same rules as a defense in depth.

-- +goose Up
CREATE TABLE stock_layers (
    id            BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id   BIGINT             NOT NULL REFERENCES businesses (id),
    ingredient_id BIGINT             NOT NULL REFERENCES ingredients (id),
    purchase_id   BIGINT             NOT NULL REFERENCES purchases (id) ON DELETE CASCADE,
    position      INTEGER            NOT NULL,
    name          TEXT               NOT NULL,
    day           DATE               NOT NULL,
    quantity      DOUBLE PRECISION   NOT NULL,
    unit          TEXT               NOT NULL,
    remaining     DOUBLE PRECISION   NOT NULL,
    cost          BIGINT             NOT NULL,
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT stock_layers_name_not_empty_check CHECK (name <> ''),
    CONSTRAINT stock_layers_unit_check CHECK (unit IN ('g', 'kg', 'paquete', 'unidad')),
    CONSTRAINT stock_layers_quantity_positive_check CHECK (quantity > 0),
    CONSTRAINT stock_layers_remaining_check CHECK (remaining >= 0 AND remaining <= quantity),
    CONSTRAINT stock_layers_cost_not_negative_check CHECK (cost >= 0),
    CONSTRAINT stock_layers_position_positive_check CHECK (position > 0)
);

CREATE UNIQUE INDEX stock_layers_one_per_line ON stock_layers (purchase_id, position);

CREATE INDEX stock_layers_of_the_business ON stock_layers (business_id, day, id);

-- +goose Down
DROP INDEX stock_layers_of_the_business;
DROP INDEX stock_layers_one_per_line;

DROP TABLE stock_layers;
