package application

import (
	"context"
	"fmt"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// MarginService is the use case that reads the margin of a product.
//
// The margin compares the price of the owner with the direct cost of the
// batch, and with the reference of the market: what she charges, what the
// ingredients cost her, and what the market says they cost.
//
// The use case never decides what a product earns and never decides what
// the market says: it reads the recipe, the layers of the stock, the
// catalog of the names, the references of the market and the price the
// owner declared, and it asks the domain to build the margin.
type MarginService struct {
	costs       *BatchCostService
	references  purchase.ReferenceRepository
	ingredients ingredient.Repository
	clock       ports.Clock
}

// NewMarginService builds the use case of the margin.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The clock is the
// port of the time: the age of a reference is counted against the day of
// the reading, and the service does not read the clock of the machine by
// itself.
//
// The cost of a batch is the base of the margin: the margin is that cost
// with the price of the owner beside it and the reference of the market
// next to it.
func NewMarginService(
	costs *BatchCostService,
	references purchase.ReferenceRepository,
	ingredients ingredient.Repository,
	clock ports.Clock,
) *MarginService {
	return &MarginService{
		costs:       costs,
		references:  references,
		ingredients: ingredients,
		clock:       clock,
	}
}

// Read answers the margin of the products of the business.
//
// The business is required: the margin belongs to a business, and the use
// case refuses the business that is not there.
//
// The recipe is optional: the identity of a recipe answers the margin of
// that recipe, the name the owner reads answers the margin of the recipes
// that carry it, and neither of them answers the margin of every recipe of
// the business.
//
// The price of the owner is optional: it is what she declared, with the
// way she charges, and it is never inferred. A margin of a price nobody
// declared says that it is not known, and the cost and the market still
// answer.
//
// The expectation of the age is optional: it is the number of days a
// reference of the market may be old. Every reference shows its age
// anyway, and a negative number of days is refused.
//
// The rules are the ones of the domain: the use case reads the cost of the
// batch, the references and the catalog, and it asks the domain to build
// the margin. It never decides what a product earns.
func (s *MarginService) Read(ctx context.Context, request ports.ReadMargin) ([]recipe.Margin, error) {
	if s == nil || s.costs == nil || s.references == nil || s.ingredients == nil || s.clock == nil {
		return nil, ports.ErrNoDatabase
	}

	// The expectation of the age is a number of days: a negative number
	// of days is not an age anybody expects, and a refused request reads
	// nothing.
	if request.MaxAgeDays < 0 {
		return nil, fmt.Errorf("%w: the expectation of the age is %d days",
			ports.ErrInvalidMaxAgeDays, request.MaxAgeDays)
	}

	price, err := ownerPriceOf(request.Price, request.PriceBasis)
	if err != nil {
		return nil, err
	}

	costs, err := s.costs.Read(ctx, ports.ReadBatchCost{
		BusinessID: request.BusinessID,
		RecipeID:   request.RecipeID,
		RecipeName: request.RecipeName,
	})
	if err != nil {
		return nil, err
	}

	// The references of the business: every one of them, because the
	// margin shows the history of what the market was seen at, and the
	// domain picks the ones of every ingredient of the recipe.
	references, err := s.references.List(ctx, request.BusinessID, ingredient.ID(0))
	if err != nil {
		return nil, err
	}

	// The catalog of the names: the weight of a package of an ingredient is
	// what the owner declared, and the market is read against it.
	items, err := s.ingredients.List(ctx, request.BusinessID)
	if err != nil {
		return nil, err
	}

	catalog, err := ingredient.NewCatalog(request.BusinessID, items)
	if err != nil {
		return nil, err
	}

	day := ledger.NewDate(s.clock.Now())
	expectation := recipe.ReferenceExpectation{MaxAgeDays: request.MaxAgeDays}

	margins := make([]recipe.Margin, 0, len(costs))

	for _, one := range costs {
		margins = append(margins, recipe.MarginOf(one, price, references, catalog, expectation, day))
	}

	return margins, nil
}
