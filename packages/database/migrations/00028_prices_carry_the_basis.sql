-- The price carries the WAY THE OWNER CHARGES.
--
-- The money of a price is only a price when the book knows what it is the
-- price of: three thousand pesos is not a price until the book knows
-- whether it is per loaf, per gram or per kilogram. The way she charges
-- belongs to the domain of the purchase, and the column carries the three
-- names the book knows.
--
-- The domain refuses a price without its basis, and the table holds the
-- same rule as a defense in depth: the check `prices_basis_check` holds
-- that the basis of a price is one of the three ways of charging.
--
-- The column is added with the default of the first way of charging
-- because the column is not null and the rows that are already there
-- carry no basis: the default fills them and it is dropped right after,
-- so every row written from now on states its basis. The migration of
-- `00024_prices.sql` is not edited: it is already applied, and the history
-- of the schema is a history that is not rewritten.

-- +goose Up
ALTER TABLE prices
    ADD COLUMN basis TEXT NOT NULL DEFAULT 'unit';

ALTER TABLE prices
    ALTER COLUMN basis DROP DEFAULT;

ALTER TABLE prices
    ADD CONSTRAINT prices_basis_check CHECK (basis IN ('unit', 'gram', 'kilogram'));

-- +goose Down
ALTER TABLE prices
    DROP CONSTRAINT prices_basis_check;

ALTER TABLE prices
    DROP COLUMN basis;
