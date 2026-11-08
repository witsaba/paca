package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/store"
)

// WasteService is the use case that writes the waste of the bakery.
//
// The owner says what she lost and confirms it; the domain decides what a
// waste is, and the use case writes it in the book of the business.
//
// The waste is a movement of the book: the units that leave the book are
// the units of the product that expire first, and the money of the waste is
// the money of those units - the cost the waste releases. The money is never
// lost from the book: it leaves the inventory and it lands in the account of
// the waste.
//
// The waste and the movement of its money are written in one transaction: a
// waste whose money did not move lies about what the bakery lost.
type WasteService struct {
	productions recipe.ProductionReader
	units       recipe.UnitInTheBookReader
	accounts    account.Repository
	book        ledger.Repository
	wastes      recipe.WasteRepository
	work        store.UnitOfWork
	clock       ports.Clock
}

// NewWasteService builds the use case of the waste.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The productions and
// the units are two of them: the money of a unit is the money of the
// production it came out of, and only the units that are still in the book
// can leave it. The transaction is the port of the work of the storage: the
// waste and the movement of its money are written whole or not at all. The
// clock is the port of the time: the day a waste names none is the day of
// today, and the service does not read the clock of the machine by itself.
func NewWasteService(
	productions recipe.ProductionReader,
	units recipe.UnitInTheBookReader,
	accounts account.Repository,
	book ledger.Repository,
	wastes recipe.WasteRepository,
	work store.UnitOfWork,
	clock ports.Clock,
) *WasteService {
	return &WasteService{
		productions: productions,
		units:       units,
		accounts:    accounts,
		book:        book,
		wastes:      wastes,
		work:        work,
		clock:       clock,
	}
}

// Record writes the waste the owner confirmed.
//
// The product is the one the owner names: her name or the identity of it.
// The name of a product is not unique inside the business, and the book
// never guesses which product she lost: a name that more than one product
// carries is refused, and the agent asks the owner which one she meant.
//
// The rules of the domain hold the recording: a waste without a product,
// without a whole number of units, without a reason of the book, without the
// day it happened and without the word of the owner is refused, and so is a
// waste of units the book does not hold. A refused waste leaves nothing
// behind: the waste and the movement of its money are written in one
// transaction.
//
// The day is the one the agent names, and the day of today when it names
// none: the day is one of the three things Paca may fill by herself.
func (s *WasteService) Record(ctx context.Context, request ports.RecordWaste) (recipe.Waste, error) {
	if s == nil || s.productions == nil || s.units == nil || s.accounts == nil ||
		s.book == nil || s.wastes == nil || s.work == nil || s.clock == nil {
		return recipe.Waste{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return recipe.Waste{}, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	reason, err := recipe.ParseWasteReason(request.Reason)
	if err != nil {
		return recipe.Waste{}, err
	}

	productions, err := s.productions.List(ctx, request.BusinessID)
	if err != nil {
		return recipe.Waste{}, err
	}

	held, err := s.units.ListInTheBook(ctx, request.BusinessID)
	if err != nil {
		return recipe.Waste{}, err
	}

	waste, err := recipe.WasteOf(request.BusinessID, productions, held, recipe.WasteDefinition{
		RecipeID:      recipe.ID(request.RecipeID),
		RecipeName:    request.RecipeName,
		Quantity:      request.Quantity,
		Unit:          request.Unit,
		Reason:        reason,
		Day:           day,
		Utterance:     request.Utterance,
		ConfirmedWith: request.ConfirmedWith,
	})
	if err != nil {
		return recipe.Waste{}, err
	}

	movement, err := s.movementOf(ctx, waste)
	if err != nil {
		return recipe.Waste{}, err
	}

	var recorded recipe.Waste

	err = s.work.Do(ctx, func(ctx context.Context) error {
		written, err := s.wastes.Record(ctx, waste)
		if err != nil {
			return err
		}

		recorded = written

		if movement == nil {
			return nil
		}

		_, err = s.book.Record(ctx, *movement)

		return err
	})
	if err != nil {
		return recipe.Waste{}, err
	}

	return recorded, nil
}

// movementOf answers the movement of the money of the waste.
//
// The money of the waste lands in the account of the waste and it comes from
// the inventory: the product never reached a client, and the money it was
// worth is not lost from the book.
//
// The answer is nothing when the waste costs nothing: there is no money to
// move, and the book never writes a movement of nothing.
//
// The error is account.ErrNotFound when the book of the business holds no
// account of the waste or no inventory, and ledger.ErrInvalidAmount when
// the two accounts are the same one.
func (s *WasteService) movementOf(ctx context.Context, waste recipe.Waste) (*ledger.Entry, error) {
	if waste.Money <= 0 {
		return nil, nil
	}

	accounts, err := s.accounts.List(ctx, waste.BusinessID)
	if err != nil {
		return nil, err
	}

	ofWaste, err := account.OfRole(accounts, account.RoleWaste)
	if err != nil {
		return nil, fmt.Errorf("%w: the business has no account for the waste", err)
	}

	inventory, err := purchase.TheInventoryAccount(accounts)
	if err != nil {
		return nil, err
	}

	movement, err := waste.Movement(ofWaste.ID, inventory.ID)
	if err != nil {
		return nil, err
	}

	return &movement, nil
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of today
// when the agent names none.
func (s *WasteService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *WasteService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
