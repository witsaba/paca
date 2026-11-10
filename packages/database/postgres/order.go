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
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The names the database gives to the constraints of the tables of the
// orders.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	// ordersPartyConstraint is the name of the foreign key that refuses a
	// party of another business.
	ordersPartyConstraint = "orders_party_of_its_business_fkey"

	// orderLinesOrderConstraint is the name of the foreign key that
	// refuses a line of an order of another business.
	orderLinesOrderConstraint = "order_lines_order_of_its_business_fkey"

	// orderLinesProductConstraint is the name of the foreign key that
	// refuses a line of a product of another business.
	orderLinesProductConstraint = "order_lines_product_of_its_business_fkey"
)

// OrderRepository is the adapter of the storage of the orders.
//
// The lines of an order are rows of their own table, and the adapter writes
// them with it: an order without its lines orders nothing.
//
// The row never decides what an order is: the model is built by the domain,
// through its own constructors.
type OrderRepository struct {
	pool *Pool
}

// NewOrderRepository returns the repository of the orders, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewOrderRepository(pool *Pool) (*OrderRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &OrderRepository{pool: pool}, nil
}

// Create writes an order with its lines, and answers it with the identity
// that the database generated.
//
// The order and its lines are written in one transaction: the set joins the
// transaction of the caller when the caller opened one, and opens its own
// when it did not. An order whose line is refused leaves nothing behind.
//
// The error is order.ErrNotFound when the business, the party or the
// product of the order is not there, order.ErrPartyOfAnotherBusiness and
// order.ErrProductOfAnotherBusiness when the party or the product is of
// another business, order.ErrInvalidValue when the row breaks a rule of
// the table, and order.ErrNoStorage when the storage is not open.
func (r *OrderRepository) Create(ctx context.Context, written order.Order) (order.Order, error) {
	if r == nil || r.pool == nil {
		return order.Order{}, order.ErrNoStorage
	}

	if _, ok := transactionIn(ctx); ok {
		return r.write(ctx, r.pool.queriesFor(ctx), written)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return order.Order{}, fmt.Errorf("the transaction of the orders could not be opened: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	recorded, err := r.write(ctx, r.pool.queriesFor(ctx).WithTx(tx), written)
	if err != nil {
		return order.Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return order.Order{}, fmt.Errorf("the transaction of the orders could not be committed: %w", err)
	}

	return recorded, nil
}

// write writes the order and its lines, and answers the order that was
// written.
func (r *OrderRepository) write(ctx context.Context, queries *db.Queries, written order.Order) (order.Order, error) {
	row, err := queries.CreateOrder(ctx, db.CreateOrderParams{
		BusinessID: int64(written.BusinessID),
		PartyID:    int64(written.PartyID),
		PartyName:  written.PartyName,
		DueOn:      dateOf(written.DueOn),
		State:      written.State.String(),
	})
	if err != nil {
		return order.Order{}, writeOrderError(err)
	}

	lines := make([]order.Line, 0, len(written.Lines))

	for position, one := range written.Lines {
		line, err := queries.CreateOrderLine(ctx, db.CreateOrderLineParams{
			OrderID:     row.ID,
			BusinessID:  int64(written.BusinessID),
			ProductID:   int64(one.ProductID),
			Position:    int32(position + 1),
			ProductName: one.ProductName,
			Quantity:    one.Quantity,
			Unit:        one.Unit.String(),
		})
		if err != nil {
			return order.Order{}, writeOrderError(err)
		}

		lines = append(lines, order.Line{
			ProductID:   product.ID(line.ProductID),
			ProductName: line.ProductName,
			Quantity:    line.Quantity,
			Unit:        purchase.PriceBasis(line.Unit),
		})
	}

	// The order is built by the domain: the row never decides what an
	// order is, and a row that another writer broke is refused.
	recorded, err := order.NewFromFacts(order.Facts{
		ID:         order.ID(row.ID),
		BusinessID: business.ID(row.BusinessID),
		PartyID:    party.ID(row.PartyID),
		PartyName:  row.PartyName,
		DueOn:      ledger.NewDate(row.DueOn.Time),
		Lines:      lines,
		State:      order.State(row.State),
	})
	if err != nil {
		return order.Order{}, err
	}

	recorded.CreatedAt = timestamp(row.CreatedAt)
	recorded.UpdatedAt = timestamp(row.UpdatedAt)

	return recorded, nil
}

// List returns the orders of the business, in the order of their identity,
// with their lines.
//
// The orders are read in the order of their identity, so the answer is the
// same every time it is read.
//
// The error is order.ErrNoStorage when the storage is not open.
func (r *OrderRepository) List(ctx context.Context, own business.ID) ([]order.Order, error) {
	if r == nil || r.pool == nil {
		return nil, order.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListOrdersByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeOrderError(err)
	}

	orders := make([]order.Order, 0, len(rows))

	for _, row := range rows {
		built, err := r.orderOf(ctx, row)
		if err != nil {
			return nil, err
		}

		orders = append(orders, built)
	}

	return orders, nil
}

// Deliver writes the delivery of the orders: the state of every order of
// the list moves to the delivered one.
//
// The delivery of an order is written with the ROUTE of the round, and the
// state that is written is the one the domain decided: the adapter never
// decides the state of an order by itself.
//
// The delivery moves an order that is OPEN, and only one. An order that is
// not open is never reopened: the delivery of an order is written once, and
// a caller that bypasses the domain cannot move a delivered order back to
// the open one through this port.
//
// The deliveries are written in one transaction: the set joins the
// transaction of the caller when the caller opened one, and opens its own
// when it did not. An order that was delivered and an order that was not is
// a round that was made and never happened.
//
// The error is order.ErrAlreadyDelivered when the order is not open any
// more, order.ErrNotFound when the order is not there, order.ErrInvalidValue
// when the row breaks a rule of the table, and order.ErrNoStorage when the
// storage is not open.
func (r *OrderRepository) Deliver(ctx context.Context, delivered []order.Order) error {
	if r == nil || r.pool == nil {
		return order.ErrNoStorage
	}

	if _, ok := transactionIn(ctx); ok {
		return r.deliver(ctx, r.pool.queriesFor(ctx), delivered)
	}

	tx, err := r.pool.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("the transaction of the deliveries could not be opened: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := r.deliver(ctx, r.pool.queriesFor(ctx).WithTx(tx), delivered); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("the transaction of the deliveries could not be committed: %w", err)
	}

	return nil
}

// deliver writes the delivery of every order of the list.
//
// The delivery moves an order that is open: the state the row carries before
// the write is the one the domain owes, and the table is asked for it.
func (r *OrderRepository) deliver(ctx context.Context, queries *db.Queries, delivered []order.Order) error {
	for _, one := range delivered {
		row, err := queries.DeliverOrder(ctx, db.DeliverOrderParams{
			State:       one.State.String(),
			ID:          int64(one.ID),
			BusinessID:  int64(one.BusinessID),
			StateBefore: order.StateOpen.String(),
		})
		if err != nil {
			return theDeliveryError(ctx, queries, one, err)
		}

		if row.State != one.State.String() {
			return fmt.Errorf("%w: the state of the order %d is %q", order.ErrInvalidValue, one.ID, row.State)
		}
	}

	return nil
}

// theDeliveryError translates the refusal of the table into the errors of the
// port of the orders.
//
// The delivery of an order is written once: the order that is not open any
// more is refused with the error of the domain, and the adapter never reopens
// it. The row is read back only when the delivery found no order to deliver,
// so the adapter can tell the order that is not there from the order that is
// not open.
func theDeliveryError(ctx context.Context, queries *db.Queries, one order.Order, err error) error {
	if !errors.Is(err, pgx.ErrNoRows) {
		return writeOrderError(err)
	}

	state, err := queries.TheStateOfTheOrder(ctx, db.TheStateOfTheOrderParams{
		ID:         int64(one.ID),
		BusinessID: int64(one.BusinessID),
	})
	if err != nil {
		return writeOrderError(err)
	}

	if order.State(state) != order.StateOpen {
		return fmt.Errorf("%w: the order %d is %q", order.ErrAlreadyDelivered, one.ID, state)
	}

	return fmt.Errorf("%w: the order %d was not delivered by this call", order.ErrInvalidValue, one.ID)
}

// orderOf builds the order of a row with its lines, and asks the domain
// whether it is whole.
//
// An order without its lines is not an order, and the domain refuses it.
func (r *OrderRepository) orderOf(ctx context.Context, row db.Order) (order.Order, error) {
	lineRows, err := r.pool.queriesFor(ctx).ListOrderLinesByOrder(ctx, row.ID)
	if err != nil {
		return order.Order{}, writeOrderError(err)
	}

	lines := make([]order.Line, 0, len(lineRows))

	for _, one := range lineRows {
		lines = append(lines, order.Line{
			ProductID:   product.ID(one.ProductID),
			ProductName: one.ProductName,
			Quantity:    one.Quantity,
			Unit:        purchase.PriceBasis(one.Unit),
		})
	}

	built, err := order.NewFromFacts(order.Facts{
		ID:         order.ID(row.ID),
		BusinessID: business.ID(row.BusinessID),
		PartyID:    party.ID(row.PartyID),
		PartyName:  row.PartyName,
		DueOn:      ledger.NewDate(row.DueOn.Time),
		Lines:      lines,
		State:      order.State(row.State),
	})
	if err != nil {
		return order.Order{}, err
	}

	built.CreatedAt = timestamp(row.CreatedAt)
	built.UpdatedAt = timestamp(row.UpdatedAt)

	return built, nil
}

// writeOrderError translates the truth of the database into the errors of
// the port of the orders.
//
// The constraints of the tables hold the rules of the domain as a defense
// in depth: the party of the order and the products of its lines are of
// the business that owns the order, the unit of a line is one of the ways
// of charging of the book, the quantity of a line by the piece is whole
// and the state of the order is one of the two of the book.
func writeOrderError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return order.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case foreignKeyViolation:
		switch pgErr.ConstraintName {
		case ordersPartyConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				order.ErrPartyOfAnotherBusiness, pgErr.ConstraintName)
		case orderLinesProductConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				order.ErrProductOfAnotherBusiness, pgErr.ConstraintName)
		case orderLinesOrderConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				order.ErrInvalidValue, pgErr.ConstraintName)
		}

		return fmt.Errorf("%w: the business, the party or the product of the order is not there", order.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", order.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
