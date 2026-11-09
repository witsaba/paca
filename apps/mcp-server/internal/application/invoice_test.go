package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The fake of the parties of the invoices.
//
// It holds the parties of the business and it remembers the ones it was
// asked for.
type fakeInvoiceParties struct {
	held   []party.Party
	listed business.ID
	fail   error
}

// Create writes the party.
func (f *fakeInvoiceParties) Create(_ context.Context, p party.Party) (party.Party, error) {
	p.ID = party.ID(len(f.held) + 100)
	f.held = append(f.held, p)

	return p, nil
}

// List answers the parties of the business.
func (f *fakeInvoiceParties) List(_ context.Context, businessID business.ID) ([]party.Party, error) {
	f.listed = businessID

	if f.fail != nil {
		return nil, f.fail
	}

	held := make([]party.Party, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == businessID {
			held = append(held, one)
		}
	}

	return held, nil
}

// The fake of the storage of the purchases.
//
// It holds the purchase the invoice comes from, and it remembers the one
// that was asked for.
type fakeInvoicePurchases struct {
	held    purchase.Purchase
	failure error
	asked   purchase.ID
}

// Find answers the purchase of the test.
func (f *fakeInvoicePurchases) Find(_ context.Context, id purchase.ID) (purchase.Purchase, error) {
	f.asked = id

	if f.failure != nil {
		return purchase.Purchase{}, f.failure
	}

	if f.held.ID != id {
		return purchase.Purchase{}, purchase.ErrNotFound
	}

	return f.held, nil
}

// The fake of the storage of the invoices.
//
// It remembers the invoices it wrote and the purchases it pointed at the
// invoices.
type fakeInvoiceStorage struct {
	written  []purchase.Invoice
	attached []purchase.Invoice
	held     []purchase.Invoice
	failure  error
	listFail error
}

// Record writes the invoice and remembers it.
func (f *fakeInvoiceStorage) Record(_ context.Context, invoice purchase.Invoice) (purchase.Invoice, error) {
	if f.failure != nil {
		return purchase.Invoice{}, f.failure
	}

	invoice.ID = purchase.InvoiceID(len(f.written) + 1)
	f.written = append(f.written, invoice)

	return invoice, nil
}

// Attach points the purchase at the invoice.
func (f *fakeInvoiceStorage) Attach(_ context.Context, invoice purchase.Invoice, bought purchase.Purchase) error {
	if f.failure != nil {
		return f.failure
	}

	f.attached = append(f.attached, invoice)

	return nil
}

// List answers the invoices of the business.
func (f *fakeInvoiceStorage) List(_ context.Context, own business.ID) ([]purchase.Invoice, error) {
	if f.listFail != nil {
		return nil, f.listFail
	}

	held := make([]purchase.Invoice, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// The fake of the transaction of the storage.
//
// It forgets what the work wrote when the work fails: a transaction that
// rolls back leaves the storage as it was.
type fakeInvoiceWork struct {
	invoices *fakeInvoiceStorage
	failure  error
}

// Do runs the work, and it forgets what the work wrote when the work
// fails.
func (f *fakeInvoiceWork) Do(ctx context.Context, work func(context.Context) error) error {
	if f.failure != nil {
		return f.failure
	}

	written := len(f.invoices.written)
	attached := len(f.invoices.attached)

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.invoices.written = f.invoices.written[:written]
	f.invoices.attached = f.invoices.attached[:attached]

	return err
}

// theProviderOfTheInvoices is the provider that issues the invoices of the
// tests.
func theProviderOfTheInvoices() party.Party {
	return party.Party{
		ID:         party.ID(10),
		BusinessID: business.ID(77),
		Kind:       party.KindFarm,
		Name:       "Finca La Esperanza",
		Roles:      []party.Role{party.RoleProvider},
	}
}

// thePurchaseOfTheInvoices is the purchase the invoices of the tests come
// from.
func thePurchaseOfTheInvoices() purchase.Purchase {
	return purchase.Purchase{
		ID:         purchase.ID(5),
		BusinessID: business.ID(77),
		Day:        ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)),
		Total:      ledger.Amount(10000),
	}
}

