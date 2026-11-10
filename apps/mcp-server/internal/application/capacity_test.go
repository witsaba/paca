package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theRecipesOfTheCapacity is a reader of the recipes that answers what the
// test tells it to answer.
//
// It answers the recipes it holds without reading the business they belong
// to, so that a test of the use case can prove that the use case decides no
// rule: the recipe of another business reaches the domain, and the domain is
// the one that refuses it.
type theRecipesOfTheCapacity struct {
	held   []recipe.Recipe
	err    error
	found  recipe.ID
	listed business.ID
}

// Find answers the recipe the test holds, and remembers the identity.
func (r *theRecipesOfTheCapacity) Find(_ context.Context, id recipe.ID) (recipe.Recipe, error) {
	r.found = id

	if r.err != nil {
		return recipe.Recipe{}, r.err
	}

	for _, one := range r.held {
		if one.ID == id {
			return one, nil
		}
	}

	return recipe.Recipe{}, recipe.ErrNotFound
}

// List answers the recipes the test holds, and remembers the business.
func (r *theRecipesOfTheCapacity) List(_ context.Context, own business.ID) ([]recipe.Recipe, error) {
	r.listed = own

	if r.err != nil {
		return nil, r.err
	}

	return r.held, nil
}

// theRecipeOfTheCapacityOfTheUseCase is a recipe of the bakery that carries
// the minutes of the hands of the owner and the yield of what it makes.
func theRecipeOfTheCapacityOfTheUseCase(t *testing.T, id recipe.ID, minutes, units int) recipe.Recipe {
	t.Helper()

	baked := theRecipeOfTheCost(t)
	baked.ID = id
	baked.Name = "Pan de molde"
	baked.Steps = []recipe.Step{
		{Instruction: "Amasar.", Minutes: minutes, Kind: recipe.LaborHandsOn},
		{Instruction: "Hornear.", Minutes: 45, Kind: recipe.LaborOven},
	}
	baked.Yield = recipe.Yield{Units: units, Grams: 90}

	return baked
}

// TestTheCapacityIsDerivedFromTheRecipesOfTheBusiness proves that the use
// case reads the recipes of the business through the reader of the book and
// asks the domain for the capacity of the weekend.
func TestTheCapacityIsDerivedFromTheRecipesOfTheBusiness(t *testing.T) {
	t.Parallel()

	recipes := &theRecipesOfTheCapacity{held: []recipe.Recipe{
		theRecipeOfTheCapacityOfTheUseCase(t, recipe.ID(9), 180, 10),
	}}

	service := application.NewCapacityService(recipes)

	capacity, err := service.Read(t.Context(), ports.ReadCapacity{
		BusinessID: business.ID(77),
		Hours:      10,
	})
	if err != nil {
		t.Fatalf("the capacity was refused: %v", err)
	}

	if recipes.listed != business.ID(77) {
		t.Errorf("the recipes were read for the business %d, want 77", recipes.listed)
	}
	if capacity.Hours != 10 {
		t.Errorf("the hours of the capacity are %v, want the 10 the owner declared", capacity.Hours)
	}
	if len(capacity.Recipes) != 1 {
		t.Fatalf("the capacity carries %d recipes, want 1", len(capacity.Recipes))
	}
	if capacity.Recipes[0].Batches != 3 || capacity.Recipes[0].Units != 30 {
		t.Errorf("the capacity carries %d batches and %d units, want 3 and 30",
			capacity.Recipes[0].Batches, capacity.Recipes[0].Units)
	}
}

// TestTheCapacityOfABusinessWithoutRecipesIsEmpty proves that the use case
// answers the capacity of a business that wrote no recipe yet: it carries
// the hours and no recipe.
func TestTheCapacityOfABusinessWithoutRecipesIsEmpty(t *testing.T) {
	t.Parallel()

	service := application.NewCapacityService(&theRecipesOfTheCapacity{})

	capacity, err := service.Read(t.Context(), ports.ReadCapacity{
		BusinessID: business.ID(77),
		Hours:      8,
	})
	if err != nil {
		t.Fatalf("the capacity was refused: %v", err)
	}
	if capacity.Hours != 8 {
		t.Errorf("the hours of the capacity are %v, want 8", capacity.Hours)
	}
	if len(capacity.Recipes) != 0 {
		t.Errorf("the capacity carries %d recipes, want none", len(capacity.Recipes))
	}
}

// TestTheCapacityOfABusinessThatIsNotThereIsRefused proves that the capacity
// belongs to a business: a request that names no business is refused with
// the refusal of the book of the business, and the recipes are never read.
func TestTheCapacityOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	recipes := &theRecipesOfTheCapacity{}

	service := application.NewCapacityService(recipes)

	_, err := service.Read(t.Context(), ports.ReadCapacity{BusinessID: business.ID(0), Hours: 10})
	if !errors.Is(err, purchase.ErrMissingBusiness) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrMissingBusiness)
	}
	if recipes.listed != business.ID(0) {
		t.Errorf("the recipes were read for the business %d, want none", recipes.listed)
	}
}

// TestTheCapacityOfNoHoursIsRefusedByTheDomain proves that the use case
// decides no rule: the hours it does not hold reach the domain, and the
// domain refuses them.
func TestTheCapacityOfNoHoursIsRefusedByTheDomain(t *testing.T) {
	t.Parallel()

	service := application.NewCapacityService(&theRecipesOfTheCapacity{held: []recipe.Recipe{
		theRecipeOfTheCapacityOfTheUseCase(t, recipe.ID(9), 60, 10),
	}})

	_, err := service.Read(t.Context(), ports.ReadCapacity{BusinessID: business.ID(77), Hours: 0})
	if !errors.Is(err, recipe.ErrInvalidCapacityHours) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidCapacityHours)
	}
}

// TestTheCapacityOfARecipeOfAnotherBusinessIsRefused proves that the use
// case decides no rule: the recipe of another business reaches the domain,
// and the domain refuses it.
func TestTheCapacityOfARecipeOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	theirs := theRecipeOfTheCapacityOfTheUseCase(t, recipe.ID(10), 60, 10)
	theirs.BusinessID = business.ID(88)

	service := application.NewCapacityService(&theRecipesOfTheCapacity{held: []recipe.Recipe{theirs}})

	_, err := service.Read(t.Context(), ports.ReadCapacity{BusinessID: business.ID(77), Hours: 10})
	if !errors.Is(err, recipe.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrWrongBusiness)
	}
}

// TestTheCapacityWithoutADatabaseSaysSo proves that the use case without the
// reader of the recipes says that the database is not there.
func TestTheCapacityWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewCapacityService(nil)

	_, err := service.Read(t.Context(), ports.ReadCapacity{BusinessID: business.ID(77), Hours: 10})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// TestTheErrorOfTheReaderOfTheRecipesReachesTheCaller proves that the use
// case does not rewrite the error of the storage of the recipes.
func TestTheErrorOfTheReaderOfTheRecipesReachesTheCaller(t *testing.T) {
	t.Parallel()

	failure := errors.New("the storage of the recipes is on fire")
	recipes := &theRecipesOfTheCapacity{err: failure}

	service := application.NewCapacityService(recipes)

	_, err := service.Read(t.Context(), ports.ReadCapacity{BusinessID: business.ID(77), Hours: 10})
	if !errors.Is(err, failure) {
		t.Fatalf("the error is %v, want %v", err, failure)
	}
}
