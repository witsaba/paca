-- A purchase carries ONE invoice: the one that came with it.
--
-- The domain refuses the second invoice of a purchase that already carries
-- one, and the table holds the same rule as a defense in depth: the unique
-- index `invoices_one_per_purchase` refuses a second invoice of the same
-- purchase, whatever number it carries.
--
-- Without the rule, a second invoice of the same purchase re-points the
-- purchase at the newest invoice and leaves the first one naming the same
-- purchase: the book would carry two invoices of one purchase and the
-- purchase would point at neither of the two truths.
--
-- The migration of `00023_invoices.sql` is not edited: it is already
-- applied, and the history of the schema is a history that is not
-- rewritten. The Down of this migration restores the rules of
-- `00023_invoices.sql`.

-- +goose Up
CREATE UNIQUE INDEX invoices_one_per_purchase ON invoices (purchase_id);

-- +goose Down
DROP INDEX invoices_one_per_purchase;
