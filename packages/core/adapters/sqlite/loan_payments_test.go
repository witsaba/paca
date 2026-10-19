package sqlite_test

import (
	"context"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/core/adapters/sqlite"
	"github.com/witsaba/paca/packages/core/domain"
)

// storedLoan stores a contact and a loan over them, and returns the identity
// of the loan.
//
// The loan is stored with raw SQL: the repositories of the contacts and of
// the loans are written elsewhere, and this test needs a loan, not a
// repository.
func storedLoan(t *testing.T, db *sqlite.DB) int64 {
	t.Helper()

	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO contacts (kind, name, phone, id_document)
		VALUES ('person', 'Luis', '300999', 'CC123')`); err != nil {
		t.Fatalf("store the contact: %v", err)
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO loans (contact_id, direction, kind, principal, granted_on)
		VALUES (1, 'borrowed', 'informal', 500000, '2024-02-01')`)
	if err != nil {
		t.Fatalf("store the loan: %v", err)
	}

	loanID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read the identity of the loan: %v", err)
	}

	return loanID
}

// storedMovement stores the account and the movement that settle a payment,
// and returns the identity of the movement.
func storedMovement(t *testing.T, db *sqlite.DB) int64 {
	t.Helper()

	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO accounts (kind, name, opened_on)
		VALUES ('cash', 'Cash in the drawer', '2024-01-15')`); err != nil {
		t.Fatalf("store the account: %v", err)
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO movements (account_id, kind, amount, occurred_on)
		VALUES (1, 'loan_repayment', 50000, '2024-03-05')`)
	if err != nil {
		t.Fatalf("store the movement: %v", err)
	}

	movementID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read the identity of the movement: %v", err)
	}

	return movementID
}

// firstPayment returns an obligation of the given loan, ready to be stored.
func firstPayment(loanID int64, seq int, amount int64, dueOn string) domain.LoanPayment {
	return domain.LoanPayment{
		LoanID: loanID,
		Seq:    seq,
		Amount: amount,
		DueOn:  dueOn,
	}
}

// A stored obligation comes back with its identity, its creation moment and
// its fields.
func TestLoanPaymentRepositoryStoresAndReadsAPayment(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanPaymentRepository(db)

	loanID := storedLoan(t, db)

	stored, err := repository.Create(ctx, domain.LoanPayment{
		LoanID: loanID,
		Seq:    1,
		Amount: 166666,
		DueOn:  "2024-03-01",
		Notes:  "the first quota",
	})
	if err != nil {
		t.Fatalf("store the payment: %v", err)
	}

	if stored.ID == 0 {
		t.Error("the stored payment has no identity")
	}

	if stored.CreatedAt == "" {
		t.Error("the stored payment has no creation moment")
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the payment: %v", err)
	}

	if read.LoanID != loanID {
		t.Errorf("the loan of the payment is %d, want %d", read.LoanID, loanID)
	}

	if read.Seq != 1 {
		t.Errorf("the position of the payment is %d, want %d", read.Seq, 1)
	}

	if read.Amount != 166666 {
		t.Errorf("the amount of the payment is %d, want %d", read.Amount, 166666)
	}

	if read.DueOn != "2024-03-01" {
		t.Errorf("the day the payment falls due is %q, want %q", read.DueOn, "2024-03-01")
	}

	if read.PaidOn != "" {
		t.Errorf("the outstanding payment carries a day of payment: %q", read.PaidOn)
	}

	if read.MovementID != nil {
		t.Errorf("the outstanding payment carries a movement: %d", *read.MovementID)
	}

	if read.Notes != "the first quota" {
		t.Errorf("the notes of the payment are %q, want %q", read.Notes, "the first quota")
	}
}

// The obligations of a loan come back in the order of their position in the
// schedule.
func TestLoanPaymentRepositoryListsThePaymentsOfALoanInOrder(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanPaymentRepository(db)

	loanID := storedLoan(t, db)

	for _, seq := range []int{3, 1, 2} {
		payment := firstPayment(loanID, seq, 100000, "2024-04-01")

		if _, err := repository.Create(ctx, payment); err != nil {
			t.Fatalf("store the payment %d: %v", seq, err)
		}
	}

	payments, err := repository.ListByLoan(ctx, loanID)
	if err != nil {
		t.Fatalf("list the payments of the loan: %v", err)
	}

	if len(payments) != 3 {
		t.Fatalf("the list holds %d payments, want 3", len(payments))
	}

	for i, want := range []int{1, 2, 3} {
		if payments[i].Seq != want {
			t.Errorf("the payment %d of the list is at the position %d, want %d", i, payments[i].Seq, want)
		}
	}
}

