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

// The fake of the storage of the people.
type fakeUsers struct {
	existing *user.User
	created  []user.User
	changed  []user.Status
	err      error
}

// Find answers the person that is there, or the error of the test.
func (f *fakeUsers) Find(_ context.Context, id user.ID) (user.User, error) {
	if f.err != nil {
		return user.User{}, f.err
	}
	if f.existing == nil {
		return user.User{}, user.ErrNotFound
	}

	return *f.existing, nil
}

// Create registers the person and remembers it.
func (f *fakeUsers) Create(_ context.Context, id user.ID, registration user.Registration) (user.User, error) {
	if f.err != nil {
		return user.User{}, f.err
	}

	created := user.User{
		ID:       id,
		FullName: registration.FullName,
		Phone:    registration.Phone,
		Email:    registration.Email,
		Status:   user.StatusOnboarding,
	}
	f.created = append(f.created, created)

	return created, nil
}

// ChangeStatus moves the person and remembers the move.
func (f *fakeUsers) ChangeStatus(_ context.Context, id user.ID, status user.Status, _ string) (user.User, error) {
	if f.err != nil {
		return user.User{}, f.err
	}

	f.changed = append(f.changed, status)

	return user.User{ID: id, FullName: "Ana María Pérez", Status: status}, nil
}

// UpdateDetails is not used by the onboarding.
func (f *fakeUsers) UpdateDetails(_ context.Context, id user.ID, _ user.Details) (user.User, error) {
	return user.User{ID: id}, nil
}

// List is not used by the onboarding.
func (f *fakeUsers) List(_ context.Context, _ int) ([]user.User, error) {
	return nil, nil
}

// The fake of the storage of the businesses.
type fakeBusinesses struct {
	existing *business.Business
	created  []business.Business
	err      error
	conflict bool
}

// Find answers the business of the identity.
func (f *fakeBusinesses) Find(_ context.Context, id business.ID) (business.Business, error) {
	if f.existing == nil {
		return business.Business{}, business.ErrNotFound
	}

	return *f.existing, nil
}

// FindPersonalByOwner answers the personal business of the owner, or the
// error that says there is none.
func (f *fakeBusinesses) FindPersonalByOwner(_ context.Context, owner user.ID) (business.Business, error) {
	if f.err != nil {
		return business.Business{}, f.err
	}
	if f.existing == nil {
		return business.Business{}, business.ErrNotFound
	}

	return *f.existing, nil
}

// Create writes the business and remembers it.
//
// The conflict is the race of two calls of the same person: the storage
// refuses the second one because the business of the first one is already
// there.
func (f *fakeBusinesses) Create(_ context.Context, owner user.ID, written business.Business) (business.Business, error) {
	if f.err != nil {
		return business.Business{}, f.err
	}
	if f.conflict {
		f.conflict = false

		winner := written
		winner.ID = 42
		winner.OwnerID = owner
		f.existing = &winner

		return business.Business{}, business.ErrAlreadyExists
	}

	created := written
	created.ID = business.ID(77)
	created.OwnerID = owner
	f.created = append(f.created, created)

	return created, nil
}

// The fake of the storage of the accounts.
type fakeAccounts struct {
	existing []account.Account
	written  []account.Account
	listed   int
	err      error
}

// Create writes the account and remembers it.
func (f *fakeAccounts) Create(_ context.Context, written account.Account) (account.Account, error) {
	if f.err != nil {
		return account.Account{}, f.err
	}

	f.written = append(f.written, written)

	return written, nil
}

// CreateMany writes the set of accounts and remembers it.
func (f *fakeAccounts) CreateMany(_ context.Context, accounts []account.Account) ([]account.Account, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.written = append(f.written, accounts...)

	return accounts, nil
}

// Find answers the account of the identity.
func (f *fakeAccounts) Find(_ context.Context, id account.ID) (account.Account, error) {
	return account.Account{ID: id}, nil
}

// List answers the accounts of the business.
func (f *fakeAccounts) List(_ context.Context, _ business.ID) ([]account.Account, error) {
	f.listed++

	return f.existing, nil
}

