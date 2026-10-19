package sqlite_test

import (
	"context"
	"testing"

	"github.com/witsaba/paca/packages/core/adapters/sqlite"
	"github.com/witsaba/paca/packages/core/domain"
)

// nequiAccount returns a Nequi account, ready to be stored.
func nequiAccount(name string, phone string) domain.Account {
	return domain.Account{
		Kind:     domain.AccountKindNequi,
		Name:     name,
		OpenedOn: "2024-01-15",
		Phone:    phone,
		IsActive: true,
	}
}

// A stored account comes back with its identity and its creation moment.
func TestAccountRepositoryStoresAndReadsAnAccount(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewAccountRepository(db)

	stored, err := repository.Create(ctx, nequiAccount("Nequi principal", "3001112233"))
	if err != nil {
		t.Fatalf("store the account: %v", err)
	}

	if stored.ID == 0 {
		t.Error("the stored account has no identity")
	}

	if stored.CreatedAt == "" {
		t.Error("the stored account has no creation moment")
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the account: %v", err)
	}

	if read.Name != "Nequi principal" {
		t.Errorf("the name of the account is %q, want %q", read.Name, "Nequi principal")
	}

	if read.Phone != "3001112233" {
		t.Errorf("the phone of the account is %q, want %q", read.Phone, "3001112233")
	}

	if read.Kind != domain.AccountKindNequi {
		t.Errorf("the kind of the account is %q, want %q", read.Kind, domain.AccountKindNequi)
	}

	if !read.IsActive {
		t.Error("the account is not active")
	}

	if read.Currency != "COP" {
		t.Errorf("the currency of the account is %q, want %q", read.Currency, "COP")
	}
}

// A cash account stores none of the subtype columns of a bank account.
func TestAccountRepositoryStoresACashAccount(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewAccountRepository(db)

	stored, err := repository.Create(ctx, domain.Account{
		Kind:     domain.AccountKindCash,
		Name:     "Cash in the drawer",
		OpenedOn: "2024-01-15",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("store the account: %v", err)
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the account: %v", err)
	}

	if read.Phone != "" || read.BankCode != "" || read.AccountNumber != "" {
		t.Errorf("the cash account carries the columns of another kind: %+v", read)
	}
}

// The accounts come back in the order of their identity.
func TestAccountRepositoryListsTheAccountsInOrder(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewAccountRepository(db)

	first, err := repository.Create(ctx, nequiAccount("First", "3000000001"))
	if err != nil {
		t.Fatalf("store the first account: %v", err)
	}

	second, err := repository.Create(ctx, nequiAccount("Second", "3000000002"))
	if err != nil {
		t.Fatalf("store the second account: %v", err)
	}

	accounts, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list the accounts: %v", err)
	}

	if len(accounts) != 2 {
		t.Fatalf("the list holds %d accounts, want 2", len(accounts))
	}

	if accounts[0].ID != first.ID || accounts[1].ID != second.ID {
		t.Errorf("the list is not ordered by identity: %d, %d", accounts[0].ID, accounts[1].ID)
	}
}

// A closed account is not part of the list of the accounts that are open.
func TestAccountRepositoryListsOnlyTheOpenAccounts(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewAccountRepository(db)

	if _, err := repository.Create(ctx, nequiAccount("Open", "3000000001")); err != nil {
		t.Fatalf("store the open account: %v", err)
	}

	closed := nequiAccount("Closed", "3000000002")
	closed.IsActive = false
	closed.ClosedOn = "2024-06-30"

	if _, err := repository.Create(ctx, closed); err != nil {
		t.Fatalf("store the closed account: %v", err)
	}

	active, err := repository.ListActive(ctx)
	if err != nil {
		t.Fatalf("list the open accounts: %v", err)
	}

	if len(active) != 1 {
		t.Fatalf("the list of the open accounts holds %d accounts, want 1", len(active))
	}

	if active[0].Name != "Open" {
		t.Errorf("the open account is %q, want %q", active[0].Name, "Open")
	}

	all, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list the accounts: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("the list of every account holds %d accounts, want 2", len(all))
	}
}

// A Nequi account is identified by its phone number, so a second one with
// the same phone is refused as a validation failure.
func TestAccountRepositoryRefusesADuplicatePhone(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewAccountRepository(db)

	if _, err := repository.Create(ctx, nequiAccount("First", "3000000001")); err != nil {
		t.Fatalf("store the first account: %v", err)
	}

	_, err := repository.Create(ctx, nequiAccount("Second", "3000000001"))
	if err == nil {
		t.Fatal("the second account was stored, want a refusal")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the refusal is %q, want %q", got, domain.CodeValidationFailed)
	}
}

// An account that does not exist is a not-found failure.
func TestAccountRepositoryReportsAMissingAccount(t *testing.T) {
	db := openMigratedDB(t)

	repository := sqlite.NewAccountRepository(db)

	_, err := repository.GetByID(context.Background(), 404)
	if err == nil {
		t.Fatal("the missing account was returned, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeNotFound {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeNotFound)
	}
}

// An empty database holds no accounts, and the list is empty and not nil.
func TestAccountRepositoryListsNothingOnAnEmptyDatabase(t *testing.T) {
	db := openMigratedDB(t)

	repository := sqlite.NewAccountRepository(db)

	accounts, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list the accounts: %v", err)
	}

	if len(accounts) != 0 {
		t.Errorf("the list holds %d accounts, want 0", len(accounts))
	}
}
