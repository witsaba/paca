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
	"github.com/witsaba/paca/packages/domain/product"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/user"
)

// productsRepository returns the repository of the products, bound to the
// database of the suite.
func productsRepository(t *testing.T) *postgres.ProductRepository {
	t.Helper()

	repo, err := postgres.NewProductRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the products could not be created: %v", err)
	}

	return repo
}

// pricesRepository returns the repository of the price list, bound to the
// database of the suite.
func pricesRepository(t *testing.T) *postgres.PriceRepository {
	t.Helper()

	repo, err := postgres.NewPriceRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the price list could not be created: %v", err)
	}

	return repo
}

// theBakeryOfThePrices opens the business that owns the price list of the
// test.
func theBakeryOfThePrices(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Prices")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería De Los Precios"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theRecipeOfThePrices writes the recipe the product of the test comes
// from: the bakery sells what it bakes.
func theRecipeOfThePrices(t *testing.T, ctx context.Context, bakery business.Business) recipe.Recipe {
	t.Helper()

	flour := theFlourOfTheRecipes(t, ctx, bakery)

	written, err := recipesRepository(t).Record(ctx, theRecipeOfTheTest(bakery, flour))
	if err != nil {
		t.Fatalf("the recipe could not be written: %v", err)
	}

	return written
}

// theProductOfThePrices builds the product the tests of the table write.
func theProductOfThePrices(bakery business.Business, baked recipe.Recipe) product.Product {
	built, err := product.New(bakery.ID, baked.ID, "Pan de molde", "")
	if err != nil {
		panic("the product of the test could not be built: " + err.Error())
	}

	return built
}

// thePriceOfThePrices builds the price the tests of the table write.
func thePriceOfThePrices(
	bakery business.Business,
	of product.Product,
	channel product.Channel,
	amount ledger.Amount,
	day string,
) product.Price {
	parsed, err := ledger.ParseDate(day)
	if err != nil {
		panic("the day of the price of the test could not be read: " + err.Error())
	}

	built, err := product.NewPrice(bakery.ID, of.ID, channel, amount, parsed)
	if err != nil {
		panic("the price of the test could not be built: " + err.Error())
	}

	return built
}

// TestTheTableOfTheProductsHasTheColumnsOfTheSpecification proves that the
// table of the products exists with exactly the columns, the types, the
// nullability and the defaults the specification describes.
//
// The columns are read from the catalog of the database, and not from the
// migration file: the file is what the database was asked to be, and the
// catalog is what it became.
func TestTheTableOfTheProductsHasTheColumnsOfTheSpecification(t *testing.T) {
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
		{name: "recipe_id", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "name", dataType: "text", nullable: "NO", defaultSQL: ""},
		{name: "tax_treatment", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "created_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
		{name: "updated_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
	}

	rows, err := theSuite.db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'products'
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
}

// TestTheTableOfThePricesHasTheColumnsOfTheSpecification proves that the
// table of the prices exists with exactly the columns, the types, the
// nullability and the defaults the specification describes.
func TestTheTableOfThePricesHasTheColumnsOfTheSpecification(t *testing.T) {
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
		{name: "product_id", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "channel", dataType: "text", nullable: "NO", defaultSQL: ""},
		{name: "amount", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "day", dataType: "date", nullable: "NO", defaultSQL: ""},
		{name: "created_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
	}

	rows, err := theSuite.db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'prices'
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
}

// TestThePriceIsNeverAColumnOfTheInventory proves the rule of the domain:
// the price of a product is a fact of the business, and it never becomes a
// column of the inventory, of the recipe or of the stock.
func TestThePriceIsNeverAColumnOfTheInventory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	rows, err := theSuite.db.QueryContext(ctx, `
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name IN ('stock_layers', 'ingredients', 'recipes', 'productions',
		                      'production_layers', 'production_units', 'purchase_lines')
		  AND column_name = 'price'`)
	if err != nil {
		t.Fatalf("the columns of the inventory could not be read: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			t.Fatalf("a column of the inventory could not be read: %v", err)
		}
		t.Errorf("the table %q holds the column %q of the price of a product", table, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("the columns of the inventory could not be read: %v", err)
	}
}

// TestTheProductsOfTheBusinessBeginEmpty proves the rule of the domain
// against a real database: the products of a business begin empty, and they
// hold only the products of that business.
func TestTheProductsOfTheBusinessBeginEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := productsRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the products could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("the business holds %d products, want none", len(held))
	}
}

