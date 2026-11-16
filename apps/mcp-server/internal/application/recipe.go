package application

import (
	"context"
	"strings"

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
// The recipe names the ingredients the way the owner said them: the use
// case writes the name she said and keeps the link the agent sent when the
// catalog of the business holds it. The catalog is never consulted to
// decide what the recipe names, and it never refuses a recipe: a recipe is
// knowledge and not stock, and it is written before the first purchase.
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
// The ingredients of the recipe are the ingredients the owner named: the
// use case links them to the catalog of the business when the catalog holds
// that name, and it never refuses a recipe because the catalog does not
// hold a name. The catalog is the one owner of the ingredient, and the
// recipe is the one owner of the knowledge.
func (s *RecipeService) Record(ctx context.Context, request ports.RecordRecipe) (recipe.Recipe, error) {
	if s == nil || s.recipes == nil || s.ingredients == nil {
		return recipe.Recipe{}, ports.ErrNoDatabase
	}

	catalog, err := s.catalog(ctx, request.BusinessID)
	if err != nil {
		return recipe.Recipe{}, err
	}

	definition, err := definitionOf(request, catalog)
	if err != nil {
		return recipe.Recipe{}, err
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

// linkOfTheCatalog answers the name of an ingredient of a recipe and the
// link it carries.
//
// The rules of the use case:
//
//   - the name the owner said is the name of the ingredient, and it is the
//     only thing the recipe needs: the catalog is asked when the money or
//     the stock is needed, and a recipe written before the first purchase
//     lives unlinked and works all the same;
//   - the link the agent sent is kept when the catalog of the business
//     holds it: the link is the memory of what the owner meant when her
//     wording was not enough, and it never decides the ingredient of the
//     recipe;
//   - a link the catalog of the business does not hold is dropped: the
//     recipe is written with the name alone, and a recipe is never
//     refused for a link.
func linkOfTheCatalog(one ports.RecipeIngredient, catalog ingredient.Catalog) (string, ingredient.ID) {
	name := strings.TrimSpace(one.Name)

	if one.IngredientID == 0 {
		return name, 0
	}

	item, held := catalog.Find(ingredient.ID(one.IngredientID))
	if !held {
		return name, 0
	}

	if name == "" {
		name = item.Name
	}

	return name, item.ID
}

// definitionOf turns what the agent sent into the definition of the domain.
//
// The rules are the ones of the domain: the use case asks the domain to
// build the ingredients, the steps and the yield, and it never builds them
// by itself.
//
// The ingredients are the ingredients the owner named: the use case writes
// the name she said and keeps the link the agent sent, and it never refuses
// a recipe because the catalog does not hold a name. The catalog is the one
// owner of the ingredient, and the recipe is the one owner of the knowledge.
func definitionOf(request ports.RecordRecipe, catalog ingredient.Catalog) (recipe.Definition, error) {
	ingredients := make([]recipe.Ingredient, 0, len(request.Ingredients))
	for _, one := range request.Ingredients {
		name, link := linkOfTheCatalog(one, catalog)

		built, err := recipe.NewIngredient(name, link, one.Grams)
		if err != nil {
			return recipe.Definition{}, err
		}

		ingredients = append(ingredients, built)
	}

	steps := make([]recipe.Step, 0, len(request.Steps))
	for _, one := range request.Steps {
		kind, err := recipe.ParseLaborKind(one.LaborKind)
		if err != nil {
			return recipe.Definition{}, err
		}

		built, err := recipe.NewStep(one.Instruction, one.Cue, one.Minutes, one.Temperature, kind)
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
