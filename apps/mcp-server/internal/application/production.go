package application

import (
	"context"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/store"
)

// ProductionService is the use case that writes the production of the
// bakery.
//
// The owner says what she made and confirms it; the domain decides what a
// production is, and the use case writes it in the book of the business.
//
// The production is derived, never typed: the grams that went in are the
// grams the layers of the stock gave, the money is the money of those
// layers, and the units that came out are the yield of the recipe. The book
// never asks the owner for a number it can read.
//
// The production and the layers it consumes are written in one
// transaction: a production whose layers did not leave the stock is not a
// production, and a refused production leaves nothing behind.
type ProductionService struct {
	recipes     recipe.Reader
	corrections recipe.ShelfLifeReader
	layers      purchase.LayerRepository
	ingredients ingredient.Repository
	productions recipe.ProductionRepository
	work        store.UnitOfWork
	clock       ports.Clock
}

// NewProductionService builds the use case of the productions.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The corrections of
// the shelf life are one of them: the shelf life that counts is the last
// correction the owner declared, and the units that come out of the oven
// expire the day the production plus that shelf life. The transaction is
// the port of the work of the storage: the production and the layers it
// consumes are written whole or not at all. The clock is the port of the
// time: the day a production names none is the day of today, and the
// service does not read the clock of the machine by itself.
func NewProductionService(
	recipes recipe.Reader,
	corrections recipe.ShelfLifeReader,
	layers purchase.LayerRepository,
	ingredients ingredient.Repository,
	productions recipe.ProductionRepository,
	work store.UnitOfWork,
	clock ports.Clock,
) *ProductionService {
	return &ProductionService{
		recipes:     recipes,
		corrections: corrections,
		layers:      layers,
		ingredients: ingredients,
		productions: productions,
		work:        work,
		clock:       clock,
	}
}

// Record writes the production the owner confirmed.
//
// The recipe is the one the owner names: her name or the identity of it.
// The name of a recipe is not unique inside the business, and the book never
// guesses which one she made: a name that more than one recipe carries is
// refused, and the agent asks the owner which one she meant.
//
// The rules of the domain hold the recording: a production without a recipe,
// without the day it was made and without the word of the owner is refused,
// and so is a production the stock cannot cover. A refused production leaves
// nothing behind: the production and the layers it consumes are written in
// one transaction.
//
// The day is the one the agent names, and the day of today when it names
// none: the day is one of the three things Paca may fill by herself.
func (s *ProductionService) Record(ctx context.Context, request ports.RecordProduction) (recipe.Production, error) {
	if s == nil || s.recipes == nil || s.corrections == nil || s.layers == nil ||
		s.ingredients == nil || s.productions == nil || s.work == nil || s.clock == nil {
		return recipe.Production{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return recipe.Production{}, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	baked, err := theRecipeOf(ctx, s.recipes, request.BusinessID, request.RecipeID, request.RecipeName)
	if err != nil {
		return recipe.Production{}, err
	}

	corrections, err := s.corrections.List(ctx, request.BusinessID)
	if err != nil {
		return recipe.Production{}, err
	}

	// The shelf life that counts is the last correction the owner
	// declared: the units that come out of the oven expire the day of the
	// production plus that shelf life.
	baked = baked.WithShelfLife(recipe.ShelfLifeOf(baked, corrections))

	layers, err := s.layers.List(ctx, request.BusinessID)
	if err != nil {
		return recipe.Production{}, err
	}

	catalog, err := s.catalog(ctx, request.BusinessID)
	if err != nil {
		return recipe.Production{}, err
	}

	production, err := recipe.ProductionOf(baked, layers, catalog, recipe.ProductionDefinition{
		Day:           day,
		Utterance:     request.Utterance,
		ConfirmedWith: request.ConfirmedWith,
	})
	if err != nil {
		return recipe.Production{}, err
	}

	var recorded recipe.Production

	err = s.work.Do(ctx, func(ctx context.Context) error {
		written, err := s.productions.Record(ctx, production)
		if err != nil {
			return err
		}

		recorded = written

		return s.layers.WriteRemaining(ctx, production.Remaining)
	})
	if err != nil {
		return recipe.Production{}, err
	}

	return recorded, nil
}

// catalog answers the catalog of the names the business buys.
//
// The catalog is what the use case asks when the owner made a recipe: the
// conversion of a package into grams is the one the owner declared, and it
// never assumes the weight of a piece.
func (s *ProductionService) catalog(ctx context.Context, own business.ID) (ingredient.Catalog, error) {
	items, err := s.ingredients.List(ctx, own)
	if err != nil {
		return ingredient.Catalog{}, err
	}

	return ingredient.NewCatalog(own, items)
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of today
// when the agent names none.
func (s *ProductionService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *ProductionService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
