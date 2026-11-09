-- The queries of the invoices.
--
-- The invoice is written against the purchase it came from, and the
-- purchase points at it: the update that points the purchase carries the
-- business of the invoice, and it touches only the purchase of that
-- business. An invoice of one business is never attached to the purchase
-- of another one.
--
-- The rate of the tax travels in hundredths of a percent: the rate of the
-- tax authority of the owner may not be a whole percent, and a rate that
-- is rounded away is a rate nobody charged.
--
-- The optional column is NULL when the value is not set. The query uses
-- sqlc.narg for it, so the library passes NULL and the column keeps the
-- rule of the domain: an empty value is an absent value.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.

-- name: CreateInvoice :one
INSERT INTO invoices (
    business_id, provider_id, purchase_id, number, day, base, rate, tax, total, reference
)
VALUES (
    sqlc.arg(business_id), sqlc.arg(provider_id), sqlc.arg(purchase_id), sqlc.arg(number),
    sqlc.arg(day), sqlc.arg(base), sqlc.arg(rate), sqlc.arg(tax), sqlc.arg(total),
    sqlc.narg(reference)::text
)
RETURNING *;

-- name: ListInvoicesByBusiness :many
--
-- The invoices of a business and of no other one, in the order of their
-- identity.
SELECT * FROM invoices WHERE business_id = sqlc.arg(business_id) ORDER BY id;

-- name: AttachInvoiceToPurchase :one
--
-- The purchase points at the invoice it came from, and the row is the
-- purchase of the business of the invoice: an invoice of one business is
-- never attached to the purchase of another one. The row that does not
-- belong is not touched.
UPDATE purchases
SET invoice_id = sqlc.arg(invoice_id)
WHERE id = sqlc.arg(purchase_id) AND business_id = sqlc.arg(business_id)
RETURNING id;
