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
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/user"
)

// invoicesRepository returns the repository of the invoices, bound to the
// database of the suite.
func invoicesRepository(t *testing.T) *postgres.InvoiceRepository {
	t.Helper()

	repo, err := postgres.NewInvoiceRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the invoices could not be created: %v", err)
	}

	return repo
}

// purchasesOfTheInvoices returns the recorder of the purchases, bound to
// the database of the suite.
func purchasesOfTheInvoices(t *testing.T) *postgres.PurchaseRecorder {
	t.Helper()

	recorder, err := postgres.NewPurchaseRecorder(theSuite.pool)
	if err != nil {
		t.Fatalf("the recorder of the purchases could not be created: %v", err)
	}

	return recorder
}

// theBakeryOfTheInvoices opens the business that owns the invoices of the
// test.
func theBakeryOfTheInvoices(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Invoices")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería De Las Facturas"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theProviderOfTheInvoices writes the provider that issues the invoices of
// the test: a party that holds the role of the provider.
func theProviderOfTheInvoices(t *testing.T, ctx context.Context, bakery business.Business) party.Party {
	t.Helper()

	repo, err := postgres.NewPartyRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the parties could not be created: %v", err)
	}

	built, err := party.New(
		bakery.ID,
		party.KindFarm,
		"Finca La Esperanza",
		[]party.Role{party.RoleProvider},
		"573212539984",
		"finca@example.com",
		"Vereda La Esperanza",
		party.LegalHuman{},
	)
	if err != nil {
		t.Fatalf("the provider was refused: %v", err)
	}

	written, err := repo.Create(ctx, built)
	if err != nil {
		t.Fatalf("the provider could not be written: %v", err)
	}

	return written
}

// thePurchaseOfTheInvoices writes the purchase the invoices of the test
// come from, and answers it.
func thePurchaseOfTheInvoices(t *testing.T, ctx context.Context, bakery business.Business) purchase.Purchase {
	t.Helper()

	var draftID int64
	if err := theSuite.db.QueryRowContext(ctx,
		`INSERT INTO purchase_drafts (business_id, utterance, proposed_on, expires_on)
		 VALUES ($1, 'Compré 25 kilos de harina', '2026-03-01', '2026-03-08') RETURNING id`,
		int64(bakery.ID),
	).Scan(&draftID); err != nil {
		t.Fatalf("the draft of the purchase could not be written: %v", err)
	}

	var purchaseID int64
	if err := theSuite.db.QueryRowContext(ctx,
		`INSERT INTO purchases (business_id, draft_id, day, utterance, confirmed_with, total)
		 VALUES ($1, $2, '2026-03-01', 'Compré 25 kilos de harina', 'sí', 10000) RETURNING id`,
		int64(bakery.ID), draftID,
	).Scan(&purchaseID); err != nil {
		t.Fatalf("the purchase of the invoices could not be written: %v", err)
	}

	bought, err := purchasesOfTheInvoices(t).Find(ctx, purchase.ID(purchaseID))
	if err != nil {
		t.Fatalf("the purchase could not be read: %v", err)
	}

	return bought
}

// theInvoiceOfTheTest builds the invoice the tests of the table write.
func theInvoiceOfTheTest(
	t *testing.T,
	bakery business.Business,
	provider party.Party,
	bought purchase.Purchase,
) purchase.Invoice {
	t.Helper()

	rate, err := purchase.NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	day, err := ledger.ParseDate("2026-03-01")
	if err != nil {
		t.Fatalf("the day was refused: %v", err)
	}

	built, err := purchase.NewInvoice(bakery.ID, provider, purchase.InvoiceDefinition{
		PurchaseID: bought.ID,
		Number:     "FV-001",
		Day:        day,
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Total:      ledger.Amount(11900),
		Reference:  "REF-1234",
	})
	if err != nil {
		t.Fatalf("the invoice was refused: %v", err)
	}

	return built
}

