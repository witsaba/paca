package ingredient

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
)

// TestTheIngredientCarriesTheNameTheOwnerReads proves that an ingredient
// carries the name the owner reads and the names she uses for it, and that
// the name loses the spaces around it.
func TestTheIngredientCarriesTheNameTheOwnerReads(t *testing.T) {
	t.Parallel()

	built, err := New(business.ID(77), "  Harina panadera  ", []string{" harina de trigo "})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	if built.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", built.BusinessID)
	}
	if built.Name != "Harina panadera" {
		t.Errorf("the name is %q, want %q", built.Name, "Harina panadera")
	}
	if len(built.Aliases) != 1 {
		t.Fatalf("the ingredient carries %d aliases, want 1", len(built.Aliases))
	}
	if built.Aliases[0] != "harina de trigo" {
		t.Errorf("the alias is %q, want %q", built.Aliases[0], "harina de trigo")
	}
	if built.ID != 0 {
		t.Errorf("the identity is %d, want 0: the storage generates it", built.ID)
	}
}

// TestTheIngredientWithoutANameIsRefused proves that an ingredient without
// a name is refused: the name is what the person reads in her book.
func TestTheIngredientWithoutANameIsRefused(t *testing.T) {
	t.Parallel()

	_, err := New(business.ID(77), "   ", nil)
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidName)
	}
}

// TestTheAliasesAreTheNamesSheUses proves the rules of the domain: an
// alias that is empty is not an alias, the alias that is the name itself is
// not an alias, and the alias that is said twice is one alias.
func TestTheAliasesAreTheNamesSheUses(t *testing.T) {
	t.Parallel()

	built, err := New(business.ID(77), "Harina", []string{
		"  ",
		"harina",
		"Harina",
		"harina de trigo",
		" HARINA DE TRIGO ",
	})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	if len(built.Aliases) != 1 {
		t.Fatalf("the ingredient carries %v, want one alias", built.Aliases)
	}
	if built.Aliases[0] != "harina de trigo" {
		t.Errorf("the alias is %q, want %q", built.Aliases[0], "harina de trigo")
	}
}

// TestTheErrorsNameEveryError proves that the catalogue of the errors of the
// package holds every one of them: an error that no caller maps reaches the
// user unmapped.
func TestTheErrorsNameEveryError(t *testing.T) {
	t.Parallel()

	all := Errors()
	if len(all) != 6 {
		t.Fatalf("the catalogue holds %d errors, want 6", len(all))
	}

	for _, err := range all {
		if err == nil {
			t.Error("the catalogue holds an empty error")
		}
	}
}
