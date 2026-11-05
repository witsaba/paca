package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The fake of the storage of the drafts.
//
// It remembers the drafts it wrote and the drafts it was asked for: the
// use case reads a draft before it amends it, and the test proves that it
// did.
type fakeDrafts struct {
	drafts   []purchase.Draft
	read     int
	listed   int
	lastList business.ID
	err      error
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
		written.ID = purchase.DraftID(55)
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
func (f *fakeDrafts) Find(_ context.Context, id purchase.DraftID) (purchase.Draft, error) {
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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	if kept.ID != purchase.DraftID(55) {
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
	if drafted.ID != purchase.DraftID(55) {
		t.Errorf("the answer holds the draft %d, want 55", drafted.ID)
	}
}

// TestTheDraftTheOwnerNamedTheDayOfKeepsThatDay proves that the day the
// owner named is the day of the draft, and that Paca filled nothing.
func TestTheDraftTheOwnerNamedTheDayOfKeepsThatDay(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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

// TestTheUnitOfTheLineIsTheUnitOfTheIngredient proves the rule of the
// use case: the unit of a line is the unit the owner named, and the unit
// of the ingredient of the catalog when she named none.
func TestTheUnitOfTheLineIsTheUnitOfTheIngredient(t *testing.T) {
	t.Parallel()

	flour := anIngredientOfTheCatalog(t, 10, "Harina panadera", "harina")
	completed, err := ingredient.Complete(flour, ingredient.Declaration{Unit: ingredient.UnitKilogram})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	drafts := &fakeDrafts{}
	catalog := &fakeIngredients{items: []ingredient.Ingredient{completed}}
	service := application.NewPurchaseService(drafts, catalog, theClock())

	drafted, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines: []ports.PurchaseLine{{
			Name:         "Harina panadera",
			Quantity:     25,
			IngredientID: 10,
			Total:        12000,
		}},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	if len(drafted.Lines) != 1 {
		t.Fatalf("the draft holds %d lines, want 1", len(drafted.Lines))
	}
	if drafted.Lines[0].Measure.Unit != "kg" {
		t.Errorf("the unit of the line is %q, want %q", drafted.Lines[0].Measure.Unit, "kg")
	}
}

// TestTheLineThatEndsWithoutAUnitIsRefused proves that the domain still
// refuses a line that ends without a unit: the ingredient the catalog does
// not know carries none.
func TestTheLineThatEndsWithoutAUnitIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	catalog := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera"),
	}}
	service := application.NewPurchaseService(drafts, catalog, theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina",
		Lines:      []ports.PurchaseLine{theLine("Polvasa", 1, "", 3000)},
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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

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
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		DraftID:    purchase.DraftID(999),
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

	service := application.NewPurchaseService(nil, theCatalog(), theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// Confirm moves the draft to the confirmed state, and remembers it.
func (f *fakeDrafts) Confirm(_ context.Context, id purchase.DraftID, revision int) (purchase.Draft, error) {
	if f.err != nil {
		return purchase.Draft{}, f.err
	}

	for index, one := range f.drafts {
		if one.ID != id {
			continue
		}

		if one.Revision != revision {
			return purchase.Draft{}, fmt.Errorf("%w: the draft is at the revision %d", purchase.ErrStaleRevision, one.Revision)
		}

		f.drafts[index].State = purchase.StateConfirmed
		return f.drafts[index], nil
	}

	return purchase.Draft{}, purchase.ErrNotFound
}

// Cancel moves the draft to the cancelled state, and remembers it.
//
// The fake holds the rule of the table: only a draft that waits is
// cancelled.
func (f *fakeDrafts) Cancel(_ context.Context, id purchase.DraftID) (purchase.Draft, error) {
	if f.err != nil {
		return purchase.Draft{}, f.err
	}

	for index, one := range f.drafts {
		if one.ID != id {
			continue
		}

		if one.State != purchase.StateProposed {
			return purchase.Draft{}, fmt.Errorf("%w: the draft is in the state %q",
				purchase.ErrNotCancellable, one.State)
		}

		f.drafts[index].State = purchase.StateCancelled
		return f.drafts[index], nil
	}

	return purchase.Draft{}, purchase.ErrNotFound
}

// List answers the drafts of the business that wait, and remembers that it
// was asked.
//
// The fake holds the rule of the table: the drafts of the other businesses
// are not listed, and a draft that ended is not listed either.
func (f *fakeDrafts) List(_ context.Context, own business.ID) ([]purchase.Draft, error) {
	f.listed++
	f.lastList = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]purchase.Draft, 0, len(f.drafts))
	for _, one := range f.drafts {
		if one.BusinessID == own && one.State == purchase.StateProposed {
			held = append(held, one)
		}
	}

	return held, nil
}

// The fake of the storage of the purchases.
//
// It remembers the purchases it wrote and the lines they carry.
type fakePurchases struct {
	purchases []purchase.Purchase
	err       error
}

// Record writes the purchase and remembers it.
func (f *fakePurchases) Record(_ context.Context, written purchase.Purchase) (purchase.Purchase, error) {
	if f.err != nil {
		return purchase.Purchase{}, f.err
	}

	written.ID = purchase.ID(9)
	f.purchases = append(f.purchases, written)

	return written, nil
}

// TestTheDraftCarriesTheIdentityOfTheIngredientTheAgentChose proves that
// the draft keeps the identity of the ingredient the agent chose with the
// tool of the catalog, and that the name is the name the owner said.
func TestTheDraftCarriesTheIdentityOfTheIngredientTheAgentChose(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	line := theLine("Harina", 25, "kg", 12000)
	line.IngredientID = 10

	drafted, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{line},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	if len(drafted.Lines) != 1 {
		t.Fatalf("the draft holds %d lines, want 1", len(drafted.Lines))
	}
	if drafted.Lines[0].IngredientID != ingredient.ID(10) {
		t.Errorf("the identity of the ingredient is %d, want 10", drafted.Lines[0].IngredientID)
	}
	if drafted.Lines[0].Name != "Harina" {
		t.Errorf("the name of the line is %q, want %q", drafted.Lines[0].Name, "Harina")
	}
}

// TestTheDraftOfANameTheCatalogDoesNotKnowKeepsTheName proves that a name
// the catalog does not know is not an error: the draft keeps the name as
// the owner said it.
func TestTheDraftOfANameTheCatalogDoesNotKnowKeepsTheName(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	drafted, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	if drafted.Lines[0].Name != "Harina" {
		t.Errorf("the name of the line is %q, want %q", drafted.Lines[0].Name, "Harina")
	}
	if drafted.Lines[0].IngredientID != ingredient.ID(0) {
		t.Errorf("the identity of the ingredient is %d, want none", drafted.Lines[0].IngredientID)
	}
}

// TestTheConfirmationGrowsTheCatalogWithTheNameSheSaid proves that the
// catalog of the business grows with what the owner buys: a name the
// catalog does not know becomes an ingredient of the business, recorded as
// she said it, and the line of the purchase carries its identity.
func TestTheConfirmationGrowsTheCatalogWithTheNameSheSaid(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{}

	draft := theDraftOfTheBakery(t, drafts)

	_, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	if len(ingredients.written) != 1 {
		t.Fatalf("the catalog holds %d new ingredients, want 1", len(ingredients.written))
	}
	if ingredients.written[0].Name != "Harina" {
		t.Errorf("the name of the ingredient is %q, want %q", ingredients.written[0].Name, "Harina")
	}
	if ingredients.written[0].BusinessID != business.ID(77) {
		t.Errorf("the business of the ingredient is %d, want 77", ingredients.written[0].BusinessID)
	}

	written := purchases.purchases[0]
	if len(written.Lines) != 1 {
		t.Fatalf("the purchase holds %d lines, want 1", len(written.Lines))
	}
	if written.Lines[0].IngredientID != ingredients.written[0].ID {
		t.Errorf("the identity of the ingredient is %d, want %d",
			written.Lines[0].IngredientID, ingredients.written[0].ID)
	}
	if written.Lines[0].Name != "Harina" {
		t.Errorf("the name of the line is %q, want %q", written.Lines[0].Name, "Harina")
	}
}

// TestTheConfirmationKeepsTheIngredientTheCatalogKnows proves that a name
// the catalog knows is the ingredient of the line, and that the catalog
// does not grow with it.
func TestTheConfirmationKeepsTheIngredientTheCatalogKnows(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina", "harina"),
	}}

	draft := theDraftOfTheBakery(t, drafts)

	_, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	if len(ingredients.written) != 0 {
		t.Errorf("the catalog grew %v, want the name it already knew", ingredients.written)
	}

	written := purchases.purchases[0]
	if len(written.Lines) != 1 {
		t.Fatalf("the purchase holds %d lines, want 1", len(written.Lines))
	}
	if written.Lines[0].IngredientID != ingredient.ID(10) {
		t.Errorf("the identity of the ingredient is %d, want 10", written.Lines[0].IngredientID)
	}
}

