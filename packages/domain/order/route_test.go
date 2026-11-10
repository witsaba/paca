package order

import (
	"errors"
	"reflect"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
)

// theOrderDueOnTheDayOfTheTest is an order of the business of the test, due
// on the day the test asks for.
func theOrderDueOnTheDayOfTheTest(t *testing.T, id ID, dueOn ledger.Date) Order {
	t.Helper()

	built, err := New(
		business.ID(77),
		theClientOfTheTest(t),
		dueOn,
		theLinesOfTheTest(t, business.ID(77)),
	)
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	built.ID = id

	return built
}

// theRouteStopsOfTheTest are the stops the agent computed for the test: the
// orders of the round of the day, in the order it will visit them.
func theRouteStopsOfTheTest(costs ...ledger.Amount) []StopDefinition {
	stops := make([]StopDefinition, 0, len(costs))

	for index, cost := range costs {
		stops = append(stops, StopDefinition{
			OrderID: ID(10 + index),
			Cost:    cost,
		})
	}

	return stops
}

// theRouteOfTheTest is the route the agent computed for the day of the test.
func theRouteOfTheTest(day ledger.Date, stops []StopDefinition) RouteDefinition {
	return RouteDefinition{Day: day, Stops: stops}
}

// TestTheRouteCarriesTheStopsInTheOrderTheAgentWillVisitThem proves the
// rule of the domain: the route carries the stops in the order the agent
// will visit the clients in, and it carries what each one costs.
func TestTheRouteCarriesTheStopsInTheOrderTheAgentWillVisitThem(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest("2026-03-14")

	held := []Order{
		theOrderDueOnTheDayOfTheTest(t, ID(10), day),
		theOrderDueOnTheDayOfTheTest(t, ID(11), day),
	}

	built, err := TheRoute(
		business.ID(77),
		held,
		nil,
		theRouteOfTheTest(day, theRouteStopsOfTheTest(3000, 1500)),
	)
	if err != nil {
		t.Fatalf("the route was refused: %v", err)
	}

	if built.BusinessID != business.ID(77) {
		t.Errorf("the business of the route is %d, want 77", built.BusinessID)
	}
	if built.Day.String() != "2026-03-14" {
		t.Errorf("the day of the route is %q, want %q", built.Day.String(), "2026-03-14")
	}
	if len(built.Stops) != 2 {
		t.Fatalf("the route carries %d stops, want 2", len(built.Stops))
	}
	if built.Stops[0].OrderID != ID(10) {
		t.Errorf("the first stop is the order %d, want 10", built.Stops[0].OrderID)
	}
	if built.Stops[1].OrderID != ID(11) {
		t.Errorf("the second stop is the order %d, want 11", built.Stops[1].OrderID)
	}
	if built.Stops[0].Cost != ledger.Amount(3000) {
		t.Errorf("the cost of the first stop is %d, want 3000", built.Stops[0].Cost)
	}
	if built.Stops[1].Cost != ledger.Amount(1500) {
		t.Errorf("the cost of the second stop is %d, want 1500", built.Stops[1].Cost)
	}
}

// TestTheTotalOfTheRouteIsDerivedFromItsStops proves the rule of the
// domain: the total of the route is derived from the stops, and it is
// never typed: the model of the route carries no total of its own.
func TestTheTotalOfTheRouteIsDerivedFromItsStops(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest("2026-03-14")

	held := []Order{
		theOrderDueOnTheDayOfTheTest(t, ID(10), day),
		theOrderDueOnTheDayOfTheTest(t, ID(11), day),
	}

	built, err := TheRoute(
		business.ID(77),
		held,
		nil,
		theRouteOfTheTest(day, theRouteStopsOfTheTest(3000, 1500)),
	)
	if err != nil {
		t.Fatalf("the route was refused: %v", err)
	}

	if built.Total() != ledger.Amount(4500) {
		t.Errorf("the total of the route is %d, want 4500", built.Total())
	}

	model := reflect.TypeOf(Route{})
	for index := 0; index < model.NumField(); index++ {
		if model.Field(index).Name == "Total" {
			t.Error("the route carries a total of its own: the total is derived from the stops")
		}
	}
}

