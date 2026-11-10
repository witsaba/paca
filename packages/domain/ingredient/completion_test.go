package ingredient

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
)

// TestTheCertificationCarriesItsEvidence proves that "organic" is a claim
// with the document that proves it: the evidence is never empty.
func TestTheCertificationCarriesItsEvidence(t *testing.T) {
	t.Parallel()

	certification, err := NewCertification("Orgánico", "Certificado de la finca, 2026-03-01")
	if err != nil {
		t.Fatalf("the certification was refused: %v", err)
	}

	if certification.Name != "Orgánico" {
		t.Errorf("the name is %q, want %q", certification.Name, "Orgánico")
	}
	if certification.Evidence != "Certificado de la finca, 2026-03-01" {
		t.Errorf("the evidence is %q, want the document of the claim", certification.Evidence)
	}
	if !certification.Declared() {
		t.Error("the certification is not declared, want a claim with its evidence")
	}
}

// TestTheCertificationWithoutEvidenceIsRefused proves that a claim nobody
// can check is a wish, and it is refused.
func TestTheCertificationWithoutEvidenceIsRefused(t *testing.T) {
	t.Parallel()

	_, err := NewCertification("Orgánico", "   ")
	if !errors.Is(err, ErrMissingEvidence) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingEvidence)
	}
}

// TestTheCertificationWithoutAClaimIsRefused proves that a certification
// without a claim is refused: there is nothing to prove.
func TestTheCertificationWithoutAClaimIsRefused(t *testing.T) {
	t.Parallel()

	_, err := NewCertification("  ", "Certificado de la finca")
	if !errors.Is(err, ErrInvalidCertification) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidCertification)
	}
}

// TestTheConversionIsDeclaredAndNeverAssumed proves that the conversion of
// the raw into the prepared is a number the owner declares, and that the
// one to one of the raw is the decision she took for now.
func TestTheConversionIsDeclaredAndNeverAssumed(t *testing.T) {
	t.Parallel()

	conversion, err := NewConversion(0.95)
	if err != nil {
		t.Fatalf("the conversion was refused: %v", err)
	}
	if conversion.RawGrams != 0.95 {
		t.Errorf("the conversion is %v, want 0.95", conversion.RawGrams)
	}
	if !conversion.Declared() {
		t.Error("the conversion is not declared, want the factor the owner declared")
	}

	if _, err := NewConversion(0); !errors.Is(err, ErrInvalidConversion) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidConversion)
	}
}

// TestTheConversionThatIsNotPositiveIsRefused proves that the domain
// refuses a conversion that is not positive: the conversion is a number of
// grams, and the table is not the only one that says so.
//
// The ingredient keeps the conversion it had: the declaration is refused
// and the book is not changed.
func TestTheConversionThatIsNotPositiveIsRefused(t *testing.T) {
	t.Parallel()

	current := anIngredient(t, 10, business.ID(77), "Harina panadera")

	for _, declaration := range []Declaration{
		{Conversion: Conversion{RawGrams: -1}},
		{ConversionDeclared: true},
	} {
		_, err := Complete(current, declaration)
		if !errors.Is(err, ErrInvalidConversion) {
			t.Fatalf("the error is %v, want %v", err, ErrInvalidConversion)
		}
	}

	completed, err := Complete(current, Declaration{})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}
	if completed.Conversion.RawGrams != current.Conversion.RawGrams {
		t.Errorf("the conversion is %v, want the one it had: %v",
			completed.Conversion.RawGrams, current.Conversion.RawGrams)
	}
}

