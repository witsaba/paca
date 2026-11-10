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
)

// fakeRoutes is the fake of the storage of the routes.
//
// It remembers the routes it holds and the ones it wrote, and it answers
// what the test tells it to answer.
type fakeRoutes struct {
	held    []order.Route
	written []order.Route
	err     error
	listed  business.ID
}

// Create writes the route and remembers it.
func (f *fakeRoutes) Create(_ context.Context, r order.Route) (order.Route, error) {
	if f.err != nil {
		return order.Route{}, f.err
	}

	r.ID = order.RouteID(len(f.written) + 100)
	f.written = append(f.written, r)

	return r, nil
}

// List answers the routes of the business, and remembers that it was asked.
func (f *fakeRoutes) List(_ context.Context, own business.ID) ([]order.Route, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]order.Route, 0, len(f.held))

	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// fakeRouteWork is the fake of the transaction of the storage.
//
// The work that fails leaves the storage as it was: the route and the
// delivery of its stops are forgotten.
type fakeRouteWork struct {
	routes *fakeRoutes
	orders *fakeOrders
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakeRouteWork) Do(ctx context.Context, work func(context.Context) error) error {
	routes := len(f.routes.written)
	delivered := len(f.orders.delivered)

	if err := work(ctx); err != nil {
		f.routes.written = f.routes.written[:routes]
		f.orders.delivered = f.orders.delivered[:delivered]

		return err
	}

	return nil
}

// theOrderDueOnTheDayOfTheRoutes is an order of the business of the tests,
// due on the day of the round.
func theOrderDueOnTheDayOfTheRoutes(t *testing.T, id order.ID, dueOn ledger.Date) order.Order {
	t.Helper()

	built, err := order.New(
		business.ID(77),
		theClientOfTheOrders(t),
		dueOn,
		[]order.LineDefinition{
			{Product: theProductOfTheOrders(), Quantity: 12, Unit: "unidad"},
		},
	)
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	built.ID = id

	return built
}

// theDayOfTheRoutes is the day of the round of the tests.
func theDayOfTheRoutes(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day of the test could not be read: %v", err)
	}

	return parsed
}

// TestTheRouteOfTheRoundIsRecorded proves that the use case writes the
// route of the round of the day: the stops in the order the agent will
// visit the clients in, and what each one costs.
func TestTheRouteOfTheRoundIsRecorded(t *testing.T) {
	t.Parallel()

	day := theDayOfTheRoutes(t, "2026-03-14")

	orders := &fakeOrders{held: []order.Order{
		theOrderDueOnTheDayOfTheRoutes(t, 10, day),
		theOrderDueOnTheDayOfTheRoutes(t, 11, day),
	}}
	routes := &fakeRoutes{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	written, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
			{OrderID: 11, Cost: 1500},
		},
	})
	if err != nil {
		t.Fatalf("the route was refused: %v", err)
	}

	if len(routes.written) != 1 {
		t.Fatalf("the storage holds %d routes, want 1", len(routes.written))
	}
	if len(written.Stops) != 2 {
		t.Fatalf("the route carries %d stops, want 2", len(written.Stops))
	}
	if written.Stops[0].OrderID != order.ID(10) {
		t.Errorf("the first stop is the order %d, want 10", written.Stops[0].OrderID)
	}
	if written.Stops[1].OrderID != order.ID(11) {
		t.Errorf("the second stop is the order %d, want 11", written.Stops[1].OrderID)
	}
	if written.Stops[0].Cost != ledger.Amount(3000) {
		t.Errorf("the cost of the first stop is %d, want 3000", written.Stops[0].Cost)
	}
	if written.Day.String() != "2026-03-14" {
		t.Errorf("the day of the route is %q, want %q", written.Day.String(), "2026-03-14")
	}
	if written.Total() != ledger.Amount(4500) {
		t.Errorf("the total of the route is %d, want 4500", written.Total())
	}
}

