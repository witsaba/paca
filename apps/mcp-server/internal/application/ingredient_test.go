package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
)

// fakeIngredients is the fake of the storage of the catalog.
//
// It remembers the ingredients it holds and the ones it wrote, and it
// answers what the test tells it to answer.
type fakeIngredients struct {
	items    []ingredient.Ingredient
	written  []ingredient.Ingredient
	updated  []ingredient.Ingredient
	err      error
	read     int
	lastList business.ID
}

// theCatalog is the catalog the tests of the use cases use.
//
// The catalog of a business begins empty, and a test that needs a name in
// it says so.
func theCatalog() *fakeIngredients {
	return &fakeIngredients{}
}

// List answers the ingredients of the business, and remembers that it was
// asked.
func (f *fakeIngredients) List(_ context.Context, businessID business.ID) ([]ingredient.Ingredient, error) {
	f.read++
	f.lastList = businessID

	if f.err != nil {
		return nil, f.err
	}

	held := make([]ingredient.Ingredient, 0, len(f.items))
	for _, one := range f.items {
		if one.BusinessID == businessID {
			held = append(held, one)
		}
	}

	return held, nil
}

// Create writes the ingredient and remembers it.
func (f *fakeIngredients) Create(_ context.Context, item ingredient.Ingredient) (ingredient.Ingredient, error) {
	if f.err != nil {
		return ingredient.Ingredient{}, f.err
	}

	item.ID = ingredient.ID(len(f.written) + 100)
	f.written = append(f.written, item)

	return item, nil
}

// Update writes the ingredient and remembers it.
func (f *fakeIngredients) Update(_ context.Context, item ingredient.Ingredient) (ingredient.Ingredient, error) {
	if f.err != nil {
		return ingredient.Ingredient{}, f.err
	}

	f.updated = append(f.updated, item)

	return item, nil
}

// anIngredientOfTheCatalog is an ingredient the tests of the catalog use.
func anIngredientOfTheCatalog(t *testing.T, id ingredient.ID, name string, aliases ...string) ingredient.Ingredient {
	t.Helper()

	built, err := ingredient.New(business.ID(77), name, aliases)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	built.ID = id

	return built
}

// TestTheNameIsMatchedAgainstTheCatalogOfTheBusiness proves that the use
// case reads the catalog of the business the agent names, and that it
// answers the ingredient the name belongs to.
func TestTheNameIsMatchedAgainstTheCatalogOfTheBusiness(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera", "harina"),
	}}
	service := application.NewIngredientService(ingredients)

	matched, err := service.Match(t.Context(), ports.MatchItems{
		BusinessID: business.ID(77),
		Name:       "harina",
	})
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if ingredients.read != 1 {
		t.Errorf("the storage was read %d times, want 1", ingredients.read)
	}
	if ingredients.lastList != business.ID(77) {
		t.Errorf("the catalog that was read is the one of %d, want 77", ingredients.lastList)
	}
	if matched.Status != ingredient.StatusExact {
		t.Errorf("the status is %q, want %q", matched.Status, ingredient.StatusExact)
	}
	if matched.Ingredient.ID != ingredient.ID(10) {
		t.Errorf("the ingredient is %d, want 10", matched.Ingredient.ID)
	}
	if matched.New {
		t.Error("the name is new, want a name the catalog knows")
	}
}

// TestTheNameTheCatalogDoesNotKnowIsNew proves that a name the catalog
// does not know is not an error: the answer says that the name is new.
func TestTheNameTheCatalogDoesNotKnowIsNew(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{}
	service := application.NewIngredientService(ingredients)

	matched, err := service.Match(t.Context(), ports.MatchItems{
		BusinessID: business.ID(77),
		Name:       "Polvasa",
	})
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if matched.Status != ingredient.StatusUnknown {
		t.Errorf("the status is %q, want %q", matched.Status, ingredient.StatusUnknown)
	}
	if !matched.New {
		t.Error("the name is not new, want a name the catalog does not know")
	}
}

// TestTheAmbiguousNameCarriesItsOptions proves that a name several
// ingredients answer comes with the options, so the owner chooses with one
// word.
func TestTheAmbiguousNameCarriesItsOptions(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera", "harina"),
		anIngredientOfTheCatalog(t, 20, "Harina de trigo", "harina"),
	}}
	service := application.NewIngredientService(ingredients)

	matched, err := service.Match(t.Context(), ports.MatchItems{
		BusinessID: business.ID(77),
		Name:       "harina",
	})
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if matched.Status != ingredient.StatusAmbiguous {
		t.Errorf("the status is %q, want %q", matched.Status, ingredient.StatusAmbiguous)
	}
	if len(matched.Options) != 2 {
		t.Fatalf("the answer holds %d options, want 2", len(matched.Options))
	}
	if matched.Options[0].ID != ingredient.ID(10) {
		t.Errorf("the first option is %d, want 10", matched.Options[0].ID)
	}
}

