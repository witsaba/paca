//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/order"
)

// routesRepository returns the repository of the routes, bound to the
// database of the suite.
func routesRepository(t *testing.T) *postgres.RouteRepository {
	t.Helper()

	repo, err := postgres.NewRouteRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the routes could not be created: %v", err)
	}

	return repo
}

// theRouteOfTheTest builds the route the tests write: the stops of the
// orders of the round of that day.
func theRouteOfTheTest(t *testing.T, bakery business.Business, orders []order.Order, day ledger.Date) order.Route {
	t.Helper()

	built, err := order.TheRoute(
		bakery.ID,
		orders,
		nil,
		order.RouteDefinition{
			Day: day,
			Stops: []order.StopDefinition{
				{OrderID: orders[0].ID, Cost: ledger.Amount(3000)},
				{OrderID: orders[1].ID, Cost: ledger.Amount(1500)},
			},
		},
	)
	if err != nil {
		t.Fatalf("the route was refused: %v", err)
	}

	return built
}

// TestTheRouteIsKeptAndReadBackWithItsStops proves the rules of the table
// against a real database: the route is written with its stops, in the order
// the agent will visit the clients in, and it is read back as it was
// written.
func TestTheRouteIsKeptAndReadBackWithItsStops(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := routesRepository(t)
	orders := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	day := theDayOfTheOrders(t, "2026-03-14")

	first, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	second, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	written, err := repo.Create(ctx, theRouteOfTheTest(t, bakery, []order.Order{first, second}, day))
	if err != nil {
		t.Fatalf("the route could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the route has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the route is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.Day.String() != "2026-03-14" {
		t.Errorf("the day of the route is %q, want %q", written.Day.String(), "2026-03-14")
	}
	if len(written.Stops) != 2 {
		t.Fatalf("the route carries %d stops, want 2", len(written.Stops))
	}
	if written.Stops[0].OrderID != first.ID {
		t.Errorf("the first stop is the order %d, want %d", written.Stops[0].OrderID, first.ID)
	}
	if written.Stops[0].Cost != ledger.Amount(3000) {
		t.Errorf("the cost of the first stop is %d, want 3000", written.Stops[0].Cost)
	}
	if written.Stops[1].OrderID != second.ID {
		t.Errorf("the second stop is the order %d, want %d", written.Stops[1].OrderID, second.ID)
	}
	if written.Total() != ledger.Amount(4500) {
		t.Errorf("the total of the route is %d, want 4500", written.Total())
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the routes could not be read: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the business holds %d routes, want 1", len(held))
	}
	if len(held[0].Stops) != 2 {
		t.Fatalf("the route read back carries %d stops, want 2", len(held[0].Stops))
	}
	if held[0].Stops[0].OrderID != first.ID {
		t.Errorf("the first stop read back is the order %d, want %d", held[0].Stops[0].OrderID, first.ID)
	}
	if held[0].Stops[1].Cost != ledger.Amount(1500) {
		t.Errorf("the cost of the second stop is %d, want 1500", held[0].Stops[1].Cost)
	}
}

// TestTheRouteOfAnotherBusinessIsRefused proves the rule of the table: the
// stop of an order of another business is refused, and the route of a
// business holds the routes of that business and of no other one.
func TestTheRouteOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := routesRepository(t)
	orders := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	other := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, other)
	baked := theProductOfTheOrders(t, ctx, other)
	day := theDayOfTheOrders(t, "2026-03-14")

	foreign, err := orders.Create(ctx, theOrderOfTheTest(t, other, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	own, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, theClientOfTheOrders(t, ctx, bakery),
		theProductOfTheOrders(t, ctx, bakery), day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	// The domain refuses the stop of an order of another business before
	// the storage is asked: the row below is written the way another
	// writer would write it, and the table refuses it.
	_, err = repo.Create(ctx, order.Route{
		BusinessID: bakery.ID,
		Day:        day,
		Stops: []order.Stop{
			{OrderID: own.ID, Cost: ledger.Amount(3000)},
			{OrderID: foreign.ID, Cost: ledger.Amount(1500)},
		},
	})
	if err == nil {
		t.Fatal("the stop of an order of another business was written")
	}
	if !errors.Is(err, order.ErrStopNotAnOrder) {
		t.Fatalf("the error is %v, want %v", err, order.ErrStopNotAnOrder)
	}
}

// TestTheSecondRouteOfTheDayIsRefused proves the rule of the table: the
// route of a business and a day is written once, and the table refuses a
// second one.
func TestTheSecondRouteOfTheDayIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := routesRepository(t)
	orders := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	day := theDayOfTheOrders(t, "2026-03-14")

	first, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	second, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	if _, err := repo.Create(ctx, theRouteOfTheTest(t, bakery, []order.Order{first, second}, day)); err != nil {
		t.Fatalf("the route could not be written: %v", err)
	}

	_, err = repo.Create(ctx, theRouteOfTheTest(t, bakery, []order.Order{first, second}, day))
	if err == nil {
		t.Fatal("a second route for the same day was written")
	}
	if !errors.Is(err, order.ErrRouteAlreadyWritten) {
		t.Fatalf("the error is %v, want %v", err, order.ErrRouteAlreadyWritten)
	}
}

