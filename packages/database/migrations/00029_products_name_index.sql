-- The name of a product is taken inside the business, and the index is in
-- the house style of the tree.
--
-- The index of `00024_prices.sql` was over the lower case of the name, and
-- the house style for the same rule is the lower case of the TRIMMED name
-- (`00008_ingredients.sql`): the domain compares the name the way a person
-- reads it, trimmed and without case, and the index of the table holds the
-- same rule.
--
-- The difference has no behavior of its own today: the domain trims the
-- name before it is stored, and the two indexes hold the same rows. The
-- index is put in the house style anyway, so that the table and the
-- domain read the name the same way.
--
-- The migration of `00024_prices.sql` is not edited: it is already applied,
-- and the history of the schema is a history that is not rewritten.

-- +goose Up
DROP INDEX products_one_name_per_business;

CREATE UNIQUE INDEX products_one_name_per_business ON products (business_id, lower(btrim(name)));

-- +goose Down
DROP INDEX products_one_name_per_business;

CREATE UNIQUE INDEX products_one_name_per_business ON products (business_id, lower(name));
