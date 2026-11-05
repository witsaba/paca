package purchase

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// theDraftOfTheTest builds the draft the tests of the confirmation use.
func theDraftOfTheTest(t *testing.T) Draft {
	t.Helper()

	proposed := theDay(2026, time.March, 1)

	draft, err := NewDraft(business.ID(77), "Compré 25 kilos de harina", Candidate{
		Lines:   []Line{theLine(t, "Harina", 25, "kg", 12000)},
		Day:     theDay(2026, time.February, 28),
		Market:  "Plaza de Mercado",
		Invoice: "FV-001",
	}, proposed)
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	return draft
}

// TestThePurchaseCarriesWhatTheOwnerSaid proves that the purchase carries
// the business, the day, the market, the invoice, its lines, the sentence
// of the owner and the word she used to confirm it.
func TestThePurchaseCarriesWhatTheOwnerSaid(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTest(t)

	confirmation, err := Confirm(draft, Yes{Revision: 1, Word: "sí"}, theDay(2026, time.March, 2))
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	written := confirmation.Purchase
	if written.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", written.BusinessID)
	}
	if written.Day != theDay(2026, time.February, 28) {
		t.Errorf("the day is %v, want the day the owner named", written.Day)
	}
	if written.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", written.Market, "Plaza de Mercado")
	}
	if written.Invoice != "FV-001" {
		t.Errorf("the invoice is %q, want %q", written.Invoice, "FV-001")
	}
	if written.Utterance != "Compré 25 kilos de harina" {
		t.Errorf("the sentence is %q, want %q", written.Utterance, "Compré 25 kilos de harina")
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if len(written.Lines) != 1 {
		t.Fatalf("the purchase holds %d lines, want 1", len(written.Lines))
	}
	if written.Lines[0].Name != "Harina" {
		t.Errorf("the name of the line is %q, want %q", written.Lines[0].Name, "Harina")
	}
	if written.Lines[0].Measure.Quantity != 25 || written.Lines[0].Measure.Unit != "kg" {
		t.Errorf("the measure of the line is %+v, want 25 kg", written.Lines[0].Measure)
	}
	if written.Lines[0].Price.Total != 12000 {
		t.Errorf("the money of the line is %d, want 12000", written.Lines[0].Price.Total)
	}
	if written.Total != 12000 {
		t.Errorf("the money of the purchase is %d, want 12000", written.Total)
	}
}

// TestTheConfirmationCarriesTheThreeParts proves that the answer of the
// confirmation carries the three parts: what it recorded, what it did not
// understand and what it still needs.
func TestTheConfirmationCarriesTheThreeParts(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTest(t)
	draft.Ambiguous = []Ambiguity{{Name: "harina", Options: []string{"Harina panadera", "Harina de trigo"}}}

	confirmation, err := Confirm(draft, Yes{Revision: 1, Word: "sí"}, theDay(2026, time.March, 2))
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	if len(confirmation.Recorded) != 1 || confirmation.Recorded[0].Name != "Harina" {
		t.Errorf("what it recorded is %v, want the line of the flour", confirmation.Recorded)
	}
	if len(confirmation.NotUnderstood) != 1 {
		t.Fatalf("what it did not understand is %v, want the ambiguity of the flour", confirmation.NotUnderstood)
	}
	if confirmation.NotUnderstood[0].Name != "harina" {
		t.Errorf("the name of the ambiguity is %q, want %q", confirmation.NotUnderstood[0].Name, "harina")
	}
	if len(confirmation.StillNeeds) != 0 {
		t.Errorf("what it still needs is %v, want nothing", confirmation.StillNeeds)
	}
}

// TestTheConfirmationOfAnExpiredDraftIsRefused proves the rule of the
// domain: the price of a market does not wait, and a draft that expired is
// not confirmed.
func TestTheConfirmationOfAnExpiredDraftIsRefused(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTest(t)

	_, err := Confirm(draft, Yes{Revision: 1, Word: "sí"}, theDay(2026, time.March, 9))
	if !errors.Is(err, ErrExpiredDraft) {
		t.Fatalf("the error is %v, want %v", err, ErrExpiredDraft)
	}
}

// TestTheConfirmationOnTheDayOfTheExpiryIsAccepted proves the rule of the
// domain: the draft waits until the day it expires, and it is refused the
// day after.
func TestTheConfirmationOnTheDayOfTheExpiryIsAccepted(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTest(t)

	if _, err := Confirm(draft, Yes{Revision: 1, Word: "sí"}, draft.ExpiresOn); err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}
}

// TestTheConfirmationOfAnOldRevisionIsRefused proves the rule of the
// domain: a draft that changed under the eyes of the owner is refused.
func TestTheConfirmationOfAnOldRevisionIsRefused(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTest(t)

	_, err := Confirm(draft, Yes{Revision: 2, Word: "sí"}, theDay(2026, time.March, 2))
	if !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("the error is %v, want %v", err, ErrStaleRevision)
	}
}

