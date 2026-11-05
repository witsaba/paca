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
	err      error
	read     int
	lastList business.ID
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
