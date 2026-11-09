-- The units of the book in the lines of the orders.
--
-- The unit of a line of an order is THE WAY THE OWNER CHARGES: the piece,
-- the gram or the kilogram. The way she said it ("unidad", "gramo", "kilo")
-- is read by the domain, and the storage holds the way the book knows it,
-- the way the basis of a price is held (`00028_prices_carry_the_basis.sql`).
--
-- The domain refuses a unit the book does not know, and the table holds the
-- same rule as a defense in depth: the check `order_lines_unit_check` holds
-- that the unit of a line is one of the three ways of charging.
--
-- The quantity of a line travels with the way she charges: the quantity of
-- a line by the PIECE is a whole number of pieces — the book never splits a
-- piece — and the quantity of a line by the weight is any positive amount
-- of weight: the book never converts a weight into a number of pieces. The
-- check `order_lines_piece_quantity_check` holds the first of the two rules.
--
-- The column of the quantity was an integer of thirty-two bits, and a
-- weight is not a whole number of grams: it becomes a double precision, the
-- way the quantities of the lines of the purchases and of the layers of the
-- stock are held.
--
-- The migration of the orders is not edited: it is already applied, and the
-- history of the schema is a history that is not rewritten.

-- +goose Up
ALTER TABLE order_lines
    ALTER COLUMN quantity TYPE DOUBLE PRECISION;

ALTER TABLE order_lines
    DROP CONSTRAINT order_lines_unit_not_empty_check;

ALTER TABLE order_lines
    ADD CONSTRAINT order_lines_unit_check CHECK (unit IN ('unit', 'gram', 'kilogram'));

ALTER TABLE order_lines
    ADD CONSTRAINT order_lines_piece_quantity_check CHECK (unit <> 'unit' OR quantity = trunc(quantity));

-- +goose Down
ALTER TABLE order_lines
    DROP CONSTRAINT order_lines_piece_quantity_check;

ALTER TABLE order_lines
    DROP CONSTRAINT order_lines_unit_check;

ALTER TABLE order_lines
    ADD CONSTRAINT order_lines_unit_not_empty_check CHECK (unit <> '');

ALTER TABLE order_lines
    ALTER COLUMN quantity TYPE INTEGER USING round(quantity)::integer;