// The outstanding obligations come back in the order of the day they fall
// due.
func TestLoanPaymentRepositoryListsTheUnpaidPaymentsByDueDate(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanPaymentRepository(db)

	loanID := storedLoan(t, db)

	late, err := repository.Create(ctx, firstPayment(loanID, 2, 100000, "2024-05-01"))
	if err != nil {
		t.Fatalf("store the late payment: %v", err)
	}

	early, err := repository.Create(ctx, firstPayment(loanID, 1, 100000, "2024-03-01"))
	if err != nil {
		t.Fatalf("store the early payment: %v", err)
	}

	if err := repository.MarkPaid(ctx, late.ID, "2024-05-05", nil); err != nil {
		t.Fatalf("settle the late payment: %v", err)
	}

	unpaid, err := repository.ListUnpaid(ctx)
	if err != nil {
		t.Fatalf("list the outstanding payments: %v", err)
	}

	if len(unpaid) != 1 {
		t.Fatalf("the list holds %d payments, want 1", len(unpaid))
	}

	if unpaid[0].ID != early.ID {
		t.Errorf("the outstanding payment is %d, want %d", unpaid[0].ID, early.ID)
	}
}

// MarkPaid records the day of the payment and the movement that settled it.
func TestLoanPaymentRepositoryMarksAPaymentAsPaid(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanPaymentRepository(db)

	loanID := storedLoan(t, db)
	movementID := storedMovement(t, db)

	stored, err := repository.Create(ctx, firstPayment(loanID, 1, 100000, "2024-03-01"))
	if err != nil {
		t.Fatalf("store the payment: %v", err)
	}

	if err := repository.MarkPaid(ctx, stored.ID, "2024-03-05", &movementID); err != nil {
		t.Fatalf("settle the payment: %v", err)
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the payment: %v", err)
	}

	if read.PaidOn != "2024-03-05" {
		t.Errorf("the day of the payment is %q, want %q", read.PaidOn, "2024-03-05")
	}

	if read.MovementID == nil {
		t.Fatal("the settled payment carries no movement")
	}

	if *read.MovementID != movementID {
		t.Errorf("the movement of the payment is %d, want %d", *read.MovementID, movementID)
	}
}

// An obligation that does not exist is a not-found failure, both when it is
// read and when it is settled.
func TestLoanPaymentRepositoryReportsAMissingPayment(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanPaymentRepository(db)

	_, err := repository.GetByID(ctx, 404)
	if err == nil {
		t.Fatal("the missing payment was returned, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeNotFound {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeNotFound)
	}

	err = repository.MarkPaid(ctx, 404, "2024-03-05", nil)
	if err == nil {
		t.Fatal("the missing payment was settled, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeNotFound {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeNotFound)
	}
}

// A loan holds one obligation per position in its schedule.
func TestLoanPaymentRepositoryRefusesASecondPaymentAtTheSamePosition(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanPaymentRepository(db)

	loanID := storedLoan(t, db)

	if _, err := repository.Create(ctx, firstPayment(loanID, 1, 100000, "2024-03-01")); err != nil {
		t.Fatalf("store the first payment: %v", err)
	}

	_, err := repository.Create(ctx, firstPayment(loanID, 1, 100000, "2024-04-01"))
	if err == nil {
		t.Fatal("the second payment was stored, want a refusal")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the refusal is %q, want %q", got, domain.CodeValidationFailed)
	}

	detail := domain.DetailOf(err)
	if strings.Contains(detail, "UNIQUE constraint failed") || strings.Contains(detail, "loan_payments") {
		t.Errorf("the detail of the refusal carries the message of the driver: %q", detail)
	}
}

// An obligation belongs to a loan that exists.
func TestLoanPaymentRepositoryRefusesAPaymentForAMissingLoan(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanPaymentRepository(db)

	_, err := repository.Create(ctx, firstPayment(404, 1, 100000, "2024-03-01"))
	if err == nil {
		t.Fatal("the payment of a missing loan was stored, want a refusal")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the refusal is %q, want %q", got, domain.CodeValidationFailed)
	}
}