// TestTheRouteDeliversTheOrdersOfItsStops proves the rule of the domain:
// the route the agent writes back is the round it made, and writing it
// delivers the orders of its stops.
func TestTheRouteDeliversTheOrdersOfItsStops(t *testing.T) {
	t.Parallel()

	day := theDayOfTheRoutes(t, "2026-03-14")

	orders := &fakeOrders{held: []order.Order{
		theOrderDueOnTheDayOfTheRoutes(t, 10, day),
		theOrderDueOnTheDayOfTheRoutes(t, 11, day),
	}}
	routes := &fakeRoutes{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
			{OrderID: 11, Cost: 1500},
		},
	})
	if err != nil {
		t.Fatalf("the route was refused: %v", err)
	}

	if len(orders.delivered) != 2 {
		t.Fatalf("the storage holds %d deliveries, want 2", len(orders.delivered))
	}
	for _, one := range orders.delivered {
		if one.State != order.StateDelivered {
			t.Errorf("the order %d is %q, want %q", one.ID, one.State, order.StateDelivered)
		}
	}
}

// TestTheRouteAndItsDeliveryAreOneUnitOfWork proves that the route and
// the delivery of the orders of its stops are written whole or not at all:
// the delivery that fails leaves the route unwritten.
func TestTheRouteAndItsDeliveryAreOneUnitOfWork(t *testing.T) {
	t.Parallel()

	day := theDayOfTheRoutes(t, "2026-03-14")

	orders := &fakeOrders{
		held: []order.Order{
			theOrderDueOnTheDayOfTheRoutes(t, 10, day),
		},
		deliveryErr: errors.New("the delivery was refused"),
	}
	routes := &fakeRoutes{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
		},
	})
	if err == nil {
		t.Fatal("the route was written and its delivery was refused")
	}
	if len(routes.written) != 0 {
		t.Errorf("the storage holds %d routes, want none: the route and its delivery are one unit of work", len(routes.written))
	}
}

// TestTheRouteOfTheDayDeliversTheOverdueOrderThatIsOwed proves the rule of
// the domain through the use case: the stops of the route are the orders the
// business still OWES, the ones that are due on the day of the round and the
// ones that fell due on an earlier day. The order that was not delivered is
// owed no matter the day it fell due on, and the round of a later day
// delivers it.
func TestTheRouteOfTheDayDeliversTheOverdueOrderThatIsOwed(t *testing.T) {
	t.Parallel()

	orders := &fakeOrders{held: []order.Order{
		theOrderDueOnTheDayOfTheRoutes(t, 10, theDayOfTheRoutes(t, "2026-03-13")),
	}}
	routes := &fakeRoutes{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	written, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-15",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
		},
	})
	if err != nil {
		t.Fatalf("the route of the order that is owed was refused: %v", err)
	}
	if len(routes.written) != 1 {
		t.Fatalf("the storage holds %d routes, want 1", len(routes.written))
	}
	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day of the route is %q, want %q", written.Day.String(), "2026-03-15")
	}
	if len(written.Stops) != 1 || written.Stops[0].OrderID != order.ID(10) {
		t.Fatalf("the route does not visit the overdue order: %v", written.Stops)
	}
	if len(orders.delivered) != 1 {
		t.Fatalf("the storage holds %d deliveries, want 1", len(orders.delivered))
	}
	if orders.delivered[0].ID != order.ID(10) {
		t.Errorf("the delivery is of the order %d, want 10", orders.delivered[0].ID)
	}
	if orders.delivered[0].State != order.StateDelivered {
		t.Errorf("the order %d is %q, want %q", orders.delivered[0].ID, orders.delivered[0].State, order.StateDelivered)
	}
}

