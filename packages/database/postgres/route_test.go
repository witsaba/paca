package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/order"
)

// TestTheRepositoryOfTheRoutesNeedsAPool proves that the repository of the
// routes refuses a pool that is not there: a repository built on nothing
// would be a nil pointer inside an interface.
func TestTheRepositoryOfTheRoutesNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewRouteRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

// TestTheRepositoryOfTheRoutesWithoutADatabaseRefusesToWork proves that the
// repository of the routes answers that the storage is not open, and never a
// nil pointer.
func TestTheRepositoryOfTheRoutesWithoutADatabaseRefusesToWork(t *testing.T) {
	t.Parallel()

	routes := &RouteRepository{}

	if _, err := routes.Create(t.Context(), order.Route{}); !errors.Is(err, order.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, order.ErrNoStorage)
	}
	if _, err := routes.List(t.Context(), business.ID(77)); !errors.Is(err, order.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, order.ErrNoStorage)
	}
}

// TestTheRowOfARouteIsMappedOntoTheModel proves that the row is mapped
// onto the model of the domain, and that the model decides what a route is.
func TestTheRowOfARouteIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	mapped, err := routeFromRow(db.Route{
		ID:         7,
		BusinessID: 77,
		Day:        pgtype.Date{Time: mustDate(t, "2026-03-14"), Valid: true},
	}, []order.Stop{
		{OrderID: order.ID(10), Cost: ledger.Amount(3000)},
		{OrderID: order.ID(11), Cost: ledger.Amount(1500)},
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if mapped.ID != order.RouteID(7) {
		t.Errorf("the identity is %d, want 7", mapped.ID)
	}
	if mapped.BusinessID != business.ID(77) {
		t.Errorf("the business of the route is %d, want 77", mapped.BusinessID)
	}
	if mapped.Day.String() != "2026-03-14" {
		t.Errorf("the day of the route is %q, want %q", mapped.Day.String(), "2026-03-14")
	}
	if len(mapped.Stops) != 2 {
		t.Fatalf("the route read back carries %d stops, want 2", len(mapped.Stops))
	}
	if mapped.Stops[0].OrderID != order.ID(10) {
		t.Errorf("the first stop is the order %d, want 10", mapped.Stops[0].OrderID)
	}
	if mapped.Stops[0].Cost != ledger.Amount(3000) {
		t.Errorf("the cost of the first stop is %d, want 3000", mapped.Stops[0].Cost)
	}
	if mapped.Total() != ledger.Amount(4500) {
		t.Errorf("the total of the route is %d, want 4500", mapped.Total())
	}
}

// TestTheRowOfARouteThatBreaksTheRuleOfTheDomainIsRefused proves that the
// model refuses a row that another writer broke: a route without its stops
// visits nobody.
func TestTheRowOfARouteThatBreaksTheRuleOfTheDomainIsRefused(t *testing.T) {
	t.Parallel()

	_, err := routeFromRow(db.Route{
		ID:         7,
		BusinessID: 77,
		Day:        pgtype.Date{Time: mustDate(t, "2026-03-14"), Valid: true},
	}, nil)
	if err == nil {
		t.Fatal("the route without a stop was read")
	}
	if !errors.Is(err, order.ErrNoStops) {
		t.Fatalf("the error is %v, want %v", err, order.ErrNoStops)
	}
}

// TestTheRefusalsOfTheDatabaseAreTranslatedForTheRoutes proves that the
// truth of the database reaches the caller as the errors of the port.
func TestTheRefusalsOfTheDatabaseAreTranslatedForTheRoutes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     *pgconn.PgError
		want    error
		message string
	}{
		{
			name: "the second route of the day",
			err: &pgconn.PgError{
				Code:           uniqueViolation,
				ConstraintName: routesOnePerBusinessAndDayConstraint,
			},
			want:    order.ErrRouteAlreadyWritten,
			message: "refused the row",
		},
		{
			name: "the stop of an order of another business",
			err: &pgconn.PgError{
				Code:           foreignKeyViolation,
				ConstraintName: routeStopsOrderConstraint,
			},
			want:    order.ErrStopNotAnOrder,
			message: "refused the row",
		},
		{
			name: "the stop of a route of another business",
			err: &pgconn.PgError{
				Code:           foreignKeyViolation,
				ConstraintName: routeStopsRouteConstraint,
			},
			want:    order.ErrInvalidValue,
			message: "refused the row",
		},
		{
			name: "the row that breaks a rule of the table",
			err: &pgconn.PgError{
				Code:           checkViolation,
				ConstraintName: "route_stops_cost_check",
			},
			want:    order.ErrInvalidValue,
			message: "refused the row",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := writeRouteError(testCase.err)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
			if !strings.Contains(err.Error(), testCase.message) {
				t.Errorf("the error is %q, want it to say %q", err.Error(), testCase.message)
			}
		})
	}
}
