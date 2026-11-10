package application

import (
	"context"
	"fmt"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// CapacityService is the use case that reads the capacity of a weekend.
//
// The capacity is what the hours the owner declares carry of the recipes of
// the business, and it is derived from the labor of the recipes and from
// those hours: it is never typed, and it is never a promise of the calendar.
//
// The use case reads the recipes of the business through the reader of the
// book, the way the cost of a batch does, and it asks the domain what the
// weekend can carry. It never decides a rule and it never re-states one.
type CapacityService struct {
	recipes recipe.Reader
}

// NewCapacityService builds the use case of the capacity.
//
// The reader is the port of the storage: the use case does not know whether
// the rows live in PostgreSQL or somewhere else. The capacity touches no
// storage of its own: the book already holds the recipes, and the capacity is
// derived from them on the spot.
func NewCapacityService(recipes recipe.Reader) *CapacityService {
	return &CapacityService{recipes: recipes}
}

// Read answers the hours the owner declared and the capacity of every recipe
// of the business.
//
// The business is required: the capacity is read for a business, and a
// request that names none is a request of nobody. The refusal is the one of
// the recipes of the business: the capacity reads the recipes, and the
// message the agent reads for it never tells it that it is reading the
// stock.
//
// The rules are the ones of the domain: the use case reads the recipes of the
// business and asks the domain for the capacity. The hours that are not
// positive, the recipe of another business and the step that is not a step of
// the book are refused by the domain, and the use case never refuses them
// before it.
func (s *CapacityService) Read(ctx context.Context, request ports.ReadCapacity) (recipe.Capacity, error) {
	if s == nil || s.recipes == nil {
		return recipe.Capacity{}, ports.ErrNoDatabase
	}

	if request.BusinessID <= 0 {
		return recipe.Capacity{}, fmt.Errorf("%w: the capacity is read for the business %d",
			recipe.ErrMissingBusiness, request.BusinessID)
	}

	own := request.BusinessID

	baked, err := s.recipes.List(ctx, own)
	if err != nil {
		return recipe.Capacity{}, err
	}

	return recipe.TheCapacity(own, request.Hours, baked)
}
