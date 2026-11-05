package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// The names of the tools of the book.
const (
	// ToolRecordMovement writes a movement of money in the book.
	ToolRecordMovement = "record_movement"

	// ToolReadMovements reads the movements of a period: the libro diario.
	ToolReadMovements = "read_movements"

	// ToolReadBalances reads the balance of every account: the libro mayor.
	ToolReadBalances = "read_balances"
)

// The descriptions of the tools of the book.
//
// The list of the kinds of an account is read from the domain: it belongs to
// the domain, and the description never writes it a second time.
var (
	descriptionRecordMovement = buildDescriptionRecordMovement()
	descriptionReadMovements  = "Read the movements of the book of a business between two days: the libro diario.\n\n" +
		"The business is REQUIRED: it is the business the money belongs to. The days are the two ends of the period, " +
		"written the way a person writes them ('2026-03-01'), and the movement of the first day and the movement of " +
		"the last one are both in the answer. Leave a day empty and it means the day of today.\n\n" +
		"The movements come in the order of the day and of the moment they were written, and each one carries the " +
		"money that moved, the accounts it went from and to, and what it was for.\n\n" +
		"This tool changes nothing: use " + ToolRecordMovement + " to write a movement."

	descriptionReadBalances = "Read the balance of every account of the book of a business: the libro mayor.\n\n" +
		"The business is REQUIRED: it is the business the money belongs to. The day is the day the balance is read " +
		"as of, written the way a person writes it ('2026-03-01'), and it means the day of today when it is empty.\n\n" +
		"The answer carries the balance of every account of the book - the account that never moved is in the " +
		"answer too - and the money of the business as a whole: what it has, what it owes, what it earned, what it " +
		"spent, and what it is worth.\n\n" +
		"This tool changes nothing: use " + ToolRecordMovement + " to write a movement."
)

// buildDescriptionRecordMovement renders the description of the tool that
// writes a movement.
//
// The list of the kinds of an account is read from the domain: it belongs to
// the domain, and the description never writes it a second time.
func buildDescriptionRecordMovement() string {
	return "Write a movement of money in the book of a business.\n\n" +
		"The money goes FROM one account TO another one: the source account is where the money comes from, and the " +
		"destination account is where it lands. A salary lands in the cash and comes from the salary; a lunch lands " +
		"in the food and comes from the cash; a loan lands in the cash and comes from the loan of the friend.\n\n" +
		"The business is REQUIRED: it is the business the money belongs to. The accounts are the identities the " +
		"tool of the onboarding answered with.\n\n" +
		"The amount is REQUIRED and it is positive: it is the money that moves, in the currency of the book (" +
		"pesos). It is never negative, and it is never a fraction: the book counts the pesos the person has.\n\n" +
		"The date is OPTIONAL: it is the day the money moved, written the way a person writes it ('2026-03-01'), and " +
		"it means the day of today when it is empty.\n\n" +
		"The description is REQUIRED: it is what the person reads in their book. Write what the money was for, in " +
		"their words ('Lunch with the team', 'Salary of March').\n\n" +
		"The accounts of a book are of these kinds: " + theKindsOfTheAccounts() + ". An account of the kind " +
		"'liability' is what the person owes, and the money that lands in it is money they owe."
}

// recordMovementInput is the input of the tool that writes a movement.
type recordMovementInput struct {
	// BusinessID is the business the money belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the money belongs to. It is required."`

	// FromAccountID is the account the money comes from.
	FromAccountID int64 `json:"from_account_id" jsonschema:"The identity of the account the money comes from. It is required."`

	// ToAccountID is the account the money lands in.
	ToAccountID int64 `json:"to_account_id" jsonschema:"The identity of the account the money lands in. It is required."`

	// Amount is the money that moves.
	Amount int64 `json:"amount" jsonschema:"The money that moves, in pesos. It is required and it is positive."`

	// Date is the day the money moved.
	Date string `json:"date,omitempty" jsonschema:"The day the money moved, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`

	// Description is what the person reads in their book.
	Description string `json:"description" jsonschema:"What the money was for, in the words of the person ('Lunch with the team'). It is required."`
}

// movementOutput is one movement of the book.
type movementOutput struct {
	// ID is the identity of the movement.
	ID int64 `json:"id"`

	// Date is the day of the movement.
	Date string `json:"date"`

	// Amount is the money that moved.
	Amount int64 `json:"amount"`

	// FromAccountID is the account the money came from.
	FromAccountID int64 `json:"from_account_id"`

	// ToAccountID is the account the money landed in.
	ToAccountID int64 `json:"to_account_id"`

	// Description is what the person reads.
	Description string `json:"description"`
}

// recordMovementOutput is the answer of the tool that writes a movement.
type recordMovementOutput struct {
	// Movement is the movement that was written.
	Movement movementOutput `json:"movement"`
}

// readMovementsOutput is the answer of the tool that reads the movements.
type readMovementsOutput struct {
	// BusinessID is the business the movements belong to.
	BusinessID int64 `json:"business_id"`

	// FromDate is the first day of the period.
	FromDate string `json:"from_date"`

	// ToDate is the last day of the period.
	ToDate string `json:"to_date"`

	// Movements are the movements of the period.
	Movements []movementOutput `json:"movements"`
}

// balanceOutput is the balance of one account of the book.
type balanceOutput struct {
	// AccountID is the account the balance belongs to.
	AccountID int64 `json:"account_id"`

	// Kind is the kind of the account.
	Kind string `json:"kind"`

	// Name is the name the person reads in their book.
	Name string `json:"name"`

	// Balance is the balance of the account, in the sense of the person:
	// positive means the person has it, earns it or owes it.
	Balance int64 `json:"balance"`
}