// TestTheCostOfAStopIsNeverNegative proves the rule of the domain: the cost
// of a stop is the money of the stop in whole pesos, and it is never
// negative: a stop the owner walks to costs nothing, and no stop costs less
// than nothing.
func TestTheCostOfAStopIsNeverNegative(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest("2026-03-14")

	held := []Order{
		theOrderDueOnTheDayOfTheTest(t, ID(10), day),
	}

	built, err := TheRoute(
		business.ID(77),
		held,
		nil,
		theRouteOfTheTest(day, theRouteStopsOfTheTest(0)),
	)
	if err != nil {
		t.Fatalf("the route of a stop that costs nothing was refused: %v", err)
	}
	if built.Stops[0].Cost != ledger.Amount(0) {
		t.Errorf("the cost of the stop is %d, want 0", built.Stops[0].Cost)
	}
	if built.Total() != ledger.Amount(0) {
		t.Errorf("the total of the route is %d, want 0", built.Total())
	}

	_, err = TheRoute(
		business.ID(77),
		held,
		nil,
		theRouteOfTheTest(day, theRouteStopsOfTheTest(-500)),
	)
	if err == nil {
		t.Fatal("the route of a stop that costs less than nothing was built")
	}
	if !errors.Is(err, ErrInvalidCost) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidCost)
	}
}

// TestTheRouteRefusesAStopThatIsNotAnOrderOfThatBusiness proves the rule of
// the domain: the stops of a route are orders of the same business.
func TestTheRouteRefusesAStopThatIsNotAnOrderOfThatBusiness(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest("2026-03-14")

	held := []Order{
		theOrderDueOnTheDayOfTheTest(t, ID(10), day),
	}

	_, err := TheRoute(
		business.ID(77),
		held,
		nil,
		theRouteOfTheTest(day, []StopDefinition{{OrderID: ID(999), Cost: ledger.Amount(1000)}}),
	)
	if err == nil {
		t.Fatal("the route of an order of another business was built")
	}
	if !errors.Is(err, ErrStopNotAnOrder) {
		t.Fatalf("the error is %v, want %v", err, ErrStopNotAnOrder)
	}
}

// TestTheRouteRefusesAStopThatIsNotAnOrderOfThatRound proves the rule of
// the domain: the stops of a route are the orders of that round, the ones
// that are due on that day.
func TestTheRouteRefusesAStopThatIsNotAnOrderOfThatRound(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest("2026-03-14")

	held := []Order{
		theOrderDueOnTheDayOfTheTest(t, ID(10), day),
		theOrderDueOnTheDayOfTheTest(t, ID(11), theDayOfTheTest("2026-03-15")),
	}

	_, err := TheRoute(
		business.ID(77),
		held,
		nil,
		theRouteOfTheTest(day, []StopDefinition{{OrderID: ID(11), Cost: ledger.Amount(1000)}}),
	)
	if err == nil {
		t.Fatal("the route of an order that is not due on that day was built")
	}
	if !errors.Is(err, ErrStopNotOfTheRound) {
		t.Fatalf("the error is %v, want %v", err, ErrStopNotOfTheRound)
	}
}

// TestTheRouteOfADayIsWrittenOnce proves the rule of the domain: the route
// of a business and a day is written once, and a second route for the same
// day is refused.
func TestTheRouteOfADayIsWrittenOnce(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest("2026-03-14")

	held := []Order{
		theOrderDueOnTheDayOfTheTest(t, ID(10), day),
	}

	written := []Route{
		{ID: RouteID(1), BusinessID: business.ID(77), Day: day},
	}

	_, err := TheRoute(
		business.ID(77),
		held,
		written,
		theRouteOfTheTest(day, theRouteStopsOfTheTest(1000)),
	)
	if err == nil {
		t.Fatal("a second route for the same day was built")
	}
	if !errors.Is(err, ErrRouteAlreadyWritten) {
		t.Fatalf("the error is %v, want %v", err, ErrRouteAlreadyWritten)
	}

	// The route of another day is not the route of this one: the rule
	// holds for the day and the business of the route.
	_, err = TheRoute(
		business.ID(77),
		held,
		[]Route{
			{ID: RouteID(1), BusinessID: business.ID(77), Day: theDayOfTheTest("2026-03-15")},
		},
		theRouteOfTheTest(day, theRouteStopsOfTheTest(1000)),
	)
	if err != nil {
		t.Fatalf("the route of the day was refused: %v", err)
	}

	_, err = TheRoute(
		business.ID(77),
		held,
		[]Route{
			{ID: RouteID(1), BusinessID: business.ID(88), Day: day},
		},
		theRouteOfTheTest(day, theRouteStopsOfTheTest(1000)),
	)
	if err != nil {
		t.Fatalf("the route of the day was refused: %v", err)
	}
}

