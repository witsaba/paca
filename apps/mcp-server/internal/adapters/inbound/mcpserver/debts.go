package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// The names of the tools of the debts and of the closing of the month.
const (
	// ToolRecordDebt opens a debt and writes the money that opened it.
	ToolRecordDebt = "record_debt"

	// ToolListDebts reads the debts the book still holds.
	ToolListDebts = "list_debts"

	// ToolRecordDebtPayment writes the payment of a debt.
	ToolRecordDebtPayment = "record_debt_payment"

	// ToolReadMonthSummary reads the closing of a month.
	ToolReadMonthSummary = "read_month_summary"
)

// The descriptions of the tools of the debts and of the closing.
//
// The list of the kinds of an account is read from the domain: it belongs to
// the domain, and the description never writes it a second time.
var (
	descriptionRecordDebt = buildDescriptionRecordDebt()
	descriptionListDebts  = "Read the debts the book of a business still holds.\n\n" +
		"The business is REQUIRED: it is the business the money belongs to.\n\n" +
		"The answer carries every debt that is still open: what the person owes, to whom, " +
		"and the day it falls due. The debts come first by the day they fall due, and the debt " +
		"without a day comes last.\n\n" +
		"This tool changes nothing: use " + ToolRecordDebt + " to open a debt and " +
		ToolRecordDebtPayment + " to pay one."
	descriptionRecordDebtPayment = buildDescriptionRecordDebtPayment()
	descriptionReadMonthSummary  = buildDescriptionReadMonthSummary()
)

// buildDescriptionRecordDebt renders the description of the tool that opens
// a debt.
func buildDescriptionRecordDebt() string {
	return "Open a debt in the book of a business and write the money that opened it.\n\n" +
		"A debt is what the person owes: a loan of a friend, a product that will be paid later. " +
		"The money comes FROM the debt and lands in the account you name: the cash when a friend " +
		"lends the money, the expense when a product is paid later.\n\n" +
		"The business is REQUIRED: it is the business the money belongs to. The counterparty is " +
		"REQUIRED: it is the other side of the money, and a debt without it is a bucket of money " +
		"nobody waits for. The account is REQUIRED: it is the identity of the account where the " +
		"money lands.\n\n" +
		"The amount is REQUIRED and it is positive: it is the money of the debt, in pesos. It is " +
		"never negative, and it is never a fraction.\n\n" +
		"The date is OPTIONAL: it is the day the money moved, written the way a person writes it " +
		"('2026-03-01'), and it means the day of today when it is empty.\n\n" +
		"The due date is OPTIONAL: it is the day the debt falls due, written the way a person " +
		"writes it ('2026-04-30'). A debt without a due date is owed, and nobody waits for a day " +
		"that is not there.\n\n" +
		"The name is OPTIONAL: it is the name the person reads in their book for the account of " +
		"the debt, and it must be a name that is free inside the book. When it is empty, the name " +
		"is built from the counterparty.\n\n" +
		"The description is OPTIONAL: it is what the person reads in their book for the movement. " +
		"When it is empty, the description follows the name of the debt.\n\n" +
		"The accounts of a book are of these kinds: " + theKindsOfTheAccounts() + ". The debt is " +
		"an account of the kind 'liability': what the person owes."
}

// buildDescriptionRecordDebtPayment renders the description of the tool that
// pays a debt.
func buildDescriptionRecordDebtPayment() string {
	return "Write the payment of a debt in the book of a business.\n\n" +
		"The money leaves the account you name and lands in the debt: the payment is what the " +
		"person owes, and it is what the debt holds after it.\n\n" +
		"The business is REQUIRED: it is the business the money belongs to. The debt is REQUIRED: " +
		"it is the identity of the account that carries the debt, and " + ToolListDebts + " is " +
		"the tool that reads them. The account is REQUIRED: it is the identity of the account the " +
		"money leaves from, the cash or the bank.\n\n" +
		"The amount is REQUIRED and it is positive: it is the money that is paid, in pesos. It may " +
		"not pass what the debt still holds: a payment bigger than the debt is money the person " +
		"does not owe.\n\n" +
		"The date is OPTIONAL: it is the day the money moved, written the way a person writes it " +
		"('2026-03-01'), and it means the day of today when it is empty.\n\n" +
		"The description is REQUIRED: it is what the person reads in their book. Write what the " +
		"payment was for, in their words ('Payment to Ana').\n\n" +
		"For a debt that is not a payment, use " + ToolRecordMovement + "."
}

