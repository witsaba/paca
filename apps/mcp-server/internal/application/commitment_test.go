package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/commitment"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// fakeCommitments is the fake of the storage of the commitments.
//
// It remembers the promises it holds and the ones it wrote, and it answers
// what the test tells it to answer.
type fakeCommitments struct {
	held    []commitment.Commitment
	written []commitment.Commitment
	met     []commitment.Commitment
	err     error
	listed  business.ID
}

// Create writes the promise and remembers it.
func (f *fakeCommitments) Create(_ context.Context, c commitment.Commitment) (commitment.Commitment, error) {
	if f.err != nil {
		return commitment.Commitment{}, f.err
	}

	c.ID = commitment.ID(len(f.written) + 100)
	f.written = append(f.written, c)

	return c, nil
}

// Meet writes the meeting and remembers it.
func (f *fakeCommitments) Meet(_ context.Context, met commitment.Commitment) (commitment.Commitment, error) {
	if f.err != nil {
		return commitment.Commitment{}, f.err
	}

	f.met = append(f.met, met)

	return met, nil
}

// List answers the promises of the business, and remembers that it was
// asked.
func (f *fakeCommitments) List(_ context.Context, own business.ID) ([]commitment.Commitment, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]commitment.Commitment, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// theDayOfTheCommitments reads a day of the book of the tests.
func theDayOfTheCommitments(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day %q could not be read: %v", day, err)
	}

	return parsed
}

// thePromiseOfTheCommitments is the promise the tests read back from the
// storage: open, and due on a day.
func thePromiseOfTheCommitments(t *testing.T, id commitment.ID, what, dueOn string) commitment.Commitment {
	t.Helper()

	built, err := commitment.New(business.ID(77), what, theDayOfTheCommitments(t, dueOn))
	if err != nil {
		t.Fatalf("the commitment was refused: %v", err)
	}

	built.ID = id

	return built
}

// theCommitmentService wires the use case with the fakes of the test.
func theCommitmentService(commitments *fakeCommitments, existing *business.Business) *application.CommitmentService {
	return application.NewCommitmentService(commitments, &fakeBusinesses{existing: existing})
}

// TestThePromiseIsWrittenWithTheWordsOfTheOwner proves that the use case
// writes the promise of the business: what is due, in the words of the
// owner, and the day it is due.
//
// The promise is written OPEN: it carries no day of meeting, because nobody
// said it was met.
func TestThePromiseIsWrittenWithTheWordsOfTheOwner(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{}
	service := theCommitmentService(commitments, &bakery)

	written, err := service.Record(t.Context(), ports.RecordCommitment{
		BusinessID: 77,
		What:       "  El pedido de la tienda de la esquina  ",
		DueOn:      "2026-03-14",
	})
	if err != nil {
		t.Fatalf("the promise was refused: %v", err)
	}

	if len(commitments.written) != 1 {
		t.Fatalf("the storage holds %d promises, want 1", len(commitments.written))
	}

	stored := commitments.written[0]
	if stored.BusinessID != bakery.ID {
		t.Errorf("the business is %d, want %d", stored.BusinessID, bakery.ID)
	}
	if stored.What != "El pedido de la tienda de la esquina" {
		t.Errorf("the what is %q, want the words of the owner without the spaces around them", stored.What)
	}
	if stored.DueOn.String() != "2026-03-14" {
		t.Errorf("the day it is due is %s, want 2026-03-14", stored.DueOn)
	}
	if !stored.MetOn.Time().IsZero() {
		t.Errorf("the day it was met is %s, want none: the promise is written open", stored.MetOn)
	}
	if written.ID == 0 {
		t.Error("the answer carries no identity: the identity is the one the storage generated")
	}
}

