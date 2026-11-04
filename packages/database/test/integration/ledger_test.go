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
	"github.com/witsaba/paca/packages/domain/ledger"
)

// The names the database gives to the guards of the table of the movements.
const (
	// ledgerAmountConstraint is the constraint that refuses an amount that
	// is not positive.
	ledgerAmountConstraint = "ledger_entries_amount_positive_check"

	// ledgerDistinctConstraint is the constraint that refuses the two sides
	// of a movement that are the same account.
	ledgerDistinctConstraint = "ledger_entries_distinct_accounts_check"

	// ledgerDescriptionConstraint is the constraint that refuses a movement
	// without a description.
	ledgerDescriptionConstraint = "ledger_entries_description_not_empty_check"
)

// TestAMovementIsWrittenAndReadBack proves that the movement of the book is
// stored as the domain states it.
func TestAMovementIsWrittenAndReadBack(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Book")
	created := createBusiness(t, owner)

	cash := createAccount(t, created.ID, account.KindAsset, "Efectivo")
	food := createAccount(t, created.ID, account.KindExpense, "Alimentación")

	day := ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	entry := movement(t, created.ID, food.ID, cash.ID, 5000, day, "Lunch")

	repo := ledgerRepository(t)
	written, err := repo.Record(ctx, entry)
	if err != nil {
		t.Fatalf("the movement could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the identity of the movement is 0, want the one the database generated")
	}
	if written.BusinessID != created.ID {
		t.Errorf("the business is %d, want %d", written.BusinessID, created.ID)
	}
	if written.DebitAccountID != food.ID {
		t.Errorf("the debit account is %d, want %d", written.DebitAccountID, food.ID)
	}
	if written.CreditAccountID != cash.ID {
		t.Errorf("the credit account is %d, want %d", written.CreditAccountID, cash.ID)
	}
	if written.Amount != 5000 {
		t.Errorf("the amount is %d, want 5000", written.Amount)
	}
	if written.OccurredOn.String() != "2026-03-01" {
		t.Errorf("the day is %q, want %q", written.OccurredOn.String(), "2026-03-01")
	}
	if written.Description != "Lunch" {
		t.Errorf("the description is %q, want %q", written.Description, "Lunch")
	}
}

// TestTheMovementsOfAPeriodAreListed proves that the libro diario is the
// book in the order the person reads it: the day, and then the identity.
func TestTheMovementsOfAPeriodAreListed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Book")
	created := createBusiness(t, owner)

	cash := createAccount(t, created.ID, account.KindAsset, "Efectivo")
	food := createAccount(t, created.ID, account.KindExpense, "Alimentación")

	march := ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	april := ledger.NewDate(time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))

	repo := ledgerRepository(t)
	first, err := repo.Record(ctx, movement(t, created.ID, food.ID, cash.ID, 5000, march, "Lunch of March"))
	if err != nil {
		t.Fatalf("the first movement could not be written: %v", err)
	}
	second, err := repo.Record(ctx, movement(t, created.ID, food.ID, cash.ID, 7000, march, "Lunch of the second day"))
	if err != nil {
		t.Fatalf("the second movement could not be written: %v", err)
	}
	if _, err := repo.Record(ctx, movement(t, created.ID, food.ID, cash.ID, 9000, april, "Lunch of April")); err != nil {
		t.Fatalf("the third movement could not be written: %v", err)
	}

	listed, err := repo.List(ctx, created.ID, march, march)
	if err != nil {
		t.Fatalf("the movements could not be read: %v", err)
	}

	if len(listed) != 2 {
		t.Fatalf("the period holds %d movements, want 2", len(listed))
	}
	if listed[0].ID != first.ID || listed[1].ID != second.ID {
		t.Errorf("the movements are %d and %d, want %d and %d", listed[0].ID, listed[1].ID, first.ID, second.ID)
	}
}

// TestTheBalancesAreTheSumOfTheMovements proves that the libro mayor is the
// book read by account: what landed, what left, and the account that never
// moved.
func TestTheBalancesAreTheSumOfTheMovements(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Book")
	created := createBusiness(t, owner)

	cash := createAccount(t, created.ID, account.KindAsset, "Efectivo")
	food := createAccount(t, created.ID, account.KindExpense, "Alimentación")
	salary := createAccount(t, created.ID, account.KindIncome, "Salario")

	day := ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))

	repo := ledgerRepository(t)
	if _, err := repo.Record(ctx, movement(t, created.ID, cash.ID, salary.ID, 100000, day, "Salary of March")); err != nil {
		t.Fatalf("the salary could not be written: %v", err)
	}
	if _, err := repo.Record(ctx, movement(t, created.ID, food.ID, cash.ID, 5000, day, "Lunch")); err != nil {
		t.Fatalf("the lunch could not be written: %v", err)
	}

	balances, err := repo.Balances(ctx, created.ID, day)
	if err != nil {
		t.Fatalf("the balances could not be read: %v", err)
	}

	byAccount := make(map[account.ID]ledger.Balance, len(balances))
	for _, balance := range balances {
		byAccount[balance.AccountID] = balance
	}

	if len(balances) != 3 {
		t.Fatalf("the book holds %d balances, want 3", len(balances))
	}

	cashBalance := byAccount[cash.ID]
	if cashBalance.Debits != 100000 || cashBalance.Credits != 5000 {
		t.Errorf("the cash landed %d and left %d, want 100000 and 5000", cashBalance.Debits, cashBalance.Credits)
	}
	if cashBalance.Normal() != 95000 {
		t.Errorf("the cash is %d, want 95000", cashBalance.Normal())
	}

	foodBalance := byAccount[food.ID]
	if foodBalance.Normal() != 5000 {
		t.Errorf("the food is %d, want 5000", foodBalance.Normal())
	}

	salaryBalance := byAccount[salary.ID]
	if salaryBalance.Normal() != 100000 {
		t.Errorf("the salary is %d, want 100000", salaryBalance.Normal())
	}

	book := ledger.BookOf(created.ID, balances)
	if book.Assets != 95000 {
		t.Errorf("the assets are %d, want 95000", book.Assets)
	}
	if book.Expenses != 5000 {
		t.Errorf("the expenses are %d, want 5000", book.Expenses)
	}
	if book.Income != 100000 {
		t.Errorf("the income is %d, want 100000", book.Income)
	}
	if book.NetWorth() != 95000 {
		t.Errorf("the net worth is %d, want 95000", book.NetWorth())
	}
}

