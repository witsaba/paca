package commitment

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// theDayOfTheTest is a day of the book of the bakery.
func theDayOfTheTest(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day %q could not be read: %v", day, err)
	}

	return parsed
}

// thePromiseOfTheTest is the promise the tests read back from the storage:
// the bread the bakery owes to the store on a day, and nobody met it yet.
func thePromiseOfTheTest(t *testing.T) Commitment {
	t.Helper()

	built, err := New(business.ID(77), "  El pedido de la tienda de la esquina  ", theDayOfTheTest(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the commitment was refused: %v", err)
	}

	built.ID = ID(9)

	return built
}

// TestNewBuildsThePromiseOfTheBusiness proves that the domain builds the
// commitment of the business: what is due, in the words of the owner, and
// the day it is due.
//
// The promise is written OPEN: it carries no day of meeting, and the
// identity of the promise is the one the storage generates.
func TestNewBuildsThePromiseOfTheBusiness(t *testing.T) {
	t.Parallel()

	built, err := New(business.ID(77), "El pedido de la tienda de la esquina", theDayOfTheTest(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the commitment was refused: %v", err)
	}

	if built.ID != 0 {
		t.Errorf("the identity is %d, want 0: the storage generates it", built.ID)
	}
	if built.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", built.BusinessID)
	}
	if built.What != "El pedido de la tienda de la esquina" {
		t.Errorf("the what is %q, want the words of the owner", built.What)
	}
	if !built.DueOn.Time().Equal(theDayOfTheTest(t, "2026-03-14").Time()) {
		t.Errorf("the day it is due is %s, want 2026-03-14", built.DueOn)
	}
	if !built.MetOn.Time().IsZero() {
		t.Errorf("the day it was met is %s, want none: the promise is written open", built.MetOn)
	}
	if !built.IsOpen() {
		t.Error("the promise that was written carries no day of meeting, and it is not open")
	}
}

// TestNewLosesTheSpacesAroundTheWhat proves the rule of the text of the
// promise: the what loses the spaces around it, and the book keeps the
// words of the owner and not the spaces of the protocol.
func TestNewLosesTheSpacesAroundTheWhat(t *testing.T) {
	t.Parallel()

	built, err := New(business.ID(77), "   El pan del domingo   ", theDayOfTheTest(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the commitment was refused: %v", err)
	}

	if built.What != "El pan del domingo" {
		t.Errorf("the what is %q, want %q", built.What, "El pan del domingo")
	}
}

// TestNewRefusesTheCommitmentWithoutABusiness proves the rule of the
// domain: the promise belongs to a business, and the promise of no business
// is refused.
func TestNewRefusesTheCommitmentWithoutABusiness(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		own   business.ID
		wants error
	}{
		{name: "no business", own: 0, wants: ErrMissingBusiness},
		{name: "a business that is not there", own: -3, wants: ErrMissingBusiness},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(test.own, "El pan del domingo", theDayOfTheTest(t, "2026-03-14"))
			if !errors.Is(err, test.wants) {
				t.Fatalf("the error is %v, want %v", err, test.wants)
			}
		})
	}
}

// TestNewRefusesThePromiseThatSaysNothing proves the rule of the domain: a
// promise that says nothing about what is due is not a promise, and the
// spaces are not a promise either.
func TestNewRefusesThePromiseThatSaysNothing(t *testing.T) {
	t.Parallel()

	for _, what := range []string{"", " ", "\t\n"} {
		_, err := New(business.ID(77), what, theDayOfTheTest(t, "2026-03-14"))
		if !errors.Is(err, ErrMissingWhat) {
			t.Errorf("the what %q was answered with %v, want %v", what, err, ErrMissingWhat)
		}
	}
}

// TestNewRefusesThePromiseWithoutItsDay proves the rule of the domain: the
// promise carries the day it is due, and the book never invents one.
func TestNewRefusesThePromiseWithoutItsDay(t *testing.T) {
	t.Parallel()

	_, err := New(business.ID(77), "El pan del domingo", ledger.Date{})
	if !errors.Is(err, ErrMissingDay) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingDay)
	}
}

