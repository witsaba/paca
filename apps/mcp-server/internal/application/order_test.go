package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/order"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
)

// fakeOrders is the fake of the storage of the orders.
//
// It remembers the orders it holds and the ones it wrote, and it answers
// what the test tells it to answer.
type fakeOrders struct {
	held    []order.Order
	written []order.Order
	err     error
	listed  business.ID
}

// Create writes the order and remembers it.
func (f *fakeOrders) Create(_ context.Context, o order.Order) (order.Order, error) {
	if f.err != nil {
		return order.Order{}, f.err
	}

	o.ID = order.ID(len(f.written) + 100)
	f.written = append(f.written, o)

	return o, nil
}

// List answers the orders of the business, and remembers that it was
// asked.
func (f *fakeOrders) List(_ context.Context, own business.ID) ([]order.Order, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]order.Order, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// theBakeryOfTheOrders is the business that owns the orders of the tests.
func theBakeryOfTheOrders() business.Business {
	return business.Business{
		ID:       business.ID(77),
		Kind:     business.KindBusiness,
		Name:     "La Panadería",
		Currency: "COP",
	}
}

// theClientOfTheOrders is the party that orders in the tests.
func theClientOfTheOrders(t *testing.T) party.Party {
	t.Helper()

	built, err := party.New(
		business.ID(77),
		party.KindPerson,
		"Doña Rosalba",
		[]party.Role{party.RoleClient},
		"",
		"rosalba@example.com",
		"Calle 5 # 12-30",
		party.LegalHuman{},
	)
	if err != nil {
		t.Fatalf("the client was refused: %v", err)
	}

	built.ID = party.ID(10)

	return built
}

// theProviderOfTheOrders is the party that does not order in the tests.
func theProviderOfTheOrders(t *testing.T) party.Party {
	t.Helper()

	built, err := party.New(
		business.ID(77),
		party.KindFarm,
		"Finca Laesperanza",
		[]party.Role{party.RoleProvider},
		"",
		"",
		"Vereda El Carmen",
		party.LegalHuman{},
	)
	if err != nil {
		t.Fatalf("the provider was refused: %v", err)
	}

	built.ID = party.ID(11)

	return built
}

// theProductOfTheOrders is the product the client orders in the tests.
func theProductOfTheOrders() product.Product {
	return product.Product{
		ID:         product.ID(5),
		BusinessID: business.ID(77),
		RecipeID:   3,
		Name:       "Pan de molde",
	}
}

// theDayOfTheOrders is the day the order of the tests is due on.
func theDayOfTheOrders(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day of the test could not be read: %v", err)
	}

	return parsed
}

// TestTheOrderIsRecordedInTheBusinessThatOwnsIt proves that the use case
// writes the order of the business: the party that ordered, the products
// with the quantities of them, and the day it is due.
func TestTheOrderIsRecordedInTheBusinessThatOwnsIt(t *testing.T) {
	t.Parallel()

	orders := &fakeOrders{}
	parties := &fakeParties{held: []party.Party{theClientOfTheOrders(t)}}
	products := &fakeProducts{held: []product.Product{theProductOfTheOrders()}}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewOrderService(orders, parties, products, businesses)

	written, err := service.Record(t.Context(), ports.RecordOrder{
		BusinessID: 77,
		PartyID:    10,
		DueOn:      "2026-03-14",
		Lines: []ports.OrderLine{
			{ProductID: 5, Quantity: 12, Unit: "pan"},
		},
	})
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	if written.ID == 0 {
		t.Error("the order has no identity, want the one the storage generated")
	}
	if written.BusinessID != business.ID(77) {
		t.Errorf("the business of the order is %d, want 77", written.BusinessID)
	}
	if written.PartyID != party.ID(10) {
		t.Errorf("the party of the order is %d, want 10", written.PartyID)
	}
	if written.PartyName != "Doña Rosalba" {
		t.Errorf("the name of the party is %q, want %q", written.PartyName, "Doña Rosalba")
	}
	if written.DueOn.String() != "2026-03-14" {
		t.Errorf("the day it is due is %q, want %q", written.DueOn.String(), "2026-03-14")
	}
	if written.State != order.StateOpen {
		t.Errorf("the state of the order is %q, want %q", written.State, order.StateOpen)
	}
	if len(written.Lines) != 1 {
		t.Fatalf("the order carries %d lines, want 1", len(written.Lines))
	}
	if written.Lines[0].ProductID != product.ID(5) {
		t.Errorf("the product of the line is %d, want 5", written.Lines[0].ProductID)
	}
	if written.Lines[0].ProductName != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", written.Lines[0].ProductName, "Pan de molde")
	}
	if written.Lines[0].Quantity != 12 {
		t.Errorf("the quantity is %d, want 12", written.Lines[0].Quantity)
	}
	if written.Lines[0].Unit != "pan" {
		t.Errorf("the unit is %q, want %q", written.Lines[0].Unit, "pan")
	}
}

