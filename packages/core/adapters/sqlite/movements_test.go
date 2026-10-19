package sqlite_test

import (
	"context"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/core/adapters/sqlite"
	"github.com/witsaba/paca/packages/core/domain"
	"github.com/witsaba/paca/packages/core/ports"
)

// movementAccount stores a cash account and returns its identity.
//
// The account is stored with raw SQL, so that the test of the movements
// never depends on another repository.
func movementAccount(t *testing.T, db *sqlite.DB, name string) int64 {
	t.Helper()

	result, err := db.ExecContext(context.Background(),
		`INSERT INTO accounts (kind, name, currency, opened_on, is_active)
		 VALUES ('cash', ?, 'COP', '2024-01-15', 1)`, name)
	if err != nil {
		t.Fatalf("store the account: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read the identity of the account: %v", err)
	}

	return id
}

// movementLoan stores a loan and returns its identity.
//
// The loan needs a contact, and both are stored with raw SQL, so that the
// test of the movements never depends on another repository.
func movementLoan(t *testing.T, db *sqlite.DB) int64 {
	t.Helper()

	result, err := db.ExecContext(context.Background(),
		`INSERT INTO contacts (kind, name, phone, id_document)
		 VALUES ('person', 'The lender', '3000000000', 'CC 1010101010')`)
	if err != nil {
		t.Fatalf("store the contact: %v", err)
	}

	contactID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read the identity of the contact: %v", err)
	}

	result, err = db.ExecContext(context.Background(),
		`INSERT INTO loans (contact_id, direction, kind, principal, granted_on)
		 VALUES (?, 'lent', 'informal', 500000, '2024-02-01')`, contactID)
	if err != nil {
		t.Fatalf("store the loan: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read the identity of the loan: %v", err)
	}

	return id
}

// incomeMovement returns an income, ready to be stored.
func incomeMovement(accountID int64, amount int64, occurredOn string) domain.Movement {
	return domain.Movement{
		AccountID:  accountID,
		Kind:       domain.MovementKindIncome,
		Amount:     amount,
		OccurredOn: occurredOn,
	}
}

// expenseMovement returns an expense, ready to be stored.
func expenseMovement(accountID int64, amount int64, occurredOn string) domain.Movement {
	return domain.Movement{
		AccountID:  accountID,
		Kind:       domain.MovementKindExpense,
		Amount:     amount,
		OccurredOn: occurredOn,
	}
}

// A stored movement comes back with its identity, its creation moment and
// its fields.
func TestMovementRepositoryStoresAndReadsAMovement(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	stored, err := repository.Create(ctx, domain.Movement{
		AccountID:  accountID,
		Kind:       domain.MovementKindIncome,
		Amount:     100000,
		OccurredOn: "2024-03-10",
		Category:   "salary",
		Notes:      "the salary of March",
	})
	if err != nil {
		t.Fatalf("store the movement: %v", err)
	}

	if stored.ID == 0 {
		t.Error("the stored movement has no identity")
	}

	if stored.CreatedAt == "" {
		t.Error("the stored movement has no creation moment")
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the movement: %v", err)
	}

	if read.AccountID != accountID {
		t.Errorf("the account of the movement is %d, want %d", read.AccountID, accountID)
	}

	if read.Kind != domain.MovementKindIncome {
		t.Errorf("the kind of the movement is %q, want %q", read.Kind, domain.MovementKindIncome)
	}

	if read.Amount != 100000 {
		t.Errorf("the amount of the movement is %d, want %d", read.Amount, 100000)
	}

	if read.OccurredOn != "2024-03-10" {
		t.Errorf("the day of the movement is %q, want %q", read.OccurredOn, "2024-03-10")
	}

	if read.Category != "salary" {
		t.Errorf("the category of the movement is %q, want %q", read.Category, "salary")
	}

	if read.Notes != "the salary of March" {
		t.Errorf("the notes of the movement are %q, want %q", read.Notes, "the salary of March")
	}

	if read.LoanID != nil {
		t.Errorf("the movement carries the loan %d, want none", *read.LoanID)
	}

	if read.CounterpartyContactID != nil {
		t.Errorf("the movement carries the contact %d, want none", *read.CounterpartyContactID)
	}
}

