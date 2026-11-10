//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/user"
)

// The names the database gives to the guards of the tables of the book.
const (
	// accountsDebtConstraint is the constraint that binds the counterparty
	// and the due date to the accounts that can owe and be owed.
	accountsDebtConstraint = "accounts_debt_check"

	// accountsKindConstraint is the constraint that checks the kind of an
	// account.
	accountsKindConstraint = "accounts_kind_check"
)

// TestCreateBusinessWritesTheBusinessOfTheOwner proves that the database
// generates the identity of the business, and that the owner is the person
// and never its key.
func TestCreateBusinessWritesTheBusinessOfTheOwner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := businessesRepository(t)
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Business")

	created, err := repo.Create(ctx, user.ID(owner), businessOf(business.KindPersonal, "Negocio Personal"))
	if err != nil {
		t.Fatalf("the business could not be created: %v", err)
	}

	if created.ID == 0 {
		t.Error("the identity of the business is 0, want the one the database generated")
	}
	if created.OwnerID != user.ID(owner) {
		t.Errorf("the owner is %d, want %d", created.OwnerID, owner)
	}
	if created.Kind != business.KindPersonal {
		t.Errorf("the kind is %q, want %q", created.Kind, business.KindPersonal)
	}
	if created.Name != "Negocio Personal" {
		t.Errorf("the name is %q, want %q", created.Name, "Negocio Personal")
	}
	if created.Currency != business.DefaultCurrency {
		t.Errorf("the currency is %q, want %q", created.Currency, business.DefaultCurrency)
	}
}

// TestThePersonalBusinessOfAPersonIsFoundByItsOwner proves that the personal
// business of a person is read through the person that owns it.
func TestThePersonalBusinessOfAPersonIsFoundByItsOwner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := businessesRepository(t)
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Business")

	created, err := repo.Create(ctx, user.ID(owner), businessOf(business.KindPersonal, "Negocio Personal"))
	if err != nil {
		t.Fatalf("the business could not be created: %v", err)
	}

	found, err := repo.FindPersonalByOwner(ctx, user.ID(owner))
	if err != nil {
		t.Fatalf("the personal business could not be read: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("the business is %d, want %d", found.ID, created.ID)
	}

	if _, err := repo.Find(ctx, created.ID); err != nil {
		t.Fatalf("the business could not be read by its identity: %v", err)
	}
}

// TestAPersonHasOnePersonalBusinessOnly proves the rule of the domain, and
// the rule of the table that guards it: the second personal business of the
// same owner is refused, and nothing is stored.
func TestAPersonHasOnePersonalBusinessOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := businessesRepository(t)
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Business")

	if _, err := repo.Create(ctx, user.ID(owner), businessOf(business.KindPersonal, "Negocio Personal")); err != nil {
		t.Fatalf("the first business could not be created: %v", err)
	}

	_, err := repo.Create(ctx, user.ID(owner), businessOf(business.KindPersonal, "Negocio Personal"))
	if !errors.Is(err, business.ErrAlreadyExists) {
		t.Fatalf("the error is %v, want %v", err, business.ErrAlreadyExists)
	}

	if count := countBusinesses(ctx, t, owner); count != 1 {
		t.Errorf("the database stored %d businesses of the owner %d, want 1", count, owner)
	}
}

// TestTheOwnerMayHoldABusinessOfAnotherKind proves the collision that is
// coming never arrives: the bakery of the future is a second row of the
// same table, with an identity of its own.
func TestTheOwnerMayHoldABusinessOfAnotherKind(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := businessesRepository(t)
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Businesses")

	personal, err := repo.Create(ctx, user.ID(owner), businessOf(business.KindPersonal, "Negocio Personal"))
	if err != nil {
		t.Fatalf("the personal business could not be created: %v", err)
	}

	bakery, err := repo.Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería"))
	if err != nil {
		t.Fatalf("the second business could not be created: %v", err)
	}

	if bakery.ID == personal.ID {
		t.Errorf("the two businesses share the identity %d, want one of its own", bakery.ID)
	}
	if count := countBusinesses(ctx, t, owner); count != 2 {
		t.Errorf("the database stored %d businesses of the owner %d, want 2", count, owner)
	}
}

// TestThePersonalBusinessOfAPersonWithoutOneIsNotThere proves that the
// business of a person that has none is not invented.
func TestThePersonalBusinessOfAPersonWithoutOneIsNotThere(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := businessesRepository(t)
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of Nothing")

	_, err := repo.FindPersonalByOwner(ctx, user.ID(owner))
	if !errors.Is(err, business.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, business.ErrNotFound)
	}
}

