package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The names of the tools of the invoices.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records an
// invoice is not a tool that creates a row.
const (
	// ToolRecordInvoice records the invoice that came with the purchase:
	// the number it carries, its day, its total and the tax that travels
	// with it.
	ToolRecordInvoice = "record_invoice"

	// ToolReadInvoices reads the invoices of the business.
	ToolReadInvoices = "read_invoices"
)

// The descriptions of the tools of the invoices.
var (
	descriptionRecordInvoice = buildDescriptionRecordInvoice()
	descriptionReadInvoices  = buildDescriptionReadInvoices()
)

// buildDescriptionRecordInvoice renders the description of the tool that
// records an invoice.
//
// The description says what the tool needs and what it does with what it
// receives, and it says what it must not do: the tax of an invoice is
// never assumed, and the electronic invoice is never assumed either.
func buildDescriptionRecordInvoice() string {
	return "Record the invoice that came with a purchase: the provider that issued it, its number, its day, " +
		"its total and the tax that travels with it.\n\n" +
		"The business is REQUIRED: it is the business that bought, and the one that holds the invoice. The " +
		"Personal Business of a person holds the money of that person, and not the book of the bakery.\n\n" +
		"The provider ('provider_id') is REQUIRED: the provider of an invoice is a PARTY of the business — the " +
		"farm, the store or the person the bakery buys from — and it holds the role of the provider. A party that " +
		"holds only the role of the client is refused as a provider, and a party may hold both roles. Send the " +
		"identity the tool of the parties answered, and never the name of the provider.\n\n" +
		"The purchase ('purchase_id') is REQUIRED: the invoice is written against the purchase it came from, and " +
		"the purchase points at the invoice. An invoice of one business is never attached to the purchase of " +
		"another one.\n\n" +
		"The number ('number') is REQUIRED: it is the number the invoice carries, and it is what identifies it " +
		"inside the business. The number of an invoice is unique inside the business that holds it, and PACA " +
		"refuses a number the business already holds.\n\n" +
		"The base ('base') is REQUIRED: it is the money the tax applies to, in whole pesos.\n\n" +
		"The rate ('rate') is REQUIRED: it is the rate the invoice states, in percent ('19', '19.5', '0'). The " +
		"tax of an invoice is NEVER assumed: there is no default of nineteen percent and no default of zero, and " +
		"the rate is never taken from the catalogue of the country. The tax is DERIVED from the rate the invoice " +
		"states and the base the invoice states, and it is rounded to the nearest whole peso, with a half peso " +
		"going up.\n\n" +
		"The total ('total') is REQUIRED: it is the money of the whole invoice, in whole pesos, and it is the base " +
		"plus the tax. An invoice that does not add up is refused: PACA never fixes a total, and it never invents " +
		"the tax of an invoice.\n\n" +
		"The day ('day') is OPTIONAL: it is the day the invoice was issued, and it is the day of today when the " +
		"agent names none.\n\n" +
		"The reference ('reference') is OPTIONAL: it is the reference of the document the owner shows, and it is a " +
		"plain fact of the invoice. PACA never assumes an electronic invoice: there is no integration with the " +
		"tax authority, and the reference is never the mark of one.\n\n" +
		"The invoice moves NO money: the purchase already moved the money of the payment when the owner bought, " +
		"and the invoice is a fact that arrives later.\n\n" +
		"The answer carries the invoice as it was written: its identity, its provider, its purchase, its number, " +
		"its day, its base, its rate, its tax and its total."
}

// buildDescriptionReadInvoices renders the description of the tool that
// reads the invoices of a business.
func buildDescriptionReadInvoices() string {
	return "Read the invoices of the business: every invoice the business holds, with the tax that travels with " +
		"each one of them.\n\n" +
		"The business is REQUIRED: it is the business whose invoices are read, and PACA never invents one.\n\n" +
		"The answer carries every invoice of the business, in the order of their identity, with:\n\n" +
		"  - the provider that issued it, and the purchase it came from;\n" +
		"  - the number it carries, and the day it was issued;\n" +
		"  - the base the tax applies to, the rate the invoice states, the tax that travels with it and the " +
		"total of the whole invoice.\n\n" +
		"The tax of an invoice is never assumed: it is the one the invoice states.\n\n" +
		"It writes nothing."
}

// recordInvoiceInput is the input of the tool that records an invoice.
type recordInvoiceInput struct {
	// BusinessID is the business that bought and that holds the invoice.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business that bought and that holds the invoice: the business of its own, the one with the providers of the bakery. It is required, and PACA never invents one."`

	// ProviderID is the party that issued the invoice.
	ProviderID int64 `json:"provider_id" jsonschema:"The identity of the party that issued the invoice: the farm, the store or the person the bakery buys from. It holds the role of the provider, and a party that holds only the role of the client is refused. It is required."`

	// PurchaseID is the purchase the invoice came from.
	PurchaseID int64 `json:"purchase_id" jsonschema:"The identity of the purchase the invoice came from. It is required: the invoice is written against that purchase, and the purchase points at the invoice."`

	// Number is the number the invoice carries.
	Number string `json:"number" jsonschema:"The number the invoice carries ('FV-001'). It is required, and it is unique inside the business."`

	// Day is the day the invoice was issued.
	Day string `json:"day,omitempty" jsonschema:"The day the invoice was issued ('2026-03-02'). It is optional, and it is the day of today when the agent names none."`

	// Base is the money the tax applies to.
	Base int64 `json:"base" jsonschema:"The money the tax applies to, in whole pesos. It is required."`

	// Rate is the rate the invoice was issued with.
	Rate float64 `json:"rate" jsonschema:"The rate the invoice states, in percent ('19', '19.5', '0'). It is required, and PACA never assumes one: there is no default of nineteen percent and no default of zero."`

	// Total is the money of the whole invoice.
	Total int64 `json:"total" jsonschema:"The money of the whole invoice, in whole pesos: the base plus the tax. It is required, and an invoice that does not add up is refused."`

	// Reference is the reference of the document the owner shows.
	Reference string `json:"reference,omitempty" jsonschema:"The reference of the document the owner shows ('CUFE-1234'). It is optional, and it is never the mark of an electronic invoice: PACA never assumes one."`
}

