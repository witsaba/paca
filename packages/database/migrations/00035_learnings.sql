-- The learning log of the bakery: the memory of the errors and the wins of
-- the owner.
--
-- An entry says what happened, in the words of the owner, and what was
-- learned from it. It is about a DAY and about the bread: the batch the
-- business bakes, and the run of that batch when the entry is about one run
-- of it.
--
-- The table carries NO money. The log is not a sale and it is not a movement
-- of the book: it is the memory the owner reads back and the thing PACA
-- learns from, and it never moves a peso.
--
-- The log is NEVER REWRITTEN. There is no update of the words of an entry
-- and no delete of a row: a correction is a NEW entry that answers the old
-- one, and the partial unique index of this migration refuses the second
-- answer of the same entry, because the log is a chain and not a tree, and
-- two answers of the same entry would leave the owner with two truths.
--
-- The answer carries the BUSINESS with it: the composite foreign key
-- `(answers_id, business_id)` over `(id, business_id)` refuses the answer of
-- an entry of another business, the way `00027_products_of_their_own_business.sql`
-- refuses the product of a foreign recipe. The unique index the composite key
-- needs is created here: `learnings_identity_and_business`.
--
-- The batch of an entry is a recipe of the business that owns it, and the
-- production of an entry is a production of that business AND a run of the
-- NAMED batch: the two composite foreign keys of the production hold both
-- rules, and they need the unique indexes `productions_business_and_id` and
-- `productions_identity_and_recipe`. The migration of the productions is not
-- edited: it is already applied, and the history of the schema is a history
-- that is not rewritten.
--
-- What the table CANNOT prove: that the day of the entry is a day of the
-- book. The table refuses a date that is not a date, and the domain is the
-- one that says whether a day is there at all.
--
-- The `happened` and the `learned` are never empty: the checks
-- `learnings_happened_check` and `learnings_learned_check` hold the rule as a
-- defense in depth, and the domain refuses it first (`learning.ErrMissingHappened`
-- and `learning.ErrMissingLearned`).

-- +goose Up
CREATE UNIQUE INDEX productions_business_and_id ON productions (business_id, id);

CREATE UNIQUE INDEX productions_identity_and_recipe ON productions (id, recipe_id);

CREATE TABLE learnings (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id   BIGINT      NOT NULL REFERENCES businesses (id),
    day           DATE        NOT NULL,
    recipe_id     BIGINT      NOT NULL,
    production_id BIGINT,
    happened      TEXT        NOT NULL,
    learned       TEXT        NOT NULL,
    answers_id    BIGINT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT learnings_happened_check CHECK (btrim(happened) <> ''),
    CONSTRAINT learnings_learned_check CHECK (btrim(learned) <> '')
);

CREATE UNIQUE INDEX learnings_identity_and_business ON learnings (id, business_id);

CREATE UNIQUE INDEX learnings_answered_once ON learnings (answers_id) WHERE answers_id IS NOT NULL;

CREATE INDEX learnings_of_their_business ON learnings (business_id, id);

ALTER TABLE learnings
    ADD CONSTRAINT learnings_batch_of_its_business_fkey
    FOREIGN KEY (business_id, recipe_id) REFERENCES recipes (business_id, id);

ALTER TABLE learnings
    ADD CONSTRAINT learnings_production_of_its_business_fkey
    FOREIGN KEY (business_id, production_id) REFERENCES productions (business_id, id);

ALTER TABLE learnings
    ADD CONSTRAINT learnings_production_of_its_batch_fkey
    FOREIGN KEY (production_id, recipe_id) REFERENCES productions (id, recipe_id);

ALTER TABLE learnings
    ADD CONSTRAINT learnings_answer_of_its_business_fkey
    FOREIGN KEY (answers_id, business_id) REFERENCES learnings (id, business_id);

-- +goose Down
ALTER TABLE learnings
    DROP CONSTRAINT learnings_answer_of_its_business_fkey;

ALTER TABLE learnings
    DROP CONSTRAINT learnings_production_of_its_batch_fkey;

ALTER TABLE learnings
    DROP CONSTRAINT learnings_production_of_its_business_fkey;

ALTER TABLE learnings
    DROP CONSTRAINT learnings_batch_of_its_business_fkey;

DROP INDEX learnings_of_their_business;

DROP INDEX learnings_answered_once;

DROP INDEX learnings_identity_and_business;

DROP TABLE learnings;

DROP INDEX productions_identity_and_recipe;

DROP INDEX productions_business_and_id;
