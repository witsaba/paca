package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/store"
)

// ExpiryService is the use case of the expiry of what the bakery makes.
//
// The shelf life of a recipe is a declared constant, and it is approximate:
// the owner corrects it when she knows it better. The correction is a fact
// of the business, it is kept, and the last one the owner declared is the
// shelf life that counts.
//
// Every unit of a production carries the day it expires, and the day is the
// day of the production plus the shelf life of the recipe. The correction
// moves the day of the units that were already made, and the use case writes
// the correction and the days in one transaction: a correction that moved
// no unit is a correction the book cannot tell the truth about.
//
// The use case never decides what expires: the domain builds the correction
// and the days of the units, and the use case carries the names and the
// numbers to it.
type ExpiryService struct {
	recipes     recipe.Reader
	corrections recipe.ShelfLifeRepository
	units       recipe.UnitRepository
	work        store.UnitOfWork
	clock       ports.Clock
}

// NewExpiryService builds the use case of the expiry.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The transaction is
// the port of the work of the storage: the correction and the days of the
// units are written whole or not at all. The clock is the port of the time:
// the day a correction names none is the day of today, and the service does
// not read the clock of the machine by itself.
func NewExpiryService(
	recipes recipe.Reader,
	corrections recipe.ShelfLifeRepository,
	units recipe.UnitRepository,
	work store.UnitOfWork,
	clock ports.Clock,
) *ExpiryService {
	return &ExpiryService{
		recipes:     recipes,
		corrections: corrections,
		units:       units,
		work:        work,
		clock:       clock,
	}
}

// Correct writes the correction of the shelf life the owner confirmed.
//
// The recipe is the one the owner names: her name or the identity of it.
// The name of a recipe is not unique inside the business, and the book never
// guesses which one she corrected.
//
// The rules of the domain hold the correction: a correction without days,
// without the day it was declared and without the word of the owner is
// refused, and so is a correction of the recipe of another business.
//
// The correction and the days of the units are written in one transaction: a
// refused correction leaves nothing behind, and a correction that moved no
// unit is not written either.
//
// The day is the one the agent names, and the day of today when it names
// none: the day is one of the three things Paca may fill by herself.
func (s *ExpiryService) Correct(ctx context.Context, request ports.CorrectShelfLife) (ports.CorrectedShelfLife, error) {
	if s == nil || s.recipes == nil || s.corrections == nil || s.units == nil ||
		s.work == nil || s.clock == nil {
		return ports.CorrectedShelfLife{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return ports.CorrectedShelfLife{}, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	baked, err := theRecipeOf(ctx, s.recipes, request.BusinessID, request.RecipeID, request.RecipeName)
	if err != nil {
		return ports.CorrectedShelfLife{}, err
	}

	correction, err := recipe.CorrectShelfLife(request.BusinessID, baked, recipe.ShelfLifeDefinition{
		RecipeID:      baked.ID,
		ShelfLifeDays: request.ShelfLifeDays,
		Day:           day,
		Utterance:     request.Utterance,
		ConfirmedWith: request.ConfirmedWith,
	})
	if err != nil {
		return ports.CorrectedShelfLife{}, err
	}

	units, err := s.units.List(ctx, request.BusinessID)
	if err != nil {
		return ports.CorrectedShelfLife{}, err
	}

	corrected, err := recipe.CorrectedUnits(units, baked.ID, correction.ShelfLifeDays)
	if err != nil {
		return ports.CorrectedShelfLife{}, err
	}

	var recorded recipe.ShelfLifeCorrection

	err = s.work.Do(ctx, func(ctx context.Context) error {
		written, err := s.corrections.Record(ctx, correction)
		if err != nil {
			return err
		}

		recorded = written

		return s.units.WriteExpiry(ctx, corrected)
	})
	if err != nil {
		return ports.CorrectedShelfLife{}, err
	}

	return ports.CorrectedShelfLife{Correction: recorded, Units: corrected}, nil
}

// Read answers what expires, and when.
//
// The units are the ones of the business, and the answer carries the day
// every one of them expires and the days that are left of it.
//
// The business is required: the expiry belongs to a business, and the use
// case refuses the business that is not there.
//
// The answer is ordered by the day the units expire: the unit that expires
// first is the first one. The rotation - the unit that leaves a production
// - is another rule, and it does not live here.
func (s *ExpiryService) Read(ctx context.Context, request ports.ReadExpiring) (recipe.Expiry, error) {
	if s == nil || s.units == nil || s.clock == nil {
		return recipe.Expiry{}, ports.ErrNoDatabase
	}

	units, err := s.units.List(ctx, request.BusinessID)
	if err != nil {
		return recipe.Expiry{}, err
	}

	return recipe.Expiry{
		Day:   s.today(),
		Units: recipe.ExpiringUnits(units),
	}, nil
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of today
// when the agent names none.
func (s *ExpiryService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *ExpiryService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}

// theRecipeOf answers the recipe the owner named.
//
// The identity of a recipe answers that recipe alone, and the name the
// owner reads answers the recipe she named: the name of a recipe is not
// unique inside the business, and a name that more than one recipe carries
// is refused, because the book never guesses which one she meant.
//
// The error is recipe.ErrNotFound when the business holds no recipe with
// that name, recipe.ErrAmbiguousRecipe when it holds more than one, and
// recipe.ErrWrongBusiness when the recipe belongs to another business.
func theRecipeOf(
	ctx context.Context,
	recipes recipe.Reader,
	own business.ID,
	id int64,
	name string,
) (recipe.Recipe, error) {
	if id != 0 {
		one, err := recipes.Find(ctx, recipe.ID(id))
		if err != nil {
			return recipe.Recipe{}, err
		}

		if one.BusinessID != own {
			return recipe.Recipe{}, fmt.Errorf("%w: the recipe %d belongs to the business %d",
				recipe.ErrWrongBusiness, one.ID, one.BusinessID)
		}

		return one, nil
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return recipe.Recipe{}, fmt.Errorf("%w: the record names no recipe", recipe.ErrMissingRecipe)
	}

	all, err := recipes.List(ctx, own)
	if err != nil {
		return recipe.Recipe{}, err
	}

	return recipe.TheRecipeNamed(all, trimmed)
}
