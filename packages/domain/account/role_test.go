package account

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
)

// theBooksOfTheTest is the storage of the accounts of the tests of the
// roles of the book.
type theBooksOfTheTest struct {
	held    []Account
	created []Account
	next    ID
	fail    error
	race    bool
	lost    bool
}

// List returns the accounts of the business.
func (b *theBooksOfTheTest) List(_ context.Context, _ business.ID) ([]Account, error) {
	if b.fail != nil {
		return nil, b.fail
	}

	return b.held, nil
}

// Create writes the account in the book.
func (b *theBooksOfTheTest) Create(_ context.Context, written Account) (Account, error) {
	if b.fail != nil {
		return Account{}, b.fail
	}

	for _, one := range b.held {
		if one.Name == written.Name {
			return Account{}, ErrAlreadyExists
		}
	}

	b.next++
	written.ID = b.next
	b.created = append(b.created, written)

	if b.race {
		// The account was opened by someone else at the same time, and
		// the storage refuses the second one.
		b.held = append(b.held, written)

		return Account{}, ErrAlreadyExists
	}

	if b.lost {
		// The storage refused the account and the book carries no
		// account of the role.
		return Account{}, ErrAlreadyExists
	}

	b.held = append(b.held, written)

	return written, nil
}

// CreateMany writes the accounts of the book.
func (b *theBooksOfTheTest) CreateMany(_ context.Context, accounts []Account) ([]Account, error) {
	return nil, errors.New("the test does not write the accounts of the book")
}

// Find returns the account that carries that identity.
func (b *theBooksOfTheTest) Find(_ context.Context, id ID) (Account, error) {
	for _, one := range b.held {
		if one.ID == id {
			return one, nil
		}
	}

	return Account{}, ErrNotFound
}

// TestTheAccountOfTheRoleIsTheOneTheBookCarries proves that the book
// answers with the account it already holds.
func TestTheAccountOfTheRoleIsTheOneTheBookCarries(t *testing.T) {
	t.Parallel()

	held := Account{ID: 6, BusinessID: 77, Kind: KindExpense, Name: "Donación"} //nolint:misspell // "Donación" is the name the person reads in their book.
	book := &theBooksOfTheTest{held: []Account{held}}

	found, err := TheAccountOfTheRole(context.Background(), book, 77, RoleDonation)
	if err != nil {
		t.Fatalf("the account of the role was refused: %v", err)
	}

	if found.ID != held.ID {
		t.Errorf("the account is %d, want %d", found.ID, held.ID)
	}

	if len(book.created) != 0 {
		t.Errorf("the book wrote %d accounts, want none", len(book.created))
	}
}

// TestTheBookThatLacksTheAccountOfTheRoleOpensIt proves the rule of the
// catch-up: a book opened before the role of the seed existed carries it
// the first time the role is used, with the name and the kind of the seed.
func TestTheBookThatLacksTheAccountOfTheRoleOpensIt(t *testing.T) {
	t.Parallel()

	book := &theBooksOfTheTest{held: []Account{
		{ID: 3, BusinessID: 77, Kind: KindAsset, Name: "Inventario"}, //nolint:misspell // "Inventario" is the name the person reads in their book.
	}}

	found, err := TheAccountOfTheRole(context.Background(), book, 77, RoleDonation)
	if err != nil {
		t.Fatalf("the account of the role was refused: %v", err)
	}

	if found.Name != "Donación" { //nolint:misspell // "Donación" is the name the person reads in their book.
		t.Errorf("the name of the account is %q, want %q", found.Name, "Donación") //nolint:misspell // "Donación" is the name the person reads in their book.
	}

	if found.Kind != KindExpense {
		t.Errorf("the kind of the account is %q, want %q", found.Kind, KindExpense)
	}

	if found.BusinessID != business.ID(77) {
		t.Errorf("the business of the account is %d, want 77", found.BusinessID)
	}

	if len(book.created) != 1 {
		t.Fatalf("the book wrote %d accounts, want 1", len(book.created))
	}
}

// TestTheRoleThatIsNotOfTheBookIsRefused proves that the book never opens
// an account of a role it does not know.
func TestTheRoleThatIsNotOfTheBookIsRefused(t *testing.T) {
	t.Parallel()

	book := &theBooksOfTheTest{}

	_, err := TheAccountOfTheRole(context.Background(), book, 77, Role("taxes"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, ErrNotFound)
	}

	if len(book.created) != 0 {
		t.Errorf("the book wrote %d accounts, want none", len(book.created))
	}
}

// TestTheAccountThatTheStorageRefusesIsNotWritten proves that the refusal
// of the storage reaches the caller.
func TestTheAccountThatTheStorageRefusesIsNotWritten(t *testing.T) {
	t.Parallel()

	book := &theBooksOfTheTest{fail: ErrNoStorage}

	_, err := TheAccountOfTheRole(context.Background(), book, 77, RoleDonation)
	if !errors.Is(err, ErrNoStorage) {
		t.Fatalf("the error is %v, want %v", err, ErrNoStorage)
	}
}

// TestTheAccountThatSomeoneElseOpenedIsTheOneTheBookUses proves that the
// book uses the account of the role that another writer opened at the same
// time: the book of the business carries one account of the role.
func TestTheAccountThatSomeoneElseOpenedIsTheOneTheBookUses(t *testing.T) {
	t.Parallel()

	book := &theBooksOfTheTest{race: true}

	found, err := TheAccountOfTheRole(context.Background(), book, 77, RoleDonation)
	if err != nil {
		t.Fatalf("the account of the role was refused: %v", err)
	}

	if found.Name != "Donación" { //nolint:misspell // "Donación" is the name the person reads in their book.
		t.Errorf("the name of the account is %q, want %q", found.Name, "Donación") //nolint:misspell // "Donación" is the name the person reads in their book.
	}
}

// TestTheAccountThatNobodyOpenedIsNotWritten proves that the refusal of
// the storage reaches the caller when the account of the role is nowhere
// in the book.
func TestTheAccountThatNobodyOpenedIsNotWritten(t *testing.T) {
	t.Parallel()

	book := &theBooksOfTheTest{lost: true}

	_, err := TheAccountOfTheRole(context.Background(), book, 77, RoleDonation)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("the error is %v, want %v", err, ErrAlreadyExists)
	}
}