// TestTheProductIsKeptWithItsRecipeAndItsName proves the rules of the
// table against a real database: the product is written with the recipe it
// comes from and the name the owner reads, and it is read back as it was
// written.
func TestTheProductIsKeptWithItsRecipeAndItsName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := productsRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)
	baked := theRecipeOfThePrices(t, ctx, bakery)

	written, err := repo.Create(ctx, theProductOfThePrices(bakery, baked))
	if err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the product has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the product is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.RecipeID != baked.ID {
		t.Errorf("the recipe of the product is %d, want %d", written.RecipeID, baked.ID)
	}
	if written.Name != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", written.Name, "Pan de molde")
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the products could not be read: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the business holds %d products, want 1", len(held))
	}
	if held[0].ID != written.ID {
		t.Errorf("the identity is %d, want %d", held[0].ID, written.ID)
	}
	if held[0].RecipeID != baked.ID {
		t.Errorf("the recipe of the product is %d, want %d", held[0].RecipeID, baked.ID)
	}
}

// TestTheTaxTreatmentOfTheProductIsOptional proves the rule of the table:
// the tax treatment of a product is an OPTIONAL fact of it, and the column
// is NULL when the owner has not decided it.
func TestTheTaxTreatmentOfTheProductIsOptional(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := productsRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)
	baked := theRecipeOfThePrices(t, ctx, bakery)

	without, err := repo.Create(ctx, theProductOfThePrices(bakery, baked))
	if err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}

	var treatment sql.NullString
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT tax_treatment FROM products WHERE id = $1`, int64(without.ID),
	).Scan(&treatment); err != nil {
		t.Fatalf("the row of the product could not be read: %v", err)
	}
	if treatment.Valid {
		t.Errorf("the tax treatment of the row is %q, want NULL", treatment.String)
	}
}

// TestTheTableRefusesTwoProductsWithTheSameName proves the rule of the
// table: the name of a product is taken inside the business.
func TestTheTableRefusesTwoProductsWithTheSameName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := productsRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)
	baked := theRecipeOfThePrices(t, ctx, bakery)

	if _, err := repo.Create(ctx, theProductOfThePrices(bakery, baked)); err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}

	_, err := repo.Create(ctx, theProductOfThePrices(bakery, baked))
	if !errors.Is(err, product.ErrDuplicateName) {
		t.Fatalf("the error is %v, want %v", err, product.ErrDuplicateName)
	}
}

// TestTheTableRefusesTwoProductsOfTheSameRecipe proves the rule of the
// table: one recipe backs one product inside the business.
func TestTheTableRefusesTwoProductsOfTheSameRecipe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := productsRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)
	baked := theRecipeOfThePrices(t, ctx, bakery)

	if _, err := repo.Create(ctx, theProductOfThePrices(bakery, baked)); err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}

	other, err := product.New(bakery.ID, baked.ID, "Pan de molde grande", "")
	if err != nil {
		t.Fatalf("the product was refused: %v", err)
	}

	_, err = repo.Create(ctx, other)
	if !errors.Is(err, product.ErrDuplicateRecipe) {
		t.Fatalf("the error is %v, want %v", err, product.ErrDuplicateRecipe)
	}
}

// TestThePriceIsWrittenWithItsChannel proves the rule of the domain against
// a real database: the price carries the channel it was set for from the
// first row.
func TestThePriceIsWrittenWithItsChannel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	products := productsRepository(t)
	prices := pricesRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)
	baked := theRecipeOfThePrices(t, ctx, bakery)

	opened, err := products.Create(ctx, theProductOfThePrices(bakery, baked))
	if err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}

	written, err := prices.Set(ctx,
		thePriceOfThePrices(bakery, opened, product.ChannelWholesale, ledger.Amount(2500), "2026-03-01"))
	if err != nil {
		t.Fatalf("the price could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the price has no identity, want the one the database generated")
	}
	if written.Channel != product.ChannelWholesale {
		t.Errorf("the channel of the price is %q, want %q", written.Channel, product.ChannelWholesale)
	}
	if written.Amount != ledger.Amount(2500) {
		t.Errorf("the money of the price is %d, want 2500", written.Amount)
	}
	if written.ProductID != opened.ID {
		t.Errorf("the product of the price is %d, want %d", written.ProductID, opened.ID)
	}
	if written.Day.String() != "2026-03-01" {
		t.Errorf("the day of the price is %q, want %q", written.Day.String(), "2026-03-01")
	}

	var channel string
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT channel FROM prices WHERE id = $1`, int64(written.ID),
	).Scan(&channel); err != nil {
		t.Fatalf("the row of the price could not be read: %v", err)
	}
	if channel != "wholesale" {
		t.Errorf("the channel of the row is %q, want %q", channel, "wholesale")
	}
}

