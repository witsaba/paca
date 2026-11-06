-- The correction of the shelf life of a recipe.
--
-- The shelf life of a recipe is a declared constant, and it is approximate:
-- the owner does not know it yet, and she corrects it when she knows it
-- better. The correction is a fact of the business, and it is kept: every
-- correction is a row of its own, and the last one the owner declared is
-- the shelf life that counts.
--
-- The correction carries the day it was declared and the word of the owner,
-- the way the recipe and the production do. The sentence of the owner is
-- kept with it.
--
-- The rules of the domain are that the business and the recipe are there,
-- that the number of days is positive, that the day is there and that the
-- word of the owner is not empty. The table holds the same rules as a
-- defense in depth.

-- +goose Up
CREATE TABLE recipe_shelf_life_corrections (
    id              BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id     BIGINT             NOT NULL REFERENCES businesses (id),
    recipe_id       BIGINT             NOT NULL REFERENCES recipes (id),
    shelf_life_days INTEGER            NOT NULL,
    day             DATE               NOT NULL,
    utterance       TEXT               NOT NULL DEFAULT '',
    confirmed_with  TEXT               NOT NULL,
    created_at      TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT recipe_shelf_life_corrections_days_positive_check CHECK (shelf_life_days > 0),
    CONSTRAINT recipe_shelf_life_corrections_confirmed_with_not_empty_check CHECK (confirmed_with <> '')
);

-- +goose Down
DROP TABLE recipe_shelf_life_corrections;
