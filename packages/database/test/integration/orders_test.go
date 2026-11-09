//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/order"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/user"
)

// ordersRepository returns the repository of the orders, bound to the
// database of the suite.
func ordersRepository(t *testing.T) *postgres.OrderRepository {
	t.Helper()

	repo, err := postgres.NewOrderRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the orders could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheOrders opens the business that owns the orders of the test.
func theBakeryOfTheOrders(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Orders")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería De Las Pedidos"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theClientOfTheOrders writes the party that orders: a person that holds
// the role of the client of the bakery.
func theClientOfTheOrders(t *testing.T, ctx context.Context, bakery business.Business) party.Party {
	t.Helper()

	built, err := party.New(
		bakery.ID,
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

	written, err := partiesRepository(t).Create(ctx, built)
	if err != nil {
		t.Fatalf("the client could not be written: %v", err)
	}

	return written
}

// theProductOfTheOrders writes the product of the price list that the
// client orders.
func theProductOfTheOrders(t *testing.T, ctx context.Context, bakery business.Business) product.Product {
	t.Helper()

	baked := theRecipeOfThePrices(t, ctx, bakery)

	built, err := product.New(bakery.ID, baked, "Pan de molde", "")
	if err != nil {
		t.Fatalf("the product was refused: %v", err)
	}

	written, err := productsRepository(t).Create(ctx, built)
	if err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}

	return written
}

// theDayOfTheOrders is the day the order of the test is due on.
func theDayOfTheOrders(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day of the order could not be read: %v", err)
	}

	return parsed
}

// theOrderOfTheTest builds the order the tests of the table write.
func theOrderOfTheTest(
	t *testing.T,
	bakery business.Business,
	client party.Party,
	baked product.Product,
	dueOn ledger.Date,
) order.Order {
	t.Helper()

	built, err := order.New(bakery.ID, client, dueOn, []order.LineDefinition{
		{Product: baked, Quantity: 12, Unit: "unidad"},
	})
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	return built
}

// TestTheTableOfTheOrdersHasTheColumnsOfTheSpecification proves that the
// tables of the orders exist with exactly the columns, the types, the
// nullability and the defaults the specification describes.
//
// The columns are read from the catalog of the database, and not from the
// migration file: the file is what the database was asked to be, and the
// catalog is what it became.
func TestTheTableOfTheOrdersHasTheColumnsOfTheSpecification(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	type column struct {
		name       string
		dataType   string
		nullable   string
		defaultSQL string
	}

	want := []column{
		{name: "id", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "business_id", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "party_id", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "party_name", dataType: "text", nullable: "NO", defaultSQL: ""},
		{name: "due_on", dataType: "date", nullable: "NO", defaultSQL: ""},
		{name: "state", dataType: "text", nullable: "NO", defaultSQL: "'open'::text"},
		{name: "created_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
		{name: "updated_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
	}

	rows, err := theSuite.db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'orders'
		ORDER BY ordinal_position`)
	if err != nil {
		t.Fatalf("the columns of the table could not be read: %v", err)
	}
	defer rows.Close()

	got := make([]column, 0, len(want))
	for rows.Next() {
		var (
			name         string
			dataType     string
			nullable     string
			defaultValue sql.NullString
		)
		if err := rows.Scan(&name, &dataType, &nullable, &defaultValue); err != nil {
			t.Fatalf("a column of the table could not be read: %v", err)
		}
		got = append(got, column{
			name:       name,
			dataType:   dataType,
			nullable:   nullable,
			defaultSQL: defaultValue.String,
		})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("the columns of the table could not be read: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("the table has %d columns, want %d: %+v", len(got), len(want), got)
	}
	for i, expected := range want {
		if got[i] != expected {
			t.Errorf("the column %d is %+v, want %+v", i, got[i], expected)
		}
	}

	var kind string
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT relkind FROM pg_class WHERE oid = 'public.orders'::regclass`,
	).Scan(&kind); err != nil {
		t.Fatalf("the relation of the orders could not be read: %v", err)
	}
	if kind != "r" {
		t.Errorf("the relation of the orders is a %q, want an ordinary table", kind)
	}
}

// TestTheOrdersOfTheBusinessBeginEmpty proves the rule of the domain
// against a real database: the orders of a business begin empty, and they
// hold only the orders of that business.
func TestTheOrdersOfTheBusinessBeginEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the orders could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("the business holds %d orders, want none", len(held))
	}
}