// TestTheOrderOfThePartyTheOwnerNamesIsRecorded proves that the party of
// the order is named by the name the owner reads, and that the name is
// read the way a person reads it.
func TestTheOrderOfThePartyTheOwnerNamesIsRecorded(t *testing.T) {
	t.Parallel()

	orders := &fakeOrders{}
	parties := &fakeParties{held: []party.Party{theClientOfTheOrders(t)}}
	products := &fakeProducts{held: []product.Product{theProductOfTheOrders()}}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewOrderService(orders, parties, products, businesses)

	written, err := service.Record(t.Context(), ports.RecordOrder{
		BusinessID: 77,
		PartyName:  "doña rosalba",
		DueOn:      "2026-03-14",
		Lines: []ports.OrderLine{
			{ProductName: "Pan de molde", Quantity: 3, Unit: "pan"},
		},
	})
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	if written.PartyID != party.ID(10) {
		t.Errorf("the party of the order is %d, want 10", written.PartyID)
	}
	if len(written.Lines) != 1 || written.Lines[0].ProductID != product.ID(5) {
		t.Errorf("the lines of the order are %v, want the product 5", written.Lines)
	}
}

// TestTheOrderOfThePartyThatIsNotAClientIsRefused proves that the domain
// refuses the order of a party that does not hold the role of the client.
func TestTheOrderOfThePartyThatIsNotAClientIsRefused(t *testing.T) {
	t.Parallel()

	orders := &fakeOrders{}
	parties := &fakeParties{held: []party.Party{theProviderOfTheOrders(t)}}
	products := &fakeProducts{held: []product.Product{theProductOfTheOrders()}}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewOrderService(orders, parties, products, businesses)

	_, err := service.Record(t.Context(), ports.RecordOrder{
		BusinessID: 77,
		PartyID:    11,
		DueOn:      "2026-03-14",
		Lines: []ports.OrderLine{
			{ProductID: 5, Quantity: 3, Unit: "pan"},
		},
	})
	if err == nil {
		t.Fatal("the order of a party that is not a client was written")
	}
	if !errors.Is(err, order.ErrNotAClient) {
		t.Fatalf("the error is %v, want %v", err, order.ErrNotAClient)
	}
	if len(orders.written) != 0 {
		t.Errorf("the storage holds %d orders, want none", len(orders.written))
	}
}

// TestTheOrderOfThePartyOfAnotherBusinessIsRefused proves that the party
// of another business is never attached to the order of the business.
func TestTheOrderOfThePartyOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	other := theClientOfTheOrders(t)
	other.BusinessID = business.ID(88)

	orders := &fakeOrders{}
	parties := &fakeParties{held: []party.Party{other}}
	products := &fakeProducts{held: []product.Product{theProductOfTheOrders()}}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewOrderService(orders, parties, products, businesses)

	_, err := service.Record(t.Context(), ports.RecordOrder{
		BusinessID: 77,
		PartyID:    10,
		DueOn:      "2026-03-14",
		Lines: []ports.OrderLine{
			{ProductID: 5, Quantity: 3, Unit: "pan"},
		},
	})
	if err == nil {
		t.Fatal("the order of a party of another business was written")
	}
	if !errors.Is(err, party.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, party.ErrNotFound)
	}
	if len(orders.written) != 0 {
		t.Errorf("the storage holds %d orders, want none", len(orders.written))
	}
}

// TestTheOrderOfTheProductOfAnotherBusinessIsRefused proves that the
// products of the lines of the order are the products of the price list of
// the business.
func TestTheOrderOfTheProductOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	other := theProductOfTheOrders()
	other.BusinessID = business.ID(88)

	orders := &fakeOrders{}
	parties := &fakeParties{held: []party.Party{theClientOfTheOrders(t)}}
	products := &fakeProducts{held: []product.Product{other}}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewOrderService(orders, parties, products, businesses)

	_, err := service.Record(t.Context(), ports.RecordOrder{
		BusinessID: 77,
		PartyID:    10,
		DueOn:      "2026-03-14",
		Lines: []ports.OrderLine{
			{ProductID: 5, Quantity: 3, Unit: "pan"},
		},
	})
	if err == nil {
		t.Fatal("the order of a product of another business was written")
	}
	if !errors.Is(err, product.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, product.ErrNotFound)
	}
	if len(orders.written) != 0 {
		t.Errorf("the storage holds %d orders, want none", len(orders.written))
	}
}

// TestTheOrderWithoutItsDayIsRefused proves that the order carries the day
// it is due, and that the book never invents one.
func TestTheOrderWithoutItsDayIsRefused(t *testing.T) {
	t.Parallel()

	orders := &fakeOrders{}
	parties := &fakeParties{held: []party.Party{theClientOfTheOrders(t)}}
	products := &fakeProducts{held: []product.Product{theProductOfTheOrders()}}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewOrderService(orders, parties, products, businesses)

	_, err := service.Record(t.Context(), ports.RecordOrder{
		BusinessID: 77,
		PartyID:    10,
		Lines: []ports.OrderLine{
			{ProductID: 5, Quantity: 3, Unit: "pan"},
		},
	})
	if err == nil {
		t.Fatal("the order without its day was written")
	}
	if !errors.Is(err, order.ErrMissingDay) {
		t.Fatalf("the error is %v, want %v", err, order.ErrMissingDay)
	}
	if len(orders.written) != 0 {
		t.Errorf("the storage holds %d orders, want none", len(orders.written))
	}
}