// readBalancesOutput is the answer of the tool that reads the balances.
type readBalancesOutput struct {
	// BusinessID is the business the book belongs to.
	BusinessID int64 `json:"business_id"`

	// AsOf is the day the balance was read as of.
	AsOf string `json:"as_of"`

	// Accounts are the balances of the accounts of the book.
	Accounts []balanceOutput `json:"accounts"`

	// Assets is what the business has.
	Assets int64 `json:"assets"`

	// Liabilities is what the business owes.
	Liabilities int64 `json:"liabilities"`

	// Income is what the business earned.
	Income int64 `json:"income"`

	// Expenses is what the business spent.
	Expenses int64 `json:"expenses"`

	// NetWorth is what the business is worth.
	NetWorth int64 `json:"net_worth"`
}

// registerBookTools registers the tools of the book.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerBookTools(book ports.BookQuerier) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordMovement,
		Description: descriptionRecordMovement,
	}, s.recordMovement(book))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadMovements,
		Description: descriptionReadMovements,
	}, s.readMovements(book))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadBalances,
		Description: descriptionReadBalances,
	}, s.readBalances(book))
}

// recordMovement is the handler of the tool that writes a movement.
func (s *Server) recordMovement(book ports.BookQuerier) sdk.ToolHandlerFor[recordMovementInput, recordMovementOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordMovementInput) (*sdk.CallToolResult, recordMovementOutput, error) {
		if book == nil {
			return nil, recordMovementOutput{}, ports.ErrNoDatabase
		}

		written, err := book.Record(
			ctx,
			businessIDOf(input.BusinessID),
			accountIDOf(input.FromAccountID),
			accountIDOf(input.ToAccountID),
			ledger.Amount(input.Amount),
			input.Date,
			input.Description,
		)
		if err != nil {
			return nil, recordMovementOutput{}, s.toolError(err)
		}

		return nil, recordMovementOutput{Movement: outputOfMovement(written)}, nil
	}
}

// readMovements is the handler of the tool that reads the movements.
func (s *Server) readMovements(book ports.BookQuerier) sdk.ToolHandlerFor[readMovementsInput, readMovementsOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readMovementsInput) (*sdk.CallToolResult, readMovementsOutput, error) {
		if book == nil {
			return nil, readMovementsOutput{}, ports.ErrNoDatabase
		}

		movements, err := book.Movements(ctx, businessIDOf(input.BusinessID), input.FromDate, input.ToDate)
		if err != nil {
			return nil, readMovementsOutput{}, s.toolError(err)
		}

		return nil, readMovementsOutput{
			BusinessID: input.BusinessID,
			FromDate:   input.FromDate,
			ToDate:     input.ToDate,
			Movements:  outputsOfMovements(movements),
		}, nil
	}
}

// readBalances is the handler of the tool that reads the balances.
func (s *Server) readBalances(book ports.BookQuerier) sdk.ToolHandlerFor[readBalancesInput, readBalancesOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readBalancesInput) (*sdk.CallToolResult, readBalancesOutput, error) {
		if book == nil {
			return nil, readBalancesOutput{}, ports.ErrNoDatabase
		}

		book0, balances, err := book.Balances(ctx, businessIDOf(input.BusinessID), input.AsOf)
		if err != nil {
			return nil, readBalancesOutput{}, s.toolError(err)
		}

		accounts := make([]balanceOutput, 0, len(balances))
		for _, one := range balances {
			accounts = append(accounts, balanceOutput{
				AccountID: int64(one.AccountID),
				Kind:      one.Kind.String(),
				Name:      one.Name,
				Balance:   int64(one.Normal()),
			})
		}

		return nil, readBalancesOutput{
			BusinessID:  input.BusinessID,
			AsOf:        input.AsOf,
			Accounts:    accounts,
			Assets:      int64(book0.Assets),
			Liabilities: int64(book0.Liabilities),
			Income:      int64(book0.Income),
			Expenses:    int64(book0.Expenses),
			NetWorth:    int64(book0.NetWorth()),
		}, nil
	}
}

// readMovementsInput is the input of the tool that reads the movements.
type readMovementsInput struct {
	// BusinessID is the business the money belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the money belongs to. It is required."`

	// FromDate is the first day of the period.
	FromDate string `json:"from_date,omitempty" jsonschema:"The first day of the period, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`

	// ToDate is the last day of the period.
	ToDate string `json:"to_date,omitempty" jsonschema:"The last day of the period, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`
}

// readBalancesInput is the input of the tool that reads the balances.
type readBalancesInput struct {
	// BusinessID is the business the book belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the money belongs to. It is required."`

	// AsOf is the day the balance is read as of.
	AsOf string `json:"as_of,omitempty" jsonschema:"The day the balance is read as of, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`
}

// outputOfMovement maps a movement of the book onto the answer of the tool.
func outputOfMovement(entry ledger.Entry) movementOutput {
	return movementOutput{
		ID:            int64(entry.ID),
		Date:          entry.OccurredOn.String(),
		Amount:        int64(entry.Amount),
		FromAccountID: int64(entry.CreditAccountID),
		ToAccountID:   int64(entry.DebitAccountID),
		Description:   entry.Description,
	}
}

// outputsOfMovements maps the movements of the book onto the answer of the
// tool.
func outputsOfMovements(entries []ledger.Entry) []movementOutput {
	movements := make([]movementOutput, 0, len(entries))
	for _, entry := range entries {
		movements = append(movements, outputOfMovement(entry))
	}

	return movements
}

// businessIDOf returns the identity of the business the tool carries.
func businessIDOf(id int64) business.ID {
	return business.ID(id)
}

// accountIDOf returns the identity of the account the tool carries.
func accountIDOf(id int64) account.ID {
	return account.ID(id)
}