// TestTheTableOfTheInvoicesHasTheColumnsOfTheSpecification proves that the
// table of the invoices exists with exactly the columns, the types, the
// nullability and the defaults the specification describes.
//
// The columns are read from the catalog of the database, and not from the
// migration file: the file is what the database was asked to be, and the
// catalog is what it became.
func TestTheTableOfTheInvoicesHasTheColumnsOfTheSpecification(t *testing.T) {
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
		{name: "provider_id", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "purchase_id", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "number", dataType: "text", nullable: "NO", defaultSQL: ""},
		{name: "day", dataType: "date", nullable: "NO", defaultSQL: ""},
		{name: "base", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "rate", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "tax", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "total", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "reference", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "created_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
	}

	rows, err := theSuite.db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'invoices'
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

// TestThePurchaseCarriesTheInvoiceThatCameWithIt proves the rule of the
// table: the purchase points at the invoice it came from, and the column
// is empty when the invoice has not arrived yet.
func TestThePurchaseCarriesTheInvoiceThatCameWithIt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheInvoices(t, ctx)
	provider := theProviderOfTheInvoices(t, ctx, bakery)
	bought := thePurchaseOfTheInvoices(t, ctx, bakery)

	var before sql.NullInt64
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT invoice_id FROM purchases WHERE id = $1`, int64(bought.ID),
	).Scan(&before); err != nil {
		t.Fatalf("the purchase could not be read: %v", err)
	}
	if before.Valid {
		t.Errorf("the purchase already points at the invoice %d, want none", before.Int64)
	}

	repo := invoicesRepository(t)
	written, err := repo.Record(ctx, theInvoiceOfTheTest(t, bakery, provider, bought))
	if err != nil {
		t.Fatalf("the invoice could not be written: %v", err)
	}

	if err := repo.Attach(ctx, written, bought); err != nil {
		t.Fatalf("the purchase could not point at the invoice: %v", err)
	}

	var after sql.NullInt64
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT invoice_id FROM purchases WHERE id = $1`, int64(bought.ID),
	).Scan(&after); err != nil {
		t.Fatalf("the purchase could not be read: %v", err)
	}
	if !after.Valid {
		t.Fatal("the purchase does not point at the invoice")
	}
	if after.Int64 != int64(written.ID) {
		t.Errorf("the purchase points at the invoice %d, want %d", after.Int64, written.ID)
	}
}

// TestTheInvoiceIsKeptAndReadBackWithItsTax proves the rules of the table
// against a real database: the invoice is written with its number, its day,
// its base, its rate, its tax and its total, and it is read back as it was
// written.
func TestTheInvoiceIsKeptAndReadBackWithItsTax(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := invoicesRepository(t)
	bakery := theBakeryOfTheInvoices(t, ctx)
	provider := theProviderOfTheInvoices(t, ctx, bakery)
	bought := thePurchaseOfTheInvoices(t, ctx, bakery)

	written, err := repo.Record(ctx, theInvoiceOfTheTest(t, bakery, provider, bought))
	if err != nil {
		t.Fatalf("the invoice could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the invoice has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the invoice is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.ProviderID != provider.ID {
		t.Errorf("the provider of the invoice is %d, want %d", written.ProviderID, provider.ID)
	}
	if written.PurchaseID != bought.ID {
		t.Errorf("the purchase of the invoice is %d, want %d", written.PurchaseID, bought.ID)
	}
	if written.Number != "FV-001" {
		t.Errorf("the number of the invoice is %q, want %q", written.Number, "FV-001")
	}
	if written.Base != ledger.Amount(10000) {
		t.Errorf("the base of the invoice is %d, want 10000", written.Base)
	}
	if written.Rate.Hundredths() != 1900 {
		t.Errorf("the rate of the invoice is %d hundredths, want 1900", written.Rate.Hundredths())
	}
	if written.Tax != ledger.Amount(1900) {
		t.Errorf("the tax of the invoice is %d, want 1900", written.Tax)
	}
	if written.Total != ledger.Amount(11900) {
		t.Errorf("the total of the invoice is %d, want 11900", written.Total)
	}
	if written.Reference != "REF-1234" {
		t.Errorf("the reference of the invoice is %q, want %q", written.Reference, "REF-1234")
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the invoices could not be read: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the business holds %d invoices, want 1", len(held))
	}
	if held[0].Number != "FV-001" {
		t.Errorf("the number of the invoice is %q, want %q", held[0].Number, "FV-001")
	}
	if held[0].Tax != ledger.Amount(1900) {
		t.Errorf("the tax of the invoice is %d, want 1900", held[0].Tax)
	}
}

// TestTheInvoicesOfTheBusinessBeginEmpty proves the rule of the domain
// against a real database: the invoices of a business begin empty, and they
// hold only the invoices of that business.
func TestTheInvoicesOfTheBusinessBeginEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := invoicesRepository(t)
	bakery := theBakeryOfTheInvoices(t, ctx)

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the invoices could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("the business holds %d invoices, want none", len(held))
	}
}

