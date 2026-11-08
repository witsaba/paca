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

// SaleService is the use case that writes the sale of the bakery.
//
// The owner says what she sold and confirms it; the domain decides what a
// sale is, and the use case writes it in the book of the business.
//
// The sale moves the money in and it releases the cost of the units it
// sells, in one transaction: the money lands in the account that receives
// it - the cash of the business, or the account the owner names - and it
// comes from the account of the sales. The cost of the units that left the
// book leaves the inventory and lands in the account of the cost of the
// sales.
//
// A sale whose money did not move lies about what the bakery sold, and a
// sale whose cost did not move lies about what it cost: the sale and the
// two movements of its money are written whole or not at all.
type SaleService struct {
	productions recipe.ProductionReader
	units       recipe.UnitInTheBookReader
	accounts    account.Repository
	book        ledger.Repository
	sales       recipe.SaleRepository
	work        store.UnitOfWork
	clock       ports.Clock
}

// NewSaleService builds the use case of the sale.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The productions and
// the units are two of them: the cost of a unit is the money of the
// production it came out of, and only the units that are still in the book
// can be sold. The transaction is the port of the work of the storage: the
// sale and the movements of its money are written whole or not at all. The
// clock is the port of the time: the day a sale names none is the day of
// today, and the service does not read the clock of the machine by itself.
func NewSaleService(
	productions recipe.ProductionReader,
	units recipe.UnitInTheBookReader,
	accounts account.Repository,
	book ledger.Repository,
	sales recipe.SaleRepository,
	work store.UnitOfWork,
	clock ports.Clock,
) *SaleService {
	return &SaleService{
		productions: productions,
		units:       units,
		accounts:    accounts,
		book:        book,
		sales:       sales,
		work:        work,
		clock:       clock,
	}
}

// Record writes the sale the owner confirmed.
//
// The product is the one the owner names: her name or the identity of it.
// The name of a product is not unique inside the business, and the book
// never guesses which product she sold: a name that more than one product
// carries is refused, and the agent asks the owner which one she meant.
//
// The rules of the domain hold the recording: a sale without a product,
// without a whole number of units, without money, without the day it
// happened and without the word of the owner is refused, and so is the sale
// of a unit that expired. A refused sale leaves nothing behind: the sale
// and the movements of its money are written in one transaction.
//
// The day is the one the agent names, and the day of today when it names
// none: the day is one of the three things Paca may fill by herself.
func (s *SaleService) Record(ctx context.Context, request ports.RecordSale) (recipe.Sale, error) {
	if s == nil || s.productions == nil || s.units == nil || s.accounts == nil ||
		s.book == nil || s.sales == nil || s.work == nil || s.clock == nil {
		return recipe.Sale{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return recipe.Sale{}, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	productions, err := s.productions.List(ctx, request.BusinessID)
	if err != nil {
		return recipe.Sale{}, err
	}

	held, err := s.units.ListInTheBook(ctx, request.BusinessID)
	if err != nil {
		return recipe.Sale{}, err
	}

	sale, err := recipe.SaleOf(request.BusinessID, productions, held, recipe.SaleDefinition{
		RecipeID:      recipe.ID(request.RecipeID),
		RecipeName:    request.RecipeName,
		Quantity:      request.Quantity,
		Unit:          request.Unit,
		Money:         ledger.Amount(request.Money),
		Day:           day,
		Utterance:     request.Utterance,
		ConfirmedWith: request.ConfirmedWith,
	})
	if err != nil {
		return recipe.Sale{}, err
	}

	movements, err := s.movementsOf(ctx, sale, request.ReceivingAccountID)
	if err != nil {
		return recipe.Sale{}, err
	}

	var recorded recipe.Sale

	err = s.work.Do(ctx, func(ctx context.Context) error {
		written, err := s.sales.Record(ctx, sale)
		if err != nil {
			return err
		}

		recorded = written

		for _, one := range movements {
			if _, err := s.book.Record(ctx, one); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return recipe.Sale{}, err
	}

	return recorded, nil
}

// movementsOf answers the two movements of the money of the sale.
//
// The money of the sale lands in the account that receives it and it comes
// from the account of the sales. The cost of the units that left the book
// leaves the inventory and lands in the account of the cost of the sales.
//
// The answer holds the movement of the cost only when the sale released
// one: there is no money to move when the cost is nothing, and the book
// never writes a movement of nothing.
//
// The error is account.ErrNotFound when the book of the business holds no
// cash, no account of the sales, no account of the cost of the sales or no
// inventory, and ledger.ErrInvalidAmount when the two accounts of a
// movement are the same one.
func (s *SaleService) movementsOf(
	ctx context.Context,
	sale recipe.Sale,
	receiving account.ID,
) ([]ledger.Entry, error) {
	accounts, err := s.accounts.List(ctx, sale.BusinessID)
	if err != nil {
		return nil, err
	}

	into, err := recipe.TheReceivingAccount(accounts, receiving)
	if err != nil {
		return nil, err
	}

	sales, err := account.OfRole(accounts, account.RoleSales)
	if err != nil {
		return nil, fmt.Errorf("%w: the business has no account for the sales", err)
	}

	costOfSales, err := account.OfRole(accounts, account.RoleCostOfSales)
	if err != nil {
		return nil, fmt.Errorf("%w: the business has no account for the cost of the sales", err)
	}

	inventory, err := purchase.TheInventoryAccount(accounts)
	if err != nil {
		return nil, err
	}

	money, err := sale.MoneyMovement(into.ID, sales.ID)
	if err != nil {
		return nil, err
	}

	movements := []ledger.Entry{money}

	if sale.Cost <= 0 {
		return movements, nil
	}

	cost, err := sale.CostMovement(costOfSales.ID, inventory.ID)
	if err != nil {
		return nil, err
	}

	return append(movements, cost), nil
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of today
// when the agent names none.
func (s *SaleService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *SaleService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
