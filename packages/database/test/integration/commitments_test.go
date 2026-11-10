//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/commitment"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/user"
)

// commitmentsRepository returns the repository of the commitments, bound to
// the database of the suite.
func commitmentsRepository(t *testing.T) *postgres.CommitmentRepository {
	t.Helper()

	repo, err := postgres.NewCommitmentRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the commitments could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheCommitments opens the business that owes the promises of the
// test.
func theBakeryOfTheCommitments(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Commitments")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner),
		businessOf(business.KindBusiness, "La Panadería De Las Promesas"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theDayOfTheCommitments reads a day of the book of the test.
func theDayOfTheCommitments(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day of the commitment could not be read: %v", err)
	}

	return parsed
}

// thePromiseOfTheTest builds the promise the tests of the table write.
func thePromiseOfTheTest(t *testing.T, own business.ID, what, dueOn string) commitment.Commitment {
	t.Helper()

	built, err := commitment.New(own, what, theDayOfTheCommitments(t, dueOn))
	if err != nil {
		t.Fatalf("the commitment was refused: %v", err)
	}

	return built
}

// thePromiseOf reads one promise of the list by its identity.
func thePromiseOf(t *testing.T, held []commitment.Commitment, id commitment.ID) commitment.Commitment {
	t.Helper()

	for _, one := range held {
		if one.ID == id {
			return one
		}
	}

	t.Fatalf("the business holds no promise with the identity %d", id)

	return commitment.Commitment{}
}

// TestThePromiseIsKeptAndReadBackOpen proves the rules of the table against a
// real database: the promise is written with what is due and the day it is
// due, it is read back as it was written, and it is OPEN — the day of meeting
// is the truth of the state, and the row carries none.
func TestThePromiseIsKeptAndReadBackOpen(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)
	bakery := theBakeryOfTheCommitments(t, ctx)

	written, err := repo.Create(ctx, thePromiseOfTheTest(t, bakery.ID, "El pedido de la tienda", "2026-03-14"))
	if err != nil {
		t.Fatalf("the promise could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Fatal("the promise carries no identity: the storage generates it")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.What != "El pedido de la tienda" {
		t.Errorf("the what is %q, want %q", written.What, "El pedido de la tienda")
	}
	if written.DueOn.String() != "2026-03-14" {
		t.Errorf("the day it is due is %s, want 2026-03-14", written.DueOn)
	}
	if !written.MetOn.Time().IsZero() {
		t.Errorf("the day it was met is %s, want none", written.MetOn)
	}
	if !written.IsOpen() {
		t.Error("the promise that was just written is not open")
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the promises could not be read: %v", err)
	}

	read := thePromiseOf(t, held, written.ID)
	if read.What != written.What {
		t.Errorf("the what read back is %q, want %q", read.What, written.What)
	}
	if read.DueOn.String() != "2026-03-14" {
		t.Errorf("the day it is due, read back, is %s, want 2026-03-14", read.DueOn)
	}
	if !read.IsOpen() {
		t.Error("the promise read back is not open, and the book carries no day of meeting")
	}
}

// TestThePromiseThatWasMetKeepsTheDayItWasMet proves the rule of the table:
// the promise that was met is KEPT, with the day it was met, and it is not
// owed any more. There is no state column that could say otherwise.
func TestThePromiseThatWasMetKeepsTheDayItWasMet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)
	bakery := theBakeryOfTheCommitments(t, ctx)

	written, err := repo.Create(ctx, thePromiseOfTheTest(t, bakery.ID, "El pan del domingo", "2026-03-14"))
	if err != nil {
		t.Fatalf("the promise could not be written: %v", err)
	}

	met, err := written.Met(theDayOfTheCommitments(t, "2026-03-13"))
	if err != nil {
		t.Fatalf("the meeting was refused: %v", err)
	}

	kept, err := repo.Meet(ctx, met)
	if err != nil {
		t.Fatalf("the promise could not be met: %v", err)
	}

	if kept.ID != written.ID {
		t.Errorf("the identity is %d, want %d: the promise that was kept is the same promise", kept.ID, written.ID)
	}
	if kept.MetOn.String() != "2026-03-13" {
		t.Errorf("the day it was met is %s, want 2026-03-13", kept.MetOn)
	}
	if kept.IsOpen() {
		t.Error("the promise that was met is still owed")
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the promises could not be read: %v", err)
	}

	read := thePromiseOf(t, held, written.ID)
	if read.MetOn.String() != "2026-03-13" {
		t.Errorf("the day it was met, read back, is %s, want 2026-03-13", read.MetOn)
	}
	if read.What != "El pan del domingo" {
		t.Errorf("the what read back is %q, want %q: the promise that was met is kept whole", read.What, "El pan del domingo")
	}
}

