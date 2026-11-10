package purchase

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// theDraftOfTheTests is a draft that waits for the yes of the owner.
func theDraftOfTheTests(t *testing.T) Draft {
	t.Helper()

	candidate := Candidate{Lines: []Line{theLine(t, "Harina", 25, "kg", 12000)}}

	draft, err := NewDraft(business.ID(77), "Compré 25 kilos de harina", candidate, theDay(2026, time.March, 1))
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	return draft
}

// TestTheStatesOfTheDraftAreTheOnesTheDomainKnows proves that the
// vocabulary of the states belongs to the domain: the list is the one the
// table and the tools read, and it is never written a second time.
func TestTheStatesOfTheDraftAreTheOnesTheDomainKnows(t *testing.T) {
	t.Parallel()

	want := []State{StateProposed, StateConfirmed, StateCancelled, StateExpired}

	states := States()
	if len(states) != len(want) {
		t.Fatalf("the domain knows %v, want %v", states, want)
	}

	for index, one := range want {
		if states[index] != one {
			t.Errorf("the state %d is %q, want %q", index, states[index], one)
		}
	}
}

// TestTheDraftThatWaitsReadsAsProposed proves that a draft the owner did
// not answer yet reads as the state it waits in.
func TestTheDraftThatWaitsReadsAsProposed(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTests(t)

	if draft.ReadsAs(theDay(2026, time.March, 5)) != StateProposed {
		t.Errorf("the draft reads as %q, want %q", draft.ReadsAs(theDay(2026, time.March, 5)), StateProposed)
	}
}

// TestTheDraftThatExpiredReadsAsExpired proves the rule of the domain: the
// expiry is computed when the draft is read, and the price of a market does
// not wait.
func TestTheDraftThatExpiredReadsAsExpired(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTests(t)

	if draft.ReadsAs(theDay(2026, time.March, 8)) != StateProposed {
		t.Errorf("the draft reads as %q, want %q on the day of the expiry",
			draft.ReadsAs(theDay(2026, time.March, 8)), StateProposed)
	}

	if draft.ReadsAs(theDay(2026, time.March, 9)) != StateExpired {
		t.Errorf("the draft reads as %q, want %q the day after the expiry",
			draft.ReadsAs(theDay(2026, time.March, 9)), StateExpired)
	}
}

// TestTheCancelledDraftReadsAsCancelled proves that the state a draft ends
// in is the state it keeps: a cancelled draft never waits again.
func TestTheCancelledDraftReadsAsCancelled(t *testing.T) {
	t.Parallel()

	cancelled, err := Cancel(theDraftOfTheTests(t), theDay(2026, time.March, 2))
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	if cancelled.State != StateCancelled {
		t.Errorf("the state of the draft is %q, want %q", cancelled.State, StateCancelled)
	}
	if cancelled.ReadsAs(theDay(2026, time.March, 20)) != StateCancelled {
		t.Errorf("the draft reads as %q, want %q", cancelled.ReadsAs(theDay(2026, time.March, 20)), StateCancelled)
	}
}

// TestTheDraftIsCancelled proves that the owner drops the draft she does
// not want, and that the draft keeps everything it carried.
func TestTheDraftIsCancelled(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTests(t)

	cancelled, err := Cancel(draft, theDay(2026, time.March, 2))
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	if cancelled.ID != draft.ID {
		t.Errorf("the identity of the draft is %d, want %d", cancelled.ID, draft.ID)
	}
	if cancelled.Revision != draft.Revision {
		t.Errorf("the revision of the draft is %d, want %d", cancelled.Revision, draft.Revision)
	}
	if cancelled.Utterance != draft.Utterance {
		t.Errorf("the sentence is %q, want %q", cancelled.Utterance, draft.Utterance)
	}
	if len(cancelled.Lines) != 1 || cancelled.Lines[0].Name != "Harina" {
		t.Errorf("the lines of the draft are %v, want the line of the flour", cancelled.Lines)
	}
}

// TestTheDraftThatIsCancelledAlreadyIsNeverCancelledAgain proves that a
// draft never waits again: the domain refuses the second cancellation.
func TestTheDraftThatIsCancelledAlreadyIsNeverCancelledAgain(t *testing.T) {
	t.Parallel()

	cancelled, err := Cancel(theDraftOfTheTests(t), theDay(2026, time.March, 2))
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	_, err = Cancel(cancelled, theDay(2026, time.March, 3))
	if !errors.Is(err, ErrNotCancellable) {
		t.Fatalf("the error is %v, want %v", err, ErrNotCancellable)
	}
}

