package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/user"
)

// ListByOwner answers the businesses of the owner.
//
// The fake of the storage of the businesses lives in the test of the
// onboarding, and the port of the businesses grew the method: the fake
// answers it here, so the two tests of the application share one fake of
// the storage of the businesses.
func (f *fakeBusinesses) ListByOwner(_ context.Context, _ user.ID) ([]business.Business, error) {
	if f.err != nil {
		return nil, f.err
	}

	return nil, nil
}

// fakeOwnerBook is the fake of the storage of the businesses of the owner.
//
// It remembers the businesses it wrote and the businesses it was asked for:
// the use case reads the businesses of the owner before it opens one, and
// the test proves that it did.
type fakeOwnerBook struct {
	owned   []business.Business
	created []business.Business
	listed  int
	err     error
}

// Find answers the business of the identity.
func (f *fakeOwnerBook) Find(_ context.Context, _ business.ID) (business.Business, error) {
	return business.Business{}, business.ErrNotFound
}

// FindPersonalByOwner answers the personal business of the owner.
func (f *fakeOwnerBook) FindPersonalByOwner(_ context.Context, _ user.ID) (business.Business, error) {
	return business.Business{}, business.ErrNotFound
}

// Create writes the business and remembers it.
func (f *fakeOwnerBook) Create(_ context.Context, owner user.ID, written business.Business) (business.Business, error) {
	if f.err != nil {
		return business.Business{}, f.err
	}

	created := written
	created.ID = business.ID(77)
	created.OwnerID = owner
	f.created = append(f.created, created)

	return created, nil
}

// ListByOwner answers the businesses of the owner, and remembers that it was
// asked for them.
func (f *fakeOwnerBook) ListByOwner(_ context.Context, _ user.ID) ([]business.Business, error) {
	f.listed++
	if f.err != nil {
		return nil, f.err
	}

	return f.owned, nil
}

// fakeTransaction is the fake of the transaction of the storage.
//
// It keeps what the work wrote, and it gives it back when the work fails: a
// transaction that rolls back leaves the storage as it was.
type fakeTransaction struct {
	book     *fakeOwnerBook
	accounts *fakeAccounts
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakeTransaction) Do(ctx context.Context, work func(context.Context) error) error {
	created := len(f.book.created)
	written := len(f.accounts.written)

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.book.created = f.book.created[:created]
	f.accounts.written = f.accounts.written[:written]

	return err
}

// newBusinessService builds the use case of the businesses over the fakes of
// the storage.
func newBusinessService(book *fakeOwnerBook, accounts *fakeAccounts) *application.BusinessService {
	return application.NewBusinessService(book, accounts, &fakeTransaction{book: book, accounts: accounts})
}

// TestTheBusinessIsOpenedWithItsOwnBook proves that a business is opened
// with the name the person reads, with the peso when the caller has no
// opinion about the currency, and with the accounts of the book of a
// business.
func TestTheBusinessIsOpenedWithItsOwnBook(t *testing.T) {
	t.Parallel()

	book := &fakeOwnerBook{}
	accounts := &fakeAccounts{}
	service := newBusinessService(book, accounts)

	opened, err := service.Open(t.Context(), "1234567", "Panadería del barrio", "")
	if err != nil {
		t.Fatalf("the business could not be opened: %v", err)
	}

	if len(book.created) != 1 {
		t.Fatalf("the storage holds %d businesses, want 1", len(book.created))
	}
	created := book.created[0]
	if created.Kind != business.KindBusiness {
		t.Errorf("the kind of the business is %q, want %q", created.Kind, business.KindBusiness)
	}
	if created.Name != "Panadería del barrio" {
		t.Errorf("the name is %q, want %q", created.Name, "Panadería del barrio")
	}
	if created.Currency != business.DefaultCurrency {
		t.Errorf("the currency is %q, want %q", created.Currency, business.DefaultCurrency)
	}
	if created.OwnerID != user.ID(1234567) {
		t.Errorf("the owner is %d, want %d", created.OwnerID, 1234567)
	}

	seed := account.DefaultBusinessAccounts()
	if len(accounts.written) != len(seed) {
		t.Fatalf("the storage holds %d accounts, want %d", len(accounts.written), len(seed))
	}
	for index, one := range seed {
		if accounts.written[index].Name != one.Name {
			t.Errorf("the account %d is %q, want %q", index, accounts.written[index].Name, one.Name)
		}
		if accounts.written[index].Kind != one.Kind {
			t.Errorf("the account %q is of the kind %q, want %q", one.Name, accounts.written[index].Kind, one.Kind)
		}
		if accounts.written[index].BusinessID != business.ID(77) {
			t.Errorf("the account %q belongs to the business %d, want %d",
				one.Name, accounts.written[index].BusinessID, 77)
		}
	}

	if opened.Business.ID != business.ID(77) {
		t.Errorf("the business is %d, want %d", opened.Business.ID, 77)
	}
	if len(opened.Accounts) != len(seed) {
		t.Errorf("the answer holds %d accounts, want %d", len(opened.Accounts), len(seed))
	}
}

