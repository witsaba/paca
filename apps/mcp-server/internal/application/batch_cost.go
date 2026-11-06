package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// BatchCostService is the use case that reads the cost of a batch.
//
// The cost of a batch is derived from the layers that the recipe consumes,
// and never typed: the use case reads the recipe, the layers of the stock
// and the catalog of the names, and it asks the domain what the batch
// costs.
//
// The recipe consumes the layers in the order of the FIFO: the oldest layer
// is the first one that leaves, and the money of what is taken is the money
// of the layer, pro rata the quantity that was taken.
//
// The use case never decides what a batch costs: the domain builds the
// cost, and the use case carries the recipe and the layers to it.
type BatchCostService struct {
	recipes     recipe.Reader
	layers      purchase.LayerRepository
	ingredients ingredient.Repository
}

// NewBatchCostService builds the use case of the cost of a batch.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The catalog of the
// names is one of them: the answer carries the name the owner reads of
// every ingredient of the recipe.
func NewBatchCostService(
	recipes recipe.Reader,
	layers purchase.LayerRepository,
	ingredients ingredient.Repository,
) *BatchCostService {
	return &BatchCostService{recipes: recipes, layers: layers, ingredients: ingredients}
}

// Read answers the cost of the batches of the recipes of the business.
//
// The business is required: the cost of a batch belongs to a business, and
// the use case refuses the business that is not there.
//
// The recipe is optional: the identity of a recipe answers the cost of
// that recipe, the name the owner reads answers the cost of the recipes
// that carry it, and neither of them answers the cost of every recipe of
// the business.
//
// The rules are the ones of the domain: the use case reads the recipe, the
// layers and the catalog, and it asks the domain to build the cost. It
// never decides what a batch costs.
func (s *BatchCostService) Read(ctx context.Context, request ports.ReadBatchCost) ([]recipe.Cost, error) {
	if s == nil || s.recipes == nil || s.layers == nil || s.ingredients == nil {
		return nil, ports.ErrNoDatabase
	}

	own, err := purchase.TheBusinessOfTheStock(request.BusinessID)
	if err != nil {
		return nil, err
	}

	baked, err := s.recipesOf(ctx, own, request)
	if err != nil {
		return nil, err
	}

	layers, err := s.layers.List(ctx, own)
	if err != nil {
		return nil, err
	}

	items, err := s.ingredients.List(ctx, own)
	if err != nil {
		return nil, err
	}

	catalog, err := ingredient.NewCatalog(own, items)
	if err != nil {
		return nil, err
	}

	costs := make([]recipe.Cost, 0, len(baked))

	for _, one := range baked {
		cost, err := recipe.CostOf(one, layers, catalog)
		if err != nil {
			return nil, err
		}

		costs = append(costs, cost)
	}

	return costs, nil
}

// recipesOf answers the recipes the caller asks about.
//
// The identity of a recipe answers that recipe alone, and the name the
// owner reads answers the recipes that carry it: the name of a recipe is
// not unique inside the business, and the owner chooses with one word.
//
// The error is recipe.ErrNotFound when the business holds no recipe with
// that name, and recipe.ErrWrongBusiness when the recipe belongs to
// another business.
func (s *BatchCostService) recipesOf(
	ctx context.Context,
	own business.ID,
	request ports.ReadBatchCost,
) ([]recipe.Recipe, error) {
	if request.RecipeID != 0 {
		one, err := s.recipes.Find(ctx, recipe.ID(request.RecipeID))
		if err != nil {
			return nil, err
		}

		if one.BusinessID != own {
			return nil, fmt.Errorf("%w: the recipe %d belongs to the business %d",
				recipe.ErrWrongBusiness, one.ID, one.BusinessID)
		}

		return []recipe.Recipe{one}, nil
	}

	all, err := s.recipes.List(ctx, own)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(request.RecipeName)
	if name == "" {
		return all, nil
	}

	found := recipe.Named(all, name)
	if len(found) == 0 {
		return nil, fmt.Errorf("%w: the business %d holds no recipe named %q", recipe.ErrNotFound, own, name)
	}

	return found, nil
}
