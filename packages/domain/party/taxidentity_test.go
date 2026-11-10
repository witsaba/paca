package party

import (
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
)

// TestTheTaxIdentityOfThePartyIsOptional proves the rule of the domain: the
// tax identity of a party — the NIT of a person, of a store or of a farm —
// is an OPTIONAL fact of the party.
//
// The book never asks for it, and it never assumes one: a party may carry
// one and a party may carry none. The tax identity is kept the way the
// owner said it, and the book does not know the format of one: the rule of
// the tax authority and the advice of the professional of the owner are
// not read here.
func TestTheTaxIdentityOfThePartyIsOptional(t *testing.T) {
	t.Parallel()

	built, err := New(
		business.ID(77),
		KindFarm,
		"Finca La Esperanza",
		[]Role{RoleProvider},
		"",
		"",
		"",
		LegalHuman{},
	)
	if err != nil {
		t.Fatalf("the party was refused: %v", err)
	}

	if built.TaxID != "" {
		t.Errorf("the tax identity is %q, want none: the book never assumes one", built.TaxID)
	}

	withTaxIdentity := built.WithTaxIdentity("  900123456-7  ")

	if withTaxIdentity.TaxID != "900123456-7" {
		t.Errorf("the tax identity is %q, want %q", withTaxIdentity.TaxID, "900123456-7")
	}
	if built.TaxID != "" {
		t.Errorf("the party was changed: its tax identity is %q, want none", built.TaxID)
	}
}
