//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/user"
)

// partiesRepository returns the repository of the parties, bound to the
// database of the suite.
func partiesRepository(t *testing.T) *postgres.PartyRepository {
	t.Helper()

	repo, err := postgres.NewPartyRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the parties could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheParties opens the business that owns the parties of the
// test.
func theBakeryOfTheParties(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Parties")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería De Las Partes"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theStoreOfTheTest builds the store the tests of the table write: a store
// with the legal human that represents it.
func theStoreOfTheTest(t *testing.T, bakery business.Business) party.Party {
	t.Helper()

	human, err := party.LegalHumanOf(party.KindStore, "Jaime Ortiz", "S.A.S.")
	if err != nil {
		t.Fatalf("the legal human was refused: %v", err)
	}

	built, err := party.New(
		bakery.ID,
		party.KindStore,
		"La Panadería de Don Jaime",
		[]party.Role{party.RoleClient, party.RoleProvider},
		"573212539984",
		"jaime@panaderia.com",
		"Calle 5 # 12-30",
		human,
	)
	if err != nil {
		t.Fatalf("the store was refused: %v", err)
	}

	return built
}

// thePersonOfTheTest builds the person the tests of the table write: a
// person, with no legal human at all.
func thePersonOfTheTest(t *testing.T, bakery business.Business) party.Party {
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
		t.Fatalf("the person was refused: %v", err)
	}

	return built
}

// TestTheTableOfThePartiesHasTheColumnsOfTheSpecification proves that the
// table of the parties exists with exactly the columns, the types, the
// nullability and the defaults the specification describes.
//
// The columns are read from the catalog of the database, and not from the
// migration file: the file is what the database was asked to be, and the
// catalog is what it became.
func TestTheTableOfThePartiesHasTheColumnsOfTheSpecification(t *testing.T) {
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
		{name: "kind", dataType: "text", nullable: "NO", defaultSQL: ""},
		{name: "name", dataType: "text", nullable: "NO", defaultSQL: ""},
		{name: "roles", dataType: "jsonb", nullable: "NO", defaultSQL: "'[]'::jsonb"},
		{name: "phone", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "email", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "address", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "legal_human_name", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "legal_figure", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "created_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
		{name: "updated_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
		{name: "tax_id", dataType: "text", nullable: "YES", defaultSQL: ""},
	}

	rows, err := theSuite.db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'parties'
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

	// The relation is an ordinary table, and not a view over something else.
	var kind string
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT relkind FROM pg_class WHERE oid = 'public.parties'::regclass`,
	).Scan(&kind); err != nil {
		t.Fatalf("the relation of the parties could not be read: %v", err)
	}
	if kind != "r" {
		t.Errorf("the relation of the parties is a %q, want an ordinary table", kind)
	}
}

// TestThePartiesOfTheBusinessBeginEmpty proves the rule of the domain
// against a real database: the parties of a business begin empty, and they
// hold only the parties of that business.
func TestThePartiesOfTheBusinessBeginEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := partiesRepository(t)
	bakery := theBakeryOfTheParties(t, ctx)

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the parties could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("the business holds %d parties, want none", len(held))
	}
}

// TestThePartyIsKeptAndReadBackWithItsRolesAndItsFacts proves the rules of
// the table against a real database: the party is written with its kind, its
// roles, the way of reaching it, its address and the legal human that
// represents it, and it is read back as it was written.
func TestThePartyIsKeptAndReadBackWithItsRolesAndItsFacts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := partiesRepository(t)
	bakery := theBakeryOfTheParties(t, ctx)

	written, err := repo.Create(ctx, theStoreOfTheTest(t, bakery))
	if err != nil {
		t.Fatalf("the party could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the party has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the party is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.Kind != party.KindStore {
		t.Errorf("the kind of the party is %q, want %q", written.Kind, party.KindStore)
	}
	if written.Name != "La Panadería de Don Jaime" {
		t.Errorf("the name of the party is %q, want %q", written.Name, "La Panadería de Don Jaime")
	}
	if len(written.Roles) != 2 {
		t.Fatalf("the party holds %d roles, want 2", len(written.Roles))
	}
	if !written.Is(party.RoleClient) || !written.Is(party.RoleProvider) {
		t.Errorf("the roles of the party are %v, want the client and the provider", written.Roles)
	}
	if written.Phone != "573212539984" {
		t.Errorf("the phone of the party is %q, want %q", written.Phone, "573212539984")
	}
	if written.Email != "jaime@panaderia.com" {
		t.Errorf("the email of the party is %q, want %q", written.Email, "jaime@panaderia.com")
	}
	if written.Address != "Calle 5 # 12-30" {
		t.Errorf("the address of the party is %q, want %q", written.Address, "Calle 5 # 12-30")
	}
	if written.LegalHuman.Name != "Jaime Ortiz" {
		t.Errorf("the legal human is %q, want %q", written.LegalHuman.Name, "Jaime Ortiz")
	}
	if written.LegalHuman.Figure != "S.A.S." {
		t.Errorf("the legal figure is %q, want %q", written.LegalHuman.Figure, "S.A.S.")
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the parties could not be read: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the business holds %d parties, want 1", len(held))
	}
	if held[0].ID != written.ID {
		t.Errorf("the identity is %d, want %d", held[0].ID, written.ID)
	}
	if held[0].LegalHuman.Figure != "S.A.S." {
		t.Errorf("the legal figure is %q, want %q", held[0].LegalHuman.Figure, "S.A.S.")
	}
	if held[0].Phone != "573212539984" {
		t.Errorf("the phone of the party is %q, want %q", held[0].Phone, "573212539984")
	}
}

// TestThePartyOfTheTableCarriesNoEmptyValue proves the rule of the table:
// a value that is not set is NULL, and the library never writes an empty
// string.
func TestThePartyOfTheTableCarriesNoEmptyValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := partiesRepository(t)
	bakery := theBakeryOfTheParties(t, ctx)

	if _, err := repo.Create(ctx, thePersonOfTheTest(t, bakery)); err != nil {
		t.Fatalf("the party could not be written: %v", err)
	}

	var phone string
	var legalHumanName string
	err := theSuite.db.QueryRowContext(ctx,
		`SELECT COALESCE(phone, ''), COALESCE(legal_human_name, '') FROM parties WHERE business_id = $1`,
		int64(bakery.ID),
	).Scan(&phone, &legalHumanName)
	if err != nil {
		t.Fatalf("the row of the party could not be read: %v", err)
	}

	if phone != "" {
		t.Errorf("the phone of the row is %q, want NULL", phone)
	}
	if legalHumanName != "" {
		t.Errorf("the legal human of the row is %q, want NULL", legalHumanName)
	}
}

// TestTwoPartiesOfTheBusinessMayShareTheName proves the rule of the domain
// against a real database: the name of a party is not unique inside the
// business, and the reading of the parties holds both of them.
func TestTwoPartiesOfTheBusinessMayShareTheName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := partiesRepository(t)
	bakery := theBakeryOfTheParties(t, ctx)

	first := thePersonOfTheTest(t, bakery)
	second := thePersonOfTheTest(t, bakery)

	if _, err := repo.Create(ctx, first); err != nil {
		t.Fatalf("the first party could not be written: %v", err)
	}
	if _, err := repo.Create(ctx, second); err != nil {
		t.Fatalf("the second party could not be written: %v", err)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the parties could not be read: %v", err)
	}
	if len(held) != 2 {
		t.Fatalf("the business holds %d parties, want 2", len(held))
	}
	if held[0].Name != held[1].Name {
		t.Errorf("the names of the parties are %q and %q, want the same one", held[0].Name, held[1].Name)
	}
}

// TestTheTableRefusesAStoreWithoutItsLegalHuman proves the rule of the
// table: a store carries the legal human that represents it, and the table
// refuses the row that carries none.
func TestTheTableRefusesAStoreWithoutItsLegalHuman(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheParties(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO parties (business_id, kind, name, roles) VALUES ($1, 'store', 'La Panadería Sin Representante', '["client"]'::jsonb)`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a store without its legal human")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "parties_legal_human_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "parties_legal_human_check")
	}
}