// buildDescriptionReadMonthSummary renders the description of the tool that
// closes a month.
func buildDescriptionReadMonthSummary() string {
	return "Read the closing of a month of the book of a business.\n\n" +
		"The closing is what the person earned, what they spent, and what they still owe: the " +
		"three numbers a person asks for when they say 'how did this month go?'.\n\n" +
		"The business is REQUIRED: it is the business the money belongs to. The year and the month " +
		"are OPTIONAL: they are the month to close, and they mean the month that just ended when " +
		"they are empty.\n\n" +
		"The answer carries the income of the month, the expenses of it, the net - what the person " +
		"earned minus what they spent - and the debts that were open at the end of the month.\n\n" +
		"This tool changes nothing: the closing of a month reads the book, it does not close it."
}

// recordDebtInput is the input of the tool that opens a debt.
type recordDebtInput struct {
	// BusinessID is the business the money belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the money belongs to. It is required."`

	// Counterparty is the other side of the money.
	Counterparty string `json:"counterparty" jsonschema:"The other side of the money: the friend who lends it, the supplier the person owes ('Ana'). It is required."`

	// Amount is the money of the debt.
	Amount int64 `json:"amount" jsonschema:"The money of the debt, in pesos. It is required and it is positive."`

	// ToAccountID is the account the money lands in.
	ToAccountID int64 `json:"to_account_id" jsonschema:"The identity of the account where the money lands: the cash when a friend lends the money, the expense when a product is paid later. It is required."`

	// Date is the day the money moved.
	Date string `json:"date,omitempty" jsonschema:"The day the money moved, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`

	// DueDate is the day the debt falls due.
	DueDate string `json:"due_date,omitempty" jsonschema:"The day the debt falls due, written as '2026-04-30'. It is optional: a debt without a day is owed anyway."`

	// Name is the name of the account of the debt.
	Name string `json:"name,omitempty" jsonschema:"The name the person reads in their book for the account of the debt. It is optional: it is built from the counterparty when it is empty, and it must be free inside the book."`

	// Description is what the person reads in their book.
	Description string `json:"description,omitempty" jsonschema:"What the money was for, in the words of the person. It is optional: it follows the name of the debt when it is empty."`
}

// listDebtsInput is the input of the tool that reads the debts.
type listDebtsInput struct {
	// BusinessID is the business the money belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the money belongs to. It is required."`
}

// recordDebtPaymentInput is the input of the tool that pays a debt.
type recordDebtPaymentInput struct {
	// BusinessID is the business the money belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the money belongs to. It is required."`

	// DebtAccountID is the account that carries the debt.
	DebtAccountID int64 `json:"debt_account_id" jsonschema:"The identity of the account that carries the debt. It is required."`

	// Amount is the money that is paid.
	Amount int64 `json:"amount" jsonschema:"The money that is paid, in pesos. It is required, it is positive, and it may not pass what the debt still holds."`

	// FromAccountID is the account the money leaves from.
	FromAccountID int64 `json:"from_account_id" jsonschema:"The identity of the account the money leaves from: the cash or the bank. It is required."`

	// Date is the day the money moved.
	Date string `json:"date,omitempty" jsonschema:"The day the money moved, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`

	// Description is what the person reads in their book.
	Description string `json:"description" jsonschema:"What the payment was for, in the words of the person ('Payment to Ana'). It is required."`
}