// The fake of the transaction of the storage.
//
// It keeps what the work wrote, and it gives it back when the work fails: a
// transaction that rolls back leaves the storage as it was.
type fakeWork struct {
	users    *fakeUsers
	book     *fakeBusinesses
	accounts *fakeAccounts
	failure  error
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakeWork) Do(ctx context.Context, work func(context.Context) error) error {
	users := len(f.users.created)
	book := len(f.book.created)
	accounts := len(f.accounts.written)

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.users.created = f.users.created[:users]
	f.book.created = f.book.created[:book]
	f.accounts.written = f.accounts.written[:accounts]

	return err
}

func TestTheOnboardingOpensTheBookOfThePerson(t *testing.T) {
	t.Parallel()

	users := &fakeUsers{}
	book := &fakeBusinesses{}
	accounts := &fakeAccounts{}
	work := &fakeWork{users: users, book: book, accounts: accounts}

	service := application.NewOnboardingService(users, book, accounts, work)

	completed, err := service.Complete(t.Context(), "1234567", "Ana María Pérez", "+57 321-253-9984", "ana@example.com")
	if err != nil {
		t.Fatalf("the onboarding could not be completed: %v", err)
	}

	if len(users.created) != 1 {
		t.Fatalf("the storage holds %d people, want 1", len(users.created))
	}
	if users.created[0].FullName != "Ana María Pérez" {
		t.Errorf("the name is %q, want %q", users.created[0].FullName, "Ana María Pérez")
	}
	if len(book.created) != 1 {
		t.Fatalf("the storage holds %d businesses, want 1", len(book.created))
	}
	if book.created[0].Kind != business.KindPersonal {
		t.Errorf("the kind of the business is %q, want %q", book.created[0].Kind, business.KindPersonal)
	}
	if len(accounts.written) != len(account.DefaultPersonalAccounts()) {
		t.Errorf("the storage holds %d accounts, want %d", len(accounts.written), len(account.DefaultPersonalAccounts()))
	}
	if len(users.changed) != 1 || users.changed[0] != user.StatusActive {
		t.Errorf("the moves of the person are %v, want one move to %q", users.changed, user.StatusActive)
	}
	if completed.User.Status != user.StatusActive {
		t.Errorf("the state of the person is %q, want %q", completed.User.Status, user.StatusActive)
	}
	if completed.Business.ID == 0 {
		t.Error("the business has no identity")
	}
	if len(completed.Accounts) != len(account.DefaultPersonalAccounts()) {
		t.Errorf("the answer holds %d accounts, want %d", len(completed.Accounts), len(account.DefaultPersonalAccounts()))
	}
}

func TestTheOnboardingCompletesThePersonThatIsAlreadyRegistered(t *testing.T) {
	t.Parallel()

	existing := user.User{ID: 1234567, FullName: "Ana María Pérez", Status: user.StatusOnboarding}
	users := &fakeUsers{existing: &existing}
	book := &fakeBusinesses{}
	accounts := &fakeAccounts{}
	work := &fakeWork{users: users, book: book, accounts: accounts}

	service := application.NewOnboardingService(users, book, accounts, work)

	if _, err := service.Complete(t.Context(), "1234567", "Ana María Pérez", "", ""); err != nil {
		t.Fatalf("the onboarding could not be completed: %v", err)
	}

	if len(users.created) != 0 {
		t.Errorf("the person was registered again: the storage holds %d people", len(users.created))
	}
	if len(book.created) != 1 {
		t.Errorf("the storage holds %d businesses, want 1", len(book.created))
	}
}

func TestTheOnboardingKeepsTheBusinessThatIsAlreadyOpen(t *testing.T) {
	t.Parallel()

	existing := user.User{ID: 1234567, FullName: "Ana María Pérez", Status: user.StatusActive}
	opened := business.Business{ID: 42, OwnerID: 1234567, Kind: business.KindPersonal, Name: "Negocio Personal"}
	users := &fakeUsers{existing: &existing}
	book := &fakeBusinesses{existing: &opened}
	accounts := &fakeAccounts{existing: []account.Account{{ID: 1, Name: "Efectivo"}}}
	work := &fakeWork{users: users, book: book, accounts: accounts}

	service := application.NewOnboardingService(users, book, accounts, work)

	completed, err := service.Complete(t.Context(), "1234567", "Ana María Pérez", "", "")
	if err != nil {
		t.Fatalf("the onboarding could not be completed: %v", err)
	}

	if len(book.created) != 0 {
		t.Errorf("the business was opened again: the storage holds %d businesses", len(book.created))
	}
	if len(accounts.written) != 0 {
		t.Errorf("the accounts were written again: the storage holds %d accounts", len(accounts.written))
	}
	if accounts.listed != 1 {
		t.Errorf("the accounts were read %d times, want 1", accounts.listed)
	}
	if completed.Business.ID != opened.ID {
		t.Errorf("the business is %d, want %d", completed.Business.ID, opened.ID)
	}
}

