package application

import (
	"context"
	"fmt"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// RecipeService is the use case that writes the recipe of the bakery.
//
// The agent reads what the owner said about the recipe and the owner
// confirms it; the domain decides what a recipe is, and the use case writes
// it in the book of the business.
//
// The recipe names the ingredients of the catalog: the use case reads the
// catalog of the business and refuses an ingredient the catalog does not
// hold, because the recipe never carries a second copy of the flour.
//
// The use case never decides what a recipe is: it asks the domain to build
// the ingredients, the steps and the yield, and it writes what the domain
// built.
type RecipeService struct {
	recipes     recipe.Repository
	ingredients ingredient.Repository
}

// NewRecipeService builds the use case of the recipes.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else.
func NewRecipeService(recipes recipe.Repository, ingredients ingredient.Repository) *RecipeService {
	return &RecipeService{recipes: recipes, ingredients: ingredients}
}

// Record writes the recipe the owner confirmed.
//
// The rules of the domain hold the recording: a recipe without a name, a
// recipe without a yield, an ingredient without its grams and a step without
// its instruction are refused, and a refused recipe leaves nothing behind.
//
// The ingredients of the recipe are the ingredients of the catalog of the
// business: an ingredient the catalog does not hold is refused, and the
// answer tells the agent to match the name against the catalog first.
func (s *RecipeService) Record(ctx context.Context, request ports.RecordRecipe) (recipe.Recipe, error) {
	if s == nil || s.recipes == nil || s.ingredients == nil {
		return recipe.Recipe{}, ports.ErrNoDatabase
	}

	definition, err := definitionOf(request)
	if err != nil {
		return recipe.Recipe{}, err
	}

	catalog, err := s.catalog(ctx, request.BusinessID)
	if err != nil {
		return recipe.Recipe{}, err
	}

	for _, one := range definition.Ingredients {
		if !catalog.Holds(one.IngredientID) {
			return recipe.Recipe{}, fmt.Errorf("%w: the ingredient %d is not one of the catalog of the business %d",
				ingredient.ErrNotFound, one.IngredientID, request.BusinessID)
		}
	}

	built, err := recipe.New(request.BusinessID, definition)
	if err != nil {
		return recipe.Recipe{}, err
	}

	return s.recipes.Record(ctx, built)
}

// catalog reads the catalog of the names the business buys.
//
// The catalog is the one owner of the ingredient: a recipe names the
// ingredients of it, and the catalog of another business says nothing about
// them.
func (s *RecipeService) catalog(ctx context.Context, own business.ID) (ingredient.Catalog, error) {
	items, err := s.ingredients.List(ctx, own)
	if err != nil {
		return ingredient.Catalog{}, err
	}

	return ingredient.NewCatalog(own, items)
}

// definitionOf turns what the agent sent into the definition of the domain.
//
// The rules are the ones of the domain: the use case asks the domain to
// build the ingredients, the steps and the yield, and it never builds them
// by itself.
func definitionOf(request ports.RecordRecipe) (recipe.Definition, error) {
	ingredients := make([]recipe.Ingredient, 0, len(request.Ingredients))
	for _, one := range request.Ingredients {
		built, err := recipe.NewIngredient(ingredient.ID(one.IngredientID), one.Grams)
		if err != nil {
			return recipe.Definition{}, err
		}

		ingredients = append(ingredients, built)
	}

	steps := make([]recipe.Step, 0, len(request.Steps))
	for _, one := range request.Steps {
		built, err := recipe.NewStep(one.Instruction, one.Cue, one.Minutes, one.Temperature)
		if err != nil {
			return recipe.Definition{}, err
		}

		steps = append(steps, built)
	}

	yield, err := recipe.NewYield(request.YieldUnits, request.YieldGrams)
	if err != nil {
		return recipe.Definition{}, err
	}

	return recipe.Definition{
		Name:          request.Name,
		Ingredients:   ingredients,
		Steps:         steps,
		Yield:         yield,
		ShelfLifeDays: request.ShelfLifeDays,
		Utterance:     request.Utterance,
		ConfirmedWith: request.ConfirmedWith,
	}, nil
}
