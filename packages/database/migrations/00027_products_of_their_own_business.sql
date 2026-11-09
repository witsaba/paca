-- A product is backed by a recipe of ITS OWN business, and a price is of a
-- product of its own business.
--
-- The domain refuses the product backed by the recipe of another business,
-- and the table holds the same rule as a defense in depth: the composite
-- foreign keys of this migration refuse the product of a foreign recipe and
-- the price of a foreign product.
--
-- The two independent foreign keys of `00024_prices.sql` cannot hold the
-- rule: a recipe and a product may each exist, and the pair may be of two
-- different businesses. The composite foreign keys carry the business of
-- the pair, and they need a unique index on the columns they point at:
-- `recipes_business_and_id` and `products_business_and_id`.
--
-- The migration of `00024_prices.sql` is not edited: it is already applied,
-- and the history of the schema is a history that is not rewritten. The
-- Down of this migration restores the rules of `00024_prices.sql`.

-- +goose Up
CREATE UNIQUE INDEX recipes_business_and_id ON recipes (business_id, id);

CREATE UNIQUE INDEX products_business_and_id ON products (business_id, id);

ALTER TABLE products
    ADD CONSTRAINT products_recipe_of_its_business_fkey
    FOREIGN KEY (business_id, recipe_id) REFERENCES recipes (business_id, id);

ALTER TABLE prices
    ADD CONSTRAINT prices_product_of_its_business_fkey
    FOREIGN KEY (business_id, product_id) REFERENCES products (business_id, id);

-- +goose Down
ALTER TABLE prices
    DROP CONSTRAINT prices_product_of_its_business_fkey;

ALTER TABLE products
    DROP CONSTRAINT products_recipe_of_its_business_fkey;

DROP INDEX products_business_and_id;

DROP INDEX recipes_business_and_id;
