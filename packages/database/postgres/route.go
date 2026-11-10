package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/order"
)

// The names the database gives to the constraints of the tables of the
// routes.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	// routesOnePerBusinessAndDayConstraint is the name of the index that
	// refuses a second route of the same day.
	routesOnePerBusinessAndDayConstraint = "routes_one_route_per_business_and_day"

	// routeStopsRouteConstraint is the name of the foreign key that
	// refuses a stop of a route of another business.
	routeStopsRouteConstraint = "route_stops_route_of_its_business_fkey"

	// routeStopsOrderConstraint is the name of the foreign key that
	// refuses a stop of an order of another business.
	routeStopsOrderConstraint = "route_stops_order_of_its_business_fkey"
)

// RouteRepository is the adapter of the storage of the routes.
//
// The route and its stops are written in one transaction: a route without
// its stops visits nobody, and a half-written route is a route nobody can
// read.
//
// The row never decides what a route is: the model is built by the domain,
// through its own constructors.
type RouteRepository struct {
	pool *Pool
}

// NewRouteRepository returns the repository of the routes, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewRouteRepository(pool *Pool) (*RouteRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &RouteRepository{pool: pool}, nil
}

// Create writes a route with its stops, and answers it with the identity
// that the database generated.
//
// The route and its stops are written in one transaction: the set joins the
// transaction of the caller when the caller opened one, and opens its own
// when it did not. A route whose stop is refused leaves nothing behind.
//
// The error is order.ErrRouteAlreadyWritten when the business already wrote
// the route of that day, order.ErrInvalidValue when the row breaks a rule
// of the table, and order.ErrNoStorage when the storage is not open.
func (r *RouteRepository) Create(ctx context.Context, written order.Route) (order.Route, error) {
	if r == nil || r.pool == nil {
		return order.Route{}, order.ErrNoStorage
	}

	if _, ok := transactionIn(ctx); ok {
		return r.write(ctx, r.pool.queriesFor(ctx), written)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return order.Route{}, fmt.Errorf("the transaction of the routes could not be opened: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	recorded, err := r.write(ctx, r.pool.queriesFor(ctx).WithTx(tx), written)
	if err != nil {
		return order.Route{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return order.Route{}, fmt.Errorf("the transaction of the routes could not be committed: %w", err)
	}

	return recorded, nil
}

// write writes the route and its stops, and answers the route that was
// written.
func (r *RouteRepository) write(ctx context.Context, queries *db.Queries, written order.Route) (order.Route, error) {
	row, err := queries.CreateRoute(ctx, db.CreateRouteParams{
		BusinessID: int64(written.BusinessID),
		Day:        dateOf(written.Day),
	})
	if err != nil {
		return order.Route{}, writeRouteError(err)
	}

	stops := make([]order.Stop, 0, len(written.Stops))

	for position, one := range written.Stops {
		stop, err := queries.CreateRouteStop(ctx, db.CreateRouteStopParams{
			RouteID:    row.ID,
			BusinessID: int64(written.BusinessID),
			OrderID:    int64(one.OrderID),
			Position:   int32(position + 1),
			Cost:       int64(one.Cost),
		})
		if err != nil {
			return order.Route{}, writeRouteError(err)
		}

		stops = append(stops, order.Stop{
			OrderID: order.ID(stop.OrderID),
			Cost:    ledger.Amount(stop.Cost),
		})
	}

	// The route is built by the domain: the row never decides what a
	// route is, and a row that another writer broke is refused.
	recorded, err := routeFromRow(row, stops)
	if err != nil {
		return order.Route{}, err
	}

	return recorded, nil
}

// List returns the routes of the business, in the order of their identity,
// with their stops.
//
// The error is order.ErrNoStorage when the storage is not open.
func (r *RouteRepository) List(ctx context.Context, own business.ID) ([]order.Route, error) {
	if r == nil || r.pool == nil {
		return nil, order.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListRoutesByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeRouteError(err)
	}

	routes := make([]order.Route, 0, len(rows))

	for _, row := range rows {
		found, err := r.routeOf(ctx, row)
		if err != nil {
			return nil, err
		}

		routes = append(routes, found)
	}

	return routes, nil
}

// routeOf builds the route of a row with its stops, and asks the domain
// whether it is whole.
//
// A route without its stops visits nobody, and the domain refuses it.
func (r *RouteRepository) routeOf(ctx context.Context, row db.Route) (order.Route, error) {
	stopRows, err := r.pool.queriesFor(ctx).ListRouteStopsByRoute(ctx, row.ID)
	if err != nil {
		return order.Route{}, writeRouteError(err)
	}

	stops := make([]order.Stop, 0, len(stopRows))

	for _, one := range stopRows {
		stops = append(stops, order.Stop{
			OrderID: order.ID(one.OrderID),
			Cost:    ledger.Amount(one.Cost),
		})
	}

	return routeFromRow(row, stops)
}

// routeFromRow maps a row of the routes and its stops onto the model of
// the domain.
//
// The route is built by the domain: the row may come from a table that was
// changed by another writer, and the model does not trust a stranger with a
// route it does not know.
func routeFromRow(row db.Route, stops []order.Stop) (order.Route, error) {
	built, err := order.NewRouteFromFacts(order.RouteFacts{
		ID:         order.RouteID(row.ID),
		BusinessID: business.ID(row.BusinessID),
		Day:        ledger.NewDate(row.Day.Time),
		Stops:      stops,
	})
	if err != nil {
		return order.Route{}, err
	}

	built.CreatedAt = timestamp(row.CreatedAt)
	built.UpdatedAt = timestamp(row.UpdatedAt)

	return built, nil
}

// writeRouteError translates the truth of the database into the errors of
// the port of the routes.
//
// The constraints of the tables hold the rules of the domain as a defense
// in depth: the route of a business and a day is written once, the stop of
// a route is a stop of the business that owns the route, and the stop of an
// order is a stop of an order of that business.
func writeRouteError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return order.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		if pgErr.ConstraintName == routesOnePerBusinessAndDayConstraint {
			return fmt.Errorf("%w: the constraint %q refused the row",
				order.ErrRouteAlreadyWritten, pgErr.ConstraintName)
		}

		return fmt.Errorf("%w: the constraint %q refused the row", order.ErrInvalidValue, pgErr.ConstraintName)
	case foreignKeyViolation:
		switch pgErr.ConstraintName {
		case routeStopsOrderConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				order.ErrStopNotAnOrder, pgErr.ConstraintName)
		case routeStopsRouteConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				order.ErrInvalidValue, pgErr.ConstraintName)
		}

		return fmt.Errorf("%w: the business or the order of the route is not there", order.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", order.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