// TestCreateAccountWritesTheAccountOfTheBusiness proves that the account
// belongs to the business, and that the name of the person is stored as it is
// read.
func TestCreateAccountWritesTheAccountOfTheBusiness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Accounts")
	created := createBusiness(t, owner)

	repo := accountsRepository(t)
	written, err := repo.Create(ctx, accountOf(created.ID, account.KindAsset, "Efectivo"))
	if err != nil {
		t.Fatalf("the account could not be created: %v", err)
	}

	if written.ID == 0 {
		t.Error("the identity of the account is 0, want the one the database generated")
	}
	if written.BusinessID != created.ID {
		t.Errorf("the business is %d, want %d", written.BusinessID, created.ID)
	}
	if written.Kind != account.KindAsset {
		t.Errorf("the kind is %q, want %q", written.Kind, account.KindAsset)
	}
	if written.Name != "Efectivo" {
		t.Errorf("the name is %q, want %q", written.Name, "Efectivo")
	}
	if written.Counterparty != "" {
		t.Errorf("the counterparty is %q, want it absent", written.Counterparty)
	}
	if written.DueDate != nil {
		t.Errorf("the due date is %v, want it absent", written.DueDate)
	}
}

// TestTheNameOfAnAccountIsUniqueInsideItsBusiness proves the rule of the
// domain, and the rule of the table that guards it: the same name twice in
// one business is refused, and the same name in another business is fine.
func TestTheNameOfAnAccountIsUniqueInsideItsBusiness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Accounts")
	first := createBusiness(t, owner)
	second := createBusinessOfKind(t, owner, business.KindBusiness, "La Panadería")

	repo := accountsRepository(t)
	if _, err := repo.Create(ctx, accountOf(first.ID, account.KindAsset, "Efectivo")); err != nil {
		t.Fatalf("the first account could not be created: %v", err)
	}

	_, err := repo.Create(ctx, accountOf(first.ID, account.KindAsset, "Efectivo"))
	if !errors.Is(err, account.ErrAlreadyExists) {
		t.Fatalf("the error is %v, want %v", err, account.ErrAlreadyExists)
	}

	if count := countAccounts(ctx, t, first.ID); count != 1 {
		t.Errorf("the database stored %d accounts of the business, want 1", count)
	}

	if _, err := repo.Create(ctx, accountOf(second.ID, account.KindAsset, "Efectivo")); err != nil {
		t.Fatalf("the account of the other business could not be created: %v", err)
	}
}

// TestTheDebtCarriesItsCounterpartyAndItsDueDate proves that the two values
// of a debt travel from the domain to the row and back.
func TestTheDebtCarriesItsCounterpartyAndItsDueDate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Debt")
	created := createBusiness(t, owner)

	due := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	repo := accountsRepository(t)
	debt, err := repo.Create(ctx, accountOfDebt(created.ID, "Préstamo: Juan", "Juan", &due))
	if err != nil {
		t.Fatalf("the debt could not be created: %v", err)
	}

	if debt.Counterparty != "Juan" {
		t.Errorf("the counterparty is %q, want %q", debt.Counterparty, "Juan")
	}
	if debt.DueDate == nil || !debt.DueDate.Equal(due) {
		t.Errorf("the due date is %v, want %v", debt.DueDate, due)
	}
}

// TestTheSetOfAccountsIsWrittenWholeOrNotAtAll proves the rule of the
// domain: a book that starts with half of its accounts is a book that lies
// about what the person has.
func TestTheSetOfAccountsIsWrittenWholeOrNotAtAll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Set")
	created := createBusiness(t, owner)

	repo := accountsRepository(t)
	accounts := []account.Account{
		accountOf(created.ID, account.KindAsset, "Efectivo"),
		accountOf(created.ID, account.KindAsset, "Efectivo"),
	}

	_, err := repo.CreateMany(ctx, accounts)
	if !errors.Is(err, account.ErrAlreadyExists) {
		t.Fatalf("the error is %v, want %v", err, account.ErrAlreadyExists)
	}

	if count := countAccounts(ctx, t, created.ID); count != 0 {
		t.Errorf("the database stored %d accounts of the business, want none", count)
	}
}

// TestTheSetOfAccountsIsWritten proves that the whole set is written, and
// that the answers carry the identities the database generated.
func TestTheSetOfAccountsIsWritten(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Set")
	created := createBusiness(t, owner)

	repo := accountsRepository(t)
	seed := account.DefaultPersonalAccounts()
	accounts := make([]account.Account, 0, len(seed))
	for _, one := range seed {
		accounts = append(accounts, accountOf(created.ID, one.Kind, one.Name))
	}

	written, err := repo.CreateMany(ctx, accounts)
	if err != nil {
		t.Fatalf("the accounts could not be created: %v", err)
	}

	if len(written) != len(seed) {
		t.Fatalf("the storage answered %d accounts, want %d", len(written), len(seed))
	}
	for _, one := range written {
		if one.ID == 0 {
			t.Errorf("the account %q has no identity", one.Name)
		}
	}

	listed, err := repo.List(ctx, created.ID)
	if err != nil {
		t.Fatalf("the accounts could not be read: %v", err)
	}
	if len(listed) != len(seed) {
		t.Errorf("the storage holds %d accounts, want %d", len(listed), len(seed))
	}
	for index := 1; index < len(listed); index++ {
		if listed[index-1].ID >= listed[index].ID {
			t.Errorf("the accounts are not in the order of their identity: %d then %d",
				listed[index-1].ID, listed[index].ID)
		}
	}
}

