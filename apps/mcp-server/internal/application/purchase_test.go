package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The fake of the storage of the drafts.
//
// It remembers the drafts it wrote and the drafts it was asked for: the
// use case reads a draft before it amends it, and the test proves that it
// did.
type fakeDrafts struct {
	drafts []purchase.Draft
	read   int
	err    error
}

// Save writes the draft and remembers it.
//
// A draft without an identity is written for the first time, and a draft
// with one replaces the row it names.
func (f *fakeDrafts) Save(_ context.Context, draft purchase.Draft) (purchase.Draft, error) {
	if f.err != nil {
		return purchase.Draft{}, f.err
	}

	written := draft
	if written.ID == 0 {
		written.ID = purchase.ID(55)
		f.drafts = append(f.drafts, written)
		return written, nil
	}

	for index, one := range f.drafts {
		if one.ID == written.ID {
			f.drafts[index] = written
			return written, nil
		}
	}

	f.drafts = append(f.drafts, written)

	return written, nil
}

// Find answers the draft of the identity, and remembers that it was asked.
func (f *fakeDrafts) Find(_ context.Context, id purchase.ID) (purchase.Draft, error) {
	f.read++

	if f.err != nil {
		return purchase.Draft{}, f.err
	}

	for _, one := range f.drafts {
		if one.ID == id {
			return one, nil
		}
	}

	return purchase.Draft{}, purchase.ErrNotFound
}

// theLine is a line the tests of the draft send to the use case.
func theLine(name string, quantity float64, unit string, total int64) ports.PurchaseLine {
	return ports.PurchaseLine{
		Name:     name,
		Quantity: quantity,
		Unit:     unit,
		Total:    total,
	}
}

// TestTheDraftIsKeptAndTheBookIsNotWritten proves that the use case keeps
// the candidate the agent read as a draft of the business, and that it
// writes nothing else.
func TestTheDraftIsKeptAndTheBookIsNotWritten(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	drafted, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	if len(drafts.drafts) != 1 {
		t.Fatalf("the storage holds %d drafts, want 1", len(drafts.drafts))
	}

	kept := drafts.drafts[0]
	if kept.ID != purchase.ID(55) {
		t.Errorf("the identity of the draft is %d, want 55", kept.ID)
	}
	if kept.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", kept.BusinessID)
	}
	if kept.Utterance != "Compré 25 kilos de harina" {
		t.Errorf("the sentence is %q, want %q", kept.Utterance, "Compré 25 kilos de harina")
	}
	if kept.State != purchase.StateProposed {
		t.Errorf("the state is %q, want %q", kept.State, purchase.StateProposed)
	}
	if kept.Revision != 1 {
		t.Errorf("the revision is %d, want 1", kept.Revision)
	}
	if len(kept.Lines) != 1 || kept.Lines[0].Name != "Harina" {
		t.Errorf("the lines of the draft are %v, want the line of the flour", kept.Lines)
	}
	if kept.Day != ledger.NewDate(theClock().Now()) {
		t.Errorf("the day is %v, want the day of today", kept.Day)
	}
	if kept.ExpiresOn != ledger.NewDate(theClock().Now().AddDate(0, 0, purchase.DraftLifetimeDays)) {
		t.Errorf("the day of the expiry is %v, want seven days later", kept.ExpiresOn)
	}
	if drafted.ID != purchase.ID(55) {
		t.Errorf("the answer holds the draft %d, want 55", drafted.ID)
	}
}

// TestTheDraftTheOwnerNamedTheDayOfKeepsThatDay proves that the day the
// owner named is the day of the draft, and that Paca filled nothing.
func TestTheDraftTheOwnerNamedTheDayOfKeepsThatDay(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	drafted, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Ayer compré harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
		Day:        "2026-03-14",
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	if drafted.Day.String() != "2026-03-14" {
		t.Errorf("the day is %v, want 2026-03-14", drafted.Day)
	}
	if len(drafted.Inferred) != 0 {
		t.Errorf("Paca filled %v, want nothing", drafted.Inferred)
	}
}

// TestTheDayThatIsNotADayIsRefused proves that the day of the purchase is
// written the way a person writes it, and that a day that is not a day is
// refused before anything is written.
func TestTheDayThatIsNotADayIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
		Day:        "el sábado",
	})
	if !errors.Is(err, ledger.ErrInvalidDate) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidDate)
	}

	if len(drafts.drafts) != 0 {
		t.Errorf("the draft was left behind: the storage holds %d drafts", len(drafts.drafts))
	}
}

// TestTheLineWithoutAUnitIsRefused proves that the rule of the domain
// reaches the use case: the quantity and its unit travel together.
func TestTheLineWithoutAUnitIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "", 12000)},
	})
	if !errors.Is(err, purchase.ErrMissingUnit) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrMissingUnit)
	}

	if len(drafts.drafts) != 0 {
		t.Errorf("the draft was left behind: the storage holds %d drafts", len(drafts.drafts))
	}
}