// TestTheInvoiceOfAnotherBusinessIsRefused proves the rule of the domain
// against a real database: an invoice of one business is never attached to
// the purchase of another one, and the purchase keeps pointing at nothing.
func TestTheInvoiceOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := invoicesRepository(t)
	bakery := theBakeryOfTheInvoices(t, ctx)
	provider := theProviderOfTheInvoices(t, ctx, bakery)
	bought := thePurchaseOfTheInvoices(t, ctx, bakery)

	written, err := repo.Record(ctx, theInvoiceOfTheTest(t, bakery, provider, bought))
	if err != nil {
		t.Fatalf("the invoice could not be written: %v", err)
	}

	otherBakery := theBakeryOfTheInvoices(t, ctx)
	other := thePurchaseOfTheInvoices(t, ctx, otherBakery)

	if err := repo.Attach(ctx, written, other); !errors.Is(err, purchase.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNotFound)
	}

	var invoiceID sql.NullInt64
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT invoice_id FROM purchases WHERE id = $1`, int64(other.ID),
	).Scan(&invoiceID); err != nil {
		t.Fatalf("the purchase could not be read: %v", err)
	}
	if invoiceID.Valid {
		t.Errorf("the purchase points at the invoice %d, want none", invoiceID.Int64)
	}
}

// TestTheNumberOfTheInvoiceIsUniqueInsideTheBusiness proves the rule of the
// table: the number of an invoice is what identifies it, and it is unique
// inside the business that holds it.
func TestTheNumberOfTheInvoiceIsUniqueInsideTheBusiness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := invoicesRepository(t)
	bakery := theBakeryOfTheInvoices(t, ctx)
	provider := theProviderOfTheInvoices(t, ctx, bakery)
	bought := thePurchaseOfTheInvoices(t, ctx, bakery)

	if _, err := repo.Record(ctx, theInvoiceOfTheTest(t, bakery, provider, bought)); err != nil {
		t.Fatalf("the invoice could not be written: %v", err)
	}

	_, err := repo.Record(ctx, theInvoiceOfTheTest(t, bakery, provider, bought))
	if !errors.Is(err, purchase.ErrDuplicateInvoiceNumber) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrDuplicateInvoiceNumber)
	}
}

// TestTheTableRefusesTheInvoiceThatDoesNotAddUp proves the rule of the
// table: the total of the invoice is the base plus the tax, and the table
// refuses the row that does not add up.
func TestTheTableRefusesTheInvoiceThatDoesNotAddUp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheInvoices(t, ctx)
	provider := theProviderOfTheInvoices(t, ctx, bakery)
	bought := thePurchaseOfTheInvoices(t, ctx, bakery)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO invoices (business_id, provider_id, purchase_id, number, day, base, rate, tax, total)
		 VALUES ($1, $2, $3, 'FV-002', '2026-03-01', 10000, 1900, 1900, 12000)`,
		int64(bakery.ID), int64(provider.ID), int64(bought.ID),
	)
	if err == nil {
		t.Fatal("the database stored an invoice that does not add up, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "invoices_total_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "invoices_total_check")
	}
}

// TestTheTableRefusesTheInvoiceOfABusinessThatIsNotThere proves the rule
// of the table: the invoice belongs to a business, and the table refuses
// the invoice of a business that does not exist.
func TestTheTableRefusesTheInvoiceOfABusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := invoicesRepository(t)
	bakery := theBakeryOfTheInvoices(t, ctx)
	provider := theProviderOfTheInvoices(t, ctx, bakery)
	bought := thePurchaseOfTheInvoices(t, ctx, bakery)

	missing := theInvoiceOfTheTest(t, bakery, provider, bought)
	missing.BusinessID = business.ID(nextIdentifier())

	_, err := repo.Record(ctx, missing)
	if !errors.Is(err, purchase.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNotFound)
	}
}