// TestTheInvoiceIsWrittenAgainstThePurchaseItCameFrom proves that the use
// case writes the invoice against the purchase it came from, and that the
// purchase points at it.
//
// The tax of the invoice is derived from the rate and the base the invoice
// states, and the invoice moves no money.
func TestTheInvoiceIsWrittenAgainstThePurchaseItCameFrom(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	written, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-001",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       19,
		Total:      11900,
		Reference:  "REF-1234",
	})
	if err != nil {
		t.Fatalf("the invoice was refused: %v", err)
	}

	if len(invoices.written) != 1 {
		t.Fatalf("the storage holds %d invoices, want 1", len(invoices.written))
	}
	if len(invoices.attached) != 1 {
		t.Fatalf("the storage attached %d invoices, want 1", len(invoices.attached))
	}

	invoice := invoices.written[0]
	if invoice.BusinessID != business.ID(77) {
		t.Errorf("the business of the invoice is %d, want 77", invoice.BusinessID)
	}
	if invoice.ProviderID != party.ID(10) {
		t.Errorf("the provider of the invoice is %d, want 10", invoice.ProviderID)
	}
	if invoice.PurchaseID != purchase.ID(5) {
		t.Errorf("the purchase of the invoice is %d, want 5", invoice.PurchaseID)
	}
	if invoice.Number != "FV-001" {
		t.Errorf("the number of the invoice is %q, want %q", invoice.Number, "FV-001")
	}
	if invoice.Tax != ledger.Amount(1900) {
		t.Errorf("the tax of the invoice is %d, want 1900", invoice.Tax)
	}
	if invoice.Total != ledger.Amount(11900) {
		t.Errorf("the total of the invoice is %d, want 11900", invoice.Total)
	}
	if invoice.Reference != "REF-1234" {
		t.Errorf("the reference of the invoice is %q, want %q", invoice.Reference, "REF-1234")
	}
	if written.ID == 0 {
		t.Error("the invoice has no identity, want the one the storage generated")
	}
}

// TestTheDayOfTheInvoiceIsTheDayOfTodayWhenTheOwnerNamedNone proves that
// the day of the invoice is the day the agent names, and the day of today
// when it names none.
func TestTheDayOfTheInvoiceIsTheDayOfTodayWhenTheOwnerNamedNone(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	written, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-002",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if err != nil {
		t.Fatalf("the invoice was refused: %v", err)
	}

	if written.Day != ledger.NewDate(theClock().Now()) {
		t.Errorf("the day of the invoice is %v, want %v", written.Day, ledger.NewDate(theClock().Now()))
	}
}

// TestTheInvoiceIsRefusedWhenTheProviderIsNotAProvider proves that the use
// case refuses the invoice of a party that does not hold the role of the
// provider, and that it writes nothing.
func TestTheInvoiceIsRefusedWhenTheProviderIsNotAProvider(t *testing.T) {
	t.Parallel()

	client := party.Party{
		ID:         party.ID(20),
		BusinessID: business.ID(77),
		Kind:       party.KindPerson,
		Name:       "Doña Rosalba",
		Roles:      []party.Role{party.RoleClient},
	}

	parties := &fakeInvoiceParties{held: []party.Party{client}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 20,
		PurchaseID: 5,
		Number:     "FV-003",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if !errors.Is(err, purchase.ErrNotAProvider) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNotAProvider)
	}

	if len(invoices.written) != 0 {
		t.Errorf("the storage holds %d invoices, want none", len(invoices.written))
	}
	if len(invoices.attached) != 0 {
		t.Errorf("the storage attached %d invoices, want none", len(invoices.attached))
	}
}

