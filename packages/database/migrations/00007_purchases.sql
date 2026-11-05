-- The purchase: the fact of what the business bought, and the lines it
-- carries.
--
-- A purchase never changes: it is what was bought, on the day it was
-- bought, at the market it was bought in, with the invoice it came with,
-- with the sentence the owner said and with the word she used to confirm
-- it.
--
-- The lines of a purchase are rows of their own table: a purchase is a
-- fact, and a fact has rows. The position of a line is the order the owner
-- said it in, and it is unique inside the purchase.
--
-- The money of a purchase is whole pesos: the book counts the pesos the
-- person has, and a price in centavos is rounded to the nearest peso when
-- it enters the book.
--
-- The rules of the domain are that the sentence is not empty, that the word
-- of the confirmation is not empty, that the money of the purchase is
-- positive and that the money of a line is not negative. The table holds
-- the same rules as a defense in depth.
--
-- A draft is confirmed once: the unique index refuses a second purchase of
-- the same draft, and the draft itself moves to the state `confirmed` when
-- the purchase is written.

-- +goose Up
CREATE TABLE purchases (
    id             BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id    BIGINT      NOT NULL REFERENCES businesses (id),
    draft_id       BIGINT      NOT NULL REFERENCES purchase_drafts (id),
    day            DATE        NOT NULL,
    market         TEXT        NULL,
    invoice        TEXT        NULL,
    utterance      TEXT        NOT NULL,
    confirmed_with TEXT        NOT NULL,
    total          BIGINT      NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT purchases_utterance_not_empty_check CHECK (utterance <> ''),
    CONSTRAINT purchases_confirmed_with_not_empty_check CHECK (confirmed_with <> ''),
    CONSTRAINT purchases_total_positive_check CHECK (total > 0)
);

CREATE UNIQUE INDEX purchases_one_per_draft ON purchases (draft_id);

CREATE TABLE purchase_lines (
    id         BIGINT             GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    purchase_id BIGINT            NOT NULL REFERENCES purchases (id) ON DELETE CASCADE,
    position   INTEGER            NOT NULL,
    name       TEXT               NOT NULL,
    quantity   DOUBLE PRECISION   NOT NULL,
    unit       TEXT               NOT NULL,
    total      BIGINT             NOT NULL,
    unit_price BIGINT             NOT NULL,

    CONSTRAINT purchase_lines_name_not_empty_check CHECK (name <> ''),
    CONSTRAINT purchase_lines_unit_not_empty_check CHECK (unit <> ''),
    CONSTRAINT purchase_lines_quantity_positive_check CHECK (quantity > 0),
    CONSTRAINT purchase_lines_total_not_negative_check CHECK (total >= 0),
    CONSTRAINT purchase_lines_unit_price_not_negative_check CHECK (unit_price >= 0),
    CONSTRAINT purchase_lines_position_positive_check CHECK (position > 0)
);

CREATE UNIQUE INDEX purchase_lines_one_position ON purchase_lines (purchase_id, position);

ALTER TABLE purchase_drafts DROP CONSTRAINT purchase_drafts_state_check;
ALTER TABLE purchase_drafts ADD CONSTRAINT purchase_drafts_state_check CHECK (state IN ('proposed', 'confirmed'));

-- +goose Down
ALTER TABLE purchase_drafts DROP CONSTRAINT purchase_drafts_state_check;
ALTER TABLE purchase_drafts ADD CONSTRAINT purchase_drafts_state_check CHECK (state IN ('proposed'));

DROP TABLE purchase_lines;
DROP TABLE purchases;