// TestTheRouteRefusesToCarryNoStop proves the rule of the domain: a route
// without a stop visits nobody, and it is not a route.
func TestTheRouteRefusesToCarryNoStop(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest("2026-03-14")

	held := []Order{
		theOrderDueOnTheDayOfTheTest(t, ID(10), day),
	}

	_, err := TheRoute(business.ID(77), held, nil, theRouteOfTheTest(day, nil))
	if err == nil {
		t.Fatal("the route without a stop was built")
	}
	if !errors.Is(err, ErrNoStops) {
		t.Fatalf("the error is %v, want %v", err, ErrNoStops)
	}
}

// TestTheRouteRefusesTheDayItCarriesNone proves the rule of the domain: the
// route is of a day, and the book never invents one.
func TestTheRouteRefusesTheDayItCarriesNone(t *testing.T) {
	t.Parallel()

	_, err := TheRoute(
		business.ID(77),
		nil,
		nil,
		RouteDefinition{Stops: theRouteStopsOfTheTest(1000)},
	)
	if err == nil {
		t.Fatal("the route without a day was built")
	}
	if !errors.Is(err, ErrMissingRouteDay) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingRouteDay)
	}
}

// TestWritingTheRouteDeliversTheOrdersOfItsStops proves the rule of the
// domain: the route the agent writes back is the round it made, and writing
// the route delivers the orders of its stops.
func TestWritingTheRouteDeliversTheOrdersOfItsStops(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest("2026-03-14")

	held := []Order{
		theOrderDueOnTheDayOfTheTest(t, ID(10), day),
		theOrderDueOnTheDayOfTheTest(t, ID(11), day),
	}

	built, err := TheRoute(
		business.ID(77),
		held,
		nil,
		theRouteOfTheTest(day, theRouteStopsOfTheTest(3000, 1500)),
	)
	if err != nil {
		t.Fatalf("the route was refused: %v", err)
	}

	delivered, err := built.TheDelivery(held)
	if err != nil {
		t.Fatalf("the delivery of the route was refused: %v", err)
	}
	if len(delivered) != 2 {
		t.Fatalf("the route delivers %d orders, want 2", len(delivered))
	}
	for _, one := range delivered {
		if one.State != StateDelivered {
			t.Errorf("the order %d is %q, want %q", one.ID, one.State, StateDelivered)
		}
	}

	// The order that was delivered already is not delivered twice.
	_, err = built.TheDelivery([]Order{
		held[0],
		theDeliveredOrderOfTheTest(t, ID(11), day),
	})
	if err == nil {
		t.Fatal("the route delivered an order that was delivered already")
	}
	if !errors.Is(err, ErrAlreadyDelivered) {
		t.Fatalf("the error is %v, want %v", err, ErrAlreadyDelivered)
	}
}

// theDeliveredOrderOfTheTest is an order that was delivered already.
func theDeliveredOrderOfTheTest(t *testing.T, id ID, dueOn ledger.Date) Order {
	t.Helper()

	built, err := NewFromFacts(Facts{
		ID:         id,
		BusinessID: business.ID(77),
		PartyID:    party.ID(10),
		PartyName:  "Doña Rosalba",
		DueOn:      dueOn,
		Lines: []Line{
			{ProductID: product.ID(5), ProductName: "Pan de molde", Quantity: 12, Unit: "unit"},
		},
		State: StateDelivered,
	})
	if err != nil {
		t.Fatalf("the delivered order was refused: %v", err)
	}

	return built
}

