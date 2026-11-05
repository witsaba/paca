package purchase

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// theDay is a day of the book the tests of the domain use.
func theDay(year int, month time.Month, day int) ledger.Date {
	return ledger.NewDate(time.Date(year, month, day, 10, 0, 0, 0, time.UTC))
}

// theLine is a line of a draft the tests of the domain use.
func theLine(t *testing.T, name string, quantity float64, unit string, total ledger.Amount) Line {
	t.Helper()

	measure, err := NewMeasure(quantity, unit)
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	line, err := NewLine(name, measure, Price{Total: total})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	return line
}

// TestTheDraftKeepsTheSentenceTheOwnerSaid proves that the draft keeps the
// sentence of the owner, its lines and the day it was proposed: the entry
// can be audited and derived again.
func TestTheDraftKeepsTheSentenceTheOwnerSaid(t *testing.T) {
	t.Parallel()

	proposed := theDay(2026, time.March, 1)
	candidate := Candidate{
		Lines:   []Line{theLine(t, "Harina", 25, "kg", 12000)},
		Day:     theDay(2026, time.February, 28),
		Market:  "Plaza de Mercado",
		Invoice: "FV-001",
	}

	draft, err := NewDraft(business.ID(77), "  Compré 25 kilos de harina  ", candidate, proposed)
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	if draft.Utterance != "Compré 25 kilos de harina" {
		t.Errorf("the sentence is %q, want %q", draft.Utterance, "Compré 25 kilos de harina")
	}
	if draft.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", draft.BusinessID)
	}
	if draft.State != StateProposed {
		t.Errorf("the state is %q, want %q", draft.State, StateProposed)
	}
	if draft.Revision != 1 {
		t.Errorf("the revision is %d, want 1", draft.Revision)
	}
	if len(draft.Lines) != 1 {
		t.Fatalf("the draft holds %d lines, want 1", len(draft.Lines))
	}
	if draft.Lines[0].Name != "Harina" {
		t.Errorf("the name of the line is %q, want %q", draft.Lines[0].Name, "Harina")
	}
	if draft.Day != candidate.Day {
		t.Errorf("the day is %v, want %v", draft.Day, candidate.Day)
	}
	if draft.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", draft.Market, "Plaza de Mercado")
	}
	if draft.Invoice != "FV-001" {
		t.Errorf("the invoice is %q, want %q", draft.Invoice, "FV-001")
	}
	if draft.ProposedOn != proposed {
		t.Errorf("the day of the proposal is %v, want %v", draft.ProposedOn, proposed)
	}
}

// TestTheDraftWithoutASentenceIsRefused proves that a draft without the
// sentence of the owner is refused: a draft that cannot be audited is not
// a draft.
func TestTheDraftWithoutASentenceIsRefused(t *testing.T) {
	t.Parallel()

	candidate := Candidate{Lines: []Line{theLine(t, "Harina", 25, "kg", 12000)}}

	_, err := NewDraft(business.ID(77), "   ", candidate, theDay(2026, time.March, 1))
	if !errors.Is(err, ErrMissingUtterance) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingUtterance)
	}
}

// TestTheDraftWithoutALineIsRefused proves that a draft without a line is
// refused: a proposal of nothing is not a proposal.
func TestTheDraftWithoutALineIsRefused(t *testing.T) {
	t.Parallel()

	_, err := NewDraft(business.ID(77), "Compré harina", Candidate{}, theDay(2026, time.March, 1))
	if !errors.Is(err, ErrNoLines) {
		t.Fatalf("the error is %v, want %v", err, ErrNoLines)
	}
}