// The fake of the transaction of the storage.
//
// It keeps what the work wrote, and it gives it back when the work fails: a
// transaction that rolls back leaves the storage as it was.
type fakeConfirmationTransaction struct {
	purchases   *fakePurchases
	book        *fakeBook
	drafts      *fakeDrafts
	ingredients *fakeIngredients
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakeConfirmationTransaction) Do(ctx context.Context, work func(context.Context) error) error {
	purchases := len(f.purchases.purchases)
	movements := len(f.book.recorded)
	written := len(f.ingredients.written)
	states := make([]purchase.State, 0, len(f.drafts.drafts))
	for _, one := range f.drafts.drafts {
		states = append(states, one.State)
	}

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.purchases.purchases = f.purchases.purchases[:purchases]
	f.book.recorded = f.book.recorded[:movements]
	f.ingredients.written = f.ingredients.written[:written]
	for index := range f.drafts.drafts {
		f.drafts.drafts[index].State = states[index]
	}

	return err
}

// expiredClock is the clock of a day after the expiry of the draft of the
// test: the price of a market does not wait.
func expiredClock() *fakeBookClock {
	return &fakeBookClock{moment: time.Date(2026, time.March, 25, 9, 0, 0, 0, time.UTC)}
}

// theAccountsOfTheBakery answers the accounts of the book of the bakery.
func theAccountsOfTheBakery(t *testing.T) []account.Account {
	t.Helper()

	seed := account.DefaultBusinessAccounts()
	accounts := make([]account.Account, 0, len(seed))

	for index, one := range seed {
		created, err := account.New(business.ID(77), one.Kind, one.Name, "", nil)
		if err != nil {
			t.Fatalf("the account was refused: %v", err)
		}

		created.ID = account.ID(index + 1)
		accounts = append(accounts, created)
	}

	return accounts
}