// TestTheTableRefusesAPersonWithItsLegalHuman proves the rule of the
// table: a person carries no legal human, and the table refuses the row that
// carries one.
func TestTheTableRefusesAPersonWithItsLegalHuman(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheParties(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO parties (business_id, kind, name, roles, legal_human_name) VALUES ($1, 'person', 'Doña Rosalba', '["client"]'::jsonb, 'Rosalba')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a person with a legal human")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "parties_legal_human_check" {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, "parties_legal_human_check")
	}
}

// TestTheTableRefusesAPartyWithoutItsRole proves the rule of the table: the
// party holds at least one of the two roles, and the table refuses the row
// that holds none.
func TestTheTableRefusesAPartyWithoutItsRole(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheParties(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO parties (business_id, kind, name, roles) VALUES ($1, 'person', 'Doña Rosalba', '[]'::jsonb)`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a party without a role")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
}

// TestTheTableRefusesALegalFigureWithoutItsHuman proves the rule of the
// table: a legal figure without the name of the human it belongs to is a
// figure of nobody, and the table refuses the row that carries it.
func TestTheTableRefusesALegalFigureWithoutItsHuman(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheParties(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO parties (business_id, kind, name, roles, legal_figure) VALUES ($1, 'farm', 'Finca Sin Representante', '["provider"]'::jsonb, 'S.A.S.')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a legal figure without the name of its human")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
}

// TestTheTableRefusesARoleThatIsNotOneOfTheTwo proves the rule of the
// table: every role of the party is the client or the provider, and the
// table refuses the row that carries another one.
func TestTheTableRefusesARoleThatIsNotOneOfTheTwo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheParties(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO parties (business_id, kind, name, roles) VALUES ($1, 'person', 'Doña Rosalba', '["client", "owner"]'::jsonb)`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the table accepted a role that is not one of the two")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
}

// TestTheTaxIdentityOfThePartyIsOptional proves the rule of the table: the
// tax identity of a party is an OPTIONAL fact of it, and the column is
// NULL when the party carries none.
func TestTheTaxIdentityOfThePartyIsOptional(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := partiesRepository(t)
	bakery := theBakeryOfTheParties(t, ctx)

	withoutTaxIdentity := thePersonOfTheTest(t, bakery)
	if _, err := repo.Create(ctx, withoutTaxIdentity); err != nil {
		t.Fatalf("the party could not be written: %v", err)
	}

	var taxID sql.NullString
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT tax_id FROM parties WHERE business_id = $1 AND name = $2`,
		int64(bakery.ID), withoutTaxIdentity.Name,
	).Scan(&taxID); err != nil {
		t.Fatalf("the row of the party could not be read: %v", err)
	}
	if taxID.Valid {
		t.Errorf("the tax identity of the row is %q, want NULL", taxID.String)
	}

	withTaxIdentity := thePersonOfTheTest(t, bakery).WithTaxIdentity("900123456-7")
	written, err := repo.Create(ctx, withTaxIdentity)
	if err != nil {
		t.Fatalf("the party could not be written: %v", err)
	}
	if written.TaxID != "900123456-7" {
		t.Errorf("the tax identity of the party is %q, want %q", written.TaxID, "900123456-7")
	}
}

// TestThePartyOfABusinessThatIsNotThereIsRefused proves the rule of the
// table: the party belongs to a business, and the table refuses the party of
// a business that does not exist.
func TestThePartyOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := partiesRepository(t)
	bakery := theBakeryOfTheParties(t, ctx)

	missing := thePersonOfTheTest(t, bakery)
	missing.BusinessID = business.ID(nextIdentifier())

	_, err := repo.Create(ctx, missing)
	if err == nil {
		t.Fatal("the party of a business that is not there was written")
	}
	if !errors.Is(err, party.ErrBusinessNotFound) {
		t.Fatalf("the error is %v, want %v", err, party.ErrBusinessNotFound)
	}
}
