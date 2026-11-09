-- The orders of the bakery: what it owes, and the round that carries them.
--
-- An order is a fact of the business: the party that ordered, the day it is
-- due and the state of the order. The lines of the order are rows of their
-- own table: the products with the quantities of them and the unit the
-- owner said.
--
-- The order is written OPEN: an order that was not delivered stays open,
-- and the delivery of an order is written with the route of the round. The
-- table holds the two states of the book as a defense in depth, and the
-- domain is the one that decides which of them an order carries.
--
-- The party of the order is a party of the business that owns it, and the
-- products of the lines are products of the price list of that business:
-- the composite foreign keys of this migration hold the rule, the way
-- `00027_products_of_their_own_business.sql` holds it for the products and
-- their recipes.
--
-- The quantity of a line is a whole number of units: the book never splits
-- a unit, and it never converts a weight into a number of units.
--
-- The migration of the parties is not edited: the unique index of the
-- business and of the identity of the party is created here, the way
-- `00027_products_of_their_own_business.sql` created the ones of the
-- recipes and of the products.

-- +goose Up
CREATE UNIQUE INDEX parties_business_and_id ON parties (business_id, id);

CREATE TABLE orders (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT      NOT NULL REFERENCES businesses (id),
    party_id    BIGINT      NOT NULL REFERENCES parties (id),
    party_name  TEXT        NOT NULL,
    due_on      DATE        NOT NULL,
    state       TEXT        NOT NULL DEFAULT 'open',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT orders_party_name_not_empty_check CHECK (party_name <> ''),
    CONSTRAINT orders_state_check CHECK (state IN ('open', 'delivered'))
);

ALTER TABLE orders
    ADD CONSTRAINT orders_party_of_its_business_fkey
    FOREIGN KEY (business_id, party_id) REFERENCES parties (business_id, id);

CREATE TABLE order_lines (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id     BIGINT      NOT NULL REFERENCES orders (id),
    business_id  BIGINT      NOT NULL REFERENCES businesses (id),
    product_id   BIGINT      NOT NULL REFERENCES products (id),
    position     INTEGER     NOT NULL,
    product_name TEXT        NOT NULL,
    quantity     INTEGER     NOT NULL,
    unit         TEXT        NOT NULL,

    CONSTRAINT order_lines_position_check CHECK (position > 0),
    CONSTRAINT order_lines_quantity_check CHECK (quantity > 0),
    CONSTRAINT order_lines_product_name_not_empty_check CHECK (product_name <> ''),
    CONSTRAINT order_lines_unit_not_empty_check CHECK (unit <> '')
);

CREATE UNIQUE INDEX orders_business_and_id ON orders (business_id, id);

ALTER TABLE order_lines
    ADD CONSTRAINT order_lines_order_of_its_business_fkey
    FOREIGN KEY (business_id, order_id) REFERENCES orders (business_id, id);

ALTER TABLE order_lines
    ADD CONSTRAINT order_lines_product_of_its_business_fkey
    FOREIGN KEY (business_id, product_id) REFERENCES products (business_id, id);

CREATE UNIQUE INDEX order_lines_one_position_per_order ON order_lines (order_id, position);

-- +goose Down
DROP INDEX order_lines_one_position_per_order;

ALTER TABLE order_lines
    DROP CONSTRAINT order_lines_product_of_its_business_fkey;

ALTER TABLE order_lines
    DROP CONSTRAINT order_lines_order_of_its_business_fkey;

DROP INDEX orders_business_and_id;

DROP TABLE order_lines;

ALTER TABLE orders
    DROP CONSTRAINT orders_party_of_its_business_fkey;

DROP TABLE orders;

DROP INDEX parties_business_and_id;