// The list holds the movements most recent first.
func TestMovementRepositoryListsTheMostRecentFirst(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	for _, day := range []string{"2024-01-05", "2024-03-10", "2024-02-20"} {
		if _, err := repository.Create(ctx, incomeMovement(accountID, 10000, day)); err != nil {
			t.Fatalf("store the movement of %s: %v", day, err)
		}
	}

	movements, err := repository.List(ctx, ports.MovementFilter{})
	if err != nil {
		t.Fatalf("list the movements: %v", err)
	}

	if len(movements) != 3 {
		t.Fatalf("the list holds %d movements, want 3", len(movements))
	}

	want := []string{"2024-03-10", "2024-02-20", "2024-01-05"}
	for i, day := range want {
		if movements[i].OccurredOn != day {
			t.Errorf("the movement %d of the list happened on %q, want %q", i, movements[i].OccurredOn, day)
		}
	}
}

// The filter narrows the list to the movements of one account.
func TestMovementRepositoryFiltersByAccount(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	first := movementAccount(t, db, "Cash in the drawer")
	second := movementAccount(t, db, "The savings")

	repository := sqlite.NewMovementRepository(db)

	if _, err := repository.Create(ctx, incomeMovement(first, 10000, "2024-01-05")); err != nil {
		t.Fatalf("store the first movement: %v", err)
	}

	if _, err := repository.Create(ctx, incomeMovement(second, 20000, "2024-01-06")); err != nil {
		t.Fatalf("store the second movement: %v", err)
	}

	movements, err := repository.List(ctx, ports.MovementFilter{AccountID: first})
	if err != nil {
		t.Fatalf("list the movements: %v", err)
	}

	if len(movements) != 1 {
		t.Fatalf("the list holds %d movements, want 1", len(movements))
	}

	if movements[0].AccountID != first {
		t.Errorf("the movement of the list belongs to the account %d, want %d", movements[0].AccountID, first)
	}
}

// The filter narrows the list to the given kinds.
func TestMovementRepositoryFiltersByKind(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	if _, err := repository.Create(ctx, incomeMovement(accountID, 10000, "2024-01-05")); err != nil {
		t.Fatalf("store the income: %v", err)
	}

	if _, err := repository.Create(ctx, expenseMovement(accountID, 4000, "2024-01-06")); err != nil {
		t.Fatalf("store the expense: %v", err)
	}

	if _, err := repository.Create(ctx, domain.Movement{
		AccountID:  accountID,
		Kind:       domain.MovementKindTransferIn,
		Amount:     3000,
		OccurredOn: "2024-01-07",
	}); err != nil {
		t.Fatalf("store the transfer: %v", err)
	}

	movements, err := repository.List(ctx, ports.MovementFilter{
		Kinds: []domain.MovementKind{domain.MovementKindIncome, domain.MovementKindExpense},
	})
	if err != nil {
		t.Fatalf("list the movements: %v", err)
	}

	if len(movements) != 2 {
		t.Fatalf("the list holds %d movements, want 2", len(movements))
	}

	for _, movement := range movements {
		if movement.Kind != domain.MovementKindIncome && movement.Kind != domain.MovementKindExpense {
			t.Errorf("the list holds the movement of the kind %q, want an income or an expense", movement.Kind)
		}
	}
}