// TestTheOrderIsKeptAndReadBackWithItsLines proves the rules of the table
// against a real database: the order is written with the party that
// ordered, the day it is due and its lines, and it is read back as it was
// written.
func TestTheOrderIsKeptAndReadBackWithItsLines(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	dueOn := theDayOfTheOrders(t, "2026-03-14")

	written, err := repo.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, dueOn))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the order has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the order is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.PartyID != client.ID {
		t.Errorf("the party of the order is %d, want %d", written.PartyID, client.ID)
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
	if written.Lines[0].ProductID != baked.ID {
		t.Errorf("the product of the line is %d, want %d", written.Lines[0].ProductID, baked.ID)
	}
	if written.Lines[0].Quantity != 12 {
		t.Errorf("the quantity of the line is %v, want 12", written.Lines[0].Quantity)
	}
	if written.Lines[0].Unit != purchase.PricePerUnit {
		t.Errorf("the unit of the line is %q, want %q", written.Lines[0].Unit, purchase.PricePerUnit)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the orders could not be read: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the business holds %d orders, want 1", len(held))
	}
	if held[0].ID != written.ID {
		t.Errorf("the identity is %d, want %d", held[0].ID, written.ID)
	}
	if len(held[0].Lines) != 1 {
		t.Fatalf("the order read back carries %d lines, want 1", len(held[0].Lines))
	}
	if held[0].Lines[0].ProductName != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", held[0].Lines[0].ProductName, "Pan de molde")
	}
	if held[0].Lines[0].Quantity != 12 {
		t.Errorf("the quantity of the line is %v, want 12", held[0].Lines[0].Quantity)
	}
}

// TestTheOrderOfAnotherBusinessIsRefused proves the rule of the table: the
// party of an order is a party of the business that owns it, and the table
// refuses the order of a foreign party.
func TestTheOrderOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	other := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, other)
	dueOn := theDayOfTheOrders(t, "2026-03-14")

	written := theOrderOfTheTest(t, bakery, theClientOfTheOrders(t, ctx, bakery), theProductOfTheOrders(t, ctx, bakery), dueOn)

	// The domain refuses the party of another business before the
	// storage is asked: the row below is written the way another writer
	// would write it, and the table refuses it.
	written.PartyID = client.ID

	_, err := repo.Create(ctx, written)
	if err == nil {
		t.Fatal("the order of a party of another business was written")
	}
	if !errors.Is(err, order.ErrPartyOfAnotherBusiness) {
		t.Fatalf("the error is %v, want %v", err, order.ErrPartyOfAnotherBusiness)
	}
}

// TestTheLineOfAProductOfAnotherBusinessIsRefused proves the rule of the
// table: the products of the lines of an order are products of the price
// list of the business that owns it, and the table refuses the line of a
// foreign product.
func TestTheLineOfAProductOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	other := theBakeryOfTheOrders(t, ctx)
	baked := theProductOfTheOrders(t, ctx, other)
	dueOn := theDayOfTheOrders(t, "2026-03-14")

	written := theOrderOfTheTest(t, bakery, theClientOfTheOrders(t, ctx, bakery), theProductOfTheOrders(t, ctx, bakery), dueOn)

	// The domain refuses the product of another business before the
	// storage is asked: the row below is written the way another writer
	// would write it, and the table refuses it.
	written.Lines[0].ProductID = baked.ID

	_, err := repo.Create(ctx, written)
	if err == nil {
		t.Fatal("the order of a product of another business was written")
	}
	if !errors.Is(err, order.ErrProductOfAnotherBusiness) {
		t.Fatalf("the error is %v, want %v", err, order.ErrProductOfAnotherBusiness)
	}
}

