//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/user"
)

// ingredientsRepository returns the repository of the catalog, bound to the
// database of the suite.
func ingredientsRepository(t *testing.T) *postgres.IngredientRepository {
	t.Helper()

	repo, err := postgres.NewIngredientRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the catalog could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheCatalog opens the business that owns the catalog of the
// test.
func theBakeryOfTheCatalog(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Catalog")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theIngredientOfTheTest builds the ingredient the tests of the table
// write.
func theIngredientOfTheTest(t *testing.T, bakery business.Business) ingredient.Ingredient {
	t.Helper()

	built, err := ingredient.New(bakery.ID, "Harina panadera", []string{"harina de trigo"})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	return built
}

// TestTheCatalogOfTheBusinessBeginsEmpty proves the rule of the domain
// against a real database: the catalog of a business begins empty, and it
// holds only the ingredients of that business.
func TestTheCatalogOfTheBusinessBeginsEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ingredientsRepository(t)
	bakery := theBakeryOfTheCatalog(t, ctx)

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the catalog could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("the catalog holds %d ingredients, want none", len(held))
	}
}

// TestTheIngredientOfTheBusinessIsKeptAndReadBack proves the rules of the
// table against a real database: the ingredient is written with its name
// and its aliases, and it is read back as it was written.
func TestTheIngredientOfTheBusinessIsKeptAndReadBack(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ingredientsRepository(t)
	bakery := theBakeryOfTheCatalog(t, ctx)

	written, err := repo.Create(ctx, theIngredientOfTheTest(t, bakery))
	if err != nil {
		t.Fatalf("the ingredient could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the ingredient has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the ingredient is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.Name != "Harina panadera" {
		t.Errorf("the name is %q, want %q", written.Name, "Harina panadera")
	}
	if len(written.Aliases) != 1 || written.Aliases[0] != "harina de trigo" {
		t.Errorf("the aliases are %v, want the one of the test", written.Aliases)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the catalog could not be read: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the catalog holds %d ingredients, want 1", len(held))
	}
	if held[0].ID != written.ID {
		t.Errorf("the identity is %d, want %d", held[0].ID, written.ID)
	}
	if held[0].Name != "Harina panadera" {
		t.Errorf("the name is %q, want %q", held[0].Name, "Harina panadera")
	}
	if len(held[0].Aliases) != 1 || held[0].Aliases[0] != "harina de trigo" {
		t.Errorf("the aliases are %v, want the one of the test", held[0].Aliases)
	}
}

// TestTheTableRefusesTheSecondIngredientWithTheSameName proves the rule of
// the domain against a real database: two ingredients of the same business
// never carry the same name, and the name is compared the way a person
// reads it.
func TestTheTableRefusesTheSecondIngredientWithTheSameName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ingredientsRepository(t)
	bakery := theBakeryOfTheCatalog(t, ctx)

	if _, err := repo.Create(ctx, theIngredientOfTheTest(t, bakery)); err != nil {
		t.Fatalf("the ingredient could not be written: %v", err)
	}

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO ingredients (business_id, name) VALUES ($1, $2)`,
		int64(bakery.ID), "  harina PANADERA ",
	)
	if err == nil {
		t.Fatal("the database stored two ingredients with the same name, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != uniqueViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, uniqueViolation)
	}
	if pgErr.ConstraintName != "ingredients_one_name_per_business" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "ingredients_one_name_per_business")
	}
}

// TestTheTableRefusesTheIngredientWithoutAName proves the rule of the
// domain against a real database: the name of an ingredient is not empty.
func TestTheTableRefusesTheIngredientWithoutAName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheCatalog(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO ingredients (business_id, name) VALUES ($1, $2)`,
		int64(bakery.ID), "",
	)
	if err == nil {
		t.Fatal("the database stored an ingredient without a name, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "ingredients_name_not_empty_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "ingredients_name_not_empty_check")
	}
}

// TestTheTableRefusesTheIngredientOfABusinessThatIsNotThere proves that
// the catalog of a business belongs to the business.
func TestTheTableRefusesTheIngredientOfABusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheCatalog(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO ingredients (business_id, name) VALUES ($1, $2)`,
		int64(bakery.ID)+1000, "Harina panadera",
	)
	if err == nil {
		t.Fatal("the database stored an ingredient of a business that is not there, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != foreignKeyViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, foreignKeyViolation)
	}
	if pgErr.ConstraintName != "ingredients_business_id_fkey" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "ingredients_business_id_fkey")
	}
}

// TestTheNameThatIsTakenIsRefusedForTheCaller proves that the truth of the
// database reaches the caller as the error of the port.
func TestTheNameThatIsTakenIsRefusedForTheCaller(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := ingredientsRepository(t)
	bakery := theBakeryOfTheCatalog(t, ctx)

	if _, err := repo.Create(ctx, theIngredientOfTheTest(t, bakery)); err != nil {
		t.Fatalf("the ingredient could not be written: %v", err)
	}

	second, err := ingredient.New(bakery.ID, "harina PANADERA", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	if _, err := repo.Create(ctx, second); !errors.Is(err, ingredient.ErrDuplicateName) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrDuplicateName)
	}
}