// The filter narrows the list to a period, and both ends of the period are
// included.
func TestMovementRepositoryFiltersByPeriod(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	for _, day := range []string{"2024-01-01", "2024-02-01", "2024-03-01"} {
		if _, err := repository.Create(ctx, incomeMovement(accountID, 10000, day)); err != nil {
			t.Fatalf("store the movement of %s: %v", day, err)
		}
	}

	movements, err := repository.List(ctx, ports.MovementFilter{
		Period: ports.Period{From: "2024-01-01", To: "2024-02-01"},
	})
	if err != nil {
		t.Fatalf("list the movements: %v", err)
	}

	if len(movements) != 2 {
		t.Fatalf("the list holds %d movements, want 2", len(movements))
	}

	days := []string{movements[0].OccurredOn, movements[1].OccurredOn}
	for _, day := range days {
		if day != "2024-02-01" && day != "2024-01-01" {
			t.Errorf("the list holds the movement of %q, want a movement of the period", day)
		}
	}
}

// The limit caps the list.
func TestMovementRepositoryCapsTheListWithTheLimit(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	for _, day := range []string{"2024-01-05", "2024-02-20", "2024-03-10"} {
		if _, err := repository.Create(ctx, incomeMovement(accountID, 10000, day)); err != nil {
			t.Fatalf("store the movement of %s: %v", day, err)
		}
	}

	movements, err := repository.List(ctx, ports.MovementFilter{Limit: 2})
	if err != nil {
		t.Fatalf("list the movements: %v", err)
	}

	if len(movements) != 2 {
		t.Fatalf("the list holds %d movements, want 2", len(movements))
	}

	if movements[0].OccurredOn != "2024-03-10" {
		t.Errorf("the first movement of the capped list happened on %q, want %q", movements[0].OccurredOn, "2024-03-10")
	}
}

// Update stores the changes of a movement, and the change is visible when
// the movement is read back.
func TestMovementRepositoryUpdatesAMovement(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	stored, err := repository.Create(ctx, domain.Movement{
		AccountID:  accountID,
		Kind:       domain.MovementKindExpense,
		Amount:     40000,
		OccurredOn: "2024-03-10",
		Category:   "food",
	})
	if err != nil {
		t.Fatalf("store the movement: %v", err)
	}

	stored.Amount = 45000
	stored.Category = "market"

	if err := repository.Update(ctx, stored); err != nil {
		t.Fatalf("update the movement: %v", err)
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the movement: %v", err)
	}

	if read.Amount != 45000 {
		t.Errorf("the amount of the movement is %d, want %d", read.Amount, 45000)
	}

	if read.Category != "market" {
		t.Errorf("the category of the movement is %q, want %q", read.Category, "market")
	}
}