// theDraftOfTheBakery keeps the draft of the test as a draft of the
// business.
func theDraftOfTheBakery(t *testing.T, drafts *fakeDrafts) purchase.Draft {
	t.Helper()

	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	drafted, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
		Market:     "Plaza de Mercado",
		Invoice:    "FV-001",
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	return drafted
}

// theConfirmation builds the use case of the confirmation with the fakes of
// the test.
func theConfirmation(
	drafts *fakeDrafts,
	purchases *fakePurchases,
	accounts *fakeAccounts,
	book *fakeBook,
	ingredients *fakeIngredients,
	clock ports.Clock,
) *application.ConfirmationService {
	work := &fakeConfirmationTransaction{
		purchases:   purchases,
		book:        book,
		drafts:      drafts,
		ingredients: ingredients,
	}

	return application.NewConfirmationService(drafts, purchases, accounts, book, ingredients, work, clock)
}

// TestTheConfirmationWritesThePurchaseAndTheMovementOfTheMoney proves that
// the confirmation writes the purchase, that the money of the payment moves
// from the cash to the inventory, and that the draft stops waiting.
func TestTheConfirmationWritesThePurchaseAndTheMovementOfTheMoney(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{}

	draft := theDraftOfTheBakery(t, drafts)

	confirmation, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	if len(purchases.purchases) != 1 {
		t.Fatalf("the storage holds %d purchases, want 1", len(purchases.purchases))
	}

	written := purchases.purchases[0]
	if written.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", written.BusinessID)
	}
	if written.DraftID != draft.ID {
		t.Errorf("the draft of the purchase is %d, want %d", written.DraftID, draft.ID)
	}
	if written.Utterance != "Compré 25 kilos de harina" {
		t.Errorf("the sentence is %q, want %q", written.Utterance, "Compré 25 kilos de harina")
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if written.Total != 12000 {
		t.Errorf("the money of the purchase is %d, want 12000", written.Total)
	}
	if len(written.Lines) != 1 || written.Lines[0].Name != "Harina" {
		t.Errorf("the lines of the purchase are %v, want the line of the flour", written.Lines)
	}

	if len(book.recorded) != 1 {
		t.Fatalf("the book holds %d movements, want 1", len(book.recorded))
	}

	movement := book.recorded[0]
	if movement.Amount != 12000 {
		t.Errorf("the money that moved is %d, want 12000", movement.Amount)
	}
	if movement.DebitAccountID != account.ID(3) {
		t.Errorf("the money lands in the account %d, want the inventory 3", movement.DebitAccountID)
	}
	if movement.CreditAccountID != account.ID(1) {
		t.Errorf("the money comes from the account %d, want the cash 1", movement.CreditAccountID)
	}
	if movement.OccurredOn != ledger.NewDate(theClock().Now()) {
		t.Errorf("the day of the movement is %v, want the day of the purchase", movement.OccurredOn)
	}

	if drafts.drafts[0].State != purchase.StateConfirmed {
		t.Errorf("the state of the draft is %q, want %q", drafts.drafts[0].State, purchase.StateConfirmed)
	}

	if len(confirmation.Recorded) != 1 {
		t.Errorf("what it recorded is %v, want the line of the flour", confirmation.Recorded)
	}
	if len(confirmation.NotUnderstood) != 0 {
		t.Errorf("what it did not understand is %v, want nothing", confirmation.NotUnderstood)
	}
	if len(confirmation.StillNeeds) != 0 {
		t.Errorf("what it still needs is %v, want nothing", confirmation.StillNeeds)
	}
}

