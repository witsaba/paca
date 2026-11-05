-- The recipe of the bakery: the guide to learn and to teach, and not a
-- list of ingredients.
--
-- A recipe carries the name the owner reads, the ingredients with their
-- grams, the steps with their times, their temperatures and their cues,
-- the yield of what it makes and the shelf life of it.
--
-- The yield is a declared constant: the number of units the recipe makes
-- and the weight of each one. It is never measured per bake, and the
-- production of a day is a fact of its own.
--
-- The ingredients of a recipe and its steps are rows of their own tables:
-- the position of a row is the order the owner said it in, and it is
-- unique inside the recipe. A recipe without its ingredients and its steps
-- is not a recipe.
--
-- The ingredient of a recipe is the ingredient of the catalog: the recipe
-- carries the identity of the ingredient and the grams of it, and never a
-- second copy of the flour. The table holds the rule as a defense in
-- depth.
--
-- The rules of the domain are that the name is not empty, that the word of
-- the confirmation is not empty, that the yield is positive, that the shelf
-- life is positive, that the grams of an ingredient are positive and that
-- the instruction of a step is not empty. The table holds the same rules as
-- a defense in depth.

-- +goose Up
CREATE TABLE recipes (
    id              BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id     BIGINT             NOT NULL REFERENCES businesses (id),
    name            TEXT               NOT NULL,
    yield_units     INTEGER            NOT NULL,
    yield_grams     DOUBLE PRECISION   NOT NULL,
    shelf_life_days INTEGER            NOT NULL,
    utterance       TEXT               NOT NULL DEFAULT '',
    confirmed_with  TEXT               NOT NULL,
    created_at      TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT recipes_name_not_empty_check CHECK (name <> ''),
    CONSTRAINT recipes_confirmed_with_not_empty_check CHECK (confirmed_with <> ''),
    CONSTRAINT recipes_yield_units_positive_check CHECK (yield_units > 0),
    CONSTRAINT recipes_yield_grams_positive_check CHECK (yield_grams > 0),
    CONSTRAINT recipes_shelf_life_days_positive_check CHECK (shelf_life_days > 0)
);

CREATE TABLE recipe_ingredients (
    id            BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    recipe_id     BIGINT             NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    position      INTEGER            NOT NULL,
    ingredient_id BIGINT             NOT NULL REFERENCES ingredients (id),
    grams         DOUBLE PRECISION   NOT NULL,

    CONSTRAINT recipe_ingredients_grams_positive_check CHECK (grams > 0),
    CONSTRAINT recipe_ingredients_position_positive_check CHECK (position > 0)
);

CREATE UNIQUE INDEX recipe_ingredients_one_position ON recipe_ingredients (recipe_id, position);

CREATE TABLE recipe_steps (
    id          BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    recipe_id   BIGINT             NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    position    INTEGER            NOT NULL,
    instruction TEXT               NOT NULL,
    cue         TEXT               NOT NULL DEFAULT '',
    minutes     INTEGER            NOT NULL DEFAULT 0,
    temperature DOUBLE PRECISION   NOT NULL DEFAULT 0,

    CONSTRAINT recipe_steps_instruction_not_empty_check CHECK (instruction <> ''),
    CONSTRAINT recipe_steps_minutes_not_negative_check CHECK (minutes >= 0),
    CONSTRAINT recipe_steps_temperature_not_below_absolute_zero_check CHECK (temperature >= -273.15),
    CONSTRAINT recipe_steps_position_positive_check CHECK (position > 0)
);

CREATE UNIQUE INDEX recipe_steps_one_position ON recipe_steps (recipe_id, position);

-- +goose Down
DROP TABLE recipe_steps;
DROP TABLE recipe_ingredients;
DROP TABLE recipes;
