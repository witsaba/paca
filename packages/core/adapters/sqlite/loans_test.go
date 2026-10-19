package sqlite_test

import (
	"context"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/core/adapters/sqlite"
	"github.com/witsaba/paca/packages/core/domain"
)

// storedContact inserts a contact with raw SQL and returns its identity.
//
// The contacts repository is written in parallel, so the test does not depend
// on it: the loan only needs a contact that exists.
func storedContact(t *testing.T, db *sqlite.DB, name string) int64 {
	t.Helper()

	ctx := context.Background()

	result, err := db.ExecContext(ctx,
		`INSERT INTO contacts (kind, name, phone, id_document) VALUES ('person', ?, ?, ?)`,
		name, "300999", "CC123")
	if err != nil {
		t.Fatalf("store the contact: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read the identity of the contact: %v", err)
	}

	return id
}

// borrowedLoan returns a loan PACA owes, ready to be stored.
func borrowedLoan(contactID int64) domain.Loan {
	return domain.Loan{
		ContactID: contactID,
		Direction: domain.LoanDirectionBorrowed,
		Kind:      domain.LoanKindInformal,
		Principal: 500000,
		GrantedOn: "2024-03-01",
		DueOn:     "2024-06-30",
		Status:    domain.LoanStatusActive,
		Notes:     "a loan for the garden",
	}
}

// A stored loan comes back with its identity, its creation moment and its
// fields.
func TestLoanRepositoryStoresAndReadsALoan(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)
	contactID := storedContact(t, db, "Luis")

	stored, err := repository.Create(ctx, borrowedLoan(contactID))
	if err != nil {
		t.Fatalf("store the loan: %v", err)
	}

	if stored.ID == 0 {
		t.Error("the stored loan has no identity")
	}

	if stored.CreatedAt == "" {
		t.Error("the stored loan has no creation moment")
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the loan: %v", err)
	}

	if read.ContactID != contactID {
		t.Errorf("the contact of the loan is %d, want %d", read.ContactID, contactID)
	}

	if read.Direction != domain.LoanDirectionBorrowed {
		t.Errorf("the direction of the loan is %q, want %q", read.Direction, domain.LoanDirectionBorrowed)
	}

	if read.Kind != domain.LoanKindInformal {
		t.Errorf("the kind of the loan is %q, want %q", read.Kind, domain.LoanKindInformal)
	}

	if read.Principal != 500000 {
		t.Errorf("the principal of the loan is %d, want %d", read.Principal, 500000)
	}

	if read.GrantedOn != "2024-03-01" {
		t.Errorf("the granting day of the loan is %q, want %q", read.GrantedOn, "2024-03-01")
	}

	if read.DueOn != "2024-06-30" {
		t.Errorf("the due day of the loan is %q, want %q", read.DueOn, "2024-06-30")
	}

	if read.Status != domain.LoanStatusActive {
		t.Errorf("the status of the loan is %q, want %q", read.Status, domain.LoanStatusActive)
	}

	if read.Notes != "a loan for the garden" {
		t.Errorf("the notes of the loan are %q, want %q", read.Notes, "a loan for the garden")
	}
}

// A loan without an interest rate is stored, and the rate comes back empty.
func TestLoanRepositoryStoresALoanWithoutAnInterestRate(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)
	contactID := storedContact(t, db, "Luis")

	stored, err := repository.Create(ctx, borrowedLoan(contactID))
	if err != nil {
		t.Fatalf("store the loan: %v", err)
	}

	if stored.InterestRateBP != nil {
		t.Errorf("the stored loan carries a rate of %d, want none", *stored.InterestRateBP)
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the loan: %v", err)
	}

	if read.InterestRateBP != nil {
		t.Errorf("the loan carries a rate of %d, want none", *read.InterestRateBP)
	}
}

// A loan without a status is active, which is the default of the schema.
func TestLoanRepositoryStoresALoanWithoutAStatusAsActive(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)
	contactID := storedContact(t, db, "Luis")

	loan := borrowedLoan(contactID)
	loan.Status = ""

	stored, err := repository.Create(ctx, loan)
	if err != nil {
		t.Fatalf("store the loan: %v", err)
	}

	if stored.Status != domain.LoanStatusActive {
		t.Errorf("the status of the stored loan is %q, want %q", stored.Status, domain.LoanStatusActive)
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the loan: %v", err)
	}

	if read.Status != domain.LoanStatusActive {
		t.Errorf("the status of the loan is %q, want %q", read.Status, domain.LoanStatusActive)
	}
}