// readInvoicesInput is the input of the tool that reads the invoices of a
// business.
type readInvoicesInput struct {
	// BusinessID is the business whose invoices are read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose invoices are read. It is required, and PACA never invents one."`
}

// invoiceOutput is what the tools of the invoices answer about one invoice.
type invoiceOutput struct {
	// ID is the identity of the invoice.
	ID int64 `json:"id"`

	// BusinessID is the business that holds the invoice.
	BusinessID int64 `json:"business_id"`

	// ProviderID is the party that issued the invoice.
	ProviderID int64 `json:"provider_id"`

	// PurchaseID is the purchase the invoice came from.
	PurchaseID int64 `json:"purchase_id"`

	// Number is the number the invoice carries.
	Number string `json:"number"`

	// Day is the day the invoice was issued.
	Day string `json:"day"`

	// Base is the money the tax applies to.
	Base int64 `json:"base"`

	// Rate is the rate the invoice states, the way a person reads it.
	Rate string `json:"rate"`

	// Tax is the tax that travels with the invoice.
	Tax int64 `json:"tax"`

	// Total is the money of the whole invoice.
	Total int64 `json:"total"`

	// Reference is the reference of the document the owner shows.
	Reference string `json:"reference,omitempty"`
}

// recordInvoiceOutput is the answer of the tool that records an invoice.
type recordInvoiceOutput struct {
	// Invoice is the invoice that was recorded.
	Invoice invoiceOutput `json:"invoice"`
}

// readInvoicesOutput is the answer of the tool that reads the invoices of a
// business.
type readInvoicesOutput struct {
	// Invoices are the invoices of the business.
	Invoices []invoiceOutput `json:"invoices"`
}

// registerInvoiceTools registers the tools of the invoices.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerInvoiceTools(recorder ports.InvoiceRecorder, reader ports.InvoicesReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordInvoice,
		Description: descriptionRecordInvoice,
	}, s.recordInvoice(recorder))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadInvoices,
		Description: descriptionReadInvoices,
	}, s.readInvoices(reader))
}

// recordInvoice is the handler of the tool that records an invoice.
//
// The tool writes the invoice against the purchase it came from and answers
// with it. It moves no money: the purchase already moved the money of the
// payment.
func (s *Server) recordInvoice(recorder ports.InvoiceRecorder) sdk.ToolHandlerFor[recordInvoiceInput, recordInvoiceOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordInvoiceInput) (*sdk.CallToolResult, recordInvoiceOutput, error) {
		if recorder == nil {
			return nil, recordInvoiceOutput{}, ports.ErrNoDatabase
		}

		recorded, err := recorder.Record(ctx, ports.RecordInvoice{
			BusinessID: businessIDOf(input.BusinessID),
			ProviderID: input.ProviderID,
			PurchaseID: input.PurchaseID,
			Number:     input.Number,
			Day:        input.Day,
			Base:       input.Base,
			Rate:       input.Rate,
			Total:      input.Total,
			Reference:  input.Reference,
		})
		if err != nil {
			return nil, recordInvoiceOutput{}, s.toolError(err)
		}

		return nil, recordInvoiceOutput{Invoice: outputOfInvoice(recorded)}, nil
	}
}

// readInvoices is the handler of the tool that reads the invoices of a
// business.
//
// The tool reads the invoices the business holds: it changes nothing, and it
// answers with every one of them.
func (s *Server) readInvoices(reader ports.InvoicesReader) sdk.ToolHandlerFor[readInvoicesInput, readInvoicesOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readInvoicesInput) (*sdk.CallToolResult, readInvoicesOutput, error) {
		if reader == nil {
			return nil, readInvoicesOutput{}, ports.ErrNoDatabase
		}

		held, err := reader.Read(ctx, ports.ReadInvoices{BusinessID: businessIDOf(input.BusinessID)})
		if err != nil {
			return nil, readInvoicesOutput{}, s.toolError(err)
		}

		invoices := make([]invoiceOutput, 0, len(held))
		for _, one := range held {
			invoices = append(invoices, outputOfInvoice(one))
		}

		return nil, readInvoicesOutput{Invoices: invoices}, nil
	}
}

// outputOfInvoice maps the model of the domain onto the answer of the tools
// of the invoices.
func outputOfInvoice(model purchase.Invoice) invoiceOutput {
	return invoiceOutput{
		ID:         int64(model.ID),
		BusinessID: int64(model.BusinessID),
		ProviderID: int64(model.ProviderID),
		PurchaseID: int64(model.PurchaseID),
		Number:     model.Number,
		Day:        model.Day.String(),
		Base:       int64(model.Base),
		Rate:       model.Rate.String(),
		Tax:        int64(model.Tax),
		Total:      int64(model.Total),
		Reference:  model.Reference,
	}
}