// TestTheNameThatIsNotThereIsRefused proves that the name is required: the
// owner said something, and the catalog never answers about nothing.
func TestTheNameThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{}
	service := application.NewIngredientService(ingredients)

	_, err := service.Match(t.Context(), ports.MatchItems{
		BusinessID: business.ID(77),
		Name:       "   ",
	})
	if !errors.Is(err, ingredient.ErrInvalidName) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrInvalidName)
	}
}

// TestTheCatalogThatCarriesTwoNamesIsRefused proves that the rule of the
// domain reaches the use case: two ingredients of the same business never
// carry the same name.
func TestTheCatalogThatCarriesTwoNamesIsRefused(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina"),
		anIngredientOfTheCatalog(t, 20, "harina"),
	}}
	service := application.NewIngredientService(ingredients)

	_, err := service.Match(t.Context(), ports.MatchItems{
		BusinessID: business.ID(77),
		Name:       "harina",
	})
	if !errors.Is(err, ingredient.ErrDuplicateName) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrDuplicateName)
	}
}

// TestTheMatchWithoutADatabaseSaysSo proves that a service without a
// database says so, and it says it in the only capability it has.
func TestTheMatchWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewIngredientService(nil)

	_, err := service.Match(t.Context(), ports.MatchItems{
		BusinessID: business.ID(77),
		Name:       "harina",
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// TestTheCatalogOfTheBusinessIsRead proves that the use case reads the
// catalog of the business the agent names, and answers every ingredient of
// it.
func TestTheCatalogOfTheBusinessIsRead(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera", "harina"),
		anIngredientOfTheCatalog(t, 20, "Azúcar"),
	}}
	service := application.NewIngredientService(ingredients)

	read, err := service.Read(t.Context(), ports.ReadIngredients{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the catalog could not be read: %v", err)
	}

	if ingredients.lastList != business.ID(77) {
		t.Errorf("the catalog that was read is the one of %d, want 77", ingredients.lastList)
	}
	if len(read) != 2 {
		t.Fatalf("the answer holds %d ingredients, want 2", len(read))
	}
	if read[0].ID != ingredient.ID(10) {
		t.Errorf("the first ingredient is %d, want 10", read[0].ID)
	}
}

// TestTheEmptyCatalogIsRead proves that the catalog of a business begins
// empty: the list is empty and it is not an error.
func TestTheEmptyCatalogIsRead(t *testing.T) {
	t.Parallel()

	service := application.NewIngredientService(&fakeIngredients{})

	read, err := service.Read(t.Context(), ports.ReadIngredients{BusinessID: business.ID(77)})
	if err != nil {
		t.Fatalf("the catalog could not be read: %v", err)
	}
	if len(read) != 0 {
		t.Errorf("the answer holds %d ingredients, want none", len(read))
	}
}

// TestTheReadOfTheCatalogWithoutADatabaseSaysSo proves that a service
// without a database says so.
func TestTheReadOfTheCatalogWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewIngredientService(nil)

	_, err := service.Read(t.Context(), ports.ReadIngredients{BusinessID: business.ID(77)})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// TestWhatTheOwnerDeclaredIsWritten proves that the use case writes what
// the owner declared about the ingredient, and answers with the ingredient
// as it is now.
func TestWhatTheOwnerDeclaredIsWritten(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera", "harina"),
	}}
	service := application.NewIngredientService(ingredients)

	completed, err := service.Complete(t.Context(), ports.CompleteIngredient{
		BusinessID:            business.ID(77),
		IngredientID:          10,
		Unit:                  "kg",
		PackageGrams:          25000,
		Allergens:             []string{"gluten"},
		CertificationName:     "Orgánico",
		CertificationEvidence: "Certificado de la finca, 2026-02-01",
		Aliases:               []string{"trigo"},
		Conversion:            0.95,
	})
	if err != nil {
		t.Fatalf("the ingredient could not be completed: %v", err)
	}

	if len(ingredients.updated) != 1 {
		t.Fatalf("the storage holds %d completions, want 1", len(ingredients.updated))
	}

	written := ingredients.updated[0]
	if written.Unit != ingredient.UnitKilogram {
		t.Errorf("the unit is %q, want %q", written.Unit, ingredient.UnitKilogram)
	}
	if written.PackageGrams != 25000 {
		t.Errorf("the weight of the package is %v, want 25000", written.PackageGrams)
	}
	if len(written.Allergens) != 1 || written.Allergens[0] != "gluten" {
		t.Errorf("the allergens are %v, want the one of the owner", written.Allergens)
	}
	if written.Certification.Name != "Orgánico" {
		t.Errorf("the certification is %q, want %q", written.Certification.Name, "Orgánico")
	}
	if written.Certification.Evidence != "Certificado de la finca, 2026-02-01" {
		t.Errorf("the evidence is %q, want the document of the owner", written.Certification.Evidence)
	}
	if len(written.Aliases) != 2 {
		t.Errorf("the aliases are %v, want the one it had and the one of the owner", written.Aliases)
	}
	if written.Conversion.RawGrams != 0.95 {
		t.Errorf("the conversion is %v, want 0.95", written.Conversion.RawGrams)
	}

	if completed.ID != ingredient.ID(10) {
		t.Errorf("the answer holds the ingredient %d, want 10", completed.ID)
	}
}