// Delete removes a movement, and deleting it again is a not-found failure.
func TestMovementRepositoryDeletesAMovement(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	stored, err := repository.Create(ctx, incomeMovement(accountID, 10000, "2024-01-05"))
	if err != nil {
		t.Fatalf("store the movement: %v", err)
	}

	if err := repository.Delete(ctx, stored.ID); err != nil {
		t.Fatalf("delete the movement: %v", err)
	}

	if _, err := repository.GetByID(ctx, stored.ID); err == nil {
		t.Fatal("the deleted movement was returned, want a failure")
	}

	err = repository.Delete(ctx, stored.ID)
	if err == nil {
		t.Fatal("the second delete succeeded, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeNotFound {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeNotFound)
	}
}

// The balance of an account is the money in minus the money out.
func TestMovementRepositoryComputesTheBalanceOfAnAccount(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	if _, err := repository.Create(ctx, incomeMovement(accountID, 100000, "2024-01-05")); err != nil {
		t.Fatalf("store the income: %v", err)
	}

	if _, err := repository.Create(ctx, expenseMovement(accountID, 40000, "2024-01-06")); err != nil {
		t.Fatalf("store the expense: %v", err)
	}

	balances, err := repository.BalanceByAccount(ctx)
	if err != nil {
		t.Fatalf("compute the balances: %v", err)
	}

	balance, ok := balances[accountID]
	if !ok {
		t.Fatalf("the balance of the account %d is not in the map: %v", accountID, balances)
	}

	if balance != 60000 {
		t.Errorf("the balance of the account is %d, want %d", balance, 60000)
	}
}

// The total of a period follows the sign of the kind of the movement.
func TestMovementRepositoryTotalsByKind(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	if _, err := repository.Create(ctx, incomeMovement(accountID, 100000, "2024-01-05")); err != nil {
		t.Fatalf("store the income: %v", err)
	}

	if _, err := repository.Create(ctx, expenseMovement(accountID, 40000, "2024-01-06")); err != nil {
		t.Fatalf("store the expense: %v", err)
	}

	totals, err := repository.TotalByKind(ctx, ports.Period{From: "2024-01-01", To: "2024-01-31"})
	if err != nil {
		t.Fatalf("total the movements by kind: %v", err)
	}

	if totals[domain.MovementKindIncome] != 100000 {
		t.Errorf("the total of the income is %d, want %d", totals[domain.MovementKindIncome], 100000)
	}

	if totals[domain.MovementKindExpense] != -40000 {
		t.Errorf("the total of the expense is %d, want %d", totals[domain.MovementKindExpense], -40000)
	}
}

// The total of a period by category groups the movements of a category, and a
// movement without a category is grouped under the empty string.
func TestMovementRepositoryTotalsByCategory(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	salary := incomeMovement(accountID, 100000, "2024-01-05")
	salary.Category = "salary"

	if _, err := repository.Create(ctx, salary); err != nil {
		t.Fatalf("store the salary: %v", err)
	}

	market := expenseMovement(accountID, 40000, "2024-01-06")
	market.Category = "food"

	if _, err := repository.Create(ctx, market); err != nil {
		t.Fatalf("store the expense: %v", err)
	}

	if _, err := repository.Create(ctx, expenseMovement(accountID, 5000, "2024-01-07")); err != nil {
		t.Fatalf("store the expense without a category: %v", err)
	}

	totals, err := repository.TotalByCategory(ctx, ports.Period{})
	if err != nil {
		t.Fatalf("total the movements by category: %v", err)
	}

	if totals["salary"] != 100000 {
		t.Errorf("the total of the salary is %d, want %d", totals["salary"], 100000)
	}

	if totals["food"] != -40000 {
		t.Errorf("the total of the food is %d, want %d", totals["food"], -40000)
	}

	if totals[""] != -5000 {
		t.Errorf("the total of the movements without a category is %d, want %d", totals[""], -5000)
	}
}

// A movement for an account that does not exist is a validation failure, and
// the message of the driver does not reach the detail.
func TestMovementRepositoryRefusesAMovementForAMissingAccount(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewMovementRepository(db)

	_, err := repository.Create(ctx, incomeMovement(404, 10000, "2024-01-05"))
	if err == nil {
		t.Fatal("the movement was stored, want a refusal")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the refusal is %q, want %q", got, domain.CodeValidationFailed)
	}

	if strings.Contains(domain.DetailOf(err), "FOREIGN KEY constraint failed") {
		t.Errorf("the detail leaks the message of the driver: %q", domain.DetailOf(err))
	}
}

// A movement that carries a loan and a kind that is not a loan kind is a
// validation failure.
func TestMovementRepositoryRefusesALoanMovementOfAnotherKind(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	accountID := movementAccount(t, db, "Cash in the drawer")
	loanID := movementLoan(t, db)

	repository := sqlite.NewMovementRepository(db)

	movement := expenseMovement(accountID, 40000, "2024-02-10")
	movement.LoanID = &loanID

	_, err := repository.Create(ctx, movement)
	if err == nil {
		t.Fatal("the movement was stored, want a refusal")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the refusal is %q, want %q", got, domain.CodeValidationFailed)
	}

	if strings.Contains(domain.DetailOf(err), "CHECK constraint failed") {
		t.Errorf("the detail leaks the message of the driver: %q", domain.DetailOf(err))
	}
}
