package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/product"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/store"
)

// PriceService is the use case of the price list of the bakery.
//
// The price list is the price of the products of the bakery in the two
// channels: the retail and the wholesale. The price is a fact of the
// business, it carries the channel it was set for, and the price list is a
// history: nothing is rewritten when a price changes, and the price that
// holds today is the last one that was set on or before the day.
//
// The product of the bakery is the thing it sells: it carries the recipe it
// comes from and the name the owner reads. The product comes into being the
// first time it is named, and the name is never guessed.
//
// The use case never decides what a product or a price is: the channel, the
// money and the day are the rules of the domain, and the use case only asks
// the domain.
type PriceService struct {
	products   product.Repository
	prices     product.PriceRepository
	recipes    recipe.Reader
	businesses business.Repository
	work       store.UnitOfWork
	clock      ports.Clock
}

// NewPriceService builds the use case of the price list.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The recipes are
// one of them: the product of the bakery comes from a recipe, and the use
// case reads the recipe before it opens the product. The clock is the port
// of the time: the day of a price that names none is the day of today.
func NewPriceService(
	products product.Repository,
	prices product.PriceRepository,
	recipes recipe.Reader,
	businesses business.Repository,
	work store.UnitOfWork,
	clock ports.Clock,
) *PriceService {
	return &PriceService{
		products:   products,
		prices:     prices,
		recipes:    recipes,
		businesses: businesses,
		work:       work,
		clock:      clock,
	}
}

// Set writes the price of the product for the channel, and answers it with
// the product and the price.
//
// The rules are the ones of the domain: the use case reads the business
// that holds the price list, asks the domain whether a business of that
// kind holds a price list, asks the domain to read the channel, resolves
// the product - the one the business already holds, or the one that comes
// into being - and asks the domain to build the price. It never decides
// what a price is.
//
// The product and the price are written in one transaction: a price of a
// product that is not there is a price of nobody.
func (s *PriceService) Set(ctx context.Context, request ports.SetPrice) (product.Listed, error) {
	if s == nil || s.products == nil || s.prices == nil || s.recipes == nil ||
		s.businesses == nil || s.work == nil || s.clock == nil {
		return product.Listed{}, ports.ErrNoDatabase
	}

	owner, err := s.businesses.Find(ctx, request.BusinessID)
	if err != nil {
		return product.Listed{}, err
	}

	if err := product.OwnedBy(owner.Kind); err != nil {
		return product.Listed{}, err
	}

	channel, err := product.ParseChannel(request.Channel)
	if err != nil {
		return product.Listed{}, err
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return product.Listed{}, err
	}
	if day.Time().IsZero() {
		day = s.today()
	}

	opened, err := s.theProduct(ctx, owner.ID, request.RecipeID, request.RecipeName, request.Name)
	if err != nil {
		return product.Listed{}, err
	}

	var written product.Listed
	err = s.work.Do(ctx, func(ctx context.Context) error {
		stored := opened
		if stored.ID == 0 {
			created, err := s.products.Create(ctx, stored.WithTaxTreatment(request.TaxTreatment))
			if err != nil {
				return err
			}

			stored = created
		} else if strings.TrimSpace(request.TaxTreatment) != "" {
			// The tax treatment is a fact the owner states, and the
			// last statement is the one that counts: the name and the
			// recipe of the product never change.
			stated, err := s.products.State(ctx, stored.WithTaxTreatment(request.TaxTreatment))
			if err != nil {
				return err
			}

			stored = stated
		}

		price, err := product.NewPrice(owner.ID, stored.ID, channel, ledger.Amount(request.Amount), day)
		if err != nil {
			return err
		}

		set, err := s.prices.Set(ctx, price)
		if err != nil {
			return err
		}

		written = product.Listed{Product: stored, Price: set}

		return nil
	})
	if err != nil {
		return product.Listed{}, err
	}

	return written, nil
}

