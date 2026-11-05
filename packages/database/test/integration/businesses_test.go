//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/user"
)

// TestTheOwnerMayHoldABusinessWithABookOfItsOwn proves the rule of the
// domain against a real database: a person may own the Personal Business
// and a second business of the kind business, and the second business has a
// book of accounts of its own.
func TestTheOwnerMayHoldABusinessWithABookOfItsOwn(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := businessesRepository(t)
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Bakery")

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

	// The book of the person and the book of the bakery are books of
	// their own: the accounts of one are not the accounts of the other.
	accounts := accountsRepository(t)

	personalBook := accountsOf(personal.ID, account.DefaultPersonalAccounts())
	if _, err := accounts.CreateMany(ctx, personalBook); err != nil {
		t.Fatalf("the accounts of the personal business could not be created: %v", err)
	}

	bakeryBook := accountsOf(bakery.ID, account.DefaultBusinessAccounts())
	written, err := accounts.CreateMany(ctx, bakeryBook)
	if err != nil {
		t.Fatalf("the accounts of the bakery could not be created: %v", err)
	}
	if len(written) != len(account.DefaultBusinessAccounts()) {
		t.Fatalf("the storage answered %d accounts, want %d", len(written), len(account.DefaultBusinessAccounts()))
	}

	if count := countAccounts(ctx, t, bakery.ID); count != len(account.DefaultBusinessAccounts()) {
		t.Errorf("the database stored %d accounts of the bakery, want %d",
			count, len(account.DefaultBusinessAccounts()))
	}
	if count := countAccounts(ctx, t, personal.ID); count != len(account.DefaultPersonalAccounts()) {
		t.Errorf("the database stored %d accounts of the personal business, want %d",
			count, len(account.DefaultPersonalAccounts()))
	}

	// The names the person reads are the ones of the seed of the domain,
	// and the two books never share an account.
	for _, one := range written {
		if one.BusinessID != bakery.ID {
			t.Errorf("the account %q belongs to the business %d, want %d", one.Name, one.BusinessID, bakery.ID)
		}
	}

	listed, err := accounts.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the accounts of the bakery could not be read: %v", err)
	}
	for index := 1; index < len(listed); index++ {
		if listed[index-1].ID >= listed[index].ID {
			t.Errorf("the accounts are not in the order of their identity: %d then %d",
				listed[index-1].ID, listed[index].ID)
		}
	}
}

// TestTheBusinessesOfTheOwnerAreReadBack proves that the businesses of a
// person are read through the person that owns them, with their kinds, and
// that the personal business is one of them.
func TestTheBusinessesOfTheOwnerAreReadBack(t *testing.T) {
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

	owned, err := repo.ListByOwner(ctx, user.ID(owner))
	if err != nil {
		t.Fatalf("the businesses of the owner could not be read: %v", err)
	}
	if len(owned) != 2 {
		t.Fatalf("the storage answered %d businesses, want 2", len(owned))
	}

	if owned[0].ID > owned[1].ID {
		t.Errorf("the businesses are not in the order of their identity: %d then %d", owned[0].ID, owned[1].ID)
	}
	if owned[0].ID != personal.ID {
		t.Errorf("the first business is %d, want %d", owned[0].ID, personal.ID)
	}
	if owned[0].Kind != business.KindPersonal {
		t.Errorf("the kind of the first business is %q, want %q", owned[0].Kind, business.KindPersonal)
	}
	if owned[1].ID != bakery.ID {
		t.Errorf("the second business is %d, want %d", owned[1].ID, bakery.ID)
	}
	if owned[1].Kind != business.KindBusiness {
		t.Errorf("the kind of the second business is %q, want %q", owned[1].Kind, business.KindBusiness)
	}
	if owned[1].Name != "La Panadería" {
		t.Errorf("the name of the second business is %q, want %q", owned[1].Name, "La Panadería")
	}
}

// TestTheOwnerWithoutABusinessHoldsNone proves that the businesses of a
// person that owns none are an empty list, and not an error.
func TestTheOwnerWithoutABusinessHoldsNone(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := businessesRepository(t)
	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of Nothing")

	owned, err := repo.ListByOwner(ctx, user.ID(owner))
	if err != nil {
		t.Fatalf("the businesses of the owner could not be read: %v", err)
	}
	if len(owned) != 0 {
		t.Errorf("the storage answered %d businesses, want none", len(owned))
	}
}

// accountsOf builds the accounts of a business out of a seed of the domain.
//
// The seed belongs to the domain: the test reads it and never writes it a
// second time.
func accountsOf(businessID business.ID, seed []account.Seed) []account.Account {
	accounts := make([]account.Account, 0, len(seed))
	for _, one := range seed {
		accounts = append(accounts, accountOf(businessID, one.Kind, one.Name))
	}

	return accounts
}