// TestTheSecondMeetingOfThePromiseIsRefused proves the rule of the table: a
// promise is kept once. The second meeting is refused, the day the book holds
// is not hidden, and the promise is still there.
func TestTheSecondMeetingOfThePromiseIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)
	bakery := theBakeryOfTheCommitments(t, ctx)

	written, err := repo.Create(ctx, thePromiseOfTheTest(t, bakery.ID, "El pan del domingo", "2026-03-14"))
	if err != nil {
		t.Fatalf("the promise could not be written: %v", err)
	}

	met, err := written.Met(theDayOfTheCommitments(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the meeting was refused: %v", err)
	}

	if _, err := repo.Meet(ctx, met); err != nil {
		t.Fatalf("the promise could not be met: %v", err)
	}

	// The caller that bypasses the domain writes the second meeting
	// straight into the port: the table is what refuses it.
	again := met
	again.MetOn = theDayOfTheCommitments(t, "2026-03-15")

	kept, err := repo.Meet(ctx, again)
	if !errors.Is(err, commitment.ErrAlreadyMet) {
		t.Fatalf("the second meeting answered %v (%+v), want %v", err, kept, commitment.ErrAlreadyMet)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the promises could not be read: %v", err)
	}

	read := thePromiseOf(t, held, written.ID)
	if read.MetOn.String() != "2026-03-14" {
		t.Errorf("the day it was met is %s, want 2026-03-14: the book never hides the day it was kept", read.MetOn)
	}
}

// TestAPromiseIsNeverDeleted proves the rule of the table: the promise that
// was kept stays in the book, and the meeting of a promise never removes it.
//
// The promise is read back after it was met and after the second meeting was
// refused: the book holds it, with its own identity, both times.
func TestAPromiseIsNeverDeleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)
	bakery := theBakeryOfTheCommitments(t, ctx)

	written, err := repo.Create(ctx, thePromiseOfTheTest(t, bakery.ID, "El pedido de la tienda", "2026-03-14"))
	if err != nil {
		t.Fatalf("the promise could not be written: %v", err)
	}

	met, err := written.Met(theDayOfTheCommitments(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the meeting was refused: %v", err)
	}

	if _, err := repo.Meet(ctx, met); err != nil {
		t.Fatalf("the promise could not be met: %v", err)
	}

	if _, err := repo.Meet(ctx, met); !errors.Is(err, commitment.ErrAlreadyMet) {
		t.Fatalf("the second meeting answered %v, want %v", err, commitment.ErrAlreadyMet)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the promises could not be read: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the business holds %d promises, want 1: a promise is never deleted", len(held))
	}

	read := thePromiseOf(t, held, written.ID)
	if read.What != "El pedido de la tienda" {
		t.Errorf("the what read back is %q, want %q", read.What, "El pedido de la tienda")
	}
	if read.MetOn.String() != "2026-03-14" {
		t.Errorf("the day it was met, read back, is %s, want 2026-03-14", read.MetOn)
	}
}

// TestTwoPromisesThatSayTheSameThingAreTwoPromises proves the rule of the
// table: the identity of a promise is its own identity, never its text. Two
// promises that say the same thing on the same day are two promises, and the
// book never refuses them and never merges them.
func TestTwoPromisesThatSayTheSameThingAreTwoPromises(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)
	bakery := theBakeryOfTheCommitments(t, ctx)

	first, err := repo.Create(ctx, thePromiseOfTheTest(t, bakery.ID, "El pan del domingo", "2026-03-14"))
	if err != nil {
		t.Fatalf("the first promise could not be written: %v", err)
	}

	second, err := repo.Create(ctx, thePromiseOfTheTest(t, bakery.ID, "El pan del domingo", "2026-03-14"))
	if err != nil {
		t.Fatalf("the second promise was refused: %v", err)
	}

	if first.ID == second.ID {
		t.Fatalf("the two promises carry the identity %d: the book merged them", first.ID)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the promises could not be read: %v", err)
	}
	if len(held) != 2 {
		t.Fatalf("the business holds %d promises, want 2: the book never merges two promises", len(held))
	}
}

