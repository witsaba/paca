package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The fake of the storage of the references.
//
// It remembers what it wrote and what it was asked for: the use case writes
// a reference, and the test proves what it wrote.
type fakeReferences struct {
	written     []purchase.Reference
	held        []purchase.Reference
	err         error
	lastList    business.ID
	lastOf      ingredient.ID
	listedTimes int
}

// Record writes the reference and remembers it.
func (f *fakeReferences) Record(_ context.Context, written purchase.Reference) (purchase.Reference, error) {
	if f.err != nil {
		return purchase.Reference{}, f.err
	}

	written.ID = purchase.ReferenceID(9)
	f.written = append(f.written, written)

	return written, nil
}

// List answers the references of the business, and remembers what it was
// asked for.
func (f *fakeReferences) List(_ context.Context, own business.ID, of ingredient.ID) ([]purchase.Reference, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.listedTimes++
	f.lastList = own
	f.lastOf = of

	held := make([]purchase.Reference, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID != own {
			continue
		}

		if of != 0 && one.IngredientID != of {
			continue
		}

		held = append(held, one)
	}

	return held, nil
}

// theReferenceOfTheTest is the reference the owner confirms.
func theReferenceOfTheTest() ports.RecordReference {
	return ports.RecordReference{
		BusinessID:    business.ID(77),
		IngredientID:  10,
		Price:         480,
		Unit:          "kg",
		Market:        "Plaza de Mercado",
		Source:        "Boletín de precios al por mayor, DANE",
		Day:           "2026-03-01",
		Utterance:     "La harina está a 480 el kilo en la plaza",
		ConfirmedWith: "sí",
	}
}

// TestTheReferenceIsWrittenWithTheConfirmationOfTheOwner proves that the use
// case writes the reference the owner confirmed, with the ingredient, the
// price, the unit, the market, the source and the day it was seen.
func TestTheReferenceIsWrittenWithTheConfirmationOfTheOwner(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	written, err := service.Record(t.Context(), theReferenceOfTheTest())
	if err != nil {
		t.Fatalf("the reference could not be written: %v", err)
	}

	if len(references.written) != 1 {
		t.Fatalf("the storage holds %d references, want 1", len(references.written))
	}

	kept := references.written[0]
	if kept.ID != purchase.ReferenceID(9) {
		t.Errorf("the identity of the reference is %d, want 9", kept.ID)
	}
	if kept.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", kept.BusinessID)
	}
	if kept.IngredientID != ingredient.ID(10) {
		t.Errorf("the ingredient is %d, want 10", kept.IngredientID)
	}
	if kept.Price != 480 {
		t.Errorf("the price is %d, want 480", kept.Price)
	}
	if kept.Unit != "kg" {
		t.Errorf("the unit is %q, want %q", kept.Unit, "kg")
	}
	if kept.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", kept.Market, "Plaza de Mercado")
	}
	if kept.Source != "Boletín de precios al por mayor, DANE" {
		t.Errorf("the source is %q, want the place the price was read", kept.Source)
	}
	if kept.SeenOn.String() != "2026-03-01" {
		t.Errorf("the day it was seen is %v, want 2026-03-01", kept.SeenOn)
	}
	if kept.Utterance != "La harina está a 480 el kilo en la plaza" {
		t.Errorf("the sentence is %q, want the one of the owner", kept.Utterance)
	}
	if kept.ConfirmedWith != "sí" {
		t.Errorf("the word of the owner is %q, want %q", kept.ConfirmedWith, "sí")
	}
	if written.Label() != purchase.ReferenceKind {
		t.Errorf("the label is %q, want %q", written.Label(), purchase.ReferenceKind)
	}
}

// TestTheReferenceCarriesTheUnitOfTheBook proves that the unit of a
// reference is the unit the book knows: the research may name the unit the
// way a person says it, and the book keeps the unit it knows. The sentence
// of the owner keeps the exact wording.
func TestTheReferenceCarriesTheUnitOfTheBook(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	request := theReferenceOfTheTest()
	request.Unit = "kilos"

	written, err := service.Record(t.Context(), request)
	if err != nil {
		t.Fatalf("the reference could not be written: %v", err)
	}

	if written.Unit != "kg" {
		t.Errorf("the unit is %q, want %q: the book stores the unit of the book", written.Unit, "kg")
	}
	if written.Utterance != "La harina está a 480 el kilo en la plaza" {
		t.Errorf("the sentence is %q, want the one of the owner", written.Utterance)
	}
}

// TestTheReferenceWithAUnitTheBookDoesNotKnowIsRefused proves that a unit
// the book does not know is refused by the domain, and that no reference is
// written: the unit of the book is a rule.
func TestTheReferenceWithAUnitTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	request := theReferenceOfTheTest()
	request.Unit = "libra"

	_, err := service.Record(t.Context(), request)
	if !errors.Is(err, ingredient.ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrUnknownUnit)
	}

	if len(references.written) != 0 {
		t.Errorf("the storage holds %d references, want none", len(references.written))
	}
}

// TestTheReferenceWithoutTheIngredientOfTheCatalogIsRefused proves that the
// use case refuses a reference of an ingredient the catalog of the business
// does not hold, and that it leaves nothing behind.
func TestTheReferenceWithoutTheIngredientOfTheCatalogIsRefused(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	request := theReferenceOfTheTest()
	request.IngredientID = 999

	_, err := service.Record(t.Context(), request)
	if !errors.Is(err, ingredient.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrNotFound)
	}

	if len(references.written) != 0 {
		t.Errorf("the storage holds %d references, want none", len(references.written))
	}
}

