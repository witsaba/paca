//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/user"
)

// referencesRepository returns the repository of the reference prices, bound
// to the database of the suite.
func referencesRepository(t *testing.T) *postgres.ReferenceRepository {
	t.Helper()

	repo, err := postgres.NewReferenceRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the reference prices could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheReferences opens the business that owns the references of
// the test.
func theBakeryOfTheReferences(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The References")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theFlourOfTheReferences writes the ingredient of the catalog that the
// reference of the test names.
func theFlourOfTheReferences(t *testing.T, ctx context.Context, bakery business.Business) ingredient.Ingredient {
	t.Helper()

	flour, err := ingredient.New(bakery.ID, "Harina panadera", []string{"harina de trigo"})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	written, err := ingredientsRepository(t).Create(ctx, flour)
	if err != nil {
		t.Fatalf("the ingredient could not be written: %v", err)
	}

	return written
}

// theReferenceOfTheTest builds the reference the tests of the table write.
func theReferenceOfTheTest(bakery business.Business, flour ingredient.Ingredient) purchase.Reference {
	built, err := purchase.NewReference(bakery.ID, purchase.ReferenceDefinition{
		IngredientID:  flour.ID,
		Price:         ledger.Amount(480),
		Unit:          "kg",
		Market:        "Plaza de Mercado",
		Source:        "Boletín de precios al por mayor, DANE",
		SeenOn:        ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)),
		Utterance:     "La harina está a 480 el kilo en la plaza",
		ConfirmedWith: "sí",
	})
	if err != nil {
		panic("the reference of the test could not be built: " + err.Error())
	}

	return built
}

