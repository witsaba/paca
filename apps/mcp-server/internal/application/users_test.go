package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/user"
)

// fakeRepository is the port of the storage, in memory.
//
// It remembers what it was asked to write, so the test can prove that the
// use case cleaned the data before it reached the storage.
type fakeRepository struct {
	created      []user.User
	updated      []user.Details
	movedTo      []user.Status
	observations []string
	err          error
}

// Create remembers the person and answers it.
func (r *fakeRepository) Create(_ context.Context, id user.ID, details user.Details) (user.User, error) {
	if r.err != nil {
		return user.User{}, r.err
	}

	created := user.User{
		ID:       id,
		FullName: details.FullName,
		Phone:    details.Phone,
		Email:    details.Email,
		Status:   user.StatusOnboarding,
	}
	r.created = append(r.created, created)

	return created, nil
}

// UpdateDetails remembers the data and answers it.
func (r *fakeRepository) UpdateDetails(_ context.Context, id user.ID, details user.Details) (user.User, error) {
	if r.err != nil {
		return user.User{}, r.err
	}

	r.updated = append(r.updated, details)

	return user.User{ID: id, FullName: details.FullName, Phone: details.Phone, Email: details.Email}, nil
}

// ChangeStatus remembers the move and answers it.
func (r *fakeRepository) ChangeStatus(_ context.Context, id user.ID, status user.Status, observation string) (user.User, error) {
	if r.err != nil {
		return user.User{}, r.err
	}

	r.movedTo = append(r.movedTo, status)
	r.observations = append(r.observations, observation)

	return user.User{ID: id, Status: status, StatusObservations: observation}, nil
}

func TestOnboardRegistersThePerson(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	service := application.NewUsersService(repository)

	created, err := service.Onboard(t.Context(), "1.234.567", "Ana María Pérez", "+57 321-253-9984", " ana@example.com ")
	if err != nil {
		t.Fatalf("the person was not registered: %v", err)
	}

	if created.ID != 1234567 {
		t.Errorf("the document is %d, want %d", created.ID, 1234567)
	}
	if len(repository.created) != 1 {
		t.Fatalf("the repository holds %d people, want 1", len(repository.created))
	}
	if repository.created[0].Phone != "573212539984" {
		t.Errorf("the phone is %q, want %q", repository.created[0].Phone, "573212539984")
	}
	if repository.created[0].Email != "ana@example.com" {
		t.Errorf("the email is %q, want %q", repository.created[0].Email, "ana@example.com")
	}
	if repository.created[0].Status != user.StatusOnboarding {
		t.Errorf("the status is %q, want %q", repository.created[0].Status, user.StatusOnboarding)
	}
}

func TestOnboardRefusesADocumentThatIsNotANumber(t *testing.T) {
	t.Parallel()

	service := application.NewUsersService(&fakeRepository{})

	_, err := service.Onboard(t.Context(), "CC1234567", "Ana", "", "")

	if !errors.Is(err, user.ErrInvalidID) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidID)
	}
}

func TestOnboardRefusesAPhoneThatIsNotInternational(t *testing.T) {
	t.Parallel()

	service := application.NewUsersService(&fakeRepository{})

	_, err := service.Onboard(t.Context(), "1234567", "Ana", "3212539984", "")

	if !errors.Is(err, user.ErrInvalidPhone) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidPhone)
	}
}

func TestOnboardRefusesAnEmailThatIsNotAnAddress(t *testing.T) {
	t.Parallel()

	service := application.NewUsersService(&fakeRepository{})

	_, err := service.Onboard(t.Context(), "1234567", "Ana", "", "not an address")

	if !errors.Is(err, user.ErrInvalidEmail) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidEmail)
	}
}

func TestTheUseCaseWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewUsersService(nil)

	if _, err := service.Onboard(t.Context(), "1234567", "", "", ""); !errors.Is(err, ports.ErrNoDatabase) {
		t.Errorf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
	if _, err := service.ChangeStatus(t.Context(), "1234567", "active", ""); !errors.Is(err, ports.ErrNoDatabase) {
		t.Errorf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
	if _, err := service.UpdateContactData(t.Context(), "1234567", "", "", ""); !errors.Is(err, ports.ErrNoDatabase) {
		t.Errorf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

func TestChangeStatusMovesTheUser(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	service := application.NewUsersService(repository)

	moved, err := service.ChangeStatus(t.Context(), "1234567", "inactive", "the person asked to leave")
	if err != nil {
		t.Fatalf("the user was not moved: %v", err)
	}

	if moved.Status != user.StatusInactive {
		t.Errorf("the status is %q, want %q", moved.Status, user.StatusInactive)
	}
	if len(repository.movedTo) != 1 || repository.movedTo[0] != user.StatusInactive {
		t.Errorf("the repository moved the user to %v, want [%q]", repository.movedTo, user.StatusInactive)
	}
	if repository.observations[0] != "the person asked to leave" {
		t.Errorf("the observation is %q, want %q", repository.observations[0], "the person asked to leave")
	}
}

func TestChangeStatusRefusesAStatusThatDoesNotExist(t *testing.T) {
	t.Parallel()

	service := application.NewUsersService(&fakeRepository{})

	_, err := service.ChangeStatus(t.Context(), "1234567", "unknown", "")

	if !errors.Is(err, user.ErrUnknownStatus) {
		t.Fatalf("the error is %v, want %v", err, user.ErrUnknownStatus)
	}
}

func TestUpdateContactDataWritesOnlyTheFieldsThatCarryAValue(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	service := application.NewUsersService(repository)

	updated, err := service.UpdateContactData(t.Context(), "1234567", "Ana María Pérez", "", "ana@example.com")
	if err != nil {
		t.Fatalf("the data was not written: %v", err)
	}

	if len(repository.updated) != 1 {
		t.Fatalf("the repository holds %d updates, want 1", len(repository.updated))
	}
	if repository.updated[0].FullName != "Ana María Pérez" {
		t.Errorf("the name is %q, want %q", repository.updated[0].FullName, "Ana María Pérez")
	}
	if repository.updated[0].Phone != "" {
		t.Errorf("the phone is %q, want it empty", repository.updated[0].Phone)
	}
	if repository.updated[0].Email != "ana@example.com" {
		t.Errorf("the email is %q, want %q", repository.updated[0].Email, "ana@example.com")
	}
	if updated.ID != 1234567 {
		t.Errorf("the document is %d, want %d", updated.ID, 1234567)
	}
}

func TestTheUseCaseAnswersTheErrorOfTheStorage(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{err: user.ErrAlreadyExists}
	service := application.NewUsersService(repository)

	_, err := service.Onboard(t.Context(), "1234567", "", "", "")

	if !errors.Is(err, user.ErrAlreadyExists) {
		t.Fatalf("the error is %v, want %v", err, user.ErrAlreadyExists)
	}
}

func TestTheUseCaseKeepsTheTimestampsOfTheStorage(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	service := application.NewUsersService(repository)

	created, err := service.Onboard(t.Context(), "1234567", "", "", "")
	if err != nil {
		t.Fatalf("the person was not registered: %v", err)
	}

	if !created.CreatedAt.Equal(time.Time{}) {
		t.Errorf("the moment of the creation is %v, want the zero time", created.CreatedAt)
	}
}