// TestTheMoneyComesFromTheAccountTheOwnerNames proves that the owner names
// the account that pays, and that the money comes from it.
func TestTheMoneyComesFromTheAccountTheOwnerNames(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{}

	draft := theDraftOfTheBakery(t, drafts)

	_, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
		FromAccountID: account.ID(2),
	})
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	if len(book.recorded) != 1 {
		t.Fatalf("the book holds %d movements, want 1", len(book.recorded))
	}
	if book.recorded[0].CreditAccountID != account.ID(2) {
		t.Errorf("the money comes from the account %d, want the bank 2", book.recorded[0].CreditAccountID)
	}
}

// TestTheConfirmationOfADraftOfAnotherBusinessIsRefused proves that a draft
// belongs to its business, and that the refused confirmation leaves nothing
// behind.
func TestTheConfirmationOfADraftOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{}

	draft := theDraftOfTheBakery(t, drafts)

	_, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(88),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, purchase.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrWrongBusiness)
	}

	if len(purchases.purchases) != 0 {
		t.Errorf("the purchase was left behind: the storage holds %d purchases", len(purchases.purchases))
	}
	if len(book.recorded) != 0 {
		t.Errorf("the movement was left behind: the book holds %d movements", len(book.recorded))
	}
	if drafts.drafts[0].State != purchase.StateProposed {
		t.Errorf("the state of the draft is %q, want %q", drafts.drafts[0].State, purchase.StateProposed)
	}
}