// TestTheDatabaseRefusesACounterpartyInAnAccountThatCannotOwe proves that
// the table holds the rule of the domain as a defense in depth.
func TestTheDatabaseRefusesACounterpartyInAnAccountThatCannotOwe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Accounts")
	created := createBusiness(t, owner)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO accounts (business_id, kind, name, counterparty) VALUES ($1, $2, $3, $4)`,
		int64(created.ID), string(account.KindExpense), "Alimentación", "Juan",
	)
	if err == nil {
		t.Fatal("the database stored an expense with a counterparty, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != accountsDebtConstraint {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, accountsDebtConstraint)
	}
}

// TestTheDatabaseRefusesAnUnknownKind proves that the table holds the kinds
// of the domain, and that it refuses a kind the domain does not know.
func TestTheDatabaseRefusesAnUnknownKind(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Accounts")
	created := createBusiness(t, owner)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO accounts (business_id, kind, name) VALUES ($1, $2, $3)`,
		int64(created.ID), "savings", "Ahorro",
	)
	if err == nil {
		t.Fatal("the database stored an account of an unknown kind, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != accountsKindConstraint {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, accountsKindConstraint)
	}
}

// businessesRepository returns the repository of the businesses, bound to the
// database of the suite.
func businessesRepository(t *testing.T) *postgres.BusinessRepository {
	t.Helper()

	repo, err := postgres.NewBusinessRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the businesses could not be created: %v", err)
	}

	return repo
}

// accountsRepository returns the repository of the accounts, bound to the
// database of the suite.
func accountsRepository(t *testing.T) *postgres.AccountRepository {
	t.Helper()

	repo, err := postgres.NewAccountRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the accounts could not be created: %v", err)
	}

	return repo
}

// businessOf builds the business of a kind and a name.
func businessOf(kind business.Kind, name string) business.Business {
	built, err := business.New(user.ID(1), kind, name, business.DefaultCurrency)
	if err != nil {
		panic("the business of the test could not be built: " + err.Error())
	}

	return built
}

// accountOf builds the account of a business, a kind and a name.
func accountOf(businessID business.ID, kind account.Kind, name string) account.Account {
	built, err := account.New(businessID, kind, name, "", nil)
	if err != nil {
		panic("the account of the test could not be built: " + err.Error())
	}

	return built
}

// accountOfDebt builds the account of a debt: a liability with a
// counterparty and a due date.
func accountOfDebt(businessID business.ID, name, counterparty string, dueDate *time.Time) account.Account {
	built, err := account.New(businessID, account.KindLiability, name, counterparty, dueDate)
	if err != nil {
		panic("the debt of the test could not be built: " + err.Error())
	}

	return built
}

// theAccountsOfTheBook writes the accounts of the book of a business that
// the movements of its money need, and answers them by their role.
//
// The names of the accounts are the ones the domain declares for the book
// of a business, and the role is what says which account is which: the
// money of a movement moves between the accounts of the roles, never
// between the names the owner reads.
func theAccountsOfTheBook(
	t *testing.T,
	ctx context.Context,
	bakery business.Business,
	roles ...account.Role,
) map[account.Role]account.Account {
	t.Helper()

	wanted := make(map[account.Role]bool, len(roles))

	for _, one := range roles {
		wanted[one] = true
	}

	accounts := make(map[account.Role]account.Account, len(roles))

	for _, one := range account.DefaultBusinessAccounts() {
		if !wanted[one.Role] {
			continue
		}

		created, err := accountsRepository(t).Create(ctx, accountOf(bakery.ID, one.Kind, one.Name))
		if err != nil {
			t.Fatalf("the account %q could not be created: %v", one.Name, err)
		}

		accounts[one.Role] = created
	}

	for _, one := range roles {
		if _, ok := accounts[one]; !ok {
			t.Fatalf("the book of the business holds no account of the role %q", one)
		}
	}

	return accounts
}

// createBusiness opens the personal business of a person.
func createBusiness(t *testing.T, owner int64) business.Business {
	t.Helper()

	return createBusinessOfKind(t, owner, business.KindPersonal, "Negocio Personal")
}

// createBusinessOfKind opens a business of a kind for a person.
func createBusinessOfKind(t *testing.T, owner int64, kind business.Kind, name string) business.Business {
	t.Helper()

	created, err := businessesRepository(t).Create(context.Background(), user.ID(owner),
		businessOf(kind, name))
	if err != nil {
		t.Fatalf("the business could not be created: %v", err)
	}

	return created
}

// countBusinesses counts the businesses of an owner.
func countBusinesses(ctx context.Context, t *testing.T, owner int64) int {
	t.Helper()

	var count int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM businesses WHERE owner_id = $1`, owner,
	).Scan(&count); err != nil {
		t.Fatalf("the businesses of the owner %d could not be counted: %v", owner, err)
	}

	return count
}

// countAccounts counts the accounts of a business.
func countAccounts(ctx context.Context, t *testing.T, businessID business.ID) int {
	t.Helper()

	var count int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM accounts WHERE business_id = $1`, int64(businessID),
	).Scan(&count); err != nil {
		t.Fatalf("the accounts of the business %d could not be counted: %v", businessID, err)
	}

	return count
}
