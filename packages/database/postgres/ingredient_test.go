package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
)

func TestTheRepositoryOfTheCatalogNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewIngredientRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRepositoryOfTheCatalogWithoutADatabaseRefusesToWork(t *testing.T) {
	t.Parallel()

	ingredients := &IngredientRepository{}

	if _, err := ingredients.List(t.Context(), business.ID(77)); !errors.Is(err, ingredient.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, ingredient.ErrNoStorage)
	}
	if _, err := ingredients.Create(t.Context(), ingredient.Ingredient{}); !errors.Is(err, ingredient.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, ingredient.ErrNoStorage)
	}
}

// TestTheAliasesOfAnIngredientTravelBothWays proves that the names the
// owner uses for the thing she buys survive the trip to the storage and
// back.
func TestTheAliasesOfAnIngredientTravelBothWays(t *testing.T) {
	t.Parallel()

	encoded, err := encodeAliases([]string{"harina de trigo", "trigo"})
	if err != nil {
		t.Fatalf("the aliases could not be encoded: %v", err)
	}

	decoded, err := decodeAliases(encoded)
	if err != nil {
		t.Fatalf("the aliases could not be decoded: %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("the storage answered %d aliases, want 2", len(decoded))
	}
	if decoded[0] != "harina de trigo" {
		t.Errorf("the first alias is %q, want %q", decoded[0], "harina de trigo")
	}
}

// TestTheAliasesThatAreNotAListAreRefused proves that a row that another
// writer broke is refused, and not read as a list of nothing.
func TestTheAliasesThatAreNotAListAreRefused(t *testing.T) {
	t.Parallel()

	if _, err := decodeAliases([]byte("not a list")); !errors.Is(err, ingredient.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrInvalidValue)
	}
}

// TestTheRowOfAnIngredientIsMappedOntoTheModel proves that the row is
// mapped onto the model of the domain, and that the model decides what an
// ingredient is.
func TestTheRowOfAnIngredientIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	mapped, err := ingredientFromRow(db.Ingredient{
		ID:         10,
		BusinessID: 77,
		Name:       "Harina panadera",
		Aliases:    []byte(`["harina de trigo"]`),
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if mapped.ID != ingredient.ID(10) {
		t.Errorf("the identity is %d, want 10", mapped.ID)
	}
	if mapped.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", mapped.BusinessID)
	}
	if mapped.Name != "Harina panadera" {
		t.Errorf("the name is %q, want %q", mapped.Name, "Harina panadera")
	}
	if len(mapped.Aliases) != 1 || mapped.Aliases[0] != "harina de trigo" {
		t.Errorf("the aliases are %v, want the one of the row", mapped.Aliases)
	}
}

// TestTheRowWithoutANameIsRefused proves that the model refuses a row that
// breaks its rule: the name of an ingredient is not empty.
func TestTheRowWithoutANameIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := ingredientFromRow(db.Ingredient{ID: 10, BusinessID: 77, Name: "  "}); !errors.Is(err, ingredient.ErrInvalidName) {
		t.Fatalf("the error is %v, want %v", err, ingredient.ErrInvalidName)
	}
}

// TestTheRefusalsOfTheDatabaseAreTranslatedForTheIngredient proves that the
// truth of the database reaches the caller as the errors of the port.
func TestTheRefusalsOfTheDatabaseAreTranslatedForTheIngredient(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     *pgconn.PgError
		want    error
		message string
	}{
		{
			name: "the name that is taken",
			err: &pgconn.PgError{
				Code:           uniqueViolation,
				ConstraintName: ingredientsNamePerBusinessIndex,
			},
			want:    ingredient.ErrDuplicateName,
			message: "the business already holds an ingredient with that name",
		},
		{
			name: "the business that is not there",
			err: &pgconn.PgError{
				Code:           foreignKeyViolation,
				ConstraintName: "ingredients_business_id_fkey",
			},
			want:    ingredient.ErrNotFound,
			message: "the business of the ingredient is not there",
		},
		{
			name: "the row that breaks a rule of the table",
			err: &pgconn.PgError{
				Code:           checkViolation,
				ConstraintName: "ingredients_name_not_empty_check",
			},
			want:    ingredient.ErrInvalidValue,
			message: "refused the row",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := writeIngredientError(testCase.err)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
			if !strings.Contains(err.Error(), testCase.message) {
				t.Errorf("the error is %q, want it to say %q", err.Error(), testCase.message)
			}
		})
	}
}
