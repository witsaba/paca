package purchase

import (
	"fmt"

	"github.com/witsaba/paca/packages/domain/account"
)

// ThePayingAccount answers the account the money of the payment comes
// from.
//
// The money comes from the cash of the business, and from the account the
// owner names when she names another one: the cash is what the bakery
// pays with, and the bank is what she pays with when she paid with the
// bank.
//
// The error is account.ErrNotFound when the account the owner names is not
// one of the book of the business, and when the book holds no cash.
func ThePayingAccount(accounts []account.Account, named account.ID) (account.Account, error) {
	if named == 0 {
		found, err := account.OfRole(accounts, account.RoleCash)
		if err != nil {
			return account.Account{}, fmt.Errorf("%w: the business has no cash to pay with", err)
		}

		return found, nil
	}

	for _, one := range accounts {
		if one.ID == named {
			return one, nil
		}
	}

	return account.Account{}, fmt.Errorf("%w: the account %d is not one of the book of the business",
		account.ErrNotFound, named)
}

// TheInventoryAccount answers the account of the inventory of the business.
//
// The money of a purchase lands in the inventory: the business bought
// something, and the inventory is what it has.
//
// The error is account.ErrNotFound when the book of the business holds no
// inventory.
func TheInventoryAccount(accounts []account.Account) (account.Account, error) {
	found, err := account.OfRole(accounts, account.RoleInventory)
	if err != nil {
		return account.Account{}, fmt.Errorf("%w: the business has no inventory to buy into", err)
	}

	return found, nil
}