// TestThePromiseWithoutADayIsRefusedByTheDomain proves the rule of the use
// case: it never decides a rule. The day the agent names none of travels as
// the empty day, and the domain refuses it in its own words.
//
// The observable difference is the one the rule has: the use case ASKS THE
// DOMAIN, so it reads the business before the refusal — a use case that
// decided the rule by itself would refuse the day without ever asking.
func TestThePromiseWithoutADayIsRefusedByTheDomain(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{}
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewCommitmentService(commitments, businesses)

	_, err := service.Record(t.Context(), ports.RecordCommitment{
		BusinessID: 77,
		What:       "El pedido de la tienda",
	})
	if !errors.Is(err, commitment.ErrMissingDay) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrMissingDay)
	}

	// The refusal is the one of the domain, with the words of the domain.
	_, refusal := commitment.New(bakery.ID, "El pedido de la tienda", ledger.Date{})
	if err.Error() != refusal.Error() {
		t.Errorf("the error is %q, want the one of the domain %q", err.Error(), refusal.Error())
	}

	if businesses.found != bakery.ID {
		t.Errorf("the business read is %d, want %d: the use case asks the domain", businesses.found, bakery.ID)
	}
	if len(commitments.written) != 0 {
		t.Errorf("the storage holds %d promises, want none", len(commitments.written))
	}
}

// TestThePromiseThatSaysNothingIsRefusedByTheDomain proves that the rule of
// the text of the promise is the rule of the domain: the promise that says
// nothing about what is due never enters the book.
func TestThePromiseThatSaysNothingIsRefusedByTheDomain(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{}
	service := theCommitmentService(commitments, &bakery)

	_, err := service.Record(t.Context(), ports.RecordCommitment{
		BusinessID: 77,
		What:       "   ",
		DueOn:      "2026-03-14",
	})
	if !errors.Is(err, commitment.ErrMissingWhat) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrMissingWhat)
	}
	if len(commitments.written) != 0 {
		t.Errorf("the storage holds %d promises, want none", len(commitments.written))
	}
}

// TestThePromiseOfABusinessThatDoesNotHoldPromisesIsRefused proves that the
// promise belongs to a business of its own: the Personal Business of a
// person holds the money of that person, and not the promises of the bakery.
func TestThePromiseOfABusinessThatDoesNotHoldPromisesIsRefused(t *testing.T) {
	t.Parallel()

	personal := business.Business{
		ID:       business.ID(77),
		Kind:     business.KindPersonal,
		Name:     "Ana Pérez",
		Currency: "COP",
	}
	commitments := &fakeCommitments{}
	service := theCommitmentService(commitments, &personal)

	_, err := service.Record(t.Context(), ports.RecordCommitment{
		BusinessID: 77,
		What:       "El pedido de la tienda",
		DueOn:      "2026-03-14",
	})
	if !errors.Is(err, commitment.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrWrongBusiness)
	}
	if len(commitments.written) != 0 {
		t.Errorf("the storage holds %d promises, want none", len(commitments.written))
	}
}

// TestThePromiseOfABusinessThatIsNotThereIsRefused proves that the business
// of the promise that is not there has its own refusal: the agent is told
// about the business of the promise, and never about the personal business
// of a person nor about the onboarding.
func TestThePromiseOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	commitments := &fakeCommitments{}
	service := theCommitmentService(commitments, nil)

	_, err := service.Record(t.Context(), ports.RecordCommitment{
		BusinessID: 77,
		What:       "El pedido de la tienda",
		DueOn:      "2026-03-14",
	})
	if !errors.Is(err, commitment.ErrBusinessNotFound) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrBusinessNotFound)
	}
	if len(commitments.written) != 0 {
		t.Errorf("the storage holds %d promises, want none", len(commitments.written))
	}
}