// TestTheTableRefusesACostThatIsLessThanNothing proves the rule of the
// table: the money of the book counts whole pesos, and the cost of a stop
// is never negative.
//
// The row is written the way another writer would write it, and the table
// refuses it: the check `route_stops_cost_check` holds the rule of the
// domain as a defense in depth.
func TestTheTableRefusesACostThatIsLessThanNothing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orders := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	day := theDayOfTheOrders(t, "2026-03-14")

	first, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	var routeID int64
	if err := theSuite.db.QueryRowContext(ctx,
		`INSERT INTO routes (business_id, day) VALUES ($1, $2) RETURNING id`,
		int64(bakery.ID), day,
	).Scan(&routeID); err != nil {
		t.Fatalf("the route could not be written: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO route_stops (route_id, business_id, order_id, position, cost)
		 VALUES ($1, $2, $3, 1, -500)`,
		routeID, int64(bakery.ID), int64(first.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a stop that costs less than nothing")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "route_stops_cost_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "route_stops_cost_check")
	}
}

// TestTheTableKeepsAStopThatCostsNothing proves the other side of the rule
// of the table: a stop the owner walks to costs nothing, and the table
// keeps it.
func TestTheTableKeepsAStopThatCostsNothing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := routesRepository(t)
	orders := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	day := theDayOfTheOrders(t, "2026-03-14")

	first, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	written, err := repo.Create(ctx, order.Route{
		BusinessID: bakery.ID,
		Day:        day,
		Stops: []order.Stop{
			{OrderID: first.ID, Cost: ledger.Amount(0)},
		},
	})
	if err != nil {
		t.Fatalf("the route of the stop that costs nothing could not be written: %v", err)
	}
	if len(written.Stops) != 1 {
		t.Fatalf("the route carries %d stops, want 1", len(written.Stops))
	}
	if written.Stops[0].Cost != ledger.Amount(0) {
		t.Errorf("the cost of the stop is %d, want 0", written.Stops[0].Cost)
	}
	if written.Total() != ledger.Amount(0) {
		t.Errorf("the total of the route is %d, want 0", written.Total())
	}
}

// TestTheDeliveryOfTheOrdersOfTheStopsIsWritten proves the rule of the
// table: writing the route delivers the orders of its stops, and the
// delivery of an order is written with the route of the round.
func TestTheDeliveryOfTheOrdersOfTheStopsIsWritten(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := routesRepository(t)
	orders := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	day := theDayOfTheOrders(t, "2026-03-14")

	first, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	second, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	route := theRouteOfTheTest(t, bakery, []order.Order{first, second}, day)

	if _, err := repo.Create(ctx, route); err != nil {
		t.Fatalf("the route could not be written: %v", err)
	}

	delivered, err := route.TheDelivery([]order.Order{first, second})
	if err != nil {
		t.Fatalf("the delivery of the route was refused: %v", err)
	}

	if err := orders.Deliver(ctx, delivered); err != nil {
		t.Fatalf("the delivery of the orders could not be written: %v", err)
	}

	held, err := orders.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the orders could not be read: %v", err)
	}

	for _, one := range held {
		if one.State != order.StateDelivered {
			t.Errorf("the order %d is %q, want %q", one.ID, one.State, order.StateDelivered)
		}
	}

	round := order.TheRound(day, held)
	if len(round.Owed) != 0 {
		t.Errorf("the round owes %d orders, want none: the orders of the stops were delivered", len(round.Owed))
	}
}

// TestTheRouteOfTheDayDeliversTheOrderThatIsOwed proves the rule of the
// round against a real database: the stops of a route are the orders the
// business still OWES, the ones that are due on the day of the route and the
// ones that fell due on an earlier day. The order that was not delivered is
// owed no matter the day it fell due on, and the route of a later day
// delivers it.
func TestTheRouteOfTheDayDeliversTheOrderThatIsOwed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := routesRepository(t)
	orders := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	day := theDayOfTheOrders(t, "2026-03-15")

	overdue, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, theDayOfTheOrders(t, "2026-03-13")))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	route, err := order.TheRoute(
		bakery.ID,
		[]order.Order{overdue},
		nil,
		order.RouteDefinition{
			Day: day,
			Stops: []order.StopDefinition{
				{OrderID: overdue.ID, Cost: ledger.Amount(3000)},
			},
		},
	)
	if err != nil {
		t.Fatalf("the route of the order that is owed was refused: %v", err)
	}

	if _, err := repo.Create(ctx, route); err != nil {
		t.Fatalf("the route could not be written: %v", err)
	}

	delivered, err := route.TheDelivery([]order.Order{overdue})
	if err != nil {
		t.Fatalf("the delivery of the route was refused: %v", err)
	}

	if err := orders.Deliver(ctx, delivered); err != nil {
		t.Fatalf("the delivery of the order could not be written: %v", err)
	}

	held, err := orders.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the orders could not be read: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the business holds %d orders, want 1", len(held))
	}
	if held[0].State != order.StateDelivered {
		t.Errorf("the order that fell due on an earlier day is %q, want %q", held[0].State, order.StateDelivered)
	}

	round := order.TheRound(day, held)
	if len(round.Owed) != 0 {
		t.Errorf("the round owes %d orders, want none: the overdue order was delivered by the route of the day",
			len(round.Owed))
	}
}

// TestTheAdapterRefusesToReopenADeliveredOrder proves the rule of the port
// against a real database: the delivery moves an order that is open, and an
// order that was delivered already is refused. A caller that bypasses the
// domain cannot move a delivered order back to the open one.
func TestTheAdapterRefusesToReopenADeliveredOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orders := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	day := theDayOfTheOrders(t, "2026-03-14")

	first, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	delivered, err := first.Delivered()
	if err != nil {
		t.Fatalf("the domain refused the delivery: %v", err)
	}

	if err := orders.Deliver(ctx, []order.Order{delivered}); err != nil {
		t.Fatalf("the delivery of the order could not be written: %v", err)
	}

	// The order the caller hands the adapter is the open one: another
	// writer wrote the delivery between the reading and the writing.
	_, err = first.Delivered()
	if err != nil {
		t.Fatalf("the order of the test is not open: %v", err)
	}

	err = orders.Deliver(ctx, []order.Order{first})
	if err == nil {
		t.Fatal("the delivered order was moved back to the open one")
	}
	if !errors.Is(err, order.ErrAlreadyDelivered) {
		t.Fatalf("the error is %v, want %v", err, order.ErrAlreadyDelivered)
	}

	held, err := orders.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the orders could not be read: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the business holds %d orders, want 1", len(held))
	}
	if held[0].State != order.StateDelivered {
		t.Errorf("the order is %q, want %q: the delivery is written once", held[0].State, order.StateDelivered)
	}
}

// TestTheTableRefusesTheSameOrderInTwoStops proves the rule of the table:
// the route visits every order once, because the total of the route is
// derived from the stops. The row is written the way another writer would
// write it, and the unique index `route_stops_one_stop_per_order` refuses the
// second stop of the same order.
func TestTheTableRefusesTheSameOrderInTwoStops(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orders := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	day := theDayOfTheOrders(t, "2026-03-14")

	first, err := orders.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, day))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	var routeID int64
	if err := theSuite.db.QueryRowContext(ctx,
		`INSERT INTO routes (business_id, day) VALUES ($1, $2) RETURNING id`,
		int64(bakery.ID), day,
	).Scan(&routeID); err != nil {
		t.Fatalf("the route could not be written: %v", err)
	}

	if _, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO route_stops (route_id, business_id, order_id, position, cost)
		 VALUES ($1, $2, $3, 1, 3000)`,
		routeID, int64(bakery.ID), int64(first.ID),
	); err != nil {
		t.Fatalf("the first stop of the order could not be written: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO route_stops (route_id, business_id, order_id, position, cost)
		 VALUES ($1, $2, $3, 2, 1500)`,
		routeID, int64(bakery.ID), int64(first.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a route that visits the same order twice")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != uniqueViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, uniqueViolation)
	}
	if pgErr.ConstraintName != "route_stops_one_stop_per_order" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "route_stops_one_stop_per_order")
	}
}
