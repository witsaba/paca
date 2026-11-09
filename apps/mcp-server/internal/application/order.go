package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/order"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
)

// OrderService is the use case of the orders of the bakery.
//
// The order is what the bakery owes: the party that ordered, the products
// it ordered with the quantities of them, and the day it is due. The round
// is the day and the orders it carries, and it answers what is still owed:
// the orders that were not delivered.
//
// The use case never decides what an order is: the party, the products, the
// quantities, the day and the state of the order are the rules of the
// domain, and the use case only asks the domain.
type OrderService struct {
	orders     order.Repository
	parties    party.Repository
	products   product.Repository
	businesses business.Repository
}

// NewOrderService builds the use case of the orders.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The parties and
// the products are needed too: the order names the client that ordered and
// the products of the price list of the business.
func NewOrderService(
	orders order.Repository,
	parties party.Repository,
	products product.Repository,
	businesses business.Repository,
) *OrderService {
	return &OrderService{
		orders:     orders,
		parties:    parties,
		products:   products,
		businesses: businesses,
	}
}

// Record writes the order of the business and answers it with it.
//
// The rules are the ones of the domain: the use case reads the business
// that owns the order, asks the domain whether a business of that kind holds
// orders, resolves the party that ordered and the products of the lines,
// asks the domain to read the day it is due and asks the domain to build
// the order. It never decides what an order is.
//
// The order is written OPEN: an order that was not delivered stays open,
// and the delivery of an order is written with the route of the round.
func (s *OrderService) Record(ctx context.Context, request ports.RecordOrder) (order.Order, error) {
	if s == nil || s.orders == nil || s.parties == nil || s.products == nil || s.businesses == nil {
		return order.Order{}, ports.ErrNoDatabase
	}

	owner, err := s.businesses.Find(ctx, request.BusinessID)
	if err != nil {
		return order.Order{}, err
	}

	if err := order.OwnedBy(owner.Kind); err != nil {
		return order.Order{}, err
	}

	client, err := s.theClient(ctx, owner.ID, request.PartyID, request.PartyName)
	if err != nil {
		return order.Order{}, err
	}

	dueOn, err := s.theDay(request.DueOn)
	if err != nil {
		return order.Order{}, err
	}

	lines := make([]order.LineDefinition, 0, len(request.Lines))

	for _, one := range request.Lines {
		baked, err := s.theProduct(ctx, owner.ID, one.ProductID, one.ProductName)
		if err != nil {
			return order.Order{}, err
		}

		lines = append(lines, order.LineDefinition{
			Product:  baked,
			Quantity: one.Quantity,
			Unit:     one.Unit,
		})
	}

	built, err := order.New(owner.ID, client, dueOn, lines)
	if err != nil {
		return order.Order{}, err
	}

	return s.orders.Create(ctx, built)
}

// Read answers the round of the day: the orders it carries and the ones
// that are still owed.
//
// The rules are the ones of the domain: the use case reads the business
// whose orders are asked for, asks the domain whether a business of that
// kind holds orders, reads the day of the round and hands the orders of the
// business to the domain, and the domain answers the round and what is
// still owed.
func (s *OrderService) Read(ctx context.Context, request ports.ReadOrders) (order.Round, error) {
	if s == nil || s.orders == nil || s.parties == nil || s.products == nil || s.businesses == nil {
		return order.Round{}, ports.ErrNoDatabase
	}

	owner, err := s.businesses.Find(ctx, request.BusinessID)
	if err != nil {
		return order.Round{}, err
	}

	if err := order.OwnedBy(owner.Kind); err != nil {
		return order.Round{}, err
	}

	day := ledger.Date{}
	if strings.TrimSpace(request.Day) != "" {
		parsed, err := ledger.ParseDate(request.Day)
		if err != nil {
			return order.Round{}, err
		}

		day = parsed
	}

	held, err := s.orders.List(ctx, owner.ID)
	if err != nil {
		return order.Round{}, err
	}

	return order.TheRound(day, held), nil
}

// theClient answers the party that ordered.
//
// The rules are the ones of the domain: the party is a party of the
// business that owns the order, and it holds the role of the client. The
// party of another business is never in the list of the parties of the
// business, and the book never attaches it to the order.
//
// The error is party.ErrNotFound when the business holds no party with that
// identity or with that name, and order.ErrMissingParty when the agent
// names no party at all.
func (s *OrderService) theClient(ctx context.Context, own business.ID, id int64, name string) (party.Party, error) {
	held, err := s.parties.List(ctx, own)
	if err != nil {
		return party.Party{}, err
	}

	if id != 0 {
		for _, one := range held {
			if int64(one.ID) == id {
				return one, nil
			}
		}

		return party.Party{}, fmt.Errorf("%w: the business %d holds no party with the identity %d",
			party.ErrNotFound, own, id)
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return party.Party{}, fmt.Errorf("%w: the order names no party", order.ErrMissingParty)
	}

	for _, one := range held {
		if strings.EqualFold(strings.TrimSpace(one.Name), trimmed) {
			return one, nil
		}
	}

	return party.Party{}, fmt.Errorf("%w: the business %d holds no party named %q",
		party.ErrNotFound, own, trimmed)
}

// theProduct answers the product of a line of the order.
//
// The rules are the ones of the domain: the product is a product of the
// price list of the business that owns the order, and the product of
// another business is never in the list of the products of the business.
//
// The error is product.ErrNotFound when the business holds no product with
// that identity, product.ErrUnknownRecipe when it holds no product with
// that name, and order.ErrMissingProduct when the agent names no product
// at all.
func (s *OrderService) theProduct(ctx context.Context, own business.ID, id int64, name string) (product.Product, error) {
	held, err := s.products.List(ctx, own)
	if err != nil {
		return product.Product{}, err
	}

	if id != 0 {
		for _, one := range held {
			if int64(one.ID) == id {
				return one, nil
			}
		}

		return product.Product{}, fmt.Errorf("%w: the business %d holds no product with the identity %d",
			product.ErrNotFound, own, id)
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return product.Product{}, fmt.Errorf("%w: the line of the order names no product", order.ErrMissingProduct)
	}

	found := product.Named(held, trimmed)
	if len(found) == 0 {
		return product.Product{}, fmt.Errorf("%w: the business %d holds no product named %q",
			product.ErrUnknownRecipe, own, trimmed)
	}

	return found[0], nil
}

// theDay answers the day the order is due on.
//
// The rule of the domain: the order carries the day it is due, and the
// book never invents one.
//
// The error is order.ErrMissingDay when the agent names no day, and
// ledger.ErrInvalidDate when the day is not a day of the book.
func (s *OrderService) theDay(day string) (ledger.Date, error) {
	trimmed := strings.TrimSpace(day)
	if trimmed == "" {
		return ledger.Date{}, fmt.Errorf("%w: the order carries no day", order.ErrMissingDay)
	}

	return ledger.ParseDate(trimmed)
}