// TestThePromiseWithoutAStorageIsRefused proves that the use case with no
// database behind it says so, and writes nothing.
func TestThePromiseWithoutAStorageIsRefused(t *testing.T) {
	t.Parallel()

	service := application.NewCommitmentService(nil, nil)

	_, err := service.Record(t.Context(), ports.RecordCommitment{
		BusinessID: 77,
		What:       "El pedido de la tienda",
		DueOn:      "2026-03-14",
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// TestTheErrorOfTheStorageReachesTheCaller proves that the use case maps the
// error of the storage to the error of the domain: it never invents a
// failure of its own, and the agent reads the rule of the book.
func TestTheErrorOfTheStorageReachesTheCaller(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{err: commitment.ErrNoStorage}
	service := theCommitmentService(commitments, &bakery)

	_, err := service.Record(t.Context(), ports.RecordCommitment{
		BusinessID: 77,
		What:       "El pedido de la tienda",
		DueOn:      "2026-03-14",
	})
	if !errors.Is(err, commitment.ErrNoStorage) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrNoStorage)
	}
}

// TestThePromiseThatWasKeptIsWrittenWithTheDayItWasKept proves that the
// meeting of a promise is a fact of the book: the promise that was met is
// kept, with the day it was met.
func TestThePromiseThatWasKeptIsWrittenWithTheDayItWasKept(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{
		held: []commitment.Commitment{
			thePromiseOfTheCommitments(t, commitment.ID(9), "El pan del domingo", "2026-03-14"),
		},
	}
	service := theCommitmentService(commitments, &bakery)

	kept, err := service.Meet(t.Context(), ports.MeetCommitment{
		BusinessID:   77,
		CommitmentID: 9,
		MetOn:        "2026-03-13",
	})
	if err != nil {
		t.Fatalf("the meeting was refused: %v", err)
	}

	if len(commitments.met) != 1 {
		t.Fatalf("the storage holds %d meetings, want 1", len(commitments.met))
	}

	stored := commitments.met[0]
	if stored.ID != commitment.ID(9) {
		t.Errorf("the identity is %d, want 9: the promise is named by its own identity", stored.ID)
	}
	if stored.MetOn.String() != "2026-03-13" {
		t.Errorf("the day it was met is %s, want 2026-03-13", stored.MetOn)
	}
	if kept.IsOpen() {
		t.Error("the promise that was kept is still owed")
	}
}

// TestTheSecondMeetingOfThePromiseIsRefusedByTheDomain proves that a
// promise is kept once, and that the rule is the one of the domain: the use
// case reads the promise and asks the domain, and nothing is written.
func TestTheSecondMeetingOfThePromiseIsRefusedByTheDomain(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	met, err := thePromiseOfTheCommitments(t, commitment.ID(9), "El pan del domingo", "2026-03-14").
		Met(theDayOfTheCommitments(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the meeting of the test was refused: %v", err)
	}

	commitments := &fakeCommitments{held: []commitment.Commitment{met}}
	service := theCommitmentService(commitments, &bakery)

	_, err = service.Meet(t.Context(), ports.MeetCommitment{
		BusinessID:   77,
		CommitmentID: 9,
		MetOn:        "2026-03-15",
	})
	if !errors.Is(err, commitment.ErrAlreadyMet) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrAlreadyMet)
	}
	if commitments.listed != bakery.ID {
		t.Errorf("the promises of the business were not read: the use case asks the domain with the promise")
	}
	if len(commitments.met) != 0 {
		t.Errorf("the storage holds %d meetings, want none", len(commitments.met))
	}
}

// TestTheMeetingWithoutItsDayIsRefusedByTheDomain proves that the promise
// that was kept stays in the book with the day it was kept: a meeting with
// no day never enters the book, and the promise does not look open for ever.
func TestTheMeetingWithoutItsDayIsRefusedByTheDomain(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	promise := thePromiseOfTheCommitments(t, commitment.ID(9), "El pan del domingo", "2026-03-14")
	commitments := &fakeCommitments{held: []commitment.Commitment{promise}}
	service := theCommitmentService(commitments, &bakery)

	_, err := service.Meet(t.Context(), ports.MeetCommitment{
		BusinessID:   77,
		CommitmentID: 9,
	})
	if !errors.Is(err, commitment.ErrMissingMetDay) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrMissingMetDay)
	}

	_, refusal := promise.Met(ledger.Date{})
	if err.Error() != refusal.Error() {
		t.Errorf("the error is %q, want the one of the domain %q", err.Error(), refusal.Error())
	}
	if len(commitments.met) != 0 {
		t.Errorf("the storage holds %d meetings, want none", len(commitments.met))
	}
}