// TestTheDraftThatIsConfirmedIsNeverCancelled proves that a draft that is
// already a purchase is never cancelled: the domain refuses it.
func TestTheDraftThatIsConfirmedIsNeverCancelled(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTests(t)
	draft.State = StateConfirmed

	_, err := Cancel(draft, theDay(2026, time.March, 2))
	if !errors.Is(err, ErrNotCancellable) {
		t.Fatalf("the error is %v, want %v", err, ErrNotCancellable)
	}
}

// TestTheDraftThatExpiredIsNeverCancelled proves the rule of the domain: the
// price of a market does not wait, and an expired draft is dead.
func TestTheDraftThatExpiredIsNeverCancelled(t *testing.T) {
	t.Parallel()

	_, err := Cancel(theDraftOfTheTests(t), theDay(2026, time.March, 20))
	if !errors.Is(err, ErrExpiredDraft) {
		t.Fatalf("the error is %v, want %v", err, ErrExpiredDraft)
	}
}

// TestTheCancelledDraftIsNeverAmended proves that a draft never returns to
// proposed: the amendment of a cancelled draft is refused.
func TestTheCancelledDraftIsNeverAmended(t *testing.T) {
	t.Parallel()

	cancelled, err := Cancel(theDraftOfTheTests(t), theDay(2026, time.March, 2))
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	_, err = Amend(cancelled, Candidate{Lines: []Line{theLine(t, "Harina", 30, "kg", 14400)}}, theDay(2026, time.March, 3))
	if !errors.Is(err, ErrNotProposed) {
		t.Fatalf("the error is %v, want %v", err, ErrNotProposed)
	}
}

// TestTheExpiredDraftIsNeverAmended proves that an expired draft is dead
// too: the amendment is refused and the draft stays as it was.
func TestTheExpiredDraftIsNeverAmended(t *testing.T) {
	t.Parallel()

	_, err := Amend(theDraftOfTheTests(t), Candidate{Lines: []Line{theLine(t, "Harina", 30, "kg", 14400)}},
		theDay(2026, time.March, 20))
	if !errors.Is(err, ErrExpiredDraft) {
		t.Fatalf("the error is %v, want %v", err, ErrExpiredDraft)
	}
}

// TestThePendingDraftsAreTheOnesThatWait proves that the pending drafts of
// a business are the ones that wait for the yes of the owner: the expired
// ones and the cancelled ones are not pending.
func TestThePendingDraftsAreTheOnesThatWait(t *testing.T) {
	t.Parallel()

	waiting := theDraftOfTheTests(t)

	cancelled, err := Cancel(waiting, theDay(2026, time.March, 2))
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	confirmed := theDraftOfTheTests(t)
	confirmed.ID = DraftID(9)
	confirmed.State = StateConfirmed

	expired := theDraftOfTheTests(t)
	expired.ID = DraftID(10)
	expired.ProposedOn = theDay(2026, time.January, 1)
	expired.ExpiresOn = ledger.NewDate(theDay(2026, time.January, 1).Time().AddDate(0, 0, DraftLifetimeDays))

	pending := Pending([]Draft{cancelled, confirmed, expired, waiting}, theDay(2026, time.March, 2))

	if len(pending) != 1 {
		t.Fatalf("the business holds %d pending drafts, want 1", len(pending))
	}
	if pending[0].ID != waiting.ID {
		t.Errorf("the pending draft is %d, want %d", pending[0].ID, waiting.ID)
	}
}

// TestTheBusinessWithNoPendingDraftHoldsNone proves that the pending
// drafts of a business that holds none is an empty list and not nothing.
func TestTheBusinessWithNoPendingDraftHoldsNone(t *testing.T) {
	t.Parallel()

	pending := Pending(nil, theDay(2026, time.March, 2))

	if pending == nil {
		t.Fatal("the pending drafts are nothing, want an empty list")
	}
	if len(pending) != 0 {
		t.Errorf("the business holds %d pending drafts, want none", len(pending))
	}
}