// TestTheConfirmationOfAnExpiredDraftIsRefused proves that the price of a
// market does not wait, and that the refused confirmation leaves nothing
// behind.
func TestTheConfirmationOfAnExpiredDraftIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{}

	draft := theDraftOfTheBakery(t, drafts)

	_, err := theConfirmation(drafts, purchases, accounts, book, ingredients, expiredClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, purchase.ErrExpiredDraft) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrExpiredDraft)
	}

	if len(purchases.purchases) != 0 {
		t.Errorf("the purchase was left behind: the storage holds %d purchases", len(purchases.purchases))
	}
	if len(book.recorded) != 0 {
		t.Errorf("the movement was left behind: the book holds %d movements", len(book.recorded))
	}
	if drafts.drafts[0].State != purchase.StateProposed {
		t.Errorf("the state of the draft is %q, want %q", drafts.drafts[0].State, purchase.StateProposed)
	}
}

// TestTheConfirmationOfAnOldRevisionIsRefused proves that a draft that
// changed under the eyes of the owner is refused, and that the refused
// confirmation leaves nothing behind.
func TestTheConfirmationOfAnOldRevisionIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{}

	draft := theDraftOfTheBakery(t, drafts)

	_, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision + 1,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, purchase.ErrStaleRevision) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrStaleRevision)
	}

	if len(purchases.purchases) != 0 {
		t.Errorf("the purchase was left behind: the storage holds %d purchases", len(purchases.purchases))
	}
	if len(book.recorded) != 0 {
		t.Errorf("the movement was left behind: the book holds %d movements", len(book.recorded))
	}
	if drafts.drafts[0].State != purchase.StateProposed {
		t.Errorf("the state of the draft is %q, want %q", drafts.drafts[0].State, purchase.StateProposed)
	}
}

// TestTheConfirmationOfADraftThatStillNeedsTheMoneyIsRefused proves that
// the money is never inferred, and that the refused confirmation leaves
// nothing behind.
func TestTheConfirmationOfADraftThatStillNeedsTheMoneyIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{}

	draft := theDraftOfTheBakery(t, drafts)
	draft.Lines = append(draft.Lines, purchase.Line{
		Name:    "Pan",
		Measure: purchase.Measure{Quantity: 3, Unit: "unidad"},
	})
	if _, err := drafts.Save(t.Context(), draft); err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	_, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, purchase.ErrStillNeeds) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrStillNeeds)
	}

	if len(purchases.purchases) != 0 {
		t.Errorf("the purchase was left behind: the storage holds %d purchases", len(purchases.purchases))
	}
	if len(book.recorded) != 0 {
		t.Errorf("the movement was left behind: the book holds %d movements", len(book.recorded))
	}
}

// TestTheFailureOfTheBookLeavesNothingBehind proves that the purchase and
// the movement of the money are written in one transaction: a movement that
// fails takes the purchase with it.
func TestTheFailureOfTheBookLeavesNothingBehind(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{err: ledger.ErrNoStorage}
	ingredients := &fakeIngredients{}

	draft := theDraftOfTheBakery(t, drafts)

	_, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, ledger.ErrNoStorage) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrNoStorage)
	}

	if len(purchases.purchases) != 0 {
		t.Errorf("the purchase was left behind: the storage holds %d purchases", len(purchases.purchases))
	}
	if drafts.drafts[0].State != purchase.StateProposed {
		t.Errorf("the state of the draft is %q, want %q", drafts.drafts[0].State, purchase.StateProposed)
	}
}

// TestTheConfirmationWithoutADatabaseSaysSo proves that a service without a
// database says so, and it says it in the only capability it has.
func TestTheConfirmationWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewConfirmationService(nil, nil, nil, nil, nil, nil, theClock())

	_, err := service.Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       purchase.DraftID(55),
		Revision:      1,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// TestTheDraftIsCancelled proves that the owner drops the draft she does