// TestTheTableRefusesTheSecondInvoiceOfThePurchase proves the rule of the
// table: a purchase carries one invoice, and the table refuses the second
// one.
func TestTheTableRefusesTheSecondInvoiceOfThePurchase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := invoicesRepository(t)
	bakery := theBakeryOfTheInvoices(t, ctx)
	provider := theProviderOfTheInvoices(t, ctx, bakery)
	bought := thePurchaseOfTheInvoices(t, ctx, bakery)

	if _, err := repo.Record(ctx, theInvoiceOfTheTest(t, bakery, provider, bought)); err != nil {
		t.Fatalf("the invoice could not be written: %v", err)
	}

	second := theInvoiceOfTheTest(t, bakery, provider, bought)
	second.Number = "FV-002"

	_, err := repo.Record(ctx, second)
	if !errors.Is(err, purchase.ErrPurchaseAlreadyHasInvoice) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrPurchaseAlreadyHasInvoice)
	}
}

// TestTheInvoiceIsNotAttachedToAnotherPurchase proves the rule of the
// table: the purchase points at the invoice it came from, and the invoice
// is never attached to another purchase of the same business.
func TestTheInvoiceIsNotAttachedToAnotherPurchase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := invoicesRepository(t)
	bakery := theBakeryOfTheInvoices(t, ctx)
	provider := theProviderOfTheInvoices(t, ctx, bakery)
	bought := thePurchaseOfTheInvoices(t, ctx, bakery)
	other := thePurchaseOfTheInvoices(t, ctx, bakery)

	written, err := repo.Record(ctx, theInvoiceOfTheTest(t, bakery, provider, bought))
	if err != nil {
		t.Fatalf("the invoice could not be written: %v", err)
	}

	if err := repo.Attach(ctx, written, other); !errors.Is(err, purchase.ErrInvoiceOfAnotherPurchase) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrInvoiceOfAnotherPurchase)
	}

	var invoiceID sql.NullInt64
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT invoice_id FROM purchases WHERE id = $1`, int64(other.ID),
	).Scan(&invoiceID); err != nil {
		t.Fatalf("the purchase could not be read: %v", err)
	}
	if invoiceID.Valid {
		t.Errorf("the purchase points at the invoice %d, want none", invoiceID.Int64)
	}
}

// TestTheInvoiceMovesNoMoney proves that the invoice moves no money: the
// total of the purchase is the one it was, and the ledger of the business
// holds no entry for the invoice.
//
// The purchase moved the money of the payment when the owner bought, and
// the invoice is a fact that arrives later.
func TestTheInvoiceMovesNoMoney(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := invoicesRepository(t)
	bakery := theBakeryOfTheInvoices(t, ctx)
	provider := theProviderOfTheInvoices(t, ctx, bakery)
	bought := thePurchaseOfTheInvoices(t, ctx, bakery)

	totalBefore, entriesBefore := theMoneyOfThePurchase(t, ctx, bakery, bought)

	written, err := repo.Record(ctx, theInvoiceOfTheTest(t, bakery, provider, bought))
	if err != nil {
		t.Fatalf("the invoice could not be written: %v", err)
	}

	if err := repo.Attach(ctx, written, bought); err != nil {
		t.Fatalf("the purchase could not point at the invoice: %v", err)
	}

	totalAfter, entriesAfter := theMoneyOfThePurchase(t, ctx, bakery, bought)

	if totalAfter != totalBefore {
		t.Errorf("the total of the purchase is %d, want %d: the invoice moves no money", totalAfter, totalBefore)
	}
	if entriesAfter != entriesBefore {
		t.Errorf("the book of the business holds %d movements, want %d: the invoice moves no money",
			entriesAfter, entriesBefore)
	}
}

// theMoneyOfThePurchase answers the total of the purchase and the number of
// movements the book of the business holds.
func theMoneyOfThePurchase(
	t *testing.T,
	ctx context.Context,
	bakery business.Business,
	bought purchase.Purchase,
) (int64, int64) {
	t.Helper()

	var total int64
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT total FROM purchases WHERE id = $1`, int64(bought.ID),
	).Scan(&total); err != nil {
		t.Fatalf("the total of the purchase could not be read: %v", err)
	}

	var entries int64
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM ledger_entries WHERE business_id = $1`, int64(bakery.ID),
	).Scan(&entries); err != nil {
		t.Fatalf("the movements of the book could not be read: %v", err)
	}

	return total, entries
}