// TestTheBalanceOfAnAccountThatNeverMovedIsZero proves that the account
// that never moved is in the book: the person reads the accounts of their
// book, and an account that is missing reads like an account that does not
// exist.
func TestTheBalanceOfAnAccountThatNeverMovedIsZero(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Book")
	created := createBusiness(t, owner)

	unused := createAccount(t, created.ID, account.KindAsset, "Efectivo")

	repo := ledgerRepository(t)
	balances, err := repo.Balances(ctx, created.ID, ledger.NewDate(time.Now()))
	if err != nil {
		t.Fatalf("the balances could not be read: %v", err)
	}

	if len(balances) != 1 {
		t.Fatalf("the book holds %d balances, want 1", len(balances))
	}
	if balances[0].AccountID != unused.ID {
		t.Errorf("the account is %d, want %d", balances[0].AccountID, unused.ID)
	}
	if balances[0].Normal() != 0 {
		t.Errorf("the balance is %d, want 0", balances[0].Normal())
	}
}

// TestTheDatabaseRefusesTheMovementsThatBreakTheRules proves that the table
// holds the rules of the domain as a defense in depth.
func TestTheDatabaseRefusesTheMovementsThatBreakTheRules(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Book")
	created := createBusiness(t, owner)

	cash := createAccount(t, created.ID, account.KindAsset, "Efectivo")

	cases := []struct {
		name       string
		query      string
		args       []any
		constraint string
	}{
		{
			name:       "an amount that is not positive",
			query:      `INSERT INTO ledger_entries (business_id, debit_account_id, credit_account_id, amount, occurred_on, description) VALUES ($1, $2, $3, $4, $5, $6)`,
			args:       []any{int64(created.ID), int64(cash.ID), int64(cash.ID), int64(0), "2026-03-01", "Lunch"},
			constraint: ledgerAmountConstraint,
		},
		{
			name:       "the two sides of the movement are the same account",
			query:      `INSERT INTO ledger_entries (business_id, debit_account_id, credit_account_id, amount, occurred_on, description) VALUES ($1, $2, $3, $4, $5, $6)`,
			args:       []any{int64(created.ID), int64(cash.ID), int64(cash.ID), int64(5000), "2026-03-01", "Lunch"},
			constraint: ledgerDistinctConstraint,
		},
		{
			name:       "a movement without a description",
			query:      `INSERT INTO ledger_entries (business_id, debit_account_id, credit_account_id, amount, occurred_on, description) VALUES ($1, $2, $3, $4, $5, $6)`,
			args:       []any{int64(created.ID), int64(cash.ID), int64(cash.ID), int64(5000), "2026-03-01", ""},
			constraint: ledgerDescriptionConstraint,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := theSuite.db.ExecContext(ctx, testCase.query, testCase.args...)
			if err == nil {
				t.Fatal("the database stored the movement, want the row refused")
			}

			pgErr := postgresError(t, err)
			if pgErr.Code != checkViolation {
				t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
			}
			if pgErr.ConstraintName != testCase.constraint {
				t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, testCase.constraint)
			}
		})
	}
}

// TestAMovementOfABusinessThatIsNotThereIsRefused proves that the book of a
// business that does not exist cannot be written.
func TestAMovementOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Book")
	created := createBusiness(t, owner)

	cash := createAccount(t, created.ID, account.KindAsset, "Efectivo")
	food := createAccount(t, created.ID, account.KindExpense, "Alimentación")

	day := ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))

	repo := ledgerRepository(t)
	entry := movement(t, business.ID(created.ID+100000), food.ID, cash.ID, 5000, day, "Lunch")

	_, err := repo.Record(ctx, entry)
	if !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrNotFound)
	}
}

// ledgerRepository returns the repository of the book, bound to the database
// of the suite.
func ledgerRepository(t *testing.T) *postgres.LedgerRepository {
	t.Helper()

	repo, err := postgres.NewLedgerRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the book could not be created: %v", err)
	}

	return repo
}

// createAccount writes one account of a business and answers it.
func createAccount(t *testing.T, businessID business.ID, kind account.Kind, name string) account.Account {
	t.Helper()

	created, err := accountsRepository(t).Create(context.Background(), accountOf(businessID, kind, name))
	if err != nil {
		t.Fatalf("the account could not be created: %v", err)
	}

	return created
}

// movement builds the movement of the book that the tests write.
func movement(
	t *testing.T,
	businessID business.ID,
	debit account.ID,
	credit account.ID,
	amount ledger.Amount,
	day ledger.Date,
	description string,
) ledger.Entry {
	t.Helper()

	entry, err := ledger.NewEntry(businessID, debit, credit, amount, day, description)
	if err != nil {
		t.Fatalf("the movement could not be built: %v", err)
	}

	return entry
}