// not want, and that the draft of the business stops waiting.
func TestTheDraftIsCancelled(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	first, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	cancelled, err := service.Cancel(t.Context(), ports.CancelDraft{
		BusinessID: business.ID(77),
		DraftID:    first.ID,
	})
	if err != nil {
		t.Fatalf("the draft could not be cancelled: %v", err)
	}

	if cancelled.State != purchase.StateCancelled {
		t.Errorf("the state of the answer is %q, want %q", cancelled.State, purchase.StateCancelled)
	}
	if cancelled.ID != first.ID {
		t.Errorf("the identity of the draft is %d, want %d", cancelled.ID, first.ID)
	}
	if drafts.drafts[0].State != purchase.StateCancelled {
		t.Errorf("the state of the draft is %q, want %q", drafts.drafts[0].State, purchase.StateCancelled)
	}
}

// TestTheCancellationOfADraftOfAnotherBusinessIsRefused proves that a
// draft belongs to its business, and that the refused call leaves the
// draft as it was.
func TestTheCancellationOfADraftOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	first, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	_, err = service.Cancel(t.Context(), ports.CancelDraft{
		BusinessID: business.ID(88),
		DraftID:    first.ID,
	})
	if !errors.Is(err, purchase.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrWrongBusiness)
	}

	if drafts.drafts[0].State != purchase.StateProposed {
		t.Errorf("the state of the draft is %q, want %q", drafts.drafts[0].State, purchase.StateProposed)
	}
}

// TestTheCancellationOfADraftThatIsNotThereIsRefused proves that a draft
// that does not exist is not cancelled.
func TestTheCancellationOfADraftThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	_, err := service.Cancel(t.Context(), ports.CancelDraft{
		BusinessID: business.ID(77),
		DraftID:    purchase.DraftID(999),
	})
	if !errors.Is(err, purchase.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNotFound)
	}
}

// TestTheCancellationOfADraftThatIsConfirmedIsRefused proves the rule of
// the state machine: a draft that is confirmed already does not wait any
// more, and the refused call leaves it as it was.
func TestTheCancellationOfADraftThatIsConfirmedIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	first, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	drafts.drafts[0].State = purchase.StateConfirmed

	_, err = service.Cancel(t.Context(), ports.CancelDraft{
		BusinessID: business.ID(77),
		DraftID:    first.ID,
	})
	if !errors.Is(err, purchase.ErrNotCancellable) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNotCancellable)
	}

	if drafts.drafts[0].State != purchase.StateConfirmed {
		t.Errorf("the state of the draft is %q, want %q", drafts.drafts[0].State, purchase.StateConfirmed)
	}
}

// TestTheCancellationOfAnExpiredDraftIsRefused proves the rule of the
// domain: the price of a market does not wait, and an expired draft is
// dead.
func TestTheCancellationOfAnExpiredDraftIsRefused(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	first, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	_, err = application.NewPurchaseService(drafts, theCatalog(), expiredClock()).Cancel(t.Context(), ports.CancelDraft{
		BusinessID: business.ID(77),
		DraftID:    first.ID,
	})
	if !errors.Is(err, purchase.ErrExpiredDraft) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrExpiredDraft)
	}

	if drafts.drafts[0].State != purchase.StateProposed {
		t.Errorf("the state of the draft is %q, want %q", drafts.drafts[0].State, purchase.StateProposed)
	}
}