// TestTheMeetingOfAPromiseThatIsNotThereIsRefused proves that a promise of
// another business is not the promise the agent named: the promises of the
// business are read, and the one that is not in them is not met.
func TestTheMeetingOfAPromiseThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{
		held: []commitment.Commitment{
			thePromiseOfTheCommitments(t, commitment.ID(9), "El pan del domingo", "2026-03-14"),
		},
	}
	service := theCommitmentService(commitments, &bakery)

	_, err := service.Meet(t.Context(), ports.MeetCommitment{
		BusinessID:   77,
		CommitmentID: 10,
		MetOn:        "2026-03-14",
	})
	if !errors.Is(err, commitment.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrNotFound)
	}
	if len(commitments.met) != 0 {
		t.Errorf("the storage holds %d meetings, want none", len(commitments.met))
	}
}

// TestTheCalendarOfTheDayAnswersWhatIsDueAndWhatIsOwed proves that the
// reading answers the calendar: the promises that are due on the day, and
// every open promise of the business.
func TestTheCalendarOfTheDayAnswersWhatIsDueAndWhatIsOwed(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()

	overdue := thePromiseOfTheCommitments(t, commitment.ID(8), "El pan del domingo", "2026-03-07")
	met, err := overdue.Met(theDayOfTheCommitments(t, "2026-03-07"))
	if err != nil {
		t.Fatalf("the meeting of the test was refused: %v", err)
	}

	due := thePromiseOfTheCommitments(t, commitment.ID(9), "El pedido de la tienda", "2026-03-14")

	commitments := &fakeCommitments{held: []commitment.Commitment{met, due}}
	service := theCommitmentService(commitments, &bakery)

	calendar, err := service.Read(t.Context(), ports.ReadCommitments{
		BusinessID: 77,
		Day:        "2026-03-14",
	})
	if err != nil {
		t.Fatalf("the calendar was refused: %v", err)
	}

	if calendar.Day.String() != "2026-03-14" {
		t.Errorf("the day is %s, want 2026-03-14", calendar.Day)
	}
	if len(calendar.Due) != 1 || calendar.Due[0].ID != commitment.ID(9) {
		t.Fatalf("the calendar carries the promises due %+v, want the promise 9", calendar.Due)
	}
	if len(calendar.Owed) != 1 || calendar.Owed[0].ID != commitment.ID(9) {
		t.Fatalf("the calendar carries the promises owed %+v, want the promise 9", calendar.Owed)
	}
}

// TestTheCalendarWithoutADayReadsTheWholeBook proves that the book of the
// promises is read whole when the agent names no day: every promise of the
// business is due, and every open one is owed.
func TestTheCalendarWithoutADayReadsTheWholeBook(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{
		held: []commitment.Commitment{
			thePromiseOfTheCommitments(t, commitment.ID(8), "El pan del domingo", "2026-03-07"),
			thePromiseOfTheCommitments(t, commitment.ID(9), "El pedido de la tienda", "2026-03-14"),
		},
	}
	service := theCommitmentService(commitments, &bakery)

	calendar, err := service.Read(t.Context(), ports.ReadCommitments{BusinessID: 77})
	if err != nil {
		t.Fatalf("the calendar was refused: %v", err)
	}

	if !calendar.Day.Time().IsZero() {
		t.Errorf("the day is %s, want none", calendar.Day)
	}
	if len(calendar.Due) != 2 {
		t.Errorf("the calendar carries %d promises, want 2", len(calendar.Due))
	}
	if len(calendar.Owed) != 2 {
		t.Errorf("the calendar carries %d promises owed, want 2", len(calendar.Owed))
	}
}