// TestTheDayTheOwnerNamedIsKeptAndNothingWasInferred proves that the day
// the owner named is the day of the draft, and that Paca filled nothing.
func TestTheDayTheOwnerNamedIsKeptAndNothingWasInferred(t *testing.T) {
	t.Parallel()

	proposed := theDay(2026, time.March, 1)
	candidate := Candidate{
		Lines: []Line{theLine(t, "Harina", 25, "kg", 12000)},
		Day:   theDay(2026, time.February, 28),
	}

	draft, err := NewDraft(business.ID(77), "Compré harina", candidate, proposed)
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	if draft.Day != candidate.Day {
		t.Errorf("the day is %v, want %v", draft.Day, candidate.Day)
	}
	if len(draft.Inferred) != 0 {
		t.Errorf("Paca filled %v, want nothing", draft.Inferred)
	}
}

// TestTheDayTheOwnerDidNotNameIsTheDayOfTodayAndItIsInferred proves the
// rule of the domain: the day is one of the three things Paca may fill by
// herself, and the answer names it.
func TestTheDayTheOwnerDidNotNameIsTheDayOfTodayAndItIsInferred(t *testing.T) {
	t.Parallel()

	proposed := theDay(2026, time.March, 1)
	candidate := Candidate{Lines: []Line{theLine(t, "Harina", 25, "kg", 12000)}}

	draft, err := NewDraft(business.ID(77), "Compré harina", candidate, proposed)
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	if draft.Day != proposed {
		t.Errorf("the day is %v, want %v", draft.Day, proposed)
	}
	if len(draft.Inferred) != 1 {
		t.Fatalf("Paca filled %v, want the day of the purchase", draft.Inferred)
	}
	if draft.Inferred[0] != "the day of the purchase is the day of today" {
		t.Errorf("the inferred thing is %q, want the day of the purchase", draft.Inferred[0])
	}
}

// TestTheLineWithoutAPriceIsWhatTheDraftStillNeeds proves the rule of the
// domain: the money is never inferred, so a line the owner did not price
// is what the draft still needs.
func TestTheLineWithoutAPriceIsWhatTheDraftStillNeeds(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(3, "unidad")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	unpriced, err := NewLine("Pan", measure, Price{})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	candidate := Candidate{Lines: []Line{
		theLine(t, "Harina", 25, "kg", 12000),
		unpriced,
	}}

	draft, err := NewDraft(business.ID(77), "Compré harina y pan", candidate, theDay(2026, time.March, 1))
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	if len(draft.Missing) != 1 {
		t.Fatalf("the draft needs %v, want the price of the pan", draft.Missing)
	}
	if draft.Missing[0] != "the price of Pan" {
		t.Errorf("the missing thing is %q, want %q", draft.Missing[0], "the price of Pan")
	}
}

// TestTheDraftExpiresSevenDaysAfterItWasProposed proves the rule of the
// domain: a draft that was neither confirmed nor cancelled in seven days
// expires, because the price of a market does not wait.
func TestTheDraftExpiresSevenDaysAfterItWasProposed(t *testing.T) {
	t.Parallel()

	proposed := theDay(2026, time.March, 1)
	candidate := Candidate{Lines: []Line{theLine(t, "Harina", 25, "kg", 12000)}}

	draft, err := NewDraft(business.ID(77), "Compré harina", candidate, proposed)
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	want := theDay(2026, time.March, 8)
	if draft.ExpiresOn != want {
		t.Errorf("the day of the expiry is %v, want %v", draft.ExpiresOn, want)
	}
}

// TestTheMarketAndTheInvoiceAreOptional proves that a draft names the
// market and the invoice only when the owner named them.
func TestTheMarketAndTheInvoiceAreOptional(t *testing.T) {
	t.Parallel()

	candidate := Candidate{Lines: []Line{theLine(t, "Harina", 25, "kg", 12000)}}

	draft, err := NewDraft(business.ID(77), "Compré harina", candidate, theDay(2026, time.March, 1))
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	if draft.Market != "" {
		t.Errorf("the market is %q, want it absent", draft.Market)
	}
	if draft.Invoice != "" {
		t.Errorf("the invoice is %q, want it absent", draft.Invoice)
	}
}

