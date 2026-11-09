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

// ExitService is the use case that writes the exit of the book: the
// consumption of the owner and the donation.
//
// The owner says what she consumed or what she gave away, and confirms it;
// the domain decides what an exit is, and the use case writes it in the
// book of the business.
//
// The exit moves no money: nobody paid for the bread the owner consumed or
// for the bread the bakery gave away. The exit releases the cost of the
// units that left the book - the money of the production they came out of,
// divided by the units that production made - and that cost leaves the
// inventory and lands in the account of the expense of the exit: the cost
// of the sales for the consumption, the account of the donation for the
// donation.
//
// The tax treatment of the exit waits for the professional of the taxes of
// the owner: the movement is written and the question is named, and the
// book never assumes a rate.
//
// The exit and the movement of its cost are written in one transaction: an
// exit whose cost did not move lies about what left the book.
type ExitService struct {
	productions recipe.ProductionReader
	units       recipe.UnitInTheBookReader
	accounts    account.Repository
	book        ledger.Repository
	exits       recipe.ExitRepository
	work        store.UnitOfWork
	clock       ports.Clock
}

// NewExitService builds the use case of the exit.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The productions and
// the units are two of them: the cost of a unit is the money of the
// production it came out of, and only the units that are still in the book
// can leave it. The transaction is the port of the work of the storage: the
// exit and the movement of its cost are written whole or not at all. The
// clock is the port of the time: the day an exit names none is the day of
// today, and the service does not read the clock of the machine by itself.
func NewExitService(
	productions recipe.ProductionReader,
	units recipe.UnitInTheBookReader,
	accounts account.Repository,
	book ledger.Repository,
	exits recipe.ExitRepository,
	work store.UnitOfWork,
	clock ports.Clock,
) *ExitService {
	return &ExitService{
		productions: productions,
		units:       units,
		accounts:    accounts,
		book:        book,
		exits:       exits,
		work:        work,
		clock:       clock,
	}
}

// Record writes the exit the owner confirmed.
//
// The product is the one the owner names: her name or the identity of it.
// The name of a product is not unique inside the business, and the book
// never guesses which product she consumed or gave away: a name that more
// than one product carries is refused, and the agent asks the owner which
// one she meant.
//
// The rules of the domain hold the recording: an exit without a product,
// without a kind of the book, without a whole number of units, without the
// day it happened and without the word of the owner is refused, and so is
// an exit of units the book does not hold. A refused exit leaves nothing
// behind: the exit and the movement of its cost are written in one
// transaction.
//
// The day is the one the agent names, and the day of today when it names
// none: the day is one of the three things Paca may fill by herself.
func (s *ExitService) Record(ctx context.Context, request ports.RecordExit) (recipe.Exit, error) {
	if s == nil || s.productions == nil || s.units == nil || s.accounts == nil ||
		s.book == nil || s.exits == nil || s.work == nil || s.clock == nil {
		return recipe.Exit{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return recipe.Exit{}, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	kind, err := recipe.ParseExitKind(request.Kind)
	if err != nil {
		return recipe.Exit{}, err
	}

	productions, err := s.productions.List(ctx, request.BusinessID)
	if err != nil {
		return recipe.Exit{}, err
	}

	held, err := s.units.ListInTheBook(ctx, request.BusinessID)
	if err != nil {
		return recipe.Exit{}, err
	}

	exit, err := recipe.ExitOf(request.BusinessID, productions, held, recipe.ExitDefinition{
		RecipeID:      recipe.ID(request.RecipeID),
		RecipeName:    request.RecipeName,
		Kind:          kind,
		Quantity:      request.Quantity,
		Unit:          request.Unit,
		Day:           day,
		Utterance:     request.Utterance,
		ConfirmedWith: request.ConfirmedWith,
	})
	if err != nil {
		return recipe.Exit{}, err
	}

	var recorded recipe.Exit

	err = s.work.Do(ctx, func(ctx context.Context) error {
		// The movement of the cost is written in the transaction of the
		// exit: the account of the expense of the exit may be opened in
		// the same transaction, and an exit whose cost did not move lies
		// about what left the book.
		movement, err := s.movementOf(ctx, exit)
		if err != nil {
			return err
		}

		written, err := s.exits.Record(ctx, exit)
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
		return recipe.Exit{}, err
	}

	return recorded, nil
}

// movementOf answers the movement of the cost of the exit.
//
// The cost of the units that left the book leaves the inventory and lands
// in the account of the expense of the exit: the cost of the sales for the
// consumption of the owner, and the account of the donation for the
// donation, which is a social expense of the business.
//
// The answer is nothing when the exit costs nothing: there is no money to
// move, and the book never writes a movement of nothing.
//
// The account of the expense of the exit is the one the book of the
// business carries, and the book opens it when it does not carry it: a
// book that was opened before the role existed catches up with the seed of
// the book of a business.
//
// The error is account.ErrNotFound when the book of the business holds no
// inventory, and the error of the storage when it cannot open the account
// of the expense of the exit. The error is ledger.ErrInvalidAmount when
// the two accounts are the same one.
func (s *ExitService) movementOf(ctx context.Context, exit recipe.Exit) (*ledger.Entry, error) {
	if exit.Cost <= 0 {
		return nil, nil
	}

	accounts, err := s.accounts.List(ctx, exit.BusinessID)
	if err != nil {
		return nil, err
	}

	inventory, err := purchase.TheInventoryAccount(accounts)
	if err != nil {
		return nil, err
	}

	role := exit.TheExpenseRole()

	expense, err := account.TheAccountOfTheRole(ctx, s.accounts, exit.BusinessID, role)
	if err != nil {
		return nil, fmt.Errorf("%w: the book of the business cannot open the account of the role %q",
			err, role)
	}

	movement, err := exit.CostMovement(expense.ID, inventory.ID)
	if err != nil {
		return nil, err
	}

	return &movement, nil
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of today
// when the agent names none.
func (s *ExitService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *ExitService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