func TestTheOnboardingAdoptsTheBusinessThatAnotherCallOpened(t *testing.T) {
	t.Parallel()

	// Two calls of the same person arrive together: the storage refuses
	// the second one because the business of the first one is already
	// there. The onboarding completes with what the winner wrote.
	existing := user.User{ID: 1234567, FullName: "Ana María Pérez", Status: user.StatusOnboarding}
	users := &fakeUsers{existing: &existing}
	book := &fakeBusinesses{conflict: true}
	accounts := &fakeAccounts{existing: []account.Account{{ID: 1, Name: "Efectivo"}}}
	work := &fakeWork{users: users, book: book, accounts: accounts}

	service := application.NewOnboardingService(users, book, accounts, work)

	completed, err := service.Complete(t.Context(), "1234567", "Ana María Pérez", "", "")
	if err != nil {
		t.Fatalf("the onboarding could not be completed: %v", err)
	}

	if len(book.created) != 0 {
		t.Errorf("the business was opened again: the storage holds %d businesses", len(book.created))
	}
	if len(accounts.written) != 0 {
		t.Errorf("the accounts were written again: the storage holds %d accounts", len(accounts.written))
	}
	if completed.Business.ID != 42 {
		t.Errorf("the business is %d, want 42", completed.Business.ID)
	}
	if completed.User.Status != user.StatusActive {
		t.Errorf("the state of the person is %q, want %q", completed.User.Status, user.StatusActive)
	}
}

func TestTheOnboardingRefusesARegistrationWithoutAName(t *testing.T) {
	t.Parallel()

	users := &fakeUsers{}
	book := &fakeBusinesses{}
	accounts := &fakeAccounts{}
	work := &fakeWork{users: users, book: book, accounts: accounts}

	service := application.NewOnboardingService(users, book, accounts, work)

	_, err := service.Complete(t.Context(), "1234567", "   ", "", "")
	if !errors.Is(err, user.ErrNameRequired) {
		t.Fatalf("the error is %v, want %v", err, user.ErrNameRequired)
	}
	if len(users.created) != 0 {
		t.Errorf("the person was registered: the storage holds %d people", len(users.created))
	}
}

func TestTheOnboardingRefusesADocumentThatIsNotANumber(t *testing.T) {
	t.Parallel()

	users := &fakeUsers{}
	book := &fakeBusinesses{}
	accounts := &fakeAccounts{}
	work := &fakeWork{users: users, book: book, accounts: accounts}

	service := application.NewOnboardingService(users, book, accounts, work)

	_, err := service.Complete(t.Context(), "CC1234567", "Ana", "", "")
	if !errors.Is(err, user.ErrInvalidID) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidID)
	}
}

func TestTheOnboardingWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewOnboardingService(nil, nil, nil, nil)

	_, err := service.Complete(t.Context(), "1234567", "Ana", "", "")
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

func TestTheOnboardingLeavesNothingBehindWhenItFails(t *testing.T) {
	t.Parallel()

	users := &fakeUsers{}
	book := &fakeBusinesses{}
	accounts := &fakeAccounts{}
	work := &fakeWork{users: users, book: book, accounts: accounts, failure: account.ErrAlreadyExists}
	accounts.err = account.ErrAlreadyExists

	service := application.NewOnboardingService(users, book, accounts, work)

	_, err := service.Complete(t.Context(), "1234567", "Ana", "", "")
	if !errors.Is(err, account.ErrAlreadyExists) {
		t.Fatalf("the error is %v, want %v", err, account.ErrAlreadyExists)
	}

	if len(users.created) != 0 {
		t.Errorf("the person was registered: the storage holds %d people", len(users.created))
	}
	if len(book.created) != 0 {
		t.Errorf("the business was opened: the storage holds %d businesses", len(book.created))
	}
	if len(accounts.written) != 0 {
		t.Errorf("the accounts were written: the storage holds %d accounts", len(accounts.written))
	}
}
