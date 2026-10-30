//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/user"
)

// This file holds the container tests of the writing of the data of a
// person.

// TestUpdateDetailsWritesOnlyTheFieldsThatCarryAValue proves that the
// update writes the fields that carry a value and leaves the others as they
// were, and that the update time moves while the creation time does not.
func TestUpdateDetailsWritesOnlyTheFieldsThatCarryAValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)
	id := nextIdentifier()

	registered := registerUser(t, repo, id, "The Person Before The Update")

	updated, err := repo.UpdateDetails(ctx, user.ID(id), user.Details{FullName: "The Person After The Update"})
	if err != nil {
		t.Fatalf("the person could not be updated: %v", err)
	}

	if updated.FullName != "The Person After The Update" {
		t.Errorf("the name of the person is %q, want %q", updated.FullName, "The Person After The Update")
	}
	if updated.Phone != registered.Phone {
		t.Errorf("the phone of the person is %q, want the one it had: %q", updated.Phone, registered.Phone)
	}
	if updated.Email != registered.Email {
		t.Errorf("the address of the person is %q, want the one it had: %q", updated.Email, registered.Email)
	}
	if updated.Status != registered.Status {
		t.Errorf("the state of the person is %q, want the one it had: %q", updated.Status, registered.Status)
	}

	if !updated.CreatedAt.Equal(registered.CreatedAt) {
		t.Errorf("the creation time of the person moved from %s to %s, want it untouched", registered.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(registered.UpdatedAt) {
		t.Errorf("the update time of the person is %s, want it after %s", updated.UpdatedAt, registered.UpdatedAt)
	}

	// The row of the database holds the same facts.
	row := readRow(ctx, t, id)
	if row.fullName.String != "The Person After The Update" {
		t.Errorf("the name of the row is %q, want %q", row.fullName.String, "The Person After The Update")
	}
	if row.phone.String != registered.Phone {
		t.Errorf("the phone of the row is %q, want the one it had: %q", row.phone.String, registered.Phone)
	}
	if row.email.String != registered.Email {
		t.Errorf("the address of the row is %q, want the one it had: %q", row.email.String, registered.Email)
	}
	if !row.createdAt.Equal(registered.CreatedAt) {
		t.Errorf("the creation time of the row moved from %s to %s, want it untouched", registered.CreatedAt, row.createdAt)
	}
	if !row.updatedAt.After(registered.UpdatedAt) {
		t.Errorf("the update time of the row is %s, want it after %s", row.updatedAt, registered.UpdatedAt)
	}
}

// TestUpdateDetailsOfAMissingDocument proves that the update of a document
// nobody carries says that the person does not exist.
func TestUpdateDetailsOfAMissingDocument(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	_, err := repo.UpdateDetails(ctx, user.ID(nextIdentifier()), user.Details{FullName: "Nobody"})
	if !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, user.ErrNotFound)
	}
}

// TestUpdateDetailsWithAnEmailThatIsTaken proves that the update refuses an
// address that another user already carries, and that the row keeps the
// address it had.
func TestUpdateDetailsWithAnEmailThatIsTaken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	owner := nextIdentifier()
	registerUser(t, repo, owner, "The Owner Of The Address")

	id := nextIdentifier()
	registered := registerUser(t, repo, id, "The Person Of The Update")

	_, err := repo.UpdateDetails(ctx, user.ID(id), user.Details{Email: emailOf(owner)})
	if !errors.Is(err, user.ErrEmailTaken) {
		t.Fatalf("the error is %v, want %v", err, user.ErrEmailTaken)
	}

	row := readRow(ctx, t, id)
	if row.email.String != registered.Email {
		t.Errorf("the address of the row is %q, want the one it had: %q", row.email.String, registered.Email)
	}
}

// TestUpdateDetailsWithAPhoneThatIsTaken proves that the update refuses a
// phone number that another user already carries, and that the row keeps the
// number it had.
func TestUpdateDetailsWithAPhoneThatIsTaken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	owner := nextIdentifier()
	registerUser(t, repo, owner, "The Owner Of The Phone")

	id := nextIdentifier()
	registered := registerUser(t, repo, id, "The Person Of The Update")

	_, err := repo.UpdateDetails(ctx, user.ID(id), user.Details{Phone: phoneOf(owner)})
	if !errors.Is(err, user.ErrPhoneTaken) {
		t.Fatalf("the error is %v, want %v", err, user.ErrPhoneTaken)
	}

	row := readRow(ctx, t, id)
	if row.phone.String != registered.Phone {
		t.Errorf("the phone of the row is %q, want the one it had: %q", row.phone.String, registered.Phone)
	}
}

// TestChangeStatusFollowsTheMapOfTheStates proves that the change of the
// state moves the person the way the map of the domain allows, that the
// observation of the move is written for the closed states, and that the
// observation the person carried is cleared when it becomes active.