// TestTheSecondRouteOfTheDayIsRefused proves that the route of a business
// and a day is written once.
func TestTheSecondRouteOfTheDayIsRefused(t *testing.T) {
	t.Parallel()

	day := theDayOfTheRoutes(t, "2026-03-14")

	orders := &fakeOrders{held: []order.Order{
		theOrderDueOnTheDayOfTheRoutes(t, 10, day),
	}}
	routes := &fakeRoutes{held: []order.Route{
		{ID: order.RouteID(1), BusinessID: business.ID(77), Day: day},
	}}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
		},
	})
	if err == nil {
		t.Fatal("a second route for the same day was written")
	}
	if !errors.Is(err, order.ErrRouteAlreadyWritten) {
		t.Fatalf("the error is %v, want %v", err, order.ErrRouteAlreadyWritten)
	}
	if len(routes.written) != 0 {
		t.Errorf("the storage holds %d routes, want none", len(routes.written))
	}
}

// TestTheRouteOfTheBusinessThatDoesNotHoldOrdersIsRefused proves that the
// route belongs to a business of its own.
func TestTheRouteOfTheBusinessThatDoesNotHoldOrdersIsRefused(t *testing.T) {
	t.Parallel()

	orders := &fakeOrders{}
	routes := &fakeRoutes{}
	personal := business.Business{
		ID:       business.ID(77),
		Kind:     business.KindPersonal,
		Name:     "Ana Pérez",
		Currency: "COP",
	}
	businesses := &fakeBusinesses{existing: &personal}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
		},
	})
	if err == nil {
		t.Fatal("the route of a business that does not hold orders was written")
	}
	if !errors.Is(err, order.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, order.ErrWrongBusiness)
	}
	if len(routes.written) != 0 {
		t.Errorf("the storage holds %d routes, want none", len(routes.written))
	}
}

// TestTheRouteWithoutADayIsRefused proves that the route carries the day of
// the round, that the book never invents one, and that the rule is the one
// of the domain: the use case does not hold it and it does not manufacture
// its refusal.
//
// The observable difference is the one the rule has: the use case ASKS THE
// DOMAIN, so it reads the business, its orders and its routes before the
// refusal — a use case that decided the rule by itself would refuse the day
// without ever asking.
func TestTheRouteWithoutADayIsRefused(t *testing.T) {
	t.Parallel()

	held := theOrderDueOnTheDayOfTheRoutes(t, 10, theDayOfTheRoutes(t, "2026-03-14"))

	orders := &fakeOrders{held: []order.Order{held}}
	routes := &fakeRoutes{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
		},
	})
	if err == nil {
		t.Fatal("the route without a day was written")
	}
	if !errors.Is(err, order.ErrMissingRouteDay) {
		t.Fatalf("the error is %v, want %v", err, order.ErrMissingRouteDay)
	}

	// The refusal is the one of the domain, with the words of the domain:
	// the use case asked it with the empty day the agent named none of.
	_, refusal := order.TheRoute(bakery.ID, []order.Order{held}, nil, order.RouteDefinition{
		Day:   ledger.Date{},
		Stops: []order.StopDefinition{{OrderID: 10, Cost: ledger.Amount(3000)}},
	})
	if err.Error() != refusal.Error() {
		t.Errorf("the error is %q, want the one of the domain %q", err.Error(), refusal.Error())
	}

	// The use case asked the domain, and asking it took the orders of the
	// business: a use case that decided the rule by itself would have
	// refused the day without reading them.
	if orders.listed != bakery.ID {
		t.Errorf("the orders of the business were not read: a use case that refuses the day by itself never asks " +
			"the domain, and the rule of the day is the one of the domain")
	}
	if routes.listed != bakery.ID {
		t.Errorf("the routes of the business were not read: the domain is asked with the routes that were written " +
			"already, and the use case that refuses the day by itself never asks it")
	}
	if len(routes.written) != 0 {
		t.Errorf("the storage holds %d routes, want none", len(routes.written))
	}
	if len(orders.delivered) != 0 {
		t.Errorf("the storage holds %d deliveries, want none", len(orders.delivered))
	}
}