// TestTheConfirmationOfADraftThatIsNotProposedIsRefused proves the rule of
// the domain: a draft waits for the yes of the owner once, and a draft
// that is already confirmed is never confirmed again.
func TestTheConfirmationOfADraftThatIsNotProposedIsRefused(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTest(t)
	draft.State = StateConfirmed

	_, err := Confirm(draft, Yes{Revision: 1, Word: "sí"}, theDay(2026, time.March, 2))
	if !errors.Is(err, ErrNotProposed) {
		t.Fatalf("the error is %v, want %v", err, ErrNotProposed)
	}
}

// TestTheConfirmationWithoutTheWordIsRefused proves the rule of the
// domain: the word the owner used to confirm is kept with the record, and
// a confirmation without it is refused.
func TestTheConfirmationWithoutTheWordIsRefused(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTest(t)

	_, err := Confirm(draft, Yes{Revision: 1, Word: "   "}, theDay(2026, time.March, 2))
	if !errors.Is(err, ErrMissingConfirmation) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingConfirmation)
	}
}

// TestTheConfirmationOfADraftThatStillNeedsTheMoneyIsRefused proves the
// rule of the domain: the money is never inferred, and a purchase without
// the money of its lines is not a purchase.
func TestTheConfirmationOfADraftThatStillNeedsTheMoneyIsRefused(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(3, "unidad")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	unpriced, err := NewLine("Pan", measure, Price{})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	draft := theDraftOfTheTest(t)
	draft.Lines = append(draft.Lines, unpriced)

	_, err = Confirm(draft, Yes{Revision: 1, Word: "sí"}, theDay(2026, time.March, 2))
	if !errors.Is(err, ErrStillNeeds) {
		t.Fatalf("the error is %v, want %v", err, ErrStillNeeds)
	}
}

// TestTheMoneyOfTheBookCountsWholePesos proves the rule of the book: the
// money of a purchase is whole pesos, and a price in centavos is rounded to
// the nearest peso when it enters the book.
func TestTheMoneyOfTheBookCountsWholePesos(t *testing.T) {
	t.Parallel()

	measure, err := NewMeasure(1.5, "kg")
	if err != nil {
		t.Fatalf("the measure was refused: %v", err)
	}

	line, err := NewLine("Harina", measure, Price{Unit: 333})
	if err != nil {
		t.Fatalf("the line was refused: %v", err)
	}

	draft, err := NewDraft(business.ID(77), "Compré harina", Candidate{
		Lines: []Line{line},
	}, theDay(2026, time.March, 1))
	if err != nil {
		t.Fatalf("the draft was refused: %v", err)
	}

	confirmation, err := Confirm(draft, Yes{Revision: 1, Word: "sí"}, theDay(2026, time.March, 2))
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	// 333 pesos the kilo of one kilo and a half are 499 pesos and a half,
	// and the book counts whole pesos.
	if confirmation.Purchase.Lines[0].Price.Total != 500 {
		t.Errorf("the money of the line is %d, want 500", confirmation.Purchase.Lines[0].Price.Total)
	}
	if confirmation.Purchase.Total != 500 {
		t.Errorf("the money of the purchase is %d, want 500", confirmation.Purchase.Total)
	}
}

// TestTheMovementOfThePurchase proves the rule of the domain: the money of
// the payment comes from the account that pays and it lands in the
// account of the inventory of the business.
func TestTheMovementOfThePurchase(t *testing.T) {
	t.Parallel()

	draft := theDraftOfTheTest(t)

	confirmation, err := Confirm(draft, Yes{Revision: 1, Word: "sí"}, theDay(2026, time.March, 2))
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	movement, err := confirmation.Purchase.Movement(account.ID(3), account.ID(4))
	if err != nil {
		t.Fatalf("the movement was refused: %v", err)
	}

	if movement.BusinessID != business.ID(77) {
		t.Errorf("the business of the movement is %d, want 77", movement.BusinessID)
	}
	if movement.CreditAccountID != account.ID(3) {
		t.Errorf("the money comes from the account %d, want 3", movement.CreditAccountID)
	}
	if movement.DebitAccountID != account.ID(4) {
		t.Errorf("the money lands in the account %d, want 4", movement.DebitAccountID)
	}
	if movement.Amount != 12000 {
		t.Errorf("the money that moved is %d, want 12000", movement.Amount)
	}
	if movement.OccurredOn != theDay(2026, time.February, 28) {
		t.Errorf("the day of the movement is %v, want the day of the purchase", movement.OccurredOn)
	}
	if movement.Description != "Compré 25 kilos de harina" {
		t.Errorf("the description is %q, want the sentence of the owner", movement.Description)
	}
}

// TestTheMovementOfThePurchaseWithoutMoneyIsRefused proves that a purchase
// without money moves no money: the book counts what it moves.
func TestTheMovementOfThePurchaseWithoutMoneyIsRefused(t *testing.T) {
	t.Parallel()

	empty := Purchase{BusinessID: business.ID(77), Day: theDay(2026, time.March, 1)}

	_, err := empty.Movement(account.ID(3), account.ID(4))
	if !errors.Is(err, ledger.ErrInvalidAmount) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidAmount)
	}
}