// TestTheNameOfTheBusinessIsTakenInsideTheBusinessesOfItsOwner proves the
// rule of the domain: the name of a business is taken inside the businesses
// of its owner, and the refused call leaves nothing behind.
func TestTheNameOfTheBusinessIsTakenInsideTheBusinessesOfItsOwner(t *testing.T) {
	t.Parallel()

	book := &fakeOwnerBook{owned: []business.Business{
		{ID: 1, OwnerID: 1234567, Kind: business.KindPersonal, Name: business.DefaultPersonalName},
		{ID: 2, OwnerID: 1234567, Kind: business.KindBusiness, Name: "Panadería del barrio"},
	}}
	accounts := &fakeAccounts{}
	service := newBusinessService(book, accounts)

	_, err := service.Open(t.Context(), "1234567", "Panadería del barrio", "")
	if !errors.Is(err, business.ErrAlreadyExists) {
		t.Fatalf("the error is %v, want %v", err, business.ErrAlreadyExists)
	}

	if len(book.created) != 0 {
		t.Errorf("the business was opened: the storage holds %d businesses", len(book.created))
	}
	if len(accounts.written) != 0 {
		t.Errorf("the accounts were written: the storage holds %d accounts", len(accounts.written))
	}
}

// TestTheNameOfTheBusinessIsFreeInAnotherOwner proves that the name is
// taken inside the businesses of its owner, and not everywhere.
func TestTheNameOfTheBusinessIsFreeInAnotherOwner(t *testing.T) {
	t.Parallel()

	book := &fakeOwnerBook{}
	accounts := &fakeAccounts{}
	service := newBusinessService(book, accounts)

	if _, err := service.Open(t.Context(), "1234567", "Panadería del barrio", ""); err != nil {
		t.Fatalf("the business could not be opened: %v", err)
	}

	if len(book.created) != 1 {
		t.Errorf("the storage holds %d businesses, want 1", len(book.created))
	}
}

// TestTheBusinessWithoutANameIsRefused proves that a business without a name
// is refused, and that nothing is written.
func TestTheBusinessWithoutANameIsRefused(t *testing.T) {
	t.Parallel()

	book := &fakeOwnerBook{}
	accounts := &fakeAccounts{}
	service := newBusinessService(book, accounts)

	_, err := service.Open(t.Context(), "1234567", "   ", "")
	if !errors.Is(err, business.ErrInvalidName) {
		t.Fatalf("the error is %v, want %v", err, business.ErrInvalidName)
	}

	if len(book.created) != 0 {
		t.Errorf("the business was opened: the storage holds %d businesses", len(book.created))
	}
	if len(accounts.written) != 0 {
		t.Errorf("the accounts were written: the storage holds %d accounts", len(accounts.written))
	}
}

