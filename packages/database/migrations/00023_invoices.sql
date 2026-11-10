-- The invoices: the fact that comes with the purchase.
--
-- The invoice is a fact of the purchase, and it is NOT money: the purchase
-- already moved the money of the payment when the owner bought, and the
-- invoice arrives later. The invoice is written against the purchase it
-- came from, and the purchase points at it.
--
-- The provider of an invoice is a PARTY: the farm, the store or the person
-- the business buys from. The table holds the party it belongs to, and
-- nowhere else.
--
-- The tax of an invoice is NEVER ASSUMED: it comes from the invoice. The
-- invoice states the RATE it was issued with and the BASE that rate
-- applies to, and the tax is derived from the two of them. The rate is
-- counted in hundredths of a percent, because the rate of the tax authority
-- of the owner may not be a whole percent, and a rate that is rounded away
-- is a rate nobody charged. The money of the book counts whole pesos.
--
-- The electronic invoice is NEVER ASSUMED either: there is no integration
-- with the tax authority and no column that pretends to be one. The
-- reference is a plain fact of the document the owner shows, and it may be
-- NULL.
--
-- The number of an invoice is what identifies it, and it is unique inside
-- the business that holds it: the unique index refuses a second invoice
-- with the same number inside the same business.
--
-- The rules of the domain hold the columns as a defense in depth: the
-- number is not empty, the money is not negative, the rate is a rate, and
-- the total of the invoice is the base plus the tax. An invoice that does
-- not add up is never read as an invoice.

-- +goose Up
CREATE TABLE invoices (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT      NOT NULL REFERENCES businesses (id),
    provider_id BIGINT      NOT NULL REFERENCES parties (id),
    purchase_id BIGINT      NOT NULL REFERENCES purchases (id),
    number      TEXT        NOT NULL,
    day         DATE        NOT NULL,
    base        BIGINT      NOT NULL,
    rate        BIGINT      NOT NULL,
    tax         BIGINT      NOT NULL,
    total       BIGINT      NOT NULL,
    reference   TEXT        NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT invoices_number_not_empty_check CHECK (number <> ''),
    CONSTRAINT invoices_base_not_negative_check CHECK (base >= 0),
    CONSTRAINT invoices_rate_check CHECK (rate >= 0 AND rate <= 10000),
    CONSTRAINT invoices_tax_not_negative_check CHECK (tax >= 0),
    CONSTRAINT invoices_total_not_negative_check CHECK (total >= 0),
    CONSTRAINT invoices_tax_check CHECK (tax = round(base * rate::numeric / 10000)),
    CONSTRAINT invoices_total_check CHECK (total = base + tax),
    CONSTRAINT invoices_reference_not_empty_check CHECK (reference IS NULL OR reference <> '')
);

CREATE UNIQUE INDEX invoices_one_number_per_business ON invoices (business_id, number);

-- The purchase points at the invoice it came from. The column is NULL when
-- the invoice has not arrived yet: the purchase is written when the owner
-- buys, and the invoice comes later.
ALTER TABLE purchases
    ADD COLUMN invoice_id BIGINT NULL REFERENCES invoices (id);

-- The tax identity of the party: the NIT of a person, of a store or of a
-- farm.
--
-- It is an OPTIONAL fact of the party, and nothing in the book assumes
-- one: the column is NULL when the party carries no tax identity, and the
-- book does not know the format of one either. The rule of the tax
-- authority and the advice of the professional of the owner are not read
-- here.
ALTER TABLE parties
    ADD COLUMN tax_id TEXT NULL;

ALTER TABLE parties
    ADD CONSTRAINT parties_tax_id_not_empty_check CHECK (tax_id IS NULL OR tax_id <> '');

-- +goose Down
ALTER TABLE parties
    DROP CONSTRAINT parties_tax_id_not_empty_check,
    DROP COLUMN tax_id;

ALTER TABLE purchases DROP COLUMN invoice_id;

DROP INDEX invoices_one_number_per_business;

DROP TABLE invoices;