// TestTheCompletionKeepsWhatTheOwnerDidNotDeclare proves that the
// completion fills what the owner declared and keeps the rest of the
// ingredient.
func TestTheCompletionKeepsWhatTheOwnerDidNotDeclare(t *testing.T) {
	t.Parallel()

	current := anIngredientOfTheCatalog(t, 10, "Harina panadera", "harina")

	ingredients := &fakeIngredients{items: []ingredient.Ingredient{current}}
	service := application.NewIngredientService(ingredients)

	completed, err := service.Complete(t.Context(), ports.CompleteIngredient{
		BusinessID:   business.ID(77),
		IngredientID: 10,
		Unit:         "g",
	})
	if err != nil {
		t.Fatalf("the ingredient could not be completed: %v", err)
	}

	if completed.Name != "Harina panadera" {
		t.Errorf("the name is %q, want %q", completed.Name, "Harina panadera")
	}
	if len(completed.Aliases) != 1 || completed.Aliases[0] != "harina" {
		t.Errorf("the aliases are %v, want the one it had", completed.Aliases)
	}
	if completed.Conversion.RawGrams != ingredient.RawConversion {
		t.Errorf("the conversion is %v, want the one to one the owner decided",
			completed.Conversion.RawGrams)
	}
}

// TestTheCompletionOfAnIngredientTheCatalogDoesNotHoldIsRefused proves that
// the ingredient belongs to the business: the completion of an ingredient
// the catalog does not hold is refused, and nothing is written.
func TestTheCompletionOfAnIngredientTheCatalogDoesNotHoldIsRefused(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera"),
	}}
	service := application.NewIngredientService(ingredients)

	_, err := service.Complete(t.Context(), ports.CompleteIngredient{
		BusinessID:   business.ID(77),
		IngredientID: 999,
		Unit:         "kg",
	})
	if !errors.Is(err, ingredient.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrNotFound)
	}
	if len(ingredients.updated) != 0 {
		t.Errorf("the storage holds %d completions, want none", len(ingredients.updated))
	}
}

// TestTheCertificationWithoutEvidenceIsRefused proves that the rule of the
// domain reaches the use case: a claim without the document that proves it
// is refused, and nothing is written.
func TestTheCertificationWithoutEvidenceIsRefused(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Grasa de cerdo"),
	}}
	service := application.NewIngredientService(ingredients)

	_, err := service.Complete(t.Context(), ports.CompleteIngredient{
		BusinessID:        business.ID(77),
		IngredientID:      10,
		CertificationName: "Orgánico",
	})
	if !errors.Is(err, ingredient.ErrMissingEvidence) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrMissingEvidence)
	}
	if len(ingredients.updated) != 0 {
		t.Errorf("the storage holds %d completions, want none", len(ingredients.updated))
	}
}

// TestTheUnitTheBookDoesNotKnowIsRefused proves that the units of the book
// are a rule: a unit PACA does not know is refused.
func TestTheUnitTheBookDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	ingredients := &fakeIngredients{items: []ingredient.Ingredient{
		anIngredientOfTheCatalog(t, 10, "Harina panadera"),
	}}
	service := application.NewIngredientService(ingredients)

	_, err := service.Complete(t.Context(), ports.CompleteIngredient{
		BusinessID:   business.ID(77),
		IngredientID: 10,
		Unit:         "libra",
	})
	if !errors.Is(err, ingredient.ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrUnknownUnit)
	}
}

// TestTheCompletionWithoutADatabaseSaysSo proves that a service without a
// database says so.
func TestTheCompletionWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewIngredientService(nil)

	_, err := service.Complete(t.Context(), ports.CompleteIngredient{
		BusinessID:   business.ID(77),
		IngredientID: 10,
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