// TestTheRouteWithoutAStopIsRefused proves that a route without a stop
// visits nobody, and that it is not a route.
func TestTheRouteWithoutAStopIsRefused(t *testing.T) {
	t.Parallel()

	orders := &fakeOrders{}
	routes := &fakeRoutes{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
	})
	if err == nil {
		t.Fatal("the route without a stop was written")
	}
	if !errors.Is(err, order.ErrNoStops) {
		t.Fatalf("the error is %v, want %v", err, order.ErrNoStops)
	}
	if len(routes.written) != 0 {
		t.Errorf("the storage holds %d routes, want none", len(routes.written))
	}
}

// TestTheRouteOfTheStopThatCostsLessThanNothingIsRefused proves that the
// money of the book counts whole pesos, and that a stop never costs less
// than nothing.
func TestTheRouteOfTheStopThatCostsLessThanNothingIsRefused(t *testing.T) {
	t.Parallel()

	day := theDayOfTheRoutes(t, "2026-03-14")

	orders := &fakeOrders{held: []order.Order{
		theOrderDueOnTheDayOfTheRoutes(t, 10, day),
	}}
	routes := &fakeRoutes{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: -500},
		},
	})
	if err == nil {
		t.Fatal("the route of a stop that costs less than nothing was written")
	}
	if !errors.Is(err, order.ErrInvalidCost) {
		t.Fatalf("the error is %v, want %v", err, order.ErrInvalidCost)
	}
	if len(routes.written) != 0 {
		t.Errorf("the storage holds %d routes, want none", len(routes.written))
	}
}

// TestTheRouteOfTheBusinessThatIsNotThereIsRefused proves that the route
// names the business of the round, and that nothing is written when the
// business is not there.
func TestTheRouteOfTheBusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	orders := &fakeOrders{}
	routes := &fakeRoutes{}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, &fakeBusinesses{}, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
		},
	})
	if err == nil {
		t.Fatal("the route of a business that is not there was written")
	}
	if !errors.Is(err, business.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, business.ErrNotFound)
	}
	if len(routes.written) != 0 {
		t.Errorf("the storage holds %d routes, want none", len(routes.written))
	}
}

// TestTheRouteOfTheDayThatIsNotADayIsRefused proves that the day of the
// round is a day of the book, and that the book never invents one.
func TestTheRouteOfTheDayThatIsNotADayIsRefused(t *testing.T) {
	t.Parallel()

	orders := &fakeOrders{}
	routes := &fakeRoutes{}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "the day of the week",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
		},
	})
	if err == nil {
		t.Fatal("the route of a day that is not a day of the book was written")
	}
	if !errors.Is(err, ledger.ErrInvalidDate) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidDate)
	}
	if len(routes.written) != 0 {
		t.Errorf("the storage holds %d routes, want none", len(routes.written))
	}
}

// TestTheRouteOfTheStorageThatIsNotThereIsRefused proves that the route
// is written through the storage, and that a storage that refuses leaves
// nothing behind.
func TestTheRouteOfTheStorageThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	day := theDayOfTheRoutes(t, "2026-03-14")

	orders := &fakeOrders{held: []order.Order{
		theOrderDueOnTheDayOfTheRoutes(t, 10, day),
	}}
	routes := &fakeRoutes{err: errors.New("the storage is not there")}
	bakery := theBakeryOfTheOrders()
	businesses := &fakeBusinesses{existing: &bakery}
	work := &fakeRouteWork{routes: routes, orders: orders}
	service := application.NewRouteService(routes, orders, businesses, work)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
		},
	})
	if err == nil {
		t.Fatal("the route was written and the storage refused it")
	}
	if len(orders.delivered) != 0 {
		t.Errorf("the storage holds %d deliveries, want none", len(orders.delivered))
	}
}

// TestTheRouteWithoutADatabaseIsRefused proves that the tools are there
// even when the service has no database, and that they say so.
func TestTheRouteWithoutADatabaseIsRefused(t *testing.T) {
	t.Parallel()

	service := application.NewRouteService(nil, nil, nil, nil)

	_, err := service.Record(t.Context(), ports.RecordRoute{
		BusinessID: 77,
		Day:        "2026-03-14",
		Stops: []ports.RouteStop{
			{OrderID: 10, Cost: 3000},
		},
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