// TestTheIngredientIsWholeWhenTheOwnerDeclaresNothing proves that the
// ingredient of a purchase carries the one to one conversion the owner
// decided for now, and that it is still missing the unit she has not
// declared.
func TestTheIngredientIsWholeWhenTheOwnerDeclaresNothing(t *testing.T) {
	t.Parallel()

	built, err := New(business.ID(77), "Harina panadera", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	if built.Conversion.RawGrams != RawConversion {
		t.Errorf("the conversion is %v, want %v", built.Conversion.RawGrams, RawConversion)
	}

	missing := Missing(built)
	if len(missing) != 1 {
		t.Fatalf("the ingredient is missing %v, want the unit alone", missing)
	}
	if missing[0] != "the unit the ingredient is bought in" {
		t.Errorf("what is missing is %q, want the unit of the ingredient", missing[0])
	}
}

// TestThePackageWithoutItsWeightIsStillMissing proves that the weight of a
// package is what the owner declares: the book never assumes it.
func TestThePackageWithoutItsWeightIsStillMissing(t *testing.T) {
	t.Parallel()

	completed, err := Complete(anIngredient(t, 10, business.ID(77), "Harina"), Declaration{Unit: UnitPackage})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	missing := Missing(completed)
	if len(missing) != 1 {
		t.Fatalf("the ingredient is missing %v, want the weight of the package alone", missing)
	}
	if missing[0] != "the weight of a package of the ingredient" {
		t.Errorf("what is missing is %q, want the weight of the package", missing[0])
	}

	withWeight, err := Complete(completed, Declaration{PackageGrams: 50000})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}
	if len(Missing(withWeight)) != 0 {
		t.Errorf("the ingredient is missing %v, want nothing", Missing(withWeight))
	}
}

// TestTheDeclarationFillsWhatTheOwnerDeclared proves that the completion
// fills what the owner declared and keeps the rest of the ingredient.
func TestTheDeclarationFillsWhatTheOwnerDeclared(t *testing.T) {
	t.Parallel()

	current := anIngredient(t, 10, business.ID(77), "Harina panadera", "harina")

	completed, err := Complete(current, Declaration{
		Unit:         UnitKilogram,
		PackageGrams: 25000,
		Allergens:    []string{" gluten ", "Gluten", "huevo"},
		Aliases:      []string{"trigo", "harina de trigo"},
	})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	if completed.Unit != UnitKilogram {
		t.Errorf("the unit is %q, want %q", completed.Unit, UnitKilogram)
	}
	if completed.PackageGrams != 25000 {
		t.Errorf("the weight of the package is %v, want 25000", completed.PackageGrams)
	}
	if len(completed.Allergens) != 2 {
		t.Fatalf("the allergens are %v, want two", completed.Allergens)
	}
	if completed.Allergens[0] != "gluten" {
		t.Errorf("the first allergen is %q, want %q", completed.Allergens[0], "gluten")
	}
	if completed.Allergens[1] != "huevo" {
		t.Errorf("the second allergen is %q, want %q", completed.Allergens[1], "huevo")
	}
	if len(completed.Aliases) != 3 {
		t.Fatalf("the aliases are %v, want the one it had and the two she declared", completed.Aliases)
	}
	if completed.Name != current.Name {
		t.Errorf("the name is %q, want %q", completed.Name, current.Name)
	}
	if completed.BusinessID != current.BusinessID {
		t.Errorf("the business is %d, want %d", completed.BusinessID, current.BusinessID)
	}
}

// TestTheAliasesTheOwnerDeclaresAreTheNamesSheUses proves that the aliases
// are written by the completion: the name of the ingredient is never one
// of them, and the alias that is said twice is one alias.
func TestTheAliasesTheOwnerDeclaresAreTheNamesSheUses(t *testing.T) {
	t.Parallel()

	current := anIngredient(t, 10, business.ID(77), "Harina panadera", "harina")

	completed, err := Complete(current, Declaration{Aliases: []string{"Harina Panadera", " HARINA ", "trigo"}})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	if len(completed.Aliases) != 2 {
		t.Fatalf("the aliases are %v, want two", completed.Aliases)
	}
	if completed.Aliases[0] != "harina" {
		t.Errorf("the first alias is %q, want %q", completed.Aliases[0], "harina")
	}
	if completed.Aliases[1] != "trigo" {
		t.Errorf("the second alias is %q, want %q", completed.Aliases[1], "trigo")
	}
}

