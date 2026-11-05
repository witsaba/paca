package ingredient

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
)

// anIngredient is an ingredient the tests of the domain use.
func anIngredient(t *testing.T, id ID, own business.ID, name string, aliases ...string) Ingredient {
	t.Helper()

	built, err := New(own, name, aliases)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	built.ID = id

	return built
}

// theCatalogOfTheTest answers the catalog the tests of the domain use.
func theCatalogOfTheTest(t *testing.T) Catalog {
	t.Helper()

	catalog, err := NewCatalog(business.ID(77), []Ingredient{
		anIngredient(t, 10, business.ID(77), "Harina panadera", "harina"),
		anIngredient(t, 20, business.ID(77), "Harina de trigo"),
		anIngredient(t, 30, business.ID(77), "Azúcar"),
	})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	return catalog
}

// TestTheCatalogOfABusinessBeginsEmpty proves that the catalog of a
// business begins empty: a business that bought nothing yet answers that
// it does not know the name.
func TestTheCatalogOfABusinessBeginsEmpty(t *testing.T) {
	t.Parallel()

	catalog, err := NewCatalog(business.ID(77), nil)
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	matched, err := catalog.Match("harina")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if matched.Status != StatusUnknown {
		t.Errorf("the status is %q, want %q", matched.Status, StatusUnknown)
	}
	if !matched.New {
		t.Error("the name is not new, want a name the catalog does not know")
	}
	if matched.Ingredient.ID != 0 {
		t.Errorf("the ingredient is %d, want none", matched.Ingredient.ID)
	}
}

// TestTheNameOneIngredientAnswersIsExact proves that the name the owner
// said is answered exactly when one ingredient answers it, and that the
// answer carries the ingredient and its identity.
func TestTheNameOneIngredientAnswersIsExact(t *testing.T) {
	t.Parallel()

	matched, err := theCatalogOfTheTest(t).Match("  Azúcar  ")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if matched.Status != StatusExact {
		t.Errorf("the status is %q, want %q", matched.Status, StatusExact)
	}
	if matched.New {
		t.Error("the name is new, want a name the catalog knows")
	}
	if matched.Ingredient.ID != 30 {
		t.Errorf("the ingredient is %d, want 30", matched.Ingredient.ID)
	}
	if matched.Ingredient.Name != "Azúcar" {
		t.Errorf("the name of the ingredient is %q, want %q", matched.Ingredient.Name, "Azúcar")
	}
	if matched.Name != "Azúcar" {
		t.Errorf("the name is %q, want %q", matched.Name, "Azúcar")
	}
}

// TestTheAliasIsTheNameSheUsesForIt proves that the names the owner uses
// for an ingredient answer the name she said: the alias is the same
// ingredient.
func TestTheAliasIsTheNameSheUsesForIt(t *testing.T) {
	t.Parallel()

	matched, err := theCatalogOfTheTest(t).Match("HARINA")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if matched.Status != StatusExact {
		t.Errorf("the status is %q, want %q", matched.Status, StatusExact)
	}
	if matched.Ingredient.ID != 10 {
		t.Errorf("the ingredient is %d, want 10", matched.Ingredient.ID)
	}
}

// TestTheNameSeveralIngredientsAnswerIsAmbiguous proves that a name
// several ingredients answer is ambiguous, and that the options carry the
// identity and the name of every one of them.
func TestTheNameSeveralIngredientsAnswerIsAmbiguous(t *testing.T) {
	t.Parallel()

	catalog, err := NewCatalog(business.ID(77), []Ingredient{
		anIngredient(t, 20, business.ID(77), "Harina de trigo", "harina"),
		anIngredient(t, 10, business.ID(77), "Harina panadera", "harina"),
	})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	matched, err := catalog.Match("harina")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if matched.Status != StatusAmbiguous {
		t.Errorf("the status is %q, want %q", matched.Status, StatusAmbiguous)
	}
	if matched.New {
		t.Error("the name is new, want a name the catalog knows")
	}
	if len(matched.Options) != 2 {
		t.Fatalf("the answer holds %d options, want 2", len(matched.Options))
	}
	if matched.Options[0].ID != 10 || matched.Options[0].Name != "Harina panadera" {
		t.Errorf("the first option is %+v, want the flour of the bakery", matched.Options[0])
	}
	if matched.Options[1].ID != 20 || matched.Options[1].Name != "Harina de trigo" {
		t.Errorf("the second option is %+v, want the flour of the wheat", matched.Options[1])
	}
}

// TestTheOptionsAreInTheOrderOfTheIdentity proves that the answer is the
// same every time: the options of an ambiguous answer are in the order of
// the identity of the ingredients.
func TestTheOptionsAreInTheOrderOfTheIdentity(t *testing.T) {
	t.Parallel()

	catalog, err := NewCatalog(business.ID(77), []Ingredient{
		anIngredient(t, 30, business.ID(77), "Harina de trigo", "harina"),
		anIngredient(t, 10, business.ID(77), "Harina panadera", "harina"),
		anIngredient(t, 20, business.ID(77), "Harina de avena", "harina"),
	})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	matched, err := catalog.Match("harina")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	want := []ID{10, 20, 30}
	for index, one := range want {
		if matched.Options[index].ID != one {
			t.Errorf("the option %d is %d, want %d", index, matched.Options[index].ID, one)
		}
	}
}

// TestTheNameNoIngredientAnswersIsUnknown proves that a name the catalog
// does not know is not an error: the answer says that the name is new.
func TestTheNameNoIngredientAnswersIsUnknown(t *testing.T) {
	t.Parallel()

	matched, err := theCatalogOfTheTest(t).Match("Polvasa")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if matched.Status != StatusUnknown {
		t.Errorf("the status is %q, want %q", matched.Status, StatusUnknown)
	}
	if !matched.New {
		t.Error("the name is not new, want a name the catalog does not know")
	}
	if len(matched.Options) != 0 {
		t.Errorf("the answer holds %v, want no option", matched.Options)
	}
}