// Read answers the price list of the business.
//
// The rules are the ones of the domain: the use case reads the products and
// the prices of the business and hands them to the domain, and the domain
// answers the price that holds for every product and every channel on the
// day that is asked for.
func (s *PriceService) Read(ctx context.Context, request ports.ReadPrices) ([]product.Listed, error) {
	if s == nil || s.products == nil || s.prices == nil || s.recipes == nil || s.businesses == nil {
		return nil, ports.ErrNoDatabase
	}

	owner, err := s.businesses.Find(ctx, request.BusinessID)
	if err != nil {
		return nil, err
	}

	if err := product.OwnedBy(owner.Kind); err != nil {
		return nil, err
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return nil, err
	}
	if day.Time().IsZero() {
		day = s.today()
	}

	channel := product.Channel("")
	if strings.TrimSpace(request.Channel) != "" {
		channel, err = product.ParseChannel(request.Channel)
		if err != nil {
			return nil, err
		}
	}

	products, err := s.products.List(ctx, owner.ID)
	if err != nil {
		return nil, err
	}

	prices, err := s.prices.List(ctx, owner.ID)
	if err != nil {
		return nil, err
	}

	of := product.ID(0)
	if request.RecipeID != 0 || strings.TrimSpace(request.RecipeName) != "" ||
		strings.TrimSpace(request.Name) != "" {
		asked, err := s.theProduct(ctx, owner.ID, request.RecipeID, request.RecipeName, request.Name)
		if err != nil {
			return nil, err
		}

		if asked.ID == 0 {
			return nil, fmt.Errorf("%w: the business %d holds no product of that recipe",
				product.ErrNotFound, owner.ID)
		}

		of = asked.ID
	}

	return product.ThePriceList(prices, products, day, of, channel), nil
}

// theProduct answers the product the owner names.
//
// The rules are the ones of the domain: the product is named by the recipe
// it comes from or by the name the owner reads, and the name the owner
// reads is the name of the recipe when she named none.
//
// The product that the business already holds is the one that is answered,
// with the identity it carries. The product that comes into being is
// answered without one: the storage generates it when it writes the row.
//
// The error is recipe.ErrUnknownRecipe when the agent named a recipe the
// business does not hold, recipe.ErrAmbiguousRecipe when it holds more
// than one with that name, and product.ErrUnknownRecipe when the name is
// neither a product nor a recipe of the business.
func (s *PriceService) theProduct(
	ctx context.Context,
	own business.ID,
	recipeID int64,
	recipeName string,
	name string,
) (product.Product, error) {
	held, err := s.products.List(ctx, own)
	if err != nil {
		return product.Product{}, err
	}

	baked, err := s.recipeOf(ctx, own, recipeID, recipeName)
	if err != nil {
		return product.Product{}, err
	}

	trimmed := strings.TrimSpace(name)

	if baked == nil {
		known := product.Named(held, trimmed)
		if len(known) == 1 {
			return known[0], nil
		}

		baked, err = s.recipeNamed(ctx, own, trimmed)
		if err != nil {
			if errors.Is(err, recipe.ErrAmbiguousRecipe) {
				return product.Product{}, err
			}

			return product.Product{}, fmt.Errorf("%w: the business %d holds no product and no recipe named %q",
				product.ErrUnknownRecipe, own, trimmed)
		}
	}

	if trimmed == "" {
		trimmed = baked.Name
	}

	return product.TheProduct(own, held, baked.ID, trimmed)
}

// recipeOf answers the recipe the product comes from, when the agent named
// it.
//
// The rule of the domain: the product of the bakery comes from a recipe,
// and the recipe is named with its identity or with its name.
func (s *PriceService) recipeOf(
	ctx context.Context,
	own business.ID,
	recipeID int64,
	recipeName string,
) (*recipe.Recipe, error) {
	if recipeID != 0 {
		found, err := s.recipes.Find(ctx, recipe.ID(recipeID))
		if err != nil {
			return nil, err
		}

		return &found, nil
	}

	if strings.TrimSpace(recipeName) != "" {
		return s.recipeNamed(ctx, own, recipeName)
	}

	return nil, nil
}

// recipeNamed answers the recipe the business holds with that name.
//
// The error is recipe.ErrUnknownRecipe when the business holds none and
// recipe.ErrAmbiguousRecipe when it holds more than one: the book never
// guesses which recipe the owner meant.
func (s *PriceService) recipeNamed(
	ctx context.Context,
	own business.ID,
	name string,
) (*recipe.Recipe, error) {
	recipes, err := s.recipes.List(ctx, own)
	if err != nil {
		return nil, err
	}

	found, err := recipe.TheRecipeNamed(recipes, name)
	if err != nil {
		return nil, err
	}

	return &found, nil
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of
// today when the agent names none.
func (s *PriceService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *PriceService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