// TestThePromiseThatWasKeptKeepsTheDayItWasKept proves the rule of the
// domain: the commitment that was met is KEPT, with the day it was met, and
// the promise that was met is not owed any more.
func TestThePromiseThatWasKeptKeepsTheDayItWasKept(t *testing.T) {
	t.Parallel()

	promise := thePromiseOfTheTest(t)

	met, err := promise.Met(theDayOfTheTest(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the meeting was refused: %v", err)
	}

	if met.ID != promise.ID {
		t.Errorf("the identity is %d, want %d: the promise that was kept is the same promise", met.ID, promise.ID)
	}
	if met.What != promise.What {
		t.Errorf("the what is %q, want %q: the promise that was kept is kept whole", met.What, promise.What)
	}
	if !met.DueOn.Time().Equal(promise.DueOn.Time()) {
		t.Errorf("the day it is due is %s, want %s", met.DueOn, promise.DueOn)
	}
	if !met.MetOn.Time().Equal(theDayOfTheTest(t, "2026-03-14").Time()) {
		t.Errorf("the day it was met is %s, want 2026-03-14", met.MetOn)
	}
	if met.IsOpen() {
		t.Error("the promise that was met is still owed")
	}
	if !promise.IsOpen() {
		t.Error("the promise the caller held was changed: the model is a value, and the meeting answers a new one")
	}
}

// TestAPromiseKeptEarlyAndAPromiseKeptLateAreBothKept proves the rule of the
// domain: the day it was met may be before or after the day it is due. A
// promise kept early is kept, a promise kept late is late, and the book
// never hides it.
func TestAPromiseKeptEarlyAndAPromiseKeptLateAreBothKept(t *testing.T) {
	t.Parallel()

	promise := thePromiseOfTheTest(t)

	cases := []struct {
		name string
		day  string
	}{
		{name: "kept before the day it is due", day: "2026-03-07"},
		{name: "kept after the day it is due", day: "2026-03-21"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			met, err := promise.Met(theDayOfTheTest(t, test.day))
			if err != nil {
				t.Fatalf("the meeting of %s was refused: %v", test.day, err)
			}
			if met.MetOn.String() != test.day {
				t.Errorf("the day it was met is %s, want %s", met.MetOn, test.day)
			}
		})
	}
}

// TestTheDomainNeverRefusesAMeetingInTheFuture proves the rule of the
// domain: the domain does not know the day of today, so it never refuses a
// day of meeting for being in the future. The owner is the one that says
// what happened.
func TestTheDomainNeverRefusesAMeetingInTheFuture(t *testing.T) {
	t.Parallel()

	met, err := thePromiseOfTheTest(t).Met(theDayOfTheTest(t, "2099-12-31"))
	if err != nil {
		t.Fatalf("the meeting in the future was refused: %v", err)
	}
	if met.MetOn.String() != "2099-12-31" {
		t.Errorf("the day it was met is %s, want 2099-12-31", met.MetOn)
	}
}

// TestMetRefusesThePromiseThatWasMetAlready proves the rule of the domain:
// a promise is kept once, and the second meeting of the same promise is
// refused with the day the book already holds.
func TestMetRefusesThePromiseThatWasMetAlready(t *testing.T) {
	t.Parallel()

	met, err := thePromiseOfTheTest(t).Met(theDayOfTheTest(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the meeting was refused: %v", err)
	}

	again, err := met.Met(theDayOfTheTest(t, "2026-03-15"))
	if !errors.Is(err, ErrAlreadyMet) {
		t.Fatalf("the second meeting answered %v, want %v", err, ErrAlreadyMet)
	}
	if again != (Commitment{}) {
		t.Errorf("the second meeting answered the commitment %+v, want the empty one", again)
	}
	if met.MetOn.String() != "2026-03-14" {
		t.Errorf("the day it was met is %s, want 2026-03-14: the book never hides the day it was kept", met.MetOn)
	}
}

// TestMetRefusesAMeetingWithoutItsDay proves the rule of the domain: the
// promise that was kept stays in the book WITH the day it was kept, so a
// meeting without a day is refused instead of leaving the promise looking
// open for ever.
func TestMetRefusesAMeetingWithoutItsDay(t *testing.T) {
	t.Parallel()

	_, err := thePromiseOfTheTest(t).Met(ledger.Date{})
	if !errors.Is(err, ErrMissingMetDay) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingMetDay)
	}
}

