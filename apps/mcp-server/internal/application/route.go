package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/order"
	"github.com/witsaba/paca/packages/domain/store"
)

// RouteService is the use case of the route of the round of a day.
//
// The route is what the agent computes when it comes back from the round:
// the order it will visit the clients in, and what each visit costs. The
// agent is the only component of PACA that has access to the outside, and
// the route is the one fact of the bakery that it computes and writes back
// into the round.
//
// The route the agent writes back is the round it made: writing the route
// DELIVERS the orders of its stops. The domain owns that transition, and
// the use case only asks the domain.
//
// The use case never decides what a route is: the day of the round, the
// orders of its stops and the money of every stop are the rules of the
// domain, and the use case only asks the domain.
type RouteService struct {
	routes     order.RouteRepository
	orders     order.Repository
	businesses business.Repository
	work       store.UnitOfWork
}

// NewRouteService builds the use case of the route.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The orders are
// needed too: the route visits the orders of the round, and writing it
// delivers them.
//
// The work is the port of the transaction: the route and the delivery of
// the orders of its stops are written whole or not at all.
func NewRouteService(
	routes order.RouteRepository,
	orders order.Repository,
	businesses business.Repository,
	work store.UnitOfWork,
) *RouteService {
	return &RouteService{
		routes:     routes,
		orders:     orders,
		businesses: businesses,
		work:       work,
	}
}

// Record writes the route of the round of the day and answers it with it.
//
// The rules are the ones of the domain: the use case reads the business
// that owns the route, asks the domain whether a business of that kind holds
// orders, reads the day of the round, reads the orders the business holds
// and the routes it wrote already, and asks the domain to build the route.
//
// The route and the delivery of the orders of its stops are written in ONE
// unit of work: the route is the round the agent made, and the orders it
// visits are delivered with it. A route that is written and an order that
// is not delivered is a round that was made and never happened.
//
// The error is the error of the domain: the domain refuses the business
// that does not hold orders, the route without a day, the route without a
// stop, the stop that is not an order of the round, the cost that is not a
// number of pesos and the second route of a day that was already written.
func (s *RouteService) Record(ctx context.Context, request ports.RecordRoute) (order.Route, error) {
	if s == nil || s.routes == nil || s.orders == nil || s.businesses == nil || s.work == nil {
		return order.Route{}, ports.ErrNoDatabase
	}

	owner, err := s.businesses.Find(ctx, request.BusinessID)
	if err != nil {
		return order.Route{}, err
	}

	if err := order.OwnedBy(owner.Kind); err != nil {
		return order.Route{}, err
	}

	day, err := s.theDay(request.Day)
	if err != nil {
		return order.Route{}, err
	}

	held, err := s.orders.List(ctx, owner.ID)
	if err != nil {
		return order.Route{}, err
	}

	written, err := s.routes.List(ctx, owner.ID)
	if err != nil {
		return order.Route{}, err
	}

	built, err := order.TheRoute(
		owner.ID,
		held,
		written,
		order.RouteDefinition{
			Day:   day,
			Stops: stopsOf(request.Stops),
		},
	)
	if err != nil {
		return order.Route{}, err
	}

	delivered, err := built.TheDelivery(held)
	if err != nil {
		return order.Route{}, err
	}

	var recorded order.Route

	err = s.work.Do(ctx, func(ctx context.Context) error {
		stored, err := s.routes.Create(ctx, built)
		if err != nil {
			return err
		}

		if err := s.orders.Deliver(ctx, delivered); err != nil {
			return err
		}

		recorded = stored

		return nil
	})
	if err != nil {
		return order.Route{}, err
	}

	return recorded, nil
}

// theDay answers the day of the round the route is of.
//
// The rule of the domain: the route carries the day of the round, and the
// book never invents one.
//
// The error is ledger.ErrInvalidDate when the day is not a day of the book,
// and order.ErrMissingRouteDay when the agent names no day.
func (s *RouteService) theDay(day string) (ledger.Date, error) {
	trimmed := strings.TrimSpace(day)
	if trimmed == "" {
		return ledger.Date{}, fmt.Errorf("%w: the route carries no day", order.ErrMissingRouteDay)
	}

	return ledger.ParseDate(trimmed)
}

// stopsOf answers the stops the agent read, in the words of the domain.
//
// The port speaks in the words of the agent and in numbers: the use case
// turns them into the model of the domain, and the domain decides what may
// enter the book.
func stopsOf(stops []ports.RouteStop) []order.StopDefinition {
	read := make([]order.StopDefinition, 0, len(stops))

	for _, one := range stops {
		read = append(read, order.StopDefinition{
			OrderID: order.ID(one.OrderID),
			Cost:    ledger.Amount(one.Cost),
		})
	}

	return read
}
