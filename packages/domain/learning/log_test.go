package learning

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
)

// theEntryOfTheLog is one entry of the log of the tests: the day, the batch,
// what happened and what was learned, in the words of the owner.
func theEntryOfTheLog(t *testing.T, id ID, happened, learned string, answers ID) Entry {
	t.Helper()

	built, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-14"), theBatchOfTheTest(), nil,
		happened, learned, nil)
	if err != nil {
		t.Fatalf("the entry of the test was refused: %v", err)
	}

	built.ID = id
	built.Answers = answers

	return built
}

// TestTheCurrentEntriesAreTheLastWordOfTheirChain proves the rule of the
// domain: the entry that is current is the one no other entry answers. The
// corrections of the past are in the log, and the last word is what Paca
// tells the owner: the entry of a chain that nobody corrected, and the entry
// that was never corrected at all.
func TestTheCurrentEntriesAreTheLastWordOfTheirChain(t *testing.T) {
	t.Parallel()

	first := theEntryOfTheLog(t, 1, "La masa subió lento", "Leudar más", 0)
	second := theEntryOfTheLog(t, 2, "La masa quedó dura", "Amasar más", 1)
	third := theEntryOfTheLog(t, 3, "El horno corrió caliente", "Bajar el horno", 2)
	alone := theEntryOfTheLog(t, 4, "La masa subió lento", "Leudar más", 0)

	current := TheLog([]Entry{first, second, third, alone}).Current()

	if len(current) != 2 {
		t.Fatalf("the log holds %d current entries, want 2: %+v", len(current), current)
	}
	if current[0].ID != 3 {
		t.Errorf("the first current entry is %d, want 3: the last word of the chain", current[0].ID)
	}
	if current[1].ID != 4 {
		t.Errorf("the second current entry is %d, want 4: the entry that answers nothing", current[1].ID)
	}
}

// TestTheLogWithNoEntryHoldsNoCurrentEntry proves the boundary of the rule:
// the log of a business that wrote nothing is empty, and it is not nil.
func TestTheLogWithNoEntryHoldsNoCurrentEntry(t *testing.T) {
	t.Parallel()

	current := TheLog(nil).Current()
	if current == nil {
		t.Fatal("the current entries are nil, want an empty list")
	}
	if len(current) != 0 {
		t.Fatalf("the empty log holds %d current entries, want none", len(current))
	}
}

// TestTheLogAnswersTheEntryThatIsThere proves the rule of the domain: the
// entry that is answered exists, and it is of the business that writes the
// answer.
func TestTheLogAnswersTheEntryThatIsThere(t *testing.T) {
	t.Parallel()

	log := TheLog([]Entry{
		theEntryOfTheLog(t, 1, "La masa subió lento", "Leudar más", 0),
		theEntryOfTheLog(t, 2, "La masa quedó dura", "Amasar más", 0),
	})

	answered, err := log.Answering(1, business.ID(77))
	if err != nil {
		t.Fatalf("the entry 1 was refused: %v", err)
	}
	if answered.ID != 1 {
		t.Errorf("the answered entry is %d, want 1", answered.ID)
	}
}

// TestTheLogRefusesTheEntryItDoesNotHold proves the rule of the domain: the
// entry that is answered must exist.
func TestTheLogRefusesTheEntryItDoesNotHold(t *testing.T) {
	t.Parallel()

	log := TheLog([]Entry{theEntryOfTheLog(t, 1, "La masa subió lento", "Leudar más", 0)})

	if _, err := log.Answering(2, business.ID(77)); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("the unknown entry answered %v, want %v", err, ErrEntryNotFound)
	}
}

// TestTheLogRefusesTheEntryOfAnotherBusiness proves the rule of the domain:
// the entry that is answered is an entry of the business that writes the
// answer.
func TestTheLogRefusesTheEntryOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	foreign := theEntryOfTheLog(t, 1, "La masa subió lento", "Leudar más", 0)
	foreign.BusinessID = business.ID(88)

	log := TheLog([]Entry{foreign})

	if _, err := log.Answering(1, business.ID(77)); !errors.Is(err, ErrEntryOfAnotherBusiness) {
		t.Fatalf("the entry of another business answered %v, want %v", err, ErrEntryOfAnotherBusiness)
	}
}

// TestTheLogRefusesTheEntryThatIsAnsweredAlready proves the rule of the
// domain: the log is a chain and not a tree, and two answers of the same
// entry would leave the owner with two truths.
func TestTheLogRefusesTheEntryThatIsAnsweredAlready(t *testing.T) {
	t.Parallel()

	answered := theEntryOfTheLog(t, 1, "La masa subió lento", "Leudar más", 0)
	answer := theEntryOfTheLog(t, 2, "La masa quedó dura", "Amasar más", 1)

	log := TheLog([]Entry{answered, answer})

	if _, err := log.Answering(1, business.ID(77)); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("the entry that was answered already answered %v, want %v", err, ErrAlreadyAnswered)
	}
}