// TestThePriceThatWasInferredIsRefused proves the rule of the domain: the
// money is never inferred, and the refused call leaves nothing behind.
func TestThePriceThatWasInferredIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	inferred := theLine("Harina", 25, "kg", 12000)
	inferred.Inferred = true

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina",
		Lines:      []ports.PurchaseLine{inferred},
	})
	if !errors.Is(err, purchase.ErrInferredPrice) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrInferredPrice)
	}

	if len(drafts.drafts) != 0 {
		t.Errorf("the draft was left behind: the storage holds %d drafts", len(drafts.drafts))
	}
}

// TestTheDraftWithoutASentenceIsRefused proves that a draft without the
// sentence of the owner is refused, and that nothing is written.
func TestTheDraftWithoutASentenceIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "   ",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if !errors.Is(err, purchase.ErrMissingUtterance) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrMissingUtterance)
	}

	if len(drafts.drafts) != 0 {
		t.Errorf("the draft was left behind: the storage holds %d drafts", len(drafts.drafts))
	}
}

// TestTheLineWithoutAPriceIsWhatTheDraftStillNeeds proves that the money
// is never inferred: the line without a price is what the draft needs.
func TestTheLineWithoutAPriceIsWhatTheDraftStillNeeds(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	drafted, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina y pan",
		Lines: []ports.PurchaseLine{
			theLine("Harina", 25, "kg", 12000),
			theLine("Pan", 3, "unidad", 0),
		},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	if len(drafted.Missing) != 1 {
		t.Fatalf("the draft needs %v, want the price of the bread", drafted.Missing)
	}
	if drafted.Missing[0] != "the price of Pan" {
		t.Errorf("the missing thing is %q, want %q", drafted.Missing[0], "the price of Pan")
	}
}

// TestTheAmbiguityIsKeptWithItsOptions proves that the ambiguity the agent
// found is kept with the options it offers: Paca never asks an open
// question.
func TestTheAmbiguityIsKeptWithItsOptions(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	drafted, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
		Ambiguities: []ports.PurchaseAmbiguity{
			{Name: "harina", Options: []string{"Harina panadera", "Harina de trigo"}},
		},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	if len(drafted.Ambiguous) != 1 {
		t.Fatalf("the draft holds %d ambiguities, want 1", len(drafted.Ambiguous))
	}
	if len(drafted.Ambiguous[0].Options) != 2 {
		t.Errorf("the ambiguity holds %v, want the two options", drafted.Ambiguous[0].Options)
	}
}

// TestTheAmbiguityWithoutOptionsIsRefused proves that an ambiguity without
// options is refused, and that nothing is written.
func TestTheAmbiguityWithoutOptionsIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
		Ambiguities: []ports.PurchaseAmbiguity{
			{Name: "harina"},
		},
	})
	if !errors.Is(err, purchase.ErrNoOptions) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNoOptions)
	}

	if len(drafts.drafts) != 0 {
		t.Errorf("the draft was left behind: the storage holds %d drafts", len(drafts.drafts))
	}
}

// TestTheAmendmentGrowsTheRevision proves that the amendment of a draft is
// the same draft, one revision further, and that the use case reads the
// draft before it amends it.
func TestTheAmendmentGrowsTheRevision(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	first, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	amended, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		DraftID:    first.ID,
		Utterance:  "Compré 30 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 30, "kg", 14400)},
	})
	if err != nil {
		t.Fatalf("the amendment could not be kept: %v", err)
	}

	if drafts.read != 1 {
		t.Errorf("the storage was read %d times, want 1", drafts.read)
	}
	if amended.ID != first.ID {
		t.Errorf("the amended draft is %d, want %d", amended.ID, first.ID)
	}
	if amended.Revision != 2 {
		t.Errorf("the revision is %d, want 2", amended.Revision)
	}
	if amended.Utterance != first.Utterance {
		t.Errorf("the sentence is %q, want %q", amended.Utterance, first.Utterance)
	}
	if len(drafts.drafts) != 1 {
		t.Errorf("the storage holds %d drafts, want 1", len(drafts.drafts))
	}
}

// TestTheAmendmentOfADraftOfAnotherBusinessIsRefused proves that a draft
// belongs to its business, and that the refused call leaves nothing behind.
func TestTheAmendmentOfADraftOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	first, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	_, err = service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(88),
		DraftID:    first.ID,
		Utterance:  "Compré 30 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 30, "kg", 14400)},
	})
	if !errors.Is(err, purchase.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrWrongBusiness)
	}

	if len(drafts.drafts) != 1 {
		t.Errorf("the storage holds %d drafts, want 1", len(drafts.drafts))
	}
	if drafts.drafts[0].Revision != 1 {
		t.Errorf("the revision of the draft is %d, want 1", drafts.drafts[0].Revision)
	}
}

// TestTheDraftOfADraftThatIsNotThereIsRefused proves that the amendment of
// a draft that does not exist is refused.
func TestTheDraftOfADraftThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		DraftID:    purchase.ID(999),
		Utterance:  "Compré 30 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 30, "kg", 14400)},
	})
	if !errors.Is(err, purchase.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNotFound)
	}
}

// TestTheDraftWithoutADatabaseSaysSo proves that a service without a
// database says so, and it says it in the only capability it has.
func TestTheDraftWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewPurchaseService(nil, theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