// TestTheTwoFatsAreTwoIngredientsWithTheirOwnCertification proves the
// rule of the domain: the two fats are two ingredients of the catalog, each
// one with its own certification, and the domain never merges them.
func TestTheTwoFatsAreTwoIngredientsWithTheirOwnCertification(t *testing.T) {
	t.Parallel()

	pig, err := Complete(anIngredient(t, 10, business.ID(77), "Grasa de cerdo"), Declaration{
		Unit: UnitKilogram,
		Certification: Certification{
			Name:     "Orgánico",
			Evidence: "Certificado de la finca porcícola, 2026-02-01",
		},
	})
	if err != nil {
		t.Fatalf("the fat of the pig was refused: %v", err)
	}

	cattle, err := Complete(anIngredient(t, 20, business.ID(77), "Grasa de res"), Declaration{
		Unit: UnitKilogram,
		Certification: Certification{
			Name:     "Orgánico",
			Evidence: "Certificado de la finca bovina, 2026-02-11",
		},
	})
	if err != nil {
		t.Fatalf("the fat of the cattle was refused: %v", err)
	}

	catalog, err := NewCatalog(business.ID(77), []Ingredient{pig, cattle})
	if err != nil {
		t.Fatalf("the catalog was refused: %v", err)
	}

	if len(catalog.items) != 2 {
		t.Fatalf("the catalog holds %d ingredients, want the two fats", len(catalog.items))
	}
	if catalog.items[0].Certification.Evidence == catalog.items[1].Certification.Evidence {
		t.Error("the two fats carry the same evidence, want one each")
	}
	if catalog.items[0].ID == catalog.items[1].ID {
		t.Error("the two fats are one ingredient, want two")
	}
}

// TestTheCertificationWithoutEvidenceIsRefusedByTheCompletion proves that
// the domain refuses a certification that carries no evidence, even when
// the declaration was built by hand.
func TestTheCertificationWithoutEvidenceIsRefusedByTheCompletion(t *testing.T) {
	t.Parallel()

	_, err := Complete(anIngredient(t, 10, business.ID(77), "Grasa de cerdo"), Declaration{
		Certification: Certification{Name: "Orgánico"},
	})
	if !errors.Is(err, ErrMissingEvidence) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingEvidence)
	}
}

// TestTheUnitOfTheIngredientIsTheUnitOfTheBook proves that the unit the
// ingredient is bought in is the unit the book knows: the way the owner
// declared it ("kilos") is read and normalised.
func TestTheUnitOfTheIngredientIsTheUnitOfTheBook(t *testing.T) {
	t.Parallel()

	completed, err := Complete(anIngredient(t, 10, business.ID(77), "Harina"), Declaration{Unit: Unit("kilos")})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	if completed.Unit != UnitKilogram {
		t.Errorf("the unit is %q, want %q", completed.Unit, UnitKilogram)
	}
}

// TestTheUnitTheBookDoesNotKnowIsRefusedByTheCompletion proves that the
// completion refuses a unit the book does not know.
func TestTheUnitTheBookDoesNotKnowIsRefusedByTheCompletion(t *testing.T) {
	t.Parallel()

	_, err := Complete(anIngredient(t, 10, business.ID(77), "Harina"), Declaration{Unit: Unit("libra")})
	if !errors.Is(err, ErrUnknownUnit) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownUnit)
	}
}

// TestTheWeightOfThePackageThatIsNotPositiveIsRefused proves that the
// weight of a package is a positive number of grams.
func TestTheWeightOfThePackageThatIsNotPositiveIsRefused(t *testing.T) {
	t.Parallel()

	_, err := Complete(anIngredient(t, 10, business.ID(77), "Harina"), Declaration{PackageGrams: -1})
	if !errors.Is(err, ErrInvalidPackageWeight) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidPackageWeight)
	}
}

// TestTheCompletionOfAnIngredientThatIsNotThereIsRefused proves that the
// completion needs the ingredient of the catalog: the storage holds it, and
// it carries its identity.
func TestTheCompletionOfAnIngredientThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	_, err := Complete(Ingredient{}, Declaration{Unit: UnitGram})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, ErrNotFound)
	}
}