// TestTheReferenceWithoutASourceIsRefused proves that the use case refuses a
// reference without the place the price was read, and that it leaves nothing
// behind.
func TestTheReferenceWithoutASourceIsRefused(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	request := theReferenceOfTheTest()
	request.Source = "   "

	_, err := service.Record(t.Context(), request)
	if !errors.Is(err, purchase.ErrMissingSource) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrMissingSource)
	}

	if len(references.written) != 0 {
		t.Errorf("the storage holds %d references, want none", len(references.written))
	}
}

// TestTheReferenceWithoutTheWordOfTheOwnerIsRefused proves that the use case
// refuses a reference without the word of the owner, and that it leaves
// nothing behind.
func TestTheReferenceWithoutTheWordOfTheOwnerIsRefused(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	request := theReferenceOfTheTest()
	request.ConfirmedWith = "  "

	_, err := service.Record(t.Context(), request)
	if !errors.Is(err, purchase.ErrMissingConfirmation) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrMissingConfirmation)
	}

	if len(references.written) != 0 {
		t.Errorf("the storage holds %d references, want none", len(references.written))
	}
}

// TestTheDayTheReferenceNamedIsKept proves that the day the agent named is
// the day the price was seen.
func TestTheDayTheReferenceNamedIsKept(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	written, err := service.Record(t.Context(), theReferenceOfTheTest())
	if err != nil {
		t.Fatalf("the reference could not be written: %v", err)
	}

	if written.SeenOn.String() != "2026-03-01" {
		t.Errorf("the day it was seen is %v, want 2026-03-01", written.SeenOn)
	}
}

// TestTheDayTheReferenceDidNotNameIsTheDayOfToday proves the rule of the
// domain: the day is one of the three things Paca may fill by herself.
func TestTheDayTheReferenceDidNotNameIsTheDayOfToday(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	request := theReferenceOfTheTest()
	request.Day = ""

	written, err := service.Record(t.Context(), request)
	if err != nil {
		t.Fatalf("the reference could not be written: %v", err)
	}

	if written.SeenOn.String() != theClock().Now().Format("2006-01-02") {
		t.Errorf("the day it was seen is %v, want the day of today", written.SeenOn)
	}
}

// TestTheReferencesOfTheBusinessAreRead proves that the use case reads the
// references of the business with their source and their day.
func TestTheReferencesOfTheBusinessAreRead(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{held: []purchase.Reference{
		{ID: 1, BusinessID: business.ID(77), IngredientID: ingredient.ID(10), Price: 480, Unit: "kg", Source: "DANE"},
		{ID: 2, BusinessID: business.ID(77), IngredientID: ingredient.ID(20), Price: 1200, Unit: "kg", Source: "La plaza"},
	}}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	read, err := service.Read(t.Context(), ports.ReadReferences{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the references could not be read: %v", err)
	}

	if len(read) != 2 {
		t.Fatalf("the use case read %d references, want 2", len(read))
	}
	if references.lastList != business.ID(77) {
		t.Errorf("the business is %d, want 77", references.lastList)
	}
	if references.lastOf != ingredient.ID(0) {
		t.Errorf("the ingredient is %d, want none", references.lastOf)
	}
	if read[0].Source != "DANE" {
		t.Errorf("the source of the first reference is %q, want %q", read[0].Source, "DANE")
	}
}

// TestTheReferencesOfOneIngredientAreRead proves that the use case reads the
// references of one ingredient when the agent names one.
func TestTheReferencesOfOneIngredientAreRead(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{held: []purchase.Reference{
		{ID: 1, BusinessID: business.ID(77), IngredientID: ingredient.ID(10), Price: 480, Unit: "kg", Source: "DANE"},
		{ID: 2, BusinessID: business.ID(77), IngredientID: ingredient.ID(20), Price: 1200, Unit: "kg", Source: "La plaza"},
	}}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	read, err := service.Read(t.Context(), ports.ReadReferences{
		BusinessID:   business.ID(77),
		IngredientID: 10,
	})
	if err != nil {
		t.Fatalf("the references could not be read: %v", err)
	}

	if len(read) != 1 {
		t.Fatalf("the use case read %d references, want 1", len(read))
	}
	if references.lastOf != ingredient.ID(10) {
		t.Errorf("the ingredient is %d, want 10", references.lastOf)
	}
	if read[0].IngredientID != ingredient.ID(10) {
		t.Errorf("the ingredient of the reference is %d, want 10", read[0].IngredientID)
	}
}

// TestTheReferenceOfAnIngredientTheCatalogDoesNotHoldIsRefused proves that
// the use case refuses to read the references of an ingredient the catalog
// of the business does not hold.
func TestTheReferenceOfAnIngredientTheCatalogDoesNotHoldIsRefused(t *testing.T) {
	t.Parallel()

	references := &fakeReferences{}
	service := application.NewReferenceService(references, theCatalogOfTheTest(t), theClock())

	_, err := service.Read(t.Context(), ports.ReadReferences{
		BusinessID:   business.ID(77),
		IngredientID: 999,
	})
	if !errors.Is(err, ingredient.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrNotFound)
	}

	if references.listedTimes != 0 {
		t.Errorf("the storage was asked %d times, want never", references.listedTimes)
	}
}