// TestTheInvoiceOfAnotherBusinessIsRefused proves that the use case refuses
// the invoice of a purchase of another business, and that it writes
// nothing.
func TestTheInvoiceOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	other := thePurchaseOfTheInvoices()
	other.BusinessID = business.ID(88)
	purchases := &fakeInvoicePurchases{held: other}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-004",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if !errors.Is(err, purchase.ErrInvoiceOfAnotherBusiness) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrInvoiceOfAnotherBusiness)
	}

	if len(invoices.written) != 0 {
		t.Errorf("the storage holds %d invoices, want none", len(invoices.written))
	}
}

// TestTheNumberOfTheInvoiceIsUniqueInsideTheBusiness proves that the use
// case refuses the invoice whose number the business already holds, and
// that it writes nothing.
func TestTheNumberOfTheInvoiceIsUniqueInsideTheBusiness(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{
		held: []purchase.Invoice{
			{ID: purchase.InvoiceID(1), BusinessID: business.ID(77), Number: "FV-001"},
		},
	}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-001",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if !errors.Is(err, purchase.ErrDuplicateInvoiceNumber) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrDuplicateInvoiceNumber)
	}

	if len(invoices.written) != 0 {
		t.Errorf("the storage holds %d invoices, want none", len(invoices.written))
	}
}

// TestTheInvoiceIsRefusedWhenTheStorageIsNotThere proves that the use case
// without a storage says that the database is not there, and that it
// writes nothing.
func TestTheInvoiceIsRefusedWhenTheStorageIsNotThere(t *testing.T) {
	t.Parallel()

	service := application.NewInvoiceService(nil, nil, nil, nil, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-013",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}

	if _, err := service.Read(t.Context(), ports.ReadInvoices{BusinessID: business.ID(77)}); !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// TestTheInvoiceIsRefusedWhenTheDayIsNotADay proves that the use case
// refuses the day that is not a day of the book, and that it writes
// nothing.
func TestTheInvoiceIsRefusedWhenTheDayIsNotADay(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-014",
		Day:        "the day before yesterday",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if !errors.Is(err, ledger.ErrInvalidDate) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidDate)
	}

	if len(invoices.written) != 0 {
		t.Errorf("the storage holds %d invoices, want none", len(invoices.written))
	}
}

// TestTheInvoiceIsRefusedWhenTheProviderIsNotThere proves that the use case
// refuses the invoice of a provider the business does not know, and that it
// writes nothing.
func TestTheInvoiceIsRefusedWhenTheProviderIsNotThere(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 99,
		PurchaseID: 5,
		Number:     "FV-015",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if !errors.Is(err, party.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, party.ErrNotFound)
	}

	if len(invoices.written) != 0 {
		t.Errorf("the storage holds %d invoices, want none", len(invoices.written))
	}
}

// TestTheInvoiceIsRefusedWhenThePurchaseIsNotThere proves that the use
// case refuses the invoice of a purchase that is not there, and that it
// writes nothing.
func TestTheInvoiceIsRefusedWhenThePurchaseIsNotThere(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 404,
		Number:     "FV-016",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if !errors.Is(err, purchase.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNotFound)
	}

	if len(invoices.written) != 0 {
		t.Errorf("the storage holds %d invoices, want none", len(invoices.written))
	}
}

// TestTheInvoiceIsRefusedWhenTheRateIsNotARate proves that the use case
// refuses the invoice whose rate is not a rate, and that it writes nothing:
// the tax of an invoice is never assumed.
func TestTheInvoiceIsRefusedWhenTheRateIsNotARate(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-017",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       -19,
		Total:      11900,
	})
	if !errors.Is(err, purchase.ErrInvalidRate) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrInvalidRate)
	}

	if len(invoices.written) != 0 {
		t.Errorf("the storage holds %d invoices, want none", len(invoices.written))
	}
}

// TestTheInvoiceIsNotWrittenWhenTheStorageFails proves that the invoice and
// the purchase that points at it are written whole or not at all: a work
// that fails leaves the storage as it was.
func TestTheInvoiceIsNotWrittenWhenTheStorageFails(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices, failure: purchase.ErrNoStorage}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-012",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if !errors.Is(err, purchase.ErrNoStorage) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNoStorage)
	}

	if len(invoices.written) != 0 {
		t.Errorf("the storage holds %d invoices, want none", len(invoices.written))
	}
	if len(invoices.attached) != 0 {
		t.Errorf("the storage attached %d invoices, want none", len(invoices.attached))
	}
}