// TestThePendingDraftsAreReadWithWhatTheyNeed proves that the pending
// drafts of a business are read back with what each one still needs, with
// what each one was not sure about and with what Paca filled.
func TestThePendingDraftsAreReadWithWhatTheyNeed(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré harina y pan",
		Lines: []ports.PurchaseLine{
			theLine("Harina", 25, "kg", 12000),
			theLine("Pan", 3, "unidad", 0),
		},
		Ambiguities: []ports.PurchaseAmbiguity{
			{Name: "harina", Options: []string{"Harina panadera", "Harina de trigo"}},
		},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	read, err := service.Read(t.Context(), ports.ReadDrafts{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the drafts could not be read: %v", err)
	}

	if drafts.lastList != business.ID(77) {
		t.Errorf("the drafts that were read are the ones of %d, want 77", drafts.lastList)
	}
	if len(read) != 1 {
		t.Fatalf("the answer holds %d drafts, want 1", len(read))
	}

	pending := read[0]
	if pending.State != purchase.StateProposed {
		t.Errorf("the state of the draft is %q, want %q", pending.State, purchase.StateProposed)
	}
	if len(pending.Missing) != 1 || pending.Missing[0] != "the price of Pan" {
		t.Errorf("the draft needs %v, want the price of the bread", pending.Missing)
	}
	if len(pending.Ambiguous) != 1 || len(pending.Ambiguous[0].Options) != 2 {
		t.Errorf("the ambiguities of the draft are %v, want the two options", pending.Ambiguous)
	}
	if len(pending.Inferred) != 1 {
		t.Errorf("Paca filled %v, want the day of the purchase", pending.Inferred)
	}
}

// TestTheExpiredDraftIsNotPending proves the rule of the domain: the
// expiry is computed when the draft is read, and an expired draft does not
// wait any more.
func TestTheExpiredDraftIsNotPending(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(77),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	read, err := application.NewPurchaseService(drafts, theCatalog(), expiredClock()).Read(t.Context(), ports.ReadDrafts{
		BusinessID: business.ID(77),
	})
	if err != nil {
		t.Fatalf("the drafts could not be read: %v", err)
	}

	if len(read) != 0 {
		t.Errorf("the answer holds %d drafts, want none", len(read))
	}
}

// TestTheDraftsOfAnotherBusinessAreNotRead proves that a draft belongs to
// its business, and that the pending drafts of a business are its own.
func TestTheDraftsOfAnotherBusinessAreNotRead(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	service := application.NewPurchaseService(drafts, theCatalog(), theClock())

	_, err := service.Draft(t.Context(), ports.DraftPurchase{
		BusinessID: business.ID(88),
		Utterance:  "Compré 25 kilos de harina",
		Lines:      []ports.PurchaseLine{theLine("Harina", 25, "kg", 12000)},
	})
	if err != nil {
		t.Fatalf("the draft could not be kept: %v", err)
	}

	read, err := service.Read(t.Context(), ports.ReadDrafts{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the drafts could not be read: %v", err)
	}

	if len(read) != 0 {
		t.Errorf("the answer holds %d drafts, want none", len(read))
	}
}

// TestTheReadOfTheDraftsWithoutADatabaseSaysSo proves that a service
// without a database says so.
func TestTheReadOfTheDraftsWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewPurchaseService(nil, theCatalog(), theClock())

	_, err := service.Read(t.Context(), ports.ReadDrafts{BusinessID: business.ID(77)})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// TestTheConfirmationNamesTheNamesTheCatalogDidNotKnow proves that a name
// the catalog does not know is not an error: the answer names it, so the
// agent can tell the owner that the name is new.
func TestTheConfirmationNamesTheNamesTheCatalogDidNotKnow(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{}

	draft := theDraftOfTheBakery(t, drafts)

	confirmation, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	if len(confirmation.NewNames) != 1 {
		t.Fatalf("the answer names %v, want the name of the flour", confirmation.NewNames)
	}
	if confirmation.NewNames[0] != "Harina" {
		t.Errorf("the new name is %q, want %q", confirmation.NewNames[0], "Harina")
	}
}

// TestTheConfirmationNamesNoNameTheCatalogKnows proves that the answer
// names nothing when the catalog already knows every name.
func TestTheConfirmationNamesNoNameTheCatalogKnows(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	purchases := &fakePurchases{}
	accounts := &fakeAccounts{existing: theAccountsOfTheBakery(t)}
	book := &fakeBook{}
	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina", "harina"),
	}}

	draft := theDraftOfTheBakery(t, drafts)

	confirmation, err := theConfirmation(drafts, purchases, accounts, book, ingredients, theClock()).Confirm(t.Context(), ports.ConfirmPurchase{
		BusinessID:    business.ID(77),
		DraftID:       draft.ID,
		Revision:      draft.Revision,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the confirmation was refused: %v", err)
	}

	if len(confirmation.NewNames) != 0 {
		t.Errorf("the answer names %v, want nothing", confirmation.NewNames)
	}
}