// The loans come back in the order of their identity.
func TestLoanRepositoryListsTheLoansInOrder(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)
	contactID := storedContact(t, db, "Luis")

	first, err := repository.Create(ctx, borrowedLoan(contactID))
	if err != nil {
		t.Fatalf("store the first loan: %v", err)
	}

	second, err := repository.Create(ctx, borrowedLoan(contactID))
	if err != nil {
		t.Fatalf("store the second loan: %v", err)
	}

	loans, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list the loans: %v", err)
	}

	if len(loans) != 2 {
		t.Fatalf("the list holds %d loans, want 2", len(loans))
	}

	if loans[0].ID != first.ID || loans[1].ID != second.ID {
		t.Errorf("the list is not ordered by identity: %d, %d", loans[0].ID, loans[1].ID)
	}
}

// The list of a status holds only the loans of that status.
func TestLoanRepositoryListsOnlyTheLoansOfAStatus(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)
	contactID := storedContact(t, db, "Luis")

	if _, err := repository.Create(ctx, borrowedLoan(contactID)); err != nil {
		t.Fatalf("store the active loan: %v", err)
	}

	settled := borrowedLoan(contactID)
	settled.Status = domain.LoanStatusSettled

	if _, err := repository.Create(ctx, settled); err != nil {
		t.Fatalf("store the settled loan: %v", err)
	}

	active, err := repository.ListByStatus(ctx, domain.LoanStatusActive)
	if err != nil {
		t.Fatalf("list the active loans: %v", err)
	}

	if len(active) != 1 {
		t.Fatalf("the list of the active loans holds %d loans, want 1", len(active))
	}

	if active[0].Status != domain.LoanStatusActive {
		t.Errorf("the loan of the list is %q, want %q", active[0].Status, domain.LoanStatusActive)
	}

	all, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list the loans: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("the list of every loan holds %d loans, want 2", len(all))
	}
}

// The list of a contact holds only the loans of that contact.
func TestLoanRepositoryListsOnlyTheLoansOfAContact(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)
	luis := storedContact(t, db, "Luis")
	ana := storedContact(t, db, "Ana")

	if _, err := repository.Create(ctx, borrowedLoan(luis)); err != nil {
		t.Fatalf("store the loan of Luis: %v", err)
	}

	if _, err := repository.Create(ctx, borrowedLoan(ana)); err != nil {
		t.Fatalf("store the loan of Ana: %v", err)
	}

	loans, err := repository.ListByContact(ctx, luis)
	if err != nil {
		t.Fatalf("list the loans of the contact: %v", err)
	}

	if len(loans) != 1 {
		t.Fatalf("the list of the contact holds %d loans, want 1", len(loans))
	}

	if loans[0].ContactID != luis {
		t.Errorf("the loan of the list belongs to the contact %d, want %d", loans[0].ContactID, luis)
	}
}

// The changes of a loan are stored, and they are visible when it is read
// back.
func TestLoanRepositoryUpdatesALoan(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)
	contactID := storedContact(t, db, "Luis")

	stored, err := repository.Create(ctx, borrowedLoan(contactID))
	if err != nil {
		t.Fatalf("store the loan: %v", err)
	}

	stored.Status = domain.LoanStatusSettled
	stored.Notes = "the loan is paid off"

	if err := repository.Update(ctx, stored); err != nil {
		t.Fatalf("update the loan: %v", err)
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the loan: %v", err)
	}

	if read.Status != domain.LoanStatusSettled {
		t.Errorf("the status of the loan is %q, want %q", read.Status, domain.LoanStatusSettled)
	}

	if read.Notes != "the loan is paid off" {
		t.Errorf("the notes of the loan are %q, want %q", read.Notes, "the loan is paid off")
	}
}

// A loan that does not exist is a not-found failure, when it is read and
// when it is updated.
func TestLoanRepositoryReportsAMissingLoan(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)

	_, err := repository.GetByID(ctx, 404)
	if err == nil {
		t.Fatal("the missing loan was returned, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeNotFound {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeNotFound)
	}

	missing := borrowedLoan(1)
	missing.ID = 404

	err = repository.Update(ctx, missing)
	if err == nil {
		t.Fatal("the missing loan was updated, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeNotFound {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeNotFound)
	}
}

// A loan for a contact that does not exist is a validation failure, and the
// message of the driver never reaches the detail.
func TestLoanRepositoryRefusesALoanForAMissingContact(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)

	_, err := repository.Create(ctx, borrowedLoan(999))
	if err == nil {
		t.Fatal("the loan was stored, want a refusal")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the refusal is %q, want %q", got, domain.CodeValidationFailed)
	}

	detail := domain.DetailOf(err)
	if strings.Contains(detail, "FOREIGN KEY constraint failed") {
		t.Errorf("the detail leaks the message of the driver: %q", detail)
	}
}
