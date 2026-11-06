package application

import (
	"context"
	"fmt"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// StockService is the use case that reads the stock of a business.
//
// The purchase is the fact of what the business bought, and the layer is
// derived from it: the use case reads the layers of the business and asks
// the domain what the business holds.
//
// The stock of an ingredient is the sum of the layers that are still open,
// and the valuation is FIFO: the oldest layer is the first one that leaves.
// The order of the output by expiry is a different rule, and it does not
// live here.
//
// The use case never decides what the business holds: the domain builds the
// stock, and the use case carries the names and the layers to it.
type StockService struct {
	layers      purchase.LayerRepository
	ingredients ingredient.Repository
}

// NewStockService builds the use case of the stock.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The catalog of the
// names is one of them: the answer carries the name the owner reads of
// every ingredient the business holds.
func NewStockService(
	layers purchase.LayerRepository,
	ingredients ingredient.Repository,
) *StockService {
	return &StockService{layers: layers, ingredients: ingredients}
}

// Read answers what the business holds.
//
// The layers are the ones of the business, and the answer carries the
// quantity the business holds, the layers that are still open in the order
// they leave the stock, and the money they cost.
//
// The ingredient is optional: the zero identity answers the whole business,
// and an identity answers the stock of that ingredient alone.
//
// The rules are the ones of the domain: the use case reads the layers and
// the catalog, and it asks the domain to build the stock. It never decides
// what the business holds.
func (s *StockService) Read(ctx context.Context, request ports.ReadStock) (purchase.Stock, error) {
	if s == nil || s.layers == nil || s.ingredients == nil {
		return purchase.Stock{}, ports.ErrNoDatabase
	}

	layers, err := s.layers.List(ctx, request.BusinessID)
	if err != nil {
		return purchase.Stock{}, err
	}

	if request.IngredientID != 0 {
		err := s.holds(ctx, request.BusinessID, ingredient.ID(request.IngredientID))
		if err != nil {
			return purchase.Stock{}, err
		}

		layers = purchase.OfIngredient(layers, ingredient.ID(request.IngredientID))
	}

	items, err := s.ingredients.List(ctx, request.BusinessID)
	if err != nil {
		return purchase.Stock{}, err
	}

	catalog, err := ingredient.NewCatalog(request.BusinessID, items)
	if err != nil {
		return purchase.Stock{}, err
	}

	return purchase.StockOf(layers, catalog), nil
}

// holds answers whether the catalog of the business holds the ingredient.
//
// The rule of the domain: the stock names an ingredient of the catalog of
// its business, and an ingredient of another business is not in it. The
// error is ingredient.ErrNotFound, and the answer tells the agent to match
// the name against the catalog first.
func (s *StockService) holds(ctx context.Context, own business.ID, id ingredient.ID) error {
	items, err := s.ingredients.List(ctx, own)
	if err != nil {
		return err
	}

	catalog, err := ingredient.NewCatalog(own, items)
	if err != nil {
		return err
	}

	if !catalog.Holds(id) {
		return fmt.Errorf("%w: the ingredient %d is not one of the catalog of the business %d",
			ingredient.ErrNotFound, id, own)
	}

	return nil
}