// TestTheAmendmentGrowsTheRevision proves the rule of the domain: an
// amendment is a new reading of the same draft, and it grows the revision.
// The sentence the owner said the first time is kept.
func TestTheAmendmentGrowsTheRevision(t *testing.T) {
	t.Parallel()

	proposed := theDay(2026, time.March, 1)
	candidate := Candidate{Lines: []Line{theLine(t, "Harina", 25, "kg", 12000)}}

	draft, err := NewDraft(business.ID(77), "Compré 25 kilos de harina", candidate, proposed)
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	amended, err := Amend(draft, Candidate{
		Lines: []Line{theLine(t, "Harina", 30, "kg", 14400)},
		Day:   theDay(2026, time.February, 28),
	})
	if err != nil {
		t.Fatalf("the amendment was refused: %v", err)
	}

	if amended.Revision != 2 {
		t.Errorf("the revision is %d, want 2", amended.Revision)
	}
	if amended.Utterance != draft.Utterance {
		t.Errorf("the sentence is %q, want %q", amended.Utterance, draft.Utterance)
	}
	if len(amended.Lines) != 1 || amended.Lines[0].Price.Total != 14400 {
		t.Errorf("the lines of the amendment are %v, want the new reading", amended.Lines)
	}
	if amended.ProposedOn != proposed {
		t.Errorf("the day of the proposal is %v, want %v", amended.ProposedOn, proposed)
	}
	if amended.ExpiresOn != draft.ExpiresOn {
		t.Errorf("the day of the expiry is %v, want %v", amended.ExpiresOn, draft.ExpiresOn)
	}
	if amended.Day != theDay(2026, time.February, 28) {
		t.Errorf("the day is %v, want the day the owner named", amended.Day)
	}
}

// TestTheAmendmentWithoutALineIsRefused proves that an amendment that
// proposes nothing is refused.
func TestTheAmendmentWithoutALineIsRefused(t *testing.T) {
	t.Parallel()

	draft := Draft{Revision: 1, Day: theDay(2026, time.March, 1)}

	_, err := Amend(draft, Candidate{})
	if !errors.Is(err, ErrNoLines) {
		t.Fatalf("the error is %v, want %v", err, ErrNoLines)
	}
}

// TestTheAmbiguityCarriesTheOptions proves the rule of the domain: Paca
// never asks an open question when it can name the options, so an
// ambiguity carries the candidates it found.
func TestTheAmbiguityCarriesTheOptions(t *testing.T) {
	t.Parallel()

	ambiguity, err := NewAmbiguity("harina", []string{"Harina panadera", "Harina de trigo"})
	if err != nil {
		t.Fatalf("the ambiguity was refused: %v", err)
	}

	if ambiguity.Name != "harina" {
		t.Errorf("the name is %q, want %q", ambiguity.Name, "harina")
	}
	if len(ambiguity.Options) != 2 {
		t.Fatalf("the ambiguity holds %d options, want 2", len(ambiguity.Options))
	}
	if ambiguity.Options[0] != "Harina panadera" {
		t.Errorf("the first option is %q, want %q", ambiguity.Options[0], "Harina panadera")
	}
}

// TestTheAmbiguityWithoutOptionsIsRefused proves that an ambiguity without
// options is refused: it would be an open question, and Paca never asks
// one.
func TestTheAmbiguityWithoutOptionsIsRefused(t *testing.T) {
	t.Parallel()

	_, err := NewAmbiguity("harina", nil)
	if !errors.Is(err, ErrNoOptions) {
		t.Fatalf("the error is %v, want %v", err, ErrNoOptions)
	}
}

// TestTheAmbiguityWithoutANameIsRefused proves that an ambiguity without a
// name is refused: the owner would not know what she is being asked about.
func TestTheAmbiguityWithoutANameIsRefused(t *testing.T) {
	t.Parallel()

	_, err := NewAmbiguity("  ", []string{"Harina panadera"})
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidName)
	}
}