// TestTheNameThatIsNotThereIsRefused proves that a name is required: the
// owner said something, and the catalog never answers about nothing.
func TestTheNameThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	_, err := theCatalogOfTheTest(t).Match("   ")
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidName)
	}
}

// TestTheTwoIngredientsOfTheSameNameAreRefused proves the rule of the
// domain: two ingredients of the same business never carry the same name,
// and the name is compared the way a person reads it.
func TestTheTwoIngredientsOfTheSameNameAreRefused(t *testing.T) {
	t.Parallel()

	_, err := NewCatalog(business.ID(77), []Ingredient{
		anIngredient(t, 10, business.ID(77), "Harina"),
		anIngredient(t, 20, business.ID(77), " harina "),
	})
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("the error is %v, want %v", err, ErrDuplicateName)
	}
}

// TestTheIngredientOfAnotherBusinessIsRefused proves that the catalog is
// the catalog of one business: an ingredient of another business is not in
// it.
func TestTheIngredientOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	_, err := NewCatalog(business.ID(77), []Ingredient{
		anIngredient(t, 10, business.ID(88), "Harina"),
	})
	if !errors.Is(err, ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrWrongBusiness)
	}
}

// TestTheCatalogGrowsWithWhatTheOwnerBuys proves that the catalog grows
// with the names the owner buys, and that the name that is taken is
// refused.
func TestTheCatalogGrowsWithWhatTheOwnerBuys(t *testing.T) {
	t.Parallel()

	grown, err := theCatalogOfTheTest(t).With(anIngredient(t, 40, business.ID(77), "Polvasa"))
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	matched, err := grown.Match("polvasa")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}
	if matched.Status != StatusExact {
		t.Errorf("the status is %q, want %q", matched.Status, StatusExact)
	}
	if matched.Ingredient.ID != 40 {
		t.Errorf("the ingredient is %d, want 40", matched.Ingredient.ID)
	}

	if _, err := grown.With(anIngredient(t, 50, business.ID(77), "polvasa")); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("the error is %v, want %v", err, ErrDuplicateName)
	}
}

// TestTheResolutionOfALineThatCarriesAnIdentity proves that a line that
// carries the identity of an ingredient of the catalog keeps it.
func TestTheResolutionOfALineThatCarriesAnIdentity(t *testing.T) {
	t.Parallel()

	resolved, err := theCatalogOfTheTest(t).Resolve(20, "harina")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if resolved.New {
		t.Error("the catalog must grow, want the ingredient of the line")
	}
	if resolved.Ingredient.ID != 20 {
		t.Errorf("the ingredient is %d, want 20", resolved.Ingredient.ID)
	}
}

// TestTheResolutionOfANameTheCatalogKnows proves that a name the catalog
// knows exactly is the ingredient of the line.
func TestTheResolutionOfANameTheCatalogKnows(t *testing.T) {
	t.Parallel()

	resolved, err := theCatalogOfTheTest(t).Resolve(0, "Azúcar")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if resolved.New {
		t.Error("the catalog must grow, want the ingredient of the line")
	}
	if resolved.Ingredient.ID != 30 {
		t.Errorf("the ingredient is %d, want 30", resolved.Ingredient.ID)
	}
}

// TestTheResolutionOfANameTheCatalogDoesNotKnow proves that a name the
// catalog does not know is not an error: the catalog must grow with it,
// and the name is recorded as she said it.
func TestTheResolutionOfANameTheCatalogDoesNotKnow(t *testing.T) {
	t.Parallel()

	resolved, err := theCatalogOfTheTest(t).Resolve(0, "Polvasa")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if !resolved.New {
		t.Error("the catalog does not grow, want the name she said")
	}
	if resolved.Ingredient.ID != 0 {
		t.Errorf("the ingredient is %d, want none: the catalog does not know it", resolved.Ingredient.ID)
	}
}

// TestTheResolutionOfAnAmbiguousName proves that a name the catalog cannot
// answer is not resolved: the catalog does not grow with it either.
func TestTheResolutionOfAnAmbiguousName(t *testing.T) {
	t.Parallel()

	catalog, err := NewCatalog(business.ID(77), []Ingredient{
		anIngredient(t, 10, business.ID(77), "Harina panadera", "harina"),
		anIngredient(t, 20, business.ID(77), "Harina de trigo", "harina"),
	})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	resolved, err := catalog.Resolve(0, "harina")
	if err != nil {
		t.Fatalf("the name was refused: %v", err)
	}

	if resolved.New {
		t.Error("the catalog grows, want a name the catalog knows")
	}
	if resolved.Ingredient.ID != 0 {
		t.Errorf("the ingredient is %d, want none: the catalog cannot answer", resolved.Ingredient.ID)
	}
}

// TestTheStatusOfTheMatchIsItsName proves that the status is the name the
// storage and the protocol carry.
func TestTheStatusOfTheMatchIsItsName(t *testing.T) {
	t.Parallel()

	if StatusExact.String() != "exact" {
		t.Errorf("the status is %q, want %q", StatusExact.String(), "exact")
	}
	if StatusAmbiguous.String() != "ambiguous" {
		t.Errorf("the status is %q, want %q", StatusAmbiguous.String(), "ambiguous")
	}
	if StatusUnknown.String() != "unknown" {
		t.Errorf("the status is %q, want %q", StatusUnknown.String(), "unknown")
	}
}