// TestNewFromFactsBuildsThePromiseOfTheRow proves that the storage row is
// read back through the own constructor of the domain: the row never
// decides what a commitment is.
func TestNewFromFactsBuildsThePromiseOfTheRow(t *testing.T) {
	t.Parallel()

	built, err := NewFromFacts(Facts{
		ID:         ID(9),
		BusinessID: business.ID(77),
		What:       "  El pan del domingo  ",
		DueOn:      theDayOfTheTest(t, "2026-03-14"),
		MetOn:      theDayOfTheTest(t, "2026-03-13"),
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if built.ID != ID(9) {
		t.Errorf("the identity is %d, want 9", built.ID)
	}
	if built.What != "El pan del domingo" {
		t.Errorf("the what is %q, want %q", built.What, "El pan del domingo")
	}
	if built.MetOn.String() != "2026-03-13" {
		t.Errorf("the day it was met is %s, want 2026-03-13", built.MetOn)
	}
	if built.IsOpen() {
		t.Error("the row carries the day it was met, and the commitment is open")
	}
}

// TestNewFromFactsReadsTheRowThatWasNotMetAsOpen proves the rule of the
// domain: the state of the row is the state of the commitment, and a row
// with no day of meeting is an open commitment. There is no state column
// that could say otherwise.
func TestNewFromFactsReadsTheRowThatWasNotMetAsOpen(t *testing.T) {
	t.Parallel()

	built, err := NewFromFacts(Facts{
		ID:         ID(9),
		BusinessID: business.ID(77),
		What:       "El pan del domingo",
		DueOn:      theDayOfTheTest(t, "2026-03-14"),
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if !built.IsOpen() {
		t.Error("the row carries no day of meeting, and the commitment is not open")
	}
}

// TestNewFromFactsRefusesTheRowThatBreaksARule proves the rule of the
// domain: a row that another writer broke is refused by the domain, and the
// storage never decides what a commitment is.
func TestNewFromFactsRefusesTheRowThatBreaksARule(t *testing.T) {
	t.Parallel()

	day := theDayOfTheTest(t, "2026-03-14")

	cases := []struct {
		name  string
		facts Facts
		wants error
	}{
		{
			name:  "the row of no business",
			facts: Facts{ID: ID(9), What: "El pan", DueOn: day},
			wants: ErrMissingBusiness,
		},
		{
			name:  "the row that says nothing",
			facts: Facts{ID: ID(9), BusinessID: business.ID(77), What: "   ", DueOn: day},
			wants: ErrMissingWhat,
		},
		{
			name:  "the row of no day",
			facts: Facts{ID: ID(9), BusinessID: business.ID(77), What: "El pan"},
			wants: ErrMissingDay,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewFromFacts(test.facts)
			if !errors.Is(err, test.wants) {
				t.Fatalf("the error is %v, want %v", err, test.wants)
			}
		})
	}
}

// TestOwnedByRefusesABusinessOfAnotherKind proves the rule of the domain:
// the commitments belong to a business of its own, the one that sells the
// bread. The Personal Business of a person holds the money of that person
// and not the promises of the bakery.
func TestOwnedByRefusesABusinessOfAnotherKind(t *testing.T) {
	t.Parallel()

	if err := OwnedBy(business.KindBusiness); err != nil {
		t.Errorf("the business of its own was refused: %v", err)
	}

	for _, kind := range []business.Kind{business.KindPersonal, business.Kind("shop")} {
		if err := OwnedBy(kind); !errors.Is(err, ErrWrongBusiness) {
			t.Errorf("the kind %q answered %v, want %v", kind, err, ErrWrongBusiness)
		}
	}
}

// TestTheCalendarCarriesThePromisesThatAreDueOnTheDay proves the rule of
// the calendar: the promises that are due on the day, the ones that were
// met on it and the ones that are still owed.
func TestTheCalendarCarriesThePromisesThatAreDueOnTheDay(t *testing.T) {
	t.Parallel()

	due := thePromiseOfTheTest(t)
	met, err := due.Met(theDayOfTheTest(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the meeting was refused: %v", err)
	}

	later, err := New(business.ID(77), "El pan del domingo", theDayOfTheTest(t, "2026-03-21"))
	if err != nil {
		t.Fatalf("the commitment was refused: %v", err)
	}

	calendar := TheCalendar(theDayOfTheTest(t, "2026-03-14"), []Commitment{met, later})

	if !calendar.Day.Time().Equal(theDayOfTheTest(t, "2026-03-14").Time()) {
		t.Errorf("the day of the calendar is %s, want 2026-03-14", calendar.Day)
	}
	if len(calendar.Due) != 1 {
		t.Fatalf("the calendar carries %d promises due on the day, want 1", len(calendar.Due))
	}
	if calendar.Due[0].What != met.What {
		t.Errorf("the promise due on the day is %q, want %q", calendar.Due[0].What, met.What)
	}
	if len(calendar.Owed) != 1 {
		t.Fatalf("the calendar carries %d promises owed, want 1: the promise that was met is not owed any more", len(calendar.Owed))
	}
	if calendar.Owed[0].What != later.What {
		t.Errorf("the promise owed is %q, want %q", calendar.Owed[0].What, later.What)
	}
}

// TestTheCalendarWithoutADayCarriesEveryPromise proves the rule of the
// calendar: a calendar with no day carries every promise of the business,
// the book of the promises read whole.
func TestTheCalendarWithoutADayCarriesEveryPromise(t *testing.T) {
	t.Parallel()

	first := thePromiseOfTheTest(t)
	second, err := New(business.ID(77), "El pan del domingo", theDayOfTheTest(t, "2026-03-21"))
	if err != nil {
		t.Fatalf("the commitment was refused: %v", err)
	}

	calendar := TheCalendar(ledger.Date{}, []Commitment{first, second})

	if !calendar.Day.Time().IsZero() {
		t.Errorf("the day of the calendar is %s, want none", calendar.Day)
	}
	if len(calendar.Due) != 2 {
		t.Fatalf("the calendar carries %d promises, want 2: the book is read whole when no day is given", len(calendar.Due))
	}
	if len(calendar.Owed) != 2 {
		t.Errorf("the calendar carries %d promises owed, want 2", len(calendar.Owed))
	}
}

// TestTheCalendarAnswersEveryOpenPromiseItOwes proves the rule of the
// calendar: the owed is EVERY open promise of the business, the ones that
// are due on the day and the ones that fell due on an earlier day. The
// promise that was met is not owed, no matter the day it fell due on.
func TestTheCalendarAnswersEveryOpenPromiseItOwes(t *testing.T) {
	t.Parallel()

	overdue, err := New(business.ID(77), "El pan del domingo", theDayOfTheTest(t, "2026-03-07"))
	if err != nil {
		t.Fatalf("the commitment was refused: %v", err)
	}

	met, err := overdue.Met(theDayOfTheTest(t, "2026-03-07"))
	if err != nil {
		t.Fatalf("the meeting was refused: %v", err)
	}

	due, err := New(business.ID(77), "El pedido de la tienda", theDayOfTheTest(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the commitment was refused: %v", err)
	}

	calendar := TheCalendar(theDayOfTheTest(t, "2026-03-14"), []Commitment{met, due})

	if len(calendar.Due) != 1 {
		t.Fatalf("the calendar carries %d promises due on the day, want 1", len(calendar.Due))
	}
	if calendar.Due[0].What != due.What {
		t.Errorf("the promise due on the day is %q, want %q", calendar.Due[0].What, due.What)
	}
	if len(calendar.Owed) != 1 {
		t.Fatalf("the calendar carries %d promises owed, want 1", len(calendar.Owed))
	}
	if calendar.Owed[0].What != due.What {
		t.Errorf("the promise owed is %q, want %q", calendar.Owed[0].What, due.What)
	}
}

// TestErrorsNamesEveryError proves that the catalogue of the errors of the
// domain names every one of them: a caller proves that it maps them all.
func TestErrorsNamesEveryError(t *testing.T) {
	t.Parallel()

	want := []error{
		ErrNotFound,
		ErrMissingBusiness,
		ErrMissingWhat,
		ErrMissingDay,
		ErrMissingMetDay,
		ErrAlreadyMet,
		ErrWrongBusiness,
		ErrBusinessNotFound,
		ErrInvalidValue,
		ErrNoStorage,
	}

	held := Errors()
	if len(held) != len(want) {
		t.Fatalf("the catalogue names %d errors, want %d", len(held), len(want))
	}

	for _, one := range want {
		found := false
		for _, held := range held {
			if errors.Is(held, one) {
				found = true
			}
		}
		if !found {
			t.Errorf("the catalogue does not name %v", one)
		}
	}
}
