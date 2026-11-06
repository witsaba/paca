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

// RealPriceService is the use case that reads the report of the real price
// of a product.
//
// The report is the cost of the batch with its three layers apart — the
// direct cost, the labor and the overhead — and the price of the owner
// beside them.
//
// The use case never decides what a product costs and never decides what
// the owner charges: it reads the recipe, the layers of the stock, the
// catalog of the names, the labor and the overhead of the business, it
// reads the price the owner declared, and it asks the domain to build the
// report.
type RealPriceService struct {
	costs *BatchCostService
}

// NewRealPriceService builds the use case of the report of the real price.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else.
//
// The cost of a batch is the base of the report: the report of the real
// price is that cost with the three layers apart and the price of the
// owner beside them, and the use case of the cost is the one that reads
// what the batch costs right now.
func NewRealPriceService(
	recipes recipe.Reader,
	layers purchase.LayerRepository,
	ingredients ingredient.Repository,
	labors recipe.LaborReader,
	overheads recipe.OverheadReader,
) *RealPriceService {
	return &RealPriceService{
		costs: NewBatchCostService(recipes, layers, ingredients, labors, overheads),
	}
}

// Read answers the report of the real price of the products of the
// business.
//
// The business is required: the report belongs to a business, and the use
// case refuses the business that is not there.
//
// The recipe is optional: the identity of a recipe answers the report of
// that recipe, the name the owner reads answers the report of the recipes
// that carry it, and neither of them answers the report of every recipe of
// the business.
//
// The price of the owner is optional: it is what she declared, with the
// way she charges, and it is never inferred. A report of a price nobody
// declared says that it is not known, and the layers of the cost still
// answer.
//
// The rules are the ones of the domain: the use case reads the cost of the
// batch and the price of the owner, and it asks the domain to build the
// report. It never decides what a product costs.
func (s *RealPriceService) Read(ctx context.Context, request ports.ReadRealPrice) ([]recipe.RealPrice, error) {
	if s == nil || s.costs == nil {
		return nil, ports.ErrNoDatabase
	}

	price, err := priceOfTheOwner(request)
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

	reports := make([]recipe.RealPrice, 0, len(costs))

	for _, one := range costs {
		reports = append(reports, recipe.RealPriceOf(one, price))
	}

	return reports, nil
}

// priceOfTheOwner answers the price the owner charges, and the way she
// charges it.
//
// The rule of the domain: the price of the owner is hers, and it is never
// inferred. A request without a price carries no price, and the report
// says that it is not known.
//
// The error is the error of the domain: the domain refuses a price that is
// not a price and a way of charging the book does not know.
func priceOfTheOwner(request ports.ReadRealPrice) (purchase.OwnerPrice, error) {
	if request.Price < 0 {
		return purchase.OwnerPrice{}, fmt.Errorf("%w: the price of the owner is %d",
			purchase.ErrInvalidOwnerPrice, request.Price)
	}

	if request.Price == 0 {
		return purchase.OwnerPrice{}, nil
	}

	basis, err := purchase.ParsePriceBasis(request.PriceBasis)
	if err != nil {
		return purchase.OwnerPrice{}, err
	}

	return purchase.NewOwnerPrice(ledger.Amount(request.Price), basis)
}