// TestTheRoundWithoutARouteIsARoundThatWasNeverPlanned proves the rule of
// the domain: a round without a route is a round that was never planned.
func TestTheRoundWithoutARouteIsARoundThatWasNeverPlanned(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest("2026-03-14")

	held := []Order{
		theOrderDueOnTheDayOfTheTest(t, ID(10), day),
	}

	round := TheRound(day, held)
	if round.Planned() {
		t.Error("the round without a route says it was planned")
	}

	built, err := TheRoute(
		business.ID(77),
		held,
		nil,
		theRouteOfTheTest(day, theRouteStopsOfTheTest(1000)),
	)
	if err != nil {
		t.Fatalf("the route was refused: %v", err)
	}

	planned := TheRoundWithRoute(round, []Route{built}, business.ID(77))
	if !planned.Planned() {
		t.Error("the round with a route says it was never planned")
	}
	if planned.Route == nil {
		t.Fatal("the round with a route carries no route")
	}
	if planned.Route.Total() != ledger.Amount(1000) {
		t.Errorf("the total of the route of the round is %d, want 1000", planned.Route.Total())
	}
}

// TestNewRouteFromFactsBuildsTheRouteOfTheStorage proves that the domain
// builds the route the storage holds, and that the row never decides what a
// route is.
func TestNewRouteFromFactsBuildsTheRouteOfTheStorage(t *testing.T) {
	t.Parallel()

	built, err := NewRouteFromFacts(RouteFacts{
		ID:         RouteID(3),
		BusinessID: business.ID(77),
		Day:        theDayOfTheTest("2026-03-14"),
		Stops: []Stop{
			{OrderID: ID(10), Cost: ledger.Amount(3000)},
			{OrderID: ID(11), Cost: ledger.Amount(1500)},
		},
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if built.ID != RouteID(3) {
		t.Errorf("the identity is %d, want 3", built.ID)
	}
	if built.Total() != ledger.Amount(4500) {
		t.Errorf("the total of the route is %d, want 4500", built.Total())
	}
	if len(built.Stops) != 2 {
		t.Fatalf("the route read back carries %d stops, want 2", len(built.Stops))
	}
}

// TestNewRouteFromFactsRefusesTheRowThatIsNotWhole proves the rule of the
// domain: a row that breaks a rule of the route is refused, and it is never
// read as a route.
func TestNewRouteFromFactsRefusesTheRowThatIsNotWhole(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		facts RouteFacts
		want  error
	}{
		{
			name:  "the route of no business",
			facts: RouteFacts{Day: theDayOfTheTest("2026-03-14"), Stops: []Stop{{OrderID: 10}}},
			want:  ErrMissingBusiness,
		},
		{
			name:  "the route that carries no day",
			facts: RouteFacts{BusinessID: 77, Stops: []Stop{{OrderID: 10}}},
			want:  ErrMissingRouteDay,
		},
		{
			name:  "the route that carries no stop",
			facts: RouteFacts{BusinessID: 77, Day: theDayOfTheTest("2026-03-14")},
			want:  ErrNoStops,
		},
		{
			name: "the stop that costs less than nothing",
			facts: RouteFacts{
				BusinessID: 77,
				Day:        theDayOfTheTest("2026-03-14"),
				Stops:      []Stop{{OrderID: 10, Cost: ledger.Amount(-1)}},
			},
			want: ErrInvalidCost,
		},
		{
			name: "the stop that names no order",
			facts: RouteFacts{
				BusinessID: 77,
				Day:        theDayOfTheTest("2026-03-14"),
				Stops:      []Stop{{Cost: ledger.Amount(1000)}},
			},
			want: ErrStopNotAnOrder,
		},
	}

	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewRouteFromFacts(one.facts)
			if err == nil {
				t.Fatal("the row that is not whole was read")
			}
			if !errors.Is(err, one.want) {
				t.Fatalf("the error is %v, want %v", err, one.want)
			}
		})
	}
}