// readMonthSummaryInput is the input of the tool that closes a month.
type readMonthSummaryInput struct {
	// BusinessID is the business the money belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the money belongs to. It is required."`

	// Year is the year of the month to close.
	Year int `json:"year,omitempty" jsonschema:"The year of the month to close (2026). It is optional: it means the year of the month that just ended when it is empty."`

	// Month is the month to close.
	Month int `json:"month,omitempty" jsonschema:"The month to close, from 1 to 12. It is optional: it means the month that just ended when it is empty."`
}

// debtOutput is one debt of the book.
type debtOutput struct {
	// AccountID is the identity of the account that carries the debt.
	AccountID int64 `json:"account_id"`

	// Kind is the kind of the account that carries the debt.
	Kind string `json:"kind"`

	// Name is the name the person reads in their book.
	Name string `json:"name"`

	// Counterparty is the other side of the money.
	Counterparty string `json:"counterparty"`

	// DueDate is the day the debt falls due. It is empty when the debt has
	// no day.
	DueDate string `json:"due_date,omitempty"`

	// Amount is what the debt still holds, in the sense of the person:
	// positive means the person owes it, or that it is owed to them.
	Amount int64 `json:"amount"`
}

// recordDebtOutput is the answer of the tool that opens a debt.
type recordDebtOutput struct {
	// Debt is the debt that was opened.
	Debt debtOutput `json:"debt"`
}

// listDebtsOutput is the answer of the tool that reads the debts.
type listDebtsOutput struct {
	// BusinessID is the business the debts belong to.
	BusinessID int64 `json:"business_id"`

	// Debts are the debts the book still holds.
	Debts []debtOutput `json:"debts"`
}

// recordDebtPaymentOutput is the answer of the tool that pays a debt.
type recordDebtPaymentOutput struct {
	// Debt is the debt after the payment: what the person still owes.
	Debt debtOutput `json:"debt"`

	// Movement is the movement that paid the debt.
	Movement movementOutput `json:"movement"`
}

// readMonthSummaryOutput is the answer of the tool that closes a month.
type readMonthSummaryOutput struct {
	// BusinessID is the business the month belongs to.
	BusinessID int64 `json:"business_id"`

	// FromDate is the first day of the month.
	FromDate string `json:"from_date"`

	// ToDate is the last day of the month.
	ToDate string `json:"to_date"`

	// Income is what the person earned during the month.
	Income int64 `json:"income"`

	// Expenses is what the person spent during the month.
	Expenses int64 `json:"expenses"`

	// Net is what the person earned minus what they spent.
	Net int64 `json:"net"`

	// Debts are the debts that were open at the end of the month.
	Debts []debtOutput `json:"debts"`
}

// registerDebtTools registers the tools of the debts and of the closing.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerDebtTools(debts ports.DebtQuerier, summary ports.SummaryQuerier) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordDebt,
		Description: descriptionRecordDebt,
	}, s.recordDebt(debts))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolListDebts,
		Description: descriptionListDebts,
	}, s.listDebts(debts))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordDebtPayment,
		Description: descriptionRecordDebtPayment,
	}, s.recordDebtPayment(debts))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadMonthSummary,
		Description: descriptionReadMonthSummary,
	}, s.readMonthSummary(summary))
}

// recordDebt is the handler of the tool that opens a debt.
func (s *Server) recordDebt(debts ports.DebtQuerier) sdk.ToolHandlerFor[recordDebtInput, recordDebtOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordDebtInput) (*sdk.CallToolResult, recordDebtOutput, error) {
		if debts == nil {
			return nil, recordDebtOutput{}, ports.ErrNoDatabase
		}

		opened, err := debts.Record(
			ctx,
			businessIDOf(input.BusinessID),
			input.Counterparty,
			ledger.Amount(input.Amount),
			accountIDOf(input.ToAccountID),
			input.Date,
			input.Description,
			input.DueDate,
			input.Name,
		)
		if err != nil {
			return nil, recordDebtOutput{}, s.toolError(err)
		}

		return nil, recordDebtOutput{Debt: outputOfDebt(opened)}, nil
	}
}

