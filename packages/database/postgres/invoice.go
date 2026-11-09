package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The names the database gives to the constraints of the table of the
// invoices.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	// invoicesNumberConstraint is the name of the index that refuses a
	// second invoice with the same number inside the same business.
	invoicesNumberConstraint = "invoices_one_number_per_business"

	// invoicesPurchaseConstraint is the name of the index that refuses a
	// second invoice for the same purchase.
	invoicesPurchaseConstraint = "invoices_one_per_purchase"
)

// InvoiceRepository is the adapter of the storage of the invoices.
//
// An invoice is a fact: it is written once, and it never changes. It is
// written against the purchase it came from, and the purchase points at
// it.
//
// The rate of the tax travels in hundredths of a percent, and the adapter
// reads it back through the domain: the row never decides what a rate is,
// and the tax of an invoice is never assumed.
type InvoiceRepository struct {
	pool *Pool
}

// NewInvoiceRepository returns the repository of the invoices, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewInvoiceRepository(pool *Pool) (*InvoiceRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &InvoiceRepository{pool: pool}, nil
}

// Record writes the invoice and answers it with the identity that the
// database generated.
//
// The error is purchase.ErrInvalidValue when the row breaks a rule of the
// table, purchase.ErrDuplicateInvoiceNumber when the business already holds
// that number, purchase.ErrNotFound when the business, the provider or the
// purchase of the invoice is not there, and purchase.ErrNoStorage when the
// storage is not open.
func (r *InvoiceRepository) Record(ctx context.Context, invoice purchase.Invoice) (purchase.Invoice, error) {
	if r == nil || r.pool == nil {
		return purchase.Invoice{}, purchase.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreateInvoice(ctx, db.CreateInvoiceParams{
		BusinessID: int64(invoice.BusinessID),
		ProviderID: int64(invoice.ProviderID),
		PurchaseID: int64(invoice.PurchaseID),
		Number:     invoice.Number,
		Day:        dateOf(invoice.Day),
		Base:       int64(invoice.Base),
		Rate:       invoice.Rate.Hundredths(),
		Tax:        int64(invoice.Tax),
		Total:      int64(invoice.Total),
		Reference:  optionalText(invoice.Reference),
	})
	if err != nil {
		return purchase.Invoice{}, writeInvoiceError(err)
	}

	return invoiceFromRow(row)
}

// Attach points the purchase at the invoice it came from.
//
// The rule of the domain: the purchase and the invoice belong to the same
// business, and the storage writes the row of the purchase only when it is
// the purchase of that business.
//
// The rule of the link holds here too: the purchase is the one the INVOICE
// NAMES, and the invoice of the purchase 2 is never attached to the
// purchase 3. The two sides of the link must agree, and the row that is
// not the one the invoice names is not touched.
//
// The error is purchase.ErrInvoiceOfAnotherPurchase when the purchase is
// not the one the invoice names, purchase.ErrNotFound when the purchase is
// not there, and purchase.ErrNoStorage when the storage is not open.
func (r *InvoiceRepository) Attach(
	ctx context.Context,
	invoice purchase.Invoice,
	bought purchase.Purchase,
) error {
	if r == nil || r.pool == nil {
		return purchase.ErrNoStorage
	}

	if bought.BusinessID != invoice.BusinessID {
		return fmt.Errorf("%w: the purchase %d is of the business %d and the invoice is of the business %d",
			purchase.ErrNotFound, bought.ID, bought.BusinessID, invoice.BusinessID)
	}

	if bought.ID != invoice.PurchaseID {
		return fmt.Errorf("%w: the invoice names the purchase %d and the purchase is %d",
			purchase.ErrInvoiceOfAnotherPurchase, invoice.PurchaseID, bought.ID)
	}

	_, err := r.pool.queriesFor(ctx).AttachInvoiceToPurchase(ctx, db.AttachInvoiceToPurchaseParams{
		InvoiceID:  pgtype.Int8{Int64: int64(invoice.ID), Valid: true},
		PurchaseID: int64(bought.ID),
		BusinessID: int64(invoice.BusinessID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: the purchase %d is not the purchase of the business %d",
				purchase.ErrNotFound, bought.ID, invoice.BusinessID)
		}

		return writeInvoiceError(err)
	}

	return nil
}

// List returns the invoices of a business, in the order of their identity.
//
// The error is purchase.ErrNoStorage when the storage is not open.
func (r *InvoiceRepository) List(ctx context.Context, own business.ID) ([]purchase.Invoice, error) {
	if r == nil || r.pool == nil {
		return nil, purchase.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListInvoicesByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeInvoiceError(err)
	}

	held := make([]purchase.Invoice, 0, len(rows))
	for _, row := range rows {
		invoice, err := invoiceFromRow(row)
		if err != nil {
			return nil, err
		}

		held = append(held, invoice)
	}

	return held, nil
}

// invoiceFromRow maps a row of the invoices onto the model of the domain.
//
// The invoice is built by the domain: the row may come from a table that
// was changed by another writer, and the model does not trust a stranger
// with an invoice it does not know. An invoice that does not add up is
// never read as an invoice.
func invoiceFromRow(row db.Invoice) (purchase.Invoice, error) {
	rate, err := purchase.NewRateFromHundredths(row.Rate)
	if err != nil {
		return purchase.Invoice{}, fmt.Errorf("%w: the rate of the invoice is %d", purchase.ErrInvalidValue, row.Rate)
	}

	invoice, err := purchase.NewInvoiceFromFacts(purchase.InvoiceFacts{
		ID:         purchase.InvoiceID(row.ID),
		BusinessID: business.ID(row.BusinessID),
		ProviderID: party.ID(row.ProviderID),
		PurchaseID: purchase.ID(row.PurchaseID),
		Number:     row.Number,
		Day:        ledger.NewDate(row.Day.Time),
		Base:       ledger.Amount(row.Base),
		Rate:       rate,
		Tax:        ledger.Amount(row.Tax),
		Total:      ledger.Amount(row.Total),
		Reference:  row.Reference.String,
	})
	if err != nil {
		return purchase.Invoice{}, err
	}

	invoice.CreatedAt = timestamp(row.CreatedAt)

	return invoice, nil
}

// writeInvoiceError translates the truth of the database into the errors of
// the port of the invoices.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth: the number of an invoice is unique inside the business, the total
// of the invoice is the base plus the tax, and the business, the provider
// and the purchase of the invoice are not there.
func writeInvoiceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return purchase.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		if pgErr.ConstraintName == invoicesNumberConstraint {
			return fmt.Errorf("%w: the constraint %q refused the row",
				purchase.ErrDuplicateInvoiceNumber, pgErr.ConstraintName)
		}

		if pgErr.ConstraintName == invoicesPurchaseConstraint {
			return fmt.Errorf("%w: the constraint %q refused the row",
				purchase.ErrPurchaseAlreadyHasInvoice, pgErr.ConstraintName)
		}

		return fmt.Errorf("%w: the constraint %q refused the row", purchase.ErrInvalidValue, pgErr.ConstraintName)
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business, the provider or the purchase of the invoice is not there",
			purchase.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", purchase.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