// TestTheOrdersOfThePersonalBusinessAreRefused proves that the orders
// belong to a business of its own.
func TestTheOrdersOfThePersonalBusinessAreRefused(t *testing.T) {
	t.Parallel()

	personal := business.Business{
		ID:       business.ID(78),
		Kind:     business.KindPersonal,
		Name:     "El Negocio De Rosalba",
		Currency: "COP",
	}

	orders := &fakeOrders{}
	parties := &fakeParties{}
	products := &fakeProducts{}
	businesses := &fakeBusinesses{existing: &personal}
	service := application.NewOrderService(orders, parties, products, businesses)

	_, err := service.Record(t.Context(), ports.RecordOrder{
		BusinessID: 78,
		PartyID:    10,
		DueOn:      "2026-03-14",
	})
	if err == nil {
		t.Fatal("the order of the personal business was written")
	}
	if !errors.Is(err, order.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, order.ErrWrongBusiness)
	}
}

// TestTheRoundOfTheDayIsRead proves that the use case reads the round of
// the day: the orders that are due on it and the ones that are still owed.
func TestTheRoundOfTheDayIsRead(t *testing.T) {
	t.Parallel()

	today := theDayOfTheOrders(t, "2026-03-14")
	tomorrow := theDayOfTheOrders(t, "2026-03-15")

	due := order.Order{
		ID:         1,
		BusinessID: business.ID(77),
		PartyID:    party.ID(10),
		PartyName:  "Doña Rosalba",
		DueOn:      today,
		Lines: []order.Line{
			{ProductID: product.ID(5), ProductName: "Pan de molde", Quantity: 12, Unit: "pan"},
		},
		State: order.StateOpen,
	}

	later := due
	later.ID = 2
	later.DueOn = tomorrow

	delivered := due
	delivered.ID = 3
	delivered.State = order.StateDelivered

	orders := &fakeOrders{held: []order.Order{due, later, delivered}}
	parties := &fakeParties{}
	products := &fakeProducts{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewOrderService(orders, parties, products, businesses)

	round, err := service.Read(t.Context(), ports.ReadOrders{
		BusinessID: 77,
		Day:        "2026-03-14",
	})
	if err != nil {
		t.Fatalf("the round could not be read: %v", err)
	}

	if orders.listed != business.ID(77) {
		t.Errorf("the business of the reading is %d, want 77", orders.listed)
	}
	if len(round.Orders) != 2 {
		t.Fatalf("the round carries %d orders, want 2", len(round.Orders))
	}

	owed := round.Owed()
	if len(owed) != 1 {
		t.Fatalf("the round owes %d orders, want 1", len(owed))
	}
	if owed[0].ID != due.ID {
		t.Errorf("the order that is owed is %d, want %d", owed[0].ID, due.ID)
	}
}

// TestTheWholeBookOfTheOrdersIsReadWhenTheAgentNamesNoDay proves that the
// book of the orders is read whole when the agent names no day.
func TestTheWholeBookOfTheOrdersIsReadWhenTheAgentNamesNoDay(t *testing.T) {
	t.Parallel()

	today := theDayOfTheOrders(t, "2026-03-14")

	due := order.Order{
		ID:         1,
		BusinessID: business.ID(77),
		PartyID:    party.ID(10),
		PartyName:  "Doña Rosalba",
		DueOn:      today,
		Lines: []order.Line{
			{ProductID: product.ID(5), ProductName: "Pan de molde", Quantity: 12, Unit: "pan"},
		},
		State: order.StateOpen,
	}

	orders := &fakeOrders{held: []order.Order{due}}
	parties := &fakeParties{}
	products := &fakeProducts{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewOrderService(orders, parties, products, businesses)

	round, err := service.Read(t.Context(), ports.ReadOrders{BusinessID: 77})
	if err != nil {
		t.Fatalf("the book of the orders could not be read: %v", err)
	}

	if len(round.Orders) != 1 {
		t.Fatalf("the book holds %d orders, want 1", len(round.Orders))
	}
	if !round.Day.Time().IsZero() {
		t.Errorf("the day of the round is %q, want the empty one", round.Day.String())
	}
}

// TestTheOrdersWithoutADatabaseSaySo proves that the use case tells the
// agent that the database is not there.
func TestTheOrdersWithoutADatabaseSaySo(t *testing.T) {
	t.Parallel()

	service := application.NewOrderService(nil, nil, nil, nil)

	_, err := service.Record(t.Context(), ports.RecordOrder{BusinessID: 77})
	if err == nil {
		t.Fatal("the order was written without a database")
	}
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}

	_, err = service.Read(t.Context(), ports.ReadOrders{BusinessID: 77})
	if err == nil {
		t.Fatal("the round was read without a database")
	}
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
