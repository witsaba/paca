package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// ReferenceService is the use case of the reference prices.
//
// The agent researches a price in the market of the owner and proposes what
// it found; the domain decides what a reference is, and the use case writes
// it in the book of the business.
//
// A reference is never the price of the ingredient: the price the owner
// pays comes from her own purchases. The use case writes what the domain
// built, and it never turns a reference into a price.
//
// The reference names an ingredient of the catalog of the business: the use
// case reads the catalog and refuses an ingredient it does not hold.
type ReferenceService struct {
	references  purchase.ReferenceRepository
	ingredients ingredient.Repository
	clock       ports.Clock
}

// NewReferenceService builds the use case of the reference prices.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The clock is the
// port of the time: the day a reference names none is the day of today, and
// the service does not read the clock of the machine by itself.
func NewReferenceService(
	references purchase.ReferenceRepository,
	ingredients ingredient.Repository,
	clock ports.Clock,
) *ReferenceService {
	return &ReferenceService{references: references, ingredients: ingredients, clock: clock}
}

// Record writes the reference the owner confirmed.
//
// The rules of the domain hold the recording: a reference without the
// ingredient of the catalog, without a price, without its unit, without its
// source, without its day and without the word of the owner is refused, and
// a refused reference leaves nothing behind.
//
// The day is the one the agent names, and the day of today when it names
// none: the day is one of the three things Paca may fill by herself.
func (s *ReferenceService) Record(ctx context.Context, request ports.RecordReference) (purchase.Reference, error) {
	if s == nil || s.references == nil || s.ingredients == nil || s.clock == nil {
		return purchase.Reference{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return purchase.Reference{}, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	err = s.holds(ctx, request.BusinessID, ingredient.ID(request.IngredientID))
	if err != nil {
		return purchase.Reference{}, err
	}

	reference, err := purchase.NewReference(request.BusinessID, purchase.ReferenceDefinition{
		IngredientID:  ingredient.ID(request.IngredientID),
		Price:         ledger.Amount(request.Price),
		Unit:          request.Unit,
		Market:        request.Market,
		Source:        request.Source,
		SeenOn:        day,
		Utterance:     request.Utterance,
		ConfirmedWith: request.ConfirmedWith,
	})
	if err != nil {
		return purchase.Reference{}, err
	}

	return s.references.Record(ctx, reference)
}

// Read answers the references of a business.
//
// The references are read for one ingredient, or for every one of them: the
// ingredient is optional, and the zero identity answers the whole business.
//
// The references carry their source and their day: a reference without them
// is a price nobody can check.
func (s *ReferenceService) Read(ctx context.Context, request ports.ReadReferences) ([]purchase.Reference, error) {
	if s == nil || s.references == nil || s.ingredients == nil {
		return nil, ports.ErrNoDatabase
	}

	if request.IngredientID != 0 {
		err := s.holds(ctx, request.BusinessID, ingredient.ID(request.IngredientID))
		if err != nil {
			return nil, err
		}
	}

	return s.references.List(ctx, request.BusinessID, ingredient.ID(request.IngredientID))
}

// holds answers whether the catalog of the business holds the ingredient.
//
// The rule of the domain: a reference names an ingredient of the catalog of
// its business, and an ingredient of another business is not in it. The
// error is ingredient.ErrNotFound, and the answer tells the agent to match
// the name against the catalog first.
func (s *ReferenceService) holds(ctx context.Context, own business.ID, id ingredient.ID) error {
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

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of today
// when the agent names none.
func (s *ReferenceService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *ReferenceService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
