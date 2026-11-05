-- The movements of the book of a business.
--
-- Every movement is money that goes from the credit account to the debit
-- account: the money lands where it is debited and it comes from where it
-- is credited. The amount is positive, the two sides are not the same
-- account, and the description is not empty: a movement the person cannot
-- read is a movement they cannot recognise.
--
-- The amount is an integer, and never a float: the arithmetic of money does
-- not forgive a rounding error. It counts the units of the currency of the
-- book, and the currency of PACA is the Colombian peso.
--
-- The day of a movement is a day and not a moment: the time of the day is
-- not a fact of the money, and the column holds no time.
--
-- The index of the book is the one the person reads it by: the business,
-- the day and the identity of the movement.

-- +goose Up
CREATE TABLE ledger_entries (
    id                BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id       BIGINT      NOT NULL REFERENCES businesses (id),
    debit_account_id  BIGINT      NOT NULL REFERENCES accounts (id),
    credit_account_id BIGINT      NOT NULL REFERENCES accounts (id),
    amount            BIGINT      NOT NULL,
    occurred_on       DATE        NOT NULL,
    description       TEXT        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT ledger_entries_amount_positive_check CHECK (amount > 0),
    CONSTRAINT ledger_entries_distinct_accounts_check CHECK (debit_account_id <> credit_account_id),
    CONSTRAINT ledger_entries_description_not_empty_check CHECK (description <> '')
);

CREATE INDEX ledger_entries_business_day ON ledger_entries (business_id, occurred_on, id);

-- +goose Down
DROP TABLE ledger_entries;