// TestThePromisesOfAnotherBusinessAreNotListed proves the rule of the table:
// the promises of a business are the promises of that business and of no
// other one.
func TestThePromisesOfAnotherBusinessAreNotListed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)
	bakery := theBakeryOfTheCommitments(t, ctx)
	other := theBakeryOfTheCommitments(t, ctx)

	if _, err := repo.Create(ctx, thePromiseOfTheTest(t, bakery.ID, "El pan del domingo", "2026-03-14")); err != nil {
		t.Fatalf("the promise could not be written: %v", err)
	}

	held, err := repo.List(ctx, other.ID)
	if err != nil {
		t.Fatalf("the promises could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Fatalf("the other business holds %d promises, want none", len(held))
	}
}

// TestTheCommitmentOfABusinessThatIsNotThereIsRefused proves the rule of the
// table that the foreign key holds: a promise belongs to a business that is
// there, and the adapter says it with the error of the domain.
func TestTheCommitmentOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)

	// The identity of the business is one the suite never writes: the
	// businesses the tests create are the ones of the counter of the
	// database, and this one is far from it.
	_, err := repo.Create(ctx, thePromiseOfTheTest(t, business.ID(999_999_999), "El pan del domingo", "2026-03-14"))
	if !errors.Is(err, commitment.ErrBusinessNotFound) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrBusinessNotFound)
	}
}

// TestTheTableRefusesThePromiseThatSaysNothing proves the rule of the table
// that the check holds, as the defense in depth of the domain: a promise that
// says nothing about what is due is refused by the table too.
func TestTheTableRefusesThePromiseThatSaysNothing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)
	bakery := theBakeryOfTheCommitments(t, ctx)

	// The promise is written straight into the port, with the spaces that
	// the domain would have taken away: the table is what refuses it.
	broken := commitment.Commitment{
		BusinessID: bakery.ID,
		What:       "   ",
		DueOn:      theDayOfTheCommitments(t, "2026-03-14"),
	}

	if _, err := repo.Create(ctx, broken); !errors.Is(err, commitment.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrInvalidValue)
	}
}

// TestTheMeetingOfAPromiseWithNoDayIsRefusedByTheTable proves the rule of
// the table that the meeting holds as its defense in depth: a promise that
// was kept stays in the book WITH the day it was kept, so a meeting that
// carries no day matches no row and writes none.
//
// The caller here bypasses the domain and the use case, which refuse the
// meeting first: the table is what refuses it, and the adapter answers the
// refusal of the rule and never "already met" nor "not found".
//
// The promise is read back open and untouched: a meeting that wrote nothing
// bumps neither the day of meeting nor the moment the row was written.
func TestTheMeetingWithNoDayIsRefusedByTheTable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)
	bakery := theBakeryOfTheCommitments(t, ctx)

	written, err := repo.Create(ctx, thePromiseOfTheTest(t, bakery.ID, "El pan del domingo", "2026-03-14"))
	if err != nil {
		t.Fatalf("the promise could not be written: %v", err)
	}

	noDay := commitment.Commitment{ID: written.ID, BusinessID: written.BusinessID, What: written.What, DueOn: written.DueOn}

	// The answer of the adapter is the refusal of the rule and no other
	// one: not "already met" and not "not found".
	kept, err := repo.Meet(ctx, noDay)
	if !errors.Is(err, commitment.ErrMissingMetDay) {
		t.Fatalf("the meeting with no day answered %v and the promise %+v, want %v",
			err, kept, commitment.ErrMissingMetDay)
	}
	if errors.Is(err, commitment.ErrAlreadyMet) || errors.Is(err, commitment.ErrNotFound) {
		t.Errorf("the meeting with no day answered %v: it is neither a promise that is not there nor one that was met", err)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the promises could not be read: %v", err)
	}

	read := thePromiseOf(t, held, written.ID)
	if !read.IsOpen() {
		t.Error("the promise was met with no day, and the book says it was kept")
	}
	if !read.UpdatedAt.Equal(written.UpdatedAt) {
		t.Errorf("the moment the row was written is %s and was %s: the meeting with no day wrote a row",
			read.UpdatedAt, written.UpdatedAt)
	}
}

// TestTheMeetingOfAPromiseThatIsNotThereIsRefused proves the rule of the
// table: the promise that is not there is not met, and the adapter tells it
// apart from the promise that was met already.
func TestTheMeetingOfAPromiseThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := commitmentsRepository(t)
	bakery := theBakeryOfTheCommitments(t, ctx)

	missing := commitment.Commitment{
		ID:         commitment.ID(999_999_999),
		BusinessID: bakery.ID,
		What:       "El pan del domingo",
		DueOn:      theDayOfTheCommitments(t, "2026-03-14"),
		MetOn:      theDayOfTheCommitments(t, "2026-03-14"),
	}

	if _, err := repo.Meet(ctx, missing); !errors.Is(err, commitment.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrNotFound)
	}
}
