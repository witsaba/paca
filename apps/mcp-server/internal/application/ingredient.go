package application

import (
	"context"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/ingredient"
)

// IngredientService is the use case that matches a name the owner said
// against the catalog of the names the business buys.
//
// The agent reads the sentence of the owner and asks the catalog what the
// name she said is: the ingredient it names, the options it offers when
// several answer it, or the news that the catalog does not know it.
//
// The use case never decides which ingredient the owner meant: the rule of
// the domain decides, and the use case only carries the names of the
// business to the catalog and the answer back to the agent.
type IngredientService struct {
	ingredients ingredient.Repository
}

// NewIngredientService builds the use case of the catalog.
//
// The repository is the port of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else.
func NewIngredientService(ingredients ingredient.Repository) *IngredientService {
	return &IngredientService{ingredients: ingredients}
}

// Match answers the name the owner said against the catalog of the
// business.
//
// The rules are the ones of the domain: the use case reads the catalog and
// asks the domain to match the name. It never decides which ingredient the
// owner meant, and it never refuses a name the catalog does not know: a
// name the catalog does not know is not an error, it is a name that is
// new.
func (s *IngredientService) Match(ctx context.Context, request ports.MatchItems) (ingredient.Match, error) {
	if s == nil || s.ingredients == nil {
		return ingredient.Match{}, ports.ErrNoDatabase
	}

	items, err := s.ingredients.List(ctx, request.BusinessID)
	if err != nil {
		return ingredient.Match{}, err
	}

	catalog, err := ingredient.NewCatalog(request.BusinessID, items)
	if err != nil {
		return ingredient.Match{}, err
	}

	return catalog.Match(request.Name)
}
