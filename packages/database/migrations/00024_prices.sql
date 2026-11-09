-- The price list of the bakery: the product, the price and the channel that
-- buys it.
--
-- The product of the bakery IS the thing it sells: it carries the recipe
-- it comes from - the bakery sells what it bakes - and the name the owner
-- reads. The product is never a second copy of the recipe: the column
-- carries the identity of it and nothing else.
--
-- The tax treatment of a product is an ATTRIBUTE OF THE PRODUCT, and it is
-- OPTIONAL: the column is NULL when the owner has not decided it, and the
-- book never derives it from the price and never assumes a rate. The
-- treatment is the fact the owner states, and nothing more.
--
-- The name of a product is taken inside the business, and one recipe backs
-- one product: the indexes of the table hold the rules of the domain as a
-- defense in depth.
--
-- The price is a FACT of the business, and it never becomes a column of
-- the inventory, of the recipe or of the stock: it lives in this table,
-- with a row of its own.
--
-- The CHANNEL is already in the price from the first row: every price
-- carries the channel it was set for, so that the discount of the wholesale
-- never has to rebuild the history of the prices. There is no "the price
-- without a channel" and there is no default channel.
--
-- The price list is a HISTORY: the rows are kept, nothing is rewritten when
-- a price changes, and the price that holds today for a product and a
-- channel is the last one set on or before the day that is asked for. Two
-- prices of the same day are both kept: the last one that was set is the
-- one that holds.
--
-- The money of a price is whole pesos: the money of the book counts the
-- pesos the person has, and the check of the table holds the rule of the
-- domain as a defense in depth.

-- +goose Up
CREATE TABLE products (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id   BIGINT      NOT NULL REFERENCES businesses (id),
    recipe_id     BIGINT      NOT NULL REFERENCES recipes (id),
    name          TEXT        NOT NULL,
    tax_treatment TEXT        NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT products_name_not_empty_check CHECK (name <> ''),
    CONSTRAINT products_tax_treatment_not_empty_check CHECK (tax_treatment IS NULL OR tax_treatment <> '')
);

CREATE UNIQUE INDEX products_one_name_per_business ON products (business_id, lower(name));

CREATE UNIQUE INDEX products_one_recipe_per_business ON products (business_id, recipe_id);

CREATE TABLE prices (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT      NOT NULL REFERENCES businesses (id),
    product_id  BIGINT      NOT NULL REFERENCES products (id),
    channel     TEXT        NOT NULL,
    amount      BIGINT      NOT NULL,
    day         DATE        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT prices_channel_check CHECK (channel IN ('retail', 'wholesale')),
    CONSTRAINT prices_amount_positive_check CHECK (amount > 0)
);

-- The price list is read for a business, a product and a channel, and the
-- price that holds is the last one set on or before the day that is asked
-- for.
CREATE INDEX prices_of_the_business ON prices (business_id, product_id, channel, day DESC, id DESC);

-- +goose Down
DROP INDEX prices_of_the_business;

DROP TABLE prices;

DROP INDEX products_one_recipe_per_business;

DROP INDEX products_one_name_per_business;

DROP TABLE products;
