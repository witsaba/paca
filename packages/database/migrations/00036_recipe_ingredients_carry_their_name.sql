-- The ingredients of a recipe carry the name the owner says.
--
-- A recipe is a guide to learn, to teach and to publish, and it is written
-- before the first purchase: the name of an ingredient of a recipe is the
-- name the owner says, and the identity of the catalog is a link that
-- exists only when the business has already bought that ingredient.
--
-- The column 'name' holds the name the owner says, and it is not empty:
-- the table holds the rule of the domain as a defense in depth. The link
-- 'ingredient_id' may be empty, and when it is empty the cost of that
-- ingredient is unknown and never zero.
--
-- The rows that were written before this migration keep the name the
-- catalog holds for them, so no row loses the name it had.

-- +goose Up
ALTER TABLE recipe_ingredients ADD COLUMN name TEXT NOT NULL DEFAULT '';

UPDATE recipe_ingredients
SET name = ingredients.name
FROM ingredients
WHERE ingredients.id = recipe_ingredients.ingredient_id;

ALTER TABLE recipe_ingredients ALTER COLUMN name DROP DEFAULT;

ALTER TABLE recipe_ingredients ALTER COLUMN ingredient_id DROP NOT NULL;

ALTER TABLE recipe_ingredients
    ADD CONSTRAINT recipe_ingredients_name_not_empty_check CHECK (name <> '');

ALTER TABLE recipe_ingredients
    ADD CONSTRAINT recipe_ingredients_link_positive_check CHECK (ingredient_id IS NULL OR ingredient_id > 0);

-- +goose Down
ALTER TABLE recipe_ingredients DROP CONSTRAINT recipe_ingredients_link_positive_check;

ALTER TABLE recipe_ingredients DROP CONSTRAINT recipe_ingredients_name_not_empty_check;

DELETE FROM recipe_ingredients WHERE ingredient_id IS NULL;

ALTER TABLE recipe_ingredients ALTER COLUMN ingredient_id SET NOT NULL;

ALTER TABLE recipe_ingredients DROP COLUMN name;