// listDebts is the handler of the tool that reads the debts.
func (s *Server) listDebts(debts ports.DebtQuerier) sdk.ToolHandlerFor[listDebtsInput, listDebtsOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input listDebtsInput) (*sdk.CallToolResult, listDebtsOutput, error) {
		if debts == nil {
			return nil, listDebtsOutput{}, ports.ErrNoDatabase
		}

		found, err := debts.Debts(ctx, businessIDOf(input.BusinessID))
		if err != nil {
			return nil, listDebtsOutput{}, s.toolError(err)
		}

		return nil, listDebtsOutput{
			BusinessID: input.BusinessID,
			Debts:      outputsOfDebts(found),
		}, nil
	}
}

// recordDebtPayment is the handler of the tool that pays a debt.
func (s *Server) recordDebtPayment(debts ports.DebtQuerier) sdk.ToolHandlerFor[recordDebtPaymentInput, recordDebtPaymentOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordDebtPaymentInput) (*sdk.CallToolResult, recordDebtPaymentOutput, error) {
		if debts == nil {
			return nil, recordDebtPaymentOutput{}, ports.ErrNoDatabase
		}

		paid, err := debts.Pay(
			ctx,
			businessIDOf(input.BusinessID),
			accountIDOf(input.DebtAccountID),
			ledger.Amount(input.Amount),
			accountIDOf(input.FromAccountID),
			input.Date,
			input.Description,
		)
		if err != nil {
			return nil, recordDebtPaymentOutput{}, s.toolError(err)
		}

		return nil, recordDebtPaymentOutput{
			Debt:     outputOfDebt(paid.Debt),
			Movement: outputOfMovement(paid.Movement),
		}, nil
	}
}

// readMonthSummary is the handler of the tool that closes a month.
func (s *Server) readMonthSummary(summary ports.SummaryQuerier) sdk.ToolHandlerFor[readMonthSummaryInput, readMonthSummaryOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readMonthSummaryInput) (*sdk.CallToolResult, readMonthSummaryOutput, error) {
		if summary == nil {
			return nil, readMonthSummaryOutput{}, ports.ErrNoDatabase
		}

		closed, err := summary.Month(ctx, businessIDOf(input.BusinessID), input.Year, input.Month)
		if err != nil {
			return nil, readMonthSummaryOutput{}, s.toolError(err)
		}

		return nil, readMonthSummaryOutput{
			BusinessID: input.BusinessID,
			FromDate:   closed.From.String(),
			ToDate:     closed.To.String(),
			Income:     int64(closed.Summary.Income),
			Expenses:   int64(closed.Summary.Expenses),
			Net:        int64(closed.Summary.Net()),
			Debts:      outputsOfDebts(closed.Debts),
		}, nil
	}
}

// outputOfDebt maps a debt of the book onto the answer of the tool.
func outputOfDebt(debt ledger.Debt) debtOutput {
	answer := debtOutput{
		AccountID:    int64(debt.Account.ID),
		Kind:         debt.Account.Kind.String(),
		Name:         debt.Account.Name,
		Counterparty: debt.Account.Counterparty,
		Amount:       int64(debt.Amount),
	}

	if debt.Account.DueDate != nil {
		answer.DueDate = debt.Account.DueDate.Format("2006-01-02")
	}

	return answer
}

// outputsOfDebts maps the debts of the book onto the answer of the tool.
func outputsOfDebts(debts []ledger.Debt) []debtOutput {
	answers := make([]debtOutput, 0, len(debts))
	for _, debt := range debts {
		answers = append(answers, outputOfDebt(debt))
	}

	return answers
}