// TestTheBusinessOfADocumentThatIsNotANumberIsRefused proves that the
// document ID of the owner is required, and that nothing is written.
func TestTheBusinessOfADocumentThatIsNotANumberIsRefused(t *testing.T) {
	t.Parallel()

	book := &fakeOwnerBook{}
	accounts := &fakeAccounts{}
	service := newBusinessService(book, accounts)

	_, err := service.Open(t.Context(), "CC1234567", "Panadería del barrio", "")
	if !errors.Is(err, user.ErrInvalidID) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidID)
	}

	if len(book.created) != 0 {
		t.Errorf("the business was opened: the storage holds %d businesses", len(book.created))
	}
}

// TestTheBookOfTheBusinessIsWrittenWholeOrNotAtAll proves the rule of the
// domain: a book that starts with half of its accounts lies about what the
// owner has, so the business and its accounts are written in one
// transaction.
func TestTheBookOfTheBusinessIsWrittenWholeOrNotAtAll(t *testing.T) {
	t.Parallel()

	book := &fakeOwnerBook{}
	accounts := &fakeAccounts{err: account.ErrAlreadyExists}
	service := newBusinessService(book, accounts)

	_, err := service.Open(t.Context(), "1234567", "Panadería del barrio", "")
	if !errors.Is(err, account.ErrAlreadyExists) {
		t.Fatalf("the error is %v, want %v", err, account.ErrAlreadyExists)
	}

	if len(book.created) != 0 {
		t.Errorf("the business was left behind: the storage holds %d businesses", len(book.created))
	}
	if len(accounts.written) != 0 {
		t.Errorf("the accounts were left behind: the storage holds %d accounts", len(accounts.written))
	}
}

// TestTheBusinessesOfTheOwnerAreReadBack proves that the businesses of a
// person are read back with their kinds, and that the personal business is
// one of them.
func TestTheBusinessesOfTheOwnerAreReadBack(t *testing.T) {
	t.Parallel()

	book := &fakeOwnerBook{owned: []business.Business{
		{ID: 1, OwnerID: 1234567, Kind: business.KindPersonal, Name: business.DefaultPersonalName, Currency: business.DefaultCurrency},
		{ID: 2, OwnerID: 1234567, Kind: business.KindBusiness, Name: "Panadería del barrio", Currency: business.DefaultCurrency},
	}}
	accounts := &fakeAccounts{}
	service := newBusinessService(book, accounts)

	owned, err := service.List(t.Context(), "1234567")
	if err != nil {
		t.Fatalf("the businesses could not be read: %v", err)
	}

	if len(owned) != 2 {
		t.Fatalf("the answer holds %d businesses, want 2", len(owned))
	}
	if owned[0].Kind != business.KindPersonal {
		t.Errorf("the kind of the first business is %q, want %q", owned[0].Kind, business.KindPersonal)
	}
	if owned[1].Kind != business.KindBusiness {
		t.Errorf("the kind of the second business is %q, want %q", owned[1].Kind, business.KindBusiness)
	}
	if owned[1].Name != "Panadería del barrio" {
		t.Errorf("the name of the second business is %q, want %q", owned[1].Name, "Panadería del barrio")
	}
}

// TestTheOwnerWithoutABusinessHoldsNone proves that the list of the
// businesses of a person that owns none is empty, and not an error.
func TestTheOwnerWithoutABusinessHoldsNone(t *testing.T) {
	t.Parallel()

	book := &fakeOwnerBook{}
	accounts := &fakeAccounts{}
	service := newBusinessService(book, accounts)

	owned, err := service.List(t.Context(), "1234567")
	if err != nil {
		t.Fatalf("the businesses could not be read: %v", err)
	}

	if len(owned) != 0 {
		t.Errorf("the answer holds %d businesses, want none", len(owned))
	}
}

// TestTheBusinessesWithoutADatabaseSaySo proves that a service without a
// database says so, and it says it in both of its capabilities.
func TestTheBusinessesWithoutADatabaseSaySo(t *testing.T) {
	t.Parallel()

	service := application.NewBusinessService(nil, nil, nil)

	_, err := service.Open(t.Context(), "1234567", "Panadería del barrio", "")
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}

	_, err = service.List(t.Context(), "1234567")
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
