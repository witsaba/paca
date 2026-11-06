-- The labor of the owner: the hours she worked, in the three kinds of the
-- labor, and the value of the hour she declared.
--
-- The labor of the PAC is the labor of the owner alone: the payroll of the
-- employees is out of scope for now, and the model of the labor of an
-- employee does not exist yet. The labor of the bakery names no employee,
-- because it has none.
--
-- The labor is measured in hours, and the hours are held apart in the
-- three kinds of the labor: the hands, the oven and the waiting. At least
-- one of the three kinds carries hours, because a labor without hours is
-- nothing.
--
-- The value of the hour is declared by the owner, and it is never
-- inferred: the money of the work of a person is not a number PACA may
-- invent. The money of the labor is not stored: it is derived from the
-- hours and the value of the hour, the way the cost of a batch is derived
-- from the layers of the stock.
--
-- The rules of the domain are that the hours are not negative, that the
-- labor carries hours, that the value of the hour is positive and that the
-- word of the owner is not empty. The table holds the same rules as a
-- defense in depth.

-- +goose Up
CREATE TABLE labors (
    id             BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id    BIGINT      NOT NULL REFERENCES businesses (id),
    day            DATE        NOT NULL,
    hands_on_hours DOUBLE PRECISION NOT NULL DEFAULT 0,
    oven_hours     DOUBLE PRECISION NOT NULL DEFAULT 0,
    waiting_hours  DOUBLE PRECISION NOT NULL DEFAULT 0,
    hour_value     BIGINT      NOT NULL,
    utterance      TEXT        NOT NULL DEFAULT '',
    confirmed_with TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT labors_hours_not_negative_check CHECK (hands_on_hours >= 0 AND oven_hours >= 0 AND waiting_hours >= 0),
    CONSTRAINT labors_hours_present_check CHECK (hands_on_hours > 0 OR oven_hours > 0 OR waiting_hours > 0),
    CONSTRAINT labors_hour_value_positive_check CHECK (hour_value > 0),
    CONSTRAINT labors_confirmed_with_not_empty_check CHECK (confirmed_with <> '')
);

-- The step of a recipe carries the kind of the labor it is: the hands, the
-- oven or the waiting. The labor of a batch is derived from the hours of
-- the steps of the recipe, and the kind is what says in which of the three
-- kinds the hours of the step are counted.
--
-- The rows that exist already are the labor of the hands: a step the owner
-- did not qualify is a step she did with her hands.
ALTER TABLE recipe_steps
    ADD COLUMN labor_kind TEXT NOT NULL DEFAULT 'hands_on';

ALTER TABLE recipe_steps
    ADD CONSTRAINT recipe_steps_labor_kind_check CHECK (labor_kind IN ('hands_on', 'oven', 'waiting')) NOT VALID;

-- +goose Down
ALTER TABLE recipe_steps DROP CONSTRAINT recipe_steps_labor_kind_check;

ALTER TABLE recipe_steps DROP COLUMN labor_kind;

DROP TABLE labors;
