package purchase

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
)

// theAccountsOfTheBook builds the accounts of the book of the bakery, the
// way the use case reads them.
func theAccountsOfTheBook(t *testing.T) []account.Account {
	t.Helper()

	seed := account.DefaultBusinessAccounts()
	accounts := make([]account.Account, 0, len(seed))

	for index, one := range seed {
		created, err := account.New(business.ID(77), one.Kind, one.Name, "", nil)
		if err != nil {
			t.Fatalf("the account was refused: %v", err)
		}

		created.ID = account.ID(index + 1)
		accounts = append(accounts, created)
	}

	return accounts
}

// theAccountOfTheRole returns the account of the book that carries that
// role.
func theAccountOfTheRole(t *testing.T, accounts []account.Account, role account.Role) account.Account {
	t.Helper()

	found, err := account.OfRole(accounts, role)
	if err != nil {
		t.Fatalf("the account of the role %q was refused: %v", role, err)
	}

	return found
}

// TestTheMoneyComesFromTheCashWhenTheOwnerNamesNone proves the rule of the
// domain: the money of the payment comes from the cash of the business,
// unless the owner names another account.
func TestTheMoneyComesFromTheCashWhenTheOwnerNamesNone(t *testing.T) {
	t.Parallel()

	accounts := theAccountsOfTheBook(t)

	paying, err := ThePayingAccount(accounts, 0)
	if err != nil {
		t.Fatalf("the account that pays was refused: %v", err)
	}

	want := theAccountOfTheRole(t, accounts, account.RoleCash)
	if paying.ID != want.ID {
		t.Errorf("the account that pays is %d, want the cash %d", paying.ID, want.ID)
	}
	if paying.Name != "Efectivo" {
		t.Errorf("the account that pays is %q, want %q", paying.Name, "Efectivo")
	}
}

// TestTheMoneyComesFromTheAccountTheOwnerNames proves the rule of the
// domain: the owner names the account that pays, and the money comes from
// it.
func TestTheMoneyComesFromTheAccountTheOwnerNames(t *testing.T) {
	t.Parallel()

	accounts := theAccountsOfTheBook(t)
	named := theAccountOfTheRole(t, accounts, account.RoleBank)

	paying, err := ThePayingAccount(accounts, named.ID)
	if err != nil {
		t.Fatalf("the account that pays was refused: %v", err)
	}

	if paying.ID != named.ID {
		t.Errorf("the account that pays is %d, want %d", paying.ID, named.ID)
	}
}

// TestTheAccountThatPaysIsNotOfTheBusinessIsRefused proves the rule of the
// domain: the money of the payment comes from an account of the book of
// the business.
func TestTheAccountThatPaysIsNotOfTheBusinessIsRefused(t *testing.T) {
	t.Parallel()

	accounts := theAccountsOfTheBook(t)

	_, err := ThePayingAccount(accounts, account.ID(999))
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, account.ErrNotFound)
	}
}

// TestTheMoneyLandsInTheInventory proves the rule of the domain: the money
// of the payment lands in the account of the inventory of the business.
func TestTheMoneyLandsInTheInventory(t *testing.T) {
	t.Parallel()

	accounts := theAccountsOfTheBook(t)

	inventory, err := TheInventoryAccount(accounts)
	if err != nil {
		t.Fatalf("the account of the inventory was refused: %v", err)
	}

	want := theAccountOfTheRole(t, accounts, account.RoleInventory)
	if inventory.ID != want.ID {
		t.Errorf("the account of the inventory is %d, want %d", inventory.ID, want.ID)
	}
	if inventory.Name != "Inventario" { //nolint:misspell // "Inventario" is the name the person reads in their book.
		t.Errorf("the account of the inventory is %q, want %q", inventory.Name, "Inventario") //nolint:misspell // "Inventario" is the name the person reads in their book.
	}
}

// TestTheBookWithoutAnInventoryIsRefused proves the rule of the domain: a
// book without an account of the inventory cannot hold what the business
// buys.
func TestTheBookWithoutAnInventoryIsRefused(t *testing.T) {
	t.Parallel()

	accounts := theAccountsOfTheBook(t)
	withoutInventory := make([]account.Account, 0, len(accounts))
	for _, one := range accounts {
		if one.Name == "Inventario" { //nolint:misspell // "Inventario" is the name the person reads in their book.
			continue
		}
		withoutInventory = append(withoutInventory, one)
	}

	_, err := TheInventoryAccount(withoutInventory)
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, account.ErrNotFound)
	}
}