// TestTheOrderOfABusinessThatIsNotThereIsRefused proves the rule of the
// table: the order belongs to a business, and the table refuses the order
// of a business that does not exist.
func TestTheOrderOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	dueOn := theDayOfTheOrders(t, "2026-03-14")

	written := theOrderOfTheTest(t, bakery, client, baked, dueOn)
	written.BusinessID = business.ID(nextIdentifier())

	_, err := repo.Create(ctx, written)
	if err == nil {
		t.Fatal("the order of a business that is not there was written")
	}
	if !errors.Is(err, order.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, order.ErrNotFound)
	}
}

// TestTheTableRefusesAQuantityThatIsNotPositive proves the rule of the
// table: the quantity of a line is positive, and the table refuses the row
// that carries another one.
func TestTheTableRefusesAQuantityThatIsNotPositive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO orders (business_id, party_id, party_name, due_on) VALUES ($1, $2, 'Doña Rosalba', '2026-03-14')`,
		int64(bakery.ID), int64(client.ID),
	)
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	var orderID int64
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT id FROM orders WHERE business_id = $1`, int64(bakery.ID),
	).Scan(&orderID); err != nil {
		t.Fatalf("the order could not be read: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO order_lines (order_id, business_id, product_id, position, product_name, quantity, unit)
		 VALUES ($1, $2, $3, 1, 'Pan de molde', 0, 'unit')`,
		orderID, int64(bakery.ID), int64(baked.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a line with no units")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "order_lines_quantity_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "order_lines_quantity_check")
	}
}

// TestTheTableRefusesAStateThatIsNotOneOfTheTwo proves the rule of the
// table: the state of an order is the open one or the delivered one, and
// the table refuses the row that carries another one.
func TestTheTableRefusesAStateThatIsNotOneOfTheTwo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO orders (business_id, party_id, party_name, due_on, state) VALUES ($1, $2, 'Doña Rosalba', '2026-03-14', 'cancelled')`,
		int64(bakery.ID), int64(client.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a state that is not one of the two")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "orders_state_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "orders_state_check")
	}
}

// TestTheTableRefusesAUnitThatIsNotOneOfTheBook proves the rule of the
// table: the unit of a line is the way the owner charges — the piece, the
// gram or the kilogram — and the table refuses the row that carries
// another one.
func TestTheTableRefusesAUnitThatIsNotOneOfTheBook(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)

	orderID := theOrderRowOfTheTest(t, ctx, bakery, client)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO order_lines (order_id, business_id, product_id, position, product_name, quantity, unit)
		 VALUES ($1, $2, $3, 1, 'Pan de molde', 12, 'pan')`,
		orderID, int64(bakery.ID), int64(baked.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a unit that is not one of the units of the book")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "order_lines_unit_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "order_lines_unit_check")
	}
}

// TestTheTableRefusesAPieceThatIsSplit proves the rule of the table: the
// book never splits a piece, and the table refuses the row that carries
// half of one when the owner charges by the piece.
func TestTheTableRefusesAPieceThatIsSplit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)

	orderID := theOrderRowOfTheTest(t, ctx, bakery, client)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO order_lines (order_id, business_id, product_id, position, product_name, quantity, unit)
		 VALUES ($1, $2, $3, 1, 'Pan de molde', 2.5, 'unit')`,
		orderID, int64(bakery.ID), int64(baked.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a piece that is split")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "order_lines_piece_quantity_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "order_lines_piece_quantity_check")
	}
}