// TestTheCalendarOfABusinessThatIsNotThereIsRefused proves that the reading
// never answers an empty calendar for a business that does not exist: a
// bakery that promised nothing and a business that is not there are two
// different answers.
func TestTheCalendarOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	commitments := &fakeCommitments{}
	service := theCommitmentService(commitments, nil)

	_, err := service.Read(t.Context(), ports.ReadCommitments{BusinessID: 77})
	if !errors.Is(err, commitment.ErrBusinessNotFound) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrBusinessNotFound)
	}
}

// TestTheCalendarOfABusinessThatDoesNotHoldPromisesIsRefused proves that
// the Personal Business of a person holds the money of that person, and not
// the promises of the bakery.
func TestTheCalendarOfABusinessThatDoesNotHoldPromisesIsRefused(t *testing.T) {
	t.Parallel()

	personal := business.Business{
		ID:       business.ID(77),
		Kind:     business.KindPersonal,
		Name:     "Ana Pérez",
		Currency: "COP",
	}
	commitments := &fakeCommitments{}
	service := theCommitmentService(commitments, &personal)

	_, err := service.Read(t.Context(), ports.ReadCommitments{BusinessID: 77})
	if !errors.Is(err, commitment.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrWrongBusiness)
	}
}

// TestTheCalendarOfTheDayThatIsNotADayIsRefused proves that the day of the
// calendar is read in the vocabulary of the book, and a day that is not a
// day of the book is refused.
func TestTheCalendarOfTheDayThatIsNotADayIsRefused(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{}
	service := theCommitmentService(commitments, &bakery)

	_, err := service.Read(t.Context(), ports.ReadCommitments{
		BusinessID: 77,
		Day:        "el catorce de marzo",
	})
	if !errors.Is(err, ledger.ErrInvalidDate) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidDate)
	}
}

// TestTheMeetingOfTheStorageThatFailsReachesTheCaller proves that the
// meeting never invents a failure of its own: the error of the storage is
// the error the agent reads.
func TestTheMeetingOfTheStorageThatFailsReachesTheCaller(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{err: commitment.ErrNoStorage}
	service := theCommitmentService(commitments, &bakery)

	_, err := service.Meet(t.Context(), ports.MeetCommitment{
		BusinessID:   77,
		CommitmentID: 9,
		MetOn:        "2026-03-14",
	})
	if !errors.Is(err, commitment.ErrNoStorage) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrNoStorage)
	}
}

// TestTheCalendarOfTheStorageThatFailsReachesTheCaller proves that the
// reading never answers an empty calendar when the storage failed: the error
// of the storage is the error the agent reads.
func TestTheCalendarOfTheStorageThatFailsReachesTheCaller(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheOrders()
	commitments := &fakeCommitments{err: commitment.ErrNoStorage}
	service := theCommitmentService(commitments, &bakery)

	_, err := service.Read(t.Context(), ports.ReadCommitments{BusinessID: 77})
	if !errors.Is(err, commitment.ErrNoStorage) {
		t.Fatalf("the error is %v, want %v", err, commitment.ErrNoStorage)
	}
}

// TestTheCalendarWithoutAStorageIsRefused proves that the use case with no
// database behind it says so.
func TestTheCalendarWithoutAStorageIsRefused(t *testing.T) {
	t.Parallel()

	service := application.NewCommitmentService(nil, nil)

	if _, err := service.Read(t.Context(), ports.ReadCommitments{BusinessID: 77}); !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
	if _, err := service.Meet(t.Context(), ports.MeetCommitment{BusinessID: 77, CommitmentID: 9}); !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