// TestTheHistoryOfThePricesOfAChannelIsKept proves the rule of the domain
// against a real database: the price list is a history, the rows are kept
// and nothing is rewritten when a price changes.
func TestTheHistoryOfThePricesOfAChannelIsKept(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	products := productsRepository(t)
	prices := pricesRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)
	baked := theRecipeOfThePrices(t, ctx, bakery)

	opened, err := products.Create(ctx, theProductOfThePrices(bakery, baked))
	if err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}

	first, err := prices.Set(ctx,
		thePriceOfThePrices(bakery, opened, product.ChannelRetail, ledger.Amount(3000), "2026-03-01"))
	if err != nil {
		t.Fatalf("the first price could not be written: %v", err)
	}

	second, err := prices.Set(ctx,
		thePriceOfThePrices(bakery, opened, product.ChannelRetail, ledger.Amount(3500), "2026-03-05"))
	if err != nil {
		t.Fatalf("the second price could not be written: %v", err)
	}

	held, err := prices.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the prices could not be read: %v", err)
	}
	if len(held) != 2 {
		t.Fatalf("the business holds %d prices, want 2", len(held))
	}
	if held[0].ID != first.ID || held[1].ID != second.ID {
		t.Errorf("the prices of the answer are %d and %d, want %d and %d",
			held[0].ID, held[1].ID, first.ID, second.ID)
	}
	if held[0].Amount != ledger.Amount(3000) {
		t.Errorf("the money of the first price is %d, want 3000", held[0].Amount)
	}

	holds, ok := product.ThePriceOn(held, opened.ID, product.ChannelRetail, mustDay(t, "2026-03-02"))
	if !ok {
		t.Fatal("the price list holds no price on the second day")
	}
	if holds.Amount != ledger.Amount(3000) {
		t.Errorf("the price of the second day is %d, want 3000", holds.Amount)
	}
}

// TestThePriceOfAnotherBusinessIsNotRead proves the rule of the domain
// against a real database: the price list of a business holds the prices of
// that business and of no other one.
func TestThePriceOfAnotherBusinessIsNotRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	products := productsRepository(t)
	prices := pricesRepository(t)

	bakery := theBakeryOfThePrices(t, ctx)
	baked := theRecipeOfThePrices(t, ctx, bakery)
	opened, err := products.Create(ctx, theProductOfThePrices(bakery, baked))
	if err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}
	if _, err := prices.Set(ctx,
		thePriceOfThePrices(bakery, opened, product.ChannelRetail, ledger.Amount(3000), "2026-03-01")); err != nil {
		t.Fatalf("the price could not be written: %v", err)
	}

	other := theBakeryOfThePrices(t, ctx)

	held, err := prices.List(ctx, other.ID)
	if err != nil {
		t.Fatalf("the prices could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("the business holds %d prices, want none", len(held))
	}
}

// TestTheTableRefusesAPriceWithAChannelTheBookDoesNotKnow proves the rule
// of the table: the channel of a price is one of the two of the book.
func TestTheTableRefusesAPriceWithAChannelTheBookDoesNotKnow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	products := productsRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)
	baked := theRecipeOfThePrices(t, ctx, bakery)

	opened, err := products.Create(ctx, theProductOfThePrices(bakery, baked))
	if err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO prices (business_id, product_id, channel, amount, day) VALUES ($1, $2, 'store', 3000, '2026-03-01')`,
		int64(bakery.ID), int64(opened.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a price of a channel the book does not know")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "prices_channel_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "prices_channel_check")
	}
}

// TestTheTableRefusesAPriceThatIsNotAPositiveNumberOfPesos proves the rule
// of the table: the money of a price is a positive number of whole pesos.
func TestTheTableRefusesAPriceThatIsNotAPositiveNumberOfPesos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	products := productsRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)
	baked := theRecipeOfThePrices(t, ctx, bakery)

	opened, err := products.Create(ctx, theProductOfThePrices(bakery, baked))
	if err != nil {
		t.Fatalf("the product could not be written: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO prices (business_id, product_id, channel, amount, day) VALUES ($1, $2, 'retail', 0, '2026-03-01')`,
		int64(bakery.ID), int64(opened.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a price of zero pesos")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "prices_amount_positive_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "prices_amount_positive_check")
	}
}

// TestThePriceOfAProductThatIsNotThereIsRefused proves the rule of the
// table: the price belongs to a product, and the table refuses the price of
// a product that does not exist.
func TestThePriceOfAProductThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prices := pricesRepository(t)
	bakery := theBakeryOfThePrices(t, ctx)

	missing := thePriceOfThePrices(bakery, product.Product{ID: product.ID(nextIdentifier())},
		product.ChannelRetail, ledger.Amount(3000), "2026-03-01")

	_, err := prices.Set(ctx, missing)
	if err == nil {
		t.Fatal("the price of a product that is not there was written")
	}
	if !errors.Is(err, product.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, product.ErrNotFound)
	}
}

// mustDay reads a day of the book of the bakery.
func mustDay(t *testing.T, day string) (parsed ledger.Date) {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day %q could not be read: %v", day, err)
	}

	return parsed
}