// TestTheInvoiceOfTheBusinessIsRead proves that the use case reads the
// invoices of the business the agent names.
func TestTheInvoiceOfTheBusinessIsRead(t *testing.T) {
	t.Parallel()

	invoices := &fakeInvoiceStorage{
		held: []purchase.Invoice{
			{ID: purchase.InvoiceID(1), BusinessID: business.ID(77), Number: "FV-001"},
			{ID: purchase.InvoiceID(2), BusinessID: business.ID(88), Number: "FV-002"},
		},
	}
	service := application.NewInvoiceService(
		&fakeInvoiceParties{},
		&fakeInvoicePurchases{},
		invoices,
		&fakeInvoiceWork{invoices: invoices},
		theClock(),
	)

	held, err := service.Read(t.Context(), ports.ReadInvoices{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the invoices could not be read: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the business holds %d invoices, want 1", len(held))
	}
	if held[0].Number != "FV-001" {
		t.Errorf("the number of the invoice is %q, want %q", held[0].Number, "FV-001")
	}
}

// TestTheSecondInvoiceOfThePurchaseIsRefused proves that the use case
// refuses the second invoice of a purchase that already carries one, and
// that it writes nothing: the purchase keeps pointing at the first
// invoice.
func TestTheSecondInvoiceOfThePurchaseIsRefused(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{
		held: []purchase.Invoice{
			{
				ID:         purchase.InvoiceID(1),
				BusinessID: business.ID(77),
				PurchaseID: purchase.ID(5),
				Number:     "FV-001",
			},
		},
	}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	_, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-002",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if !errors.Is(err, purchase.ErrPurchaseAlreadyHasInvoice) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrPurchaseAlreadyHasInvoice)
	}

	if len(invoices.written) != 0 {
		t.Errorf("the storage holds %d invoices, want none", len(invoices.written))
	}
	if len(invoices.attached) != 0 {
		t.Errorf("the storage attached %d invoices, want none", len(invoices.attached))
	}
}

// TestTheInvoiceMovesNoMoney proves that the invoice moves no money: the
// use case reads the purchase, writes the invoice and points the purchase
// at it, and the money of the purchase is left as it was.
//
// The port of the purchases is a FINDER: the use case can only read the
// purchase, and the money of the book moved when the owner bought.
func TestTheInvoiceMovesNoMoney(t *testing.T) {
	t.Parallel()

	parties := &fakeInvoiceParties{held: []party.Party{theProviderOfTheInvoices()}}
	purchases := &fakeInvoicePurchases{held: thePurchaseOfTheInvoices()}
	invoices := &fakeInvoiceStorage{}
	work := &fakeInvoiceWork{invoices: invoices}
	service := application.NewInvoiceService(parties, purchases, invoices, work, theClock())

	written, err := service.Record(t.Context(), ports.RecordInvoice{
		BusinessID: business.ID(77),
		ProviderID: 10,
		PurchaseID: 5,
		Number:     "FV-018",
		Day:        "2026-03-02",
		Base:       10000,
		Rate:       19,
		Total:      11900,
	})
	if err != nil {
		t.Fatalf("the invoice was refused: %v", err)
	}

	if written.Total != ledger.Amount(11900) {
		t.Errorf("the total of the invoice is %d, want 11900", written.Total)
	}
	if purchases.held.Total != ledger.Amount(10000) {
		t.Errorf("the money of the purchase is %d, want 10000: the invoice moves no money", purchases.held.Total)
	}
	if len(invoices.written) != 1 {
		t.Errorf("the storage holds %d invoices, want 1", len(invoices.written))
	}
	if len(invoices.attached) != 1 {
		t.Errorf("the storage attached %d invoices, want 1", len(invoices.attached))
	}
}
