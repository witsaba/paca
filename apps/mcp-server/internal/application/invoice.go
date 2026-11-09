package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/store"
)

// InvoiceService is the use case of the invoices of the bakery.
//
// The invoice is the fact that comes with the purchase: the provider that
// issued it, its number, its day, its total and the tax that travels with
// it. The use case carries what the agent showed to the domain, and the
// answer of the domain back to the agent.
//
// The invoice moves NO money: the purchase already moved the money of the
// payment when the owner bought, and the invoice arrives later. The invoice
// is written against the purchase it came from, and the purchase points at
// it.
//
// The use case never decides what an invoice is: the tax of an invoice, the
// provider that may issue it, the business it belongs to and the number
// that identifies it are the rules of the domain, and the use case only
// asks the domain.
type InvoiceService struct {
	parties   party.Repository
	purchases purchase.Finder
	invoices  purchase.InvoiceRepository
	work      store.UnitOfWork
	clock     ports.Clock
}

// NewInvoiceService builds the use case of the invoices.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The parties are
// one of them: the provider of an invoice is a party, and the use case
// reads it before it writes the invoice. The clock is the port of the
// time: the day of an invoice that names none is the day of today.
func NewInvoiceService(
	parties party.Repository,
	purchases purchase.Finder,
	invoices purchase.InvoiceRepository,
	work store.UnitOfWork,
	clock ports.Clock,
) *InvoiceService {
	return &InvoiceService{
		parties:   parties,
		purchases: purchases,
		invoices:  invoices,
		work:      work,
		clock:     clock,
	}
}

// Record writes the invoice against the purchase it came from.
//
// The rules are the ones of the domain: the use case reads the provider,
// reads the purchase, asks the domain to read the rate of the tax and to
// build the invoice, and asks the domain whether the number of the invoice
// is free inside the business. It never decides what an invoice is.
//
// The invoice and the purchase that points at it are written in one
// transaction: an invoice that no purchase points at is a fact of nobody.
func (s *InvoiceService) Record(ctx context.Context, request ports.RecordInvoice) (purchase.Invoice, error) {
	if s == nil || s.parties == nil || s.purchases == nil || s.invoices == nil ||
		s.work == nil || s.clock == nil {
		return purchase.Invoice{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return purchase.Invoice{}, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	provider, err := s.providerOf(ctx, request.BusinessID, request.ProviderID)
	if err != nil {
		return purchase.Invoice{}, err
	}

	bought, err := s.purchases.Find(ctx, purchase.ID(request.PurchaseID))
	if err != nil {
		return purchase.Invoice{}, err
	}

	rate, err := purchase.NewRate(request.Rate)
	if err != nil {
		return purchase.Invoice{}, err
	}

	invoice, err := purchase.NewInvoice(request.BusinessID, provider, purchase.InvoiceDefinition{
		PurchaseID: purchase.ID(request.PurchaseID),
		Number:     request.Number,
		Day:        day,
		Base:       ledger.Amount(request.Base),
		Rate:       rate,
		Total:      ledger.Amount(request.Total),
		Reference:  request.Reference,
	})
	if err != nil {
		return purchase.Invoice{}, err
	}

	// The purchase points at the invoice it came from, and the invoice
	// belongs to the business of the purchase: the domain refuses the
	// invoice of another business.
	if err := invoice.AttachesTo(bought); err != nil {
		return purchase.Invoice{}, err
	}

	held, err := s.invoices.List(ctx, request.BusinessID)
	if err != nil {
		return purchase.Invoice{}, err
	}

	if err := purchase.UniqueInvoiceNumber(held, invoice); err != nil {
		return purchase.Invoice{}, err
	}

	// A purchase carries ONE invoice: the one that came with it. The
	// rule lives in the domain, and the use case asks it with the
	// invoices it read.
	if err := purchase.UniqueInvoiceForPurchase(held, invoice); err != nil {
		return purchase.Invoice{}, err
	}

	var written purchase.Invoice
	err = s.work.Do(ctx, func(ctx context.Context) error {
		recorded, err := s.invoices.Record(ctx, invoice)
		if err != nil {
			return err
		}

		if err := s.invoices.Attach(ctx, recorded, bought); err != nil {
			return err
		}

		written = recorded

		return nil
	})
	if err != nil {
		return purchase.Invoice{}, err
	}

	return written, nil
}

// Read answers the invoices of the business.
//
// The rules are the ones of the domain: the use case reads the invoices of
// the business and hands them to the agent. The tax of every invoice
// travels with it: it is the one the invoice states, and it is never
// assumed.
func (s *InvoiceService) Read(ctx context.Context, request ports.ReadInvoices) ([]purchase.Invoice, error) {
	if s == nil || s.invoices == nil {
		return nil, ports.ErrNoDatabase
	}

	return s.invoices.List(ctx, request.BusinessID)
}

// providerOf answers the party that issued the invoice.
//
// The rule of the domain: the provider of an invoice is a party of the
// business, and it holds the role of the provider. The use case reads the
// parties of the business and hands the one the agent named to the domain,
// and the domain refuses the party that may not issue an invoice.
func (s *InvoiceService) providerOf(
	ctx context.Context,
	own business.ID,
	providerID int64,
) (party.Party, error) {
	held, err := s.parties.List(ctx, own)
	if err != nil {
		return party.Party{}, err
	}

	for _, one := range held {
		if int64(one.ID) == providerID {
			return one, nil
		}
	}

	return party.Party{}, fmt.Errorf("%w: the party %d is not one of the business %d",
		party.ErrNotFound, providerID, own)
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of
// today when the agent names none.
func (s *InvoiceService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *InvoiceService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
