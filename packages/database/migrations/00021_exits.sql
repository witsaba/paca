-- The exit of the book of the bread: the consumption of the owner and the
-- donation.
--
-- The exit is a movement of the book, and it moves no money: nobody paid
-- for the bread the owner consumed or for the bread the bakery gave away.
-- What the exit does is release the cost of the units that left the book.
--
-- The rules of the domain are that the business and the product are there,
-- that the kind of the exit is one of the two of the book, that the
-- quantity is positive and whole, that the unit is the unit of a piece,
-- that the cost is not negative, that the tax treatment is the one the
-- book waits for and that the word of the owner is not empty. The table
-- holds the same rules as a defense in depth.
--
-- The tax treatment of the exit is never decided here: the column holds
-- the question that waits for the professional of the taxes of the owner,
-- and the check of the table refuses any other value. The book never
-- assumes a rate.
--
-- The units that left the book are rows of the table of the exits: the
-- trace of the rotation of the output by the expiry. The table of the
-- exits owns the rule of the book: a unit leaves the book only once,
-- whichever exit takes it - the waste, the sale, the consumption or the
-- donation.

-- +goose Up
ALTER TABLE unit_exits DROP CONSTRAINT unit_exits_kind_check;
ALTER TABLE unit_exits ADD CONSTRAINT unit_exits_kind_check
    CHECK (exit_kind IN ('waste', 'sale', 'consumption', 'donation'));

CREATE TABLE exits (
    id             BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id    BIGINT             NOT NULL REFERENCES businesses (id),
    recipe_id      BIGINT             NOT NULL REFERENCES recipes (id),
    kind           TEXT               NOT NULL,
    day            DATE               NOT NULL,
    quantity       DOUBLE PRECISION   NOT NULL,
    unit           TEXT               NOT NULL,
    cost           BIGINT             NOT NULL,
    tax_treatment  TEXT               NOT NULL,
    tax_question   TEXT               NOT NULL,
    utterance      TEXT               NOT NULL DEFAULT '',
    confirmed_with TEXT               NOT NULL,
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT exits_kind_check CHECK (kind IN ('consumption', 'donation')),
    CONSTRAINT exits_quantity_positive_check CHECK (quantity > 0),
    CONSTRAINT exits_quantity_whole_check CHECK (quantity = trunc(quantity)),
    CONSTRAINT exits_unit_check CHECK (unit = 'unidad'),
    CONSTRAINT exits_cost_not_negative_check CHECK (cost >= 0),
    CONSTRAINT exits_tax_treatment_check CHECK (tax_treatment IN ('pending_the_professional')),
    CONSTRAINT exits_tax_question_not_empty_check CHECK (tax_question <> ''),
    CONSTRAINT exits_confirmed_with_not_empty_check CHECK (confirmed_with <> '')
);

-- +goose Down
DROP TABLE exits;

ALTER TABLE unit_exits DROP CONSTRAINT unit_exits_kind_check;
ALTER TABLE unit_exits ADD CONSTRAINT unit_exits_kind_check CHECK (exit_kind IN ('waste', 'sale'));