// TestTheTableKeepsAWeightThatIsNotAWholeNumberOfPieces proves the other
// side of the rule of the table: the book never converts a weight into a
// number of pieces, and the table keeps a line by the weight whose
// quantity is not a whole number.
func TestTheTableKeepsAWeightThatIsNotAWholeNumberOfPieces(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)

	built, err := order.New(bakery.ID, client, theDayOfTheOrders(t, "2026-03-14"), []order.LineDefinition{
		{Product: baked, Quantity: 500.5, Unit: "gramo"},
	})
	if err != nil {
		t.Fatalf("the order of five hundred grams and a half was refused: %v", err)
	}

	written, err := repo.Create(ctx, built)
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}
	if written.Lines[0].Quantity != 500.5 {
		t.Errorf("the quantity of the line is %v, want 500.5", written.Lines[0].Quantity)
	}
	if written.Lines[0].Unit != purchase.PricePerGram {
		t.Errorf("the unit of the line is %q, want %q", written.Lines[0].Unit, purchase.PricePerGram)
	}
}

// TestTheRoundOfADayAnswersTheOrdersThatAreStillOwed proves the rule of
// the domain against a real database: the order that was not delivered
// stays open, and the round of the day answers what is still owed.
//
// The owed is every open order of the business: the one that is due on the
// day and the one that fell due on an earlier day.
func TestTheRoundOfADayAnswersTheOrdersThatAreStillOwed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ordersRepository(t)
	bakery := theBakeryOfTheOrders(t, ctx)
	client := theClientOfTheOrders(t, ctx, bakery)
	baked := theProductOfTheOrders(t, ctx, bakery)
	dueOn := theDayOfTheOrders(t, "2026-03-14")

	open, err := repo.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, dueOn))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	overdue, err := repo.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, theDayOfTheOrders(t, "2026-03-13")))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	delivered, err := repo.Create(ctx, theOrderOfTheTest(t, bakery, client, baked, dueOn))
	if err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	// The delivery of an order is written with the route of the round, and
	// the route is written by the increment that comes after this one: the
	// row of a delivered order is written here, the way the route writes
	// it.
	if _, err := theSuite.db.ExecContext(ctx,
		`UPDATE orders SET state = 'delivered' WHERE id = $1`, int64(delivered.ID),
	); err != nil {
		t.Fatalf("the state of the order could not be written: %v", err)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the orders could not be read: %v", err)
	}
	if len(held) != 3 {
		t.Fatalf("the business holds %d orders, want 3", len(held))
	}

	round := order.TheRound(dueOn, held)

	if len(round.Orders) != 2 {
		t.Fatalf("the round carries %d orders, want 2: the order that fell due is not due on that day", len(round.Orders))
	}

	owed := round.Owed
	if len(owed) != 2 {
		t.Fatalf("the round owes %d orders, want 2: every open order of the business is owed", len(owed))
	}
	if owed[0].ID != open.ID {
		t.Errorf("the order that is owed is %d, want %d", owed[0].ID, open.ID)
	}
	if owed[1].ID != overdue.ID {
		t.Errorf("the order that is owed is %d, want %d: the order that fell due is still owed", owed[1].ID, overdue.ID)
	}
}

// theOrderRowOfTheTest writes the row of an order and answers its
// identity.
//
// The row is written the way the table holds it: the tests of the table
// write their lines by themselves, the way another writer would write them.
func theOrderRowOfTheTest(t *testing.T, ctx context.Context, bakery business.Business, client party.Party) int64 {
	t.Helper()

	if _, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO orders (business_id, party_id, party_name, due_on) VALUES ($1, $2, 'Doña Rosalba', '2026-03-14')`,
		int64(bakery.ID), int64(client.ID),
	); err != nil {
		t.Fatalf("the order could not be written: %v", err)
	}

	var orderID int64
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT id FROM orders WHERE business_id = $1`, int64(bakery.ID),
	).Scan(&orderID); err != nil {
		t.Fatalf("the order could not be read: %v", err)
	}

	return orderID
}