// TestTheReferenceIsKeptWithItsSourceAndItsDay proves the rules of the
// table against a real database: the reference is written with its
// ingredient, its price, its unit, its market, its source and the day it
// was seen, and it is read back as it was written.
func TestTheReferenceIsKeptWithItsSourceAndItsDay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := referencesRepository(t)
	bakery := theBakeryOfTheReferences(t, ctx)
	flour := theFlourOfTheReferences(t, ctx, bakery)

	written, err := repo.Record(ctx, theReferenceOfTheTest(bakery, flour))
	if err != nil {
		t.Fatalf("the reference could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the reference has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the reference is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.IngredientID != flour.ID {
		t.Errorf("the ingredient of the reference is %d, want %d", written.IngredientID, flour.ID)
	}
	if written.Price != ledger.Amount(480) {
		t.Errorf("the price is %d, want 480", written.Price)
	}
	if written.Unit != "kg" {
		t.Errorf("the unit is %q, want %q", written.Unit, "kg")
	}
	if written.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", written.Market, "Plaza de Mercado")
	}
	if written.Source != "Boletín de precios al por mayor, DANE" {
		t.Errorf("the source is %q, want the place the price was read", written.Source)
	}
	if written.SeenOn.String() != "2026-03-01" {
		t.Errorf("the day it was seen is %v, want 2026-03-01", written.SeenOn)
	}
	if written.Utterance != "La harina está a 480 el kilo en la plaza" {
		t.Errorf("the sentence is %q, want the one of the owner", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if written.CreatedAt.IsZero() {
		t.Error("the reference carries no moment of creation, want the one the database filled")
	}
	if written.Label() != purchase.ReferenceKind {
		t.Errorf("the label is %q, want %q", written.Label(), purchase.ReferenceKind)
	}
}

// TestTheReferencesOfTheBusinessAreRead proves that the references of a
// business are read back with their source and their day, and that the
// references of one ingredient are read apart.
func TestTheReferencesOfTheBusinessAreRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := referencesRepository(t)
	bakery := theBakeryOfTheReferences(t, ctx)
	flour := theFlourOfTheReferences(t, ctx, bakery)

	salt, err := ingredient.New(bakery.ID, "Sal", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	writtenSalt, err := ingredientsRepository(t).Create(ctx, salt)
	if err != nil {
		t.Fatalf("the ingredient could not be written: %v", err)
	}

	flourReference, err := repo.Record(ctx, theReferenceOfTheTest(bakery, flour))
	if err != nil {
		t.Fatalf("the reference could not be written: %v", err)
	}

	saltReference, err := purchase.NewReference(bakery.ID, purchase.ReferenceDefinition{
		IngredientID:  writtenSalt.ID,
		Price:         ledger.Amount(1200),
		Unit:          "kg",
		Source:        "La plaza",
		SeenOn:        ledger.NewDate(time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	if _, err := repo.Record(ctx, saltReference); err != nil {
		t.Fatalf("the reference could not be written: %v", err)
	}

	held, err := repo.List(ctx, bakery.ID, ingredient.ID(0))
	if err != nil {
		t.Fatalf("the references could not be read: %v", err)
	}

	if len(held) != 2 {
		t.Fatalf("the business holds %d references, want 2", len(held))
	}
	if held[0].ID != flourReference.ID {
		t.Errorf("the first reference is %d, want %d", held[0].ID, flourReference.ID)
	}
	if held[0].Source != "Boletín de precios al por mayor, DANE" {
		t.Errorf("the source of the first reference is %q, want the place the price was read", held[0].Source)
	}
	if held[0].SeenOn.String() != "2026-03-01" {
		t.Errorf("the day of the first reference is %v, want 2026-03-01", held[0].SeenOn)
	}

	ofFlour, err := repo.List(ctx, bakery.ID, flour.ID)
	if err != nil {
		t.Fatalf("the references could not be read: %v", err)
	}

	if len(ofFlour) != 1 {
		t.Fatalf("the ingredient holds %d references, want 1", len(ofFlour))
	}
	if ofFlour[0].IngredientID != flour.ID {
		t.Errorf("the ingredient of the reference is %d, want %d", ofFlour[0].IngredientID, flour.ID)
	}
	if ofFlour[0].Price != ledger.Amount(480) {
		t.Errorf("the price is %d, want 480", ofFlour[0].Price)
	}
}

// TestTheTableRefusesTheReferenceWithoutASource proves the rule of the
// domain against a real database: a price without a source is a price
// nobody can check, and the table refuses the row.
func TestTheTableRefusesTheReferenceWithoutASource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheReferences(t, ctx)
	flour := theFlourOfTheReferences(t, ctx, bakery)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO price_references (business_id, ingredient_id, price, unit, market, source, seen_on, utterance, confirmed_with)
		 VALUES ($1, $2, 480, 'kg', 'Plaza de Mercado', '', '2026-03-01', 'La harina está a 480 el kilo', 'sí')`,
		bakery.ID, flour.ID)
	if err == nil {
		t.Fatal("the row was written, want the refusal of the table")
	}

	pgErr := postgresError(t, err)
	if pgErr.ConstraintName != "price_references_source_not_empty_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "price_references_source_not_empty_check")
	}
}

// TestTheTableRefusesTheReferenceWithoutAPrice proves the rule of the domain
// against a real database: the price of a reference is a price, and the
// table refuses the row.
func TestTheTableRefusesTheReferenceWithoutAPrice(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheReferences(t, ctx)
	flour := theFlourOfTheReferences(t, ctx, bakery)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO price_references (business_id, ingredient_id, price, unit, market, source, seen_on, utterance, confirmed_with)
		 VALUES ($1, $2, 0, 'kg', 'Plaza de Mercado', 'DANE', '2026-03-01', 'La harina está a 480 el kilo', 'sí')`,
		bakery.ID, flour.ID)
	if err == nil {
		t.Fatal("the row was written, want the refusal of the table")
	}

	pgErr := postgresError(t, err)
	if pgErr.ConstraintName != "price_references_price_positive_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "price_references_price_positive_check")
	}
}

// TestTheTableRefusesTheReferenceOfAnIngredientThatIsNotThere proves that
// the reference names an ingredient of the catalog, and that the adapter
// says that it is not there.
func TestTheTableRefusesTheReferenceOfAnIngredientThatIsNotThere(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheReferences(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO price_references (business_id, ingredient_id, price, unit, market, source, seen_on, utterance, confirmed_with)
		 VALUES ($1, 999999, 480, 'kg', 'Plaza de Mercado', 'DANE', '2026-03-01', 'La harina está a 480 el kilo', 'sí')`,
		bakery.ID)
	if err == nil {
		t.Fatal("the row was written, want the refusal of the table")
	}

	pgErr := postgresError(t, err)
	if pgErr.ConstraintName != "price_references_ingredient_id_fkey" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "price_references_ingredient_id_fkey")
	}
}

// TestTheReferenceOfAnIngredientThatIsNotThereIsRefused proves that the
// adapter says what the domain says: the ingredient of the reference is not
// there.
func TestTheReferenceOfAnIngredientThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := referencesRepository(t)
	bakery := theBakeryOfTheReferences(t, ctx)

	reference, err := purchase.NewReference(bakery.ID, purchase.ReferenceDefinition{
		IngredientID:  ingredient.ID(999999),
		Price:         ledger.Amount(480),
		Unit:          "kg",
		Source:        "DANE",
		SeenOn:        ledger.NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the reference was refused: %v", err)
	}

	_, err = repo.Record(ctx, reference)
	if !errors.Is(err, purchase.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrNotFound)
	}
}
