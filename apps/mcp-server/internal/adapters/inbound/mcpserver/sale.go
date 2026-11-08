package mcpserver

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the sale.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records a sale
// is not a tool that reads what expires.
const (
	// ToolRecordSale writes the sale of the bakery that the owner
	// confirmed.
	ToolRecordSale = "record_sale"
)

// The description of the tools of the sale.
//
// The vocabulary of the sale belongs to the domain: the description reads
// it and never writes it a second time.
var (
	descriptionRecordSale = buildDescriptionRecordSale()
)

// buildDescriptionRecordSale renders the description of the tool that
// records the sale.
//
// The sale is what the bakery sold, and the description says so in the
// words she reads: the product, the quantity, the unit and the money the
// client paid.
func buildDescriptionRecordSale() string {
	return "Record the sale of the bakery: what she sold, and what it cost her.\n\n" +
		"A sale carries:\n\n" +
		"  - the PRODUCT she sold: send the identity of it, or the name the owner reads ('Pan de molde');\n" +
		"  - the QUANTITY she sold ('3');\n" +
		"  - the UNIT of that quantity ('unidad');\n" +
		"  - the MONEY the client paid, in whole pesos ('5000').\n\n" +
		"The sale of the bakery counts the UNITS it sold: the unit is the piece ('unidad'), and PACA never " +
		"converts a weight into a number of units.\n\n" +
		"The MONEY is what the client paid, and PACA never infers it: ask the owner what the client paid, and " +
		"never fill a price by yourself. The price list of the products is a later work, and the money of the " +
		"sale is what the owner says it is.\n\n" +
		"The COST of the sale is NOT typed by the agent: PACA derives it from the book. The units that leave " +
		"the book are the units of the product that EXPIRE FIRST, and the cost of the sale is the money of " +
		"those units - the cost the sale releases. The money of the cost leaves the inventory and it lands " +
		"in the account of the cost of the sales, and the money of the sale lands in the account that " +
		"receives it and comes from the account of the sales.\n\n" +
		"A unit that EXPIRED is never sold: PACA refuses the sale of it, and the expired unit has to be " +
		"written as a waste.\n\n" +
		"The business is REQUIRED: it is the business the sale belongs to, and PACA never invents one.\n\n" +
		"The product is REQUIRED: it is the product she sold. The name of a product is not unique inside the " +
		"business, so send the identity of the product when the name carries more than one: PACA refuses to " +
		"guess which product she sold.\n\n" +
		"The quantity is REQUIRED: it is the number of units she sold, and PACA never fills it by herself.\n\n" +
		"The money is REQUIRED: it is the money the client paid, and PACA never fills it by herself.\n\n" +
		"The account that receives the money is OPTIONAL: send 'to_account_id' when the client paid with " +
		"another account than the cash of the business, and the money lands in the cash when it is left out.\n\n" +
		"The day is OPTIONAL: the day of the sale, written the way a person writes it ('2026-03-14'). It is " +
		"the day of today when it is left out.\n\n" +
		"The utterance is OPTIONAL: the sentence of the owner, verbatim. Send it as she said it when she said " +
		"one, and it is kept with the record.\n\n" +
		"confirmed_with is REQUIRED: the word she used to confirm it ('sí', 'confirmado'), and it is kept with " +
		"the record.\n\n" +
		"The answer carries the sale that was written: the product, the quantity, the unit, the money the " +
		"client paid, the cost the sale released, and the units that left the book."
}

// recordSaleInput is the input of the tool that records the sale.
type recordSaleInput struct {
	// BusinessID is the business the sale belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the sale belongs to. It is required."`

	// RecipeID is the identity of the product she sold.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the product she sold. It is optional: send the recipe_name instead when the name carries only one product."`

	// RecipeName is the name the owner reads of the product she sold.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name the owner reads of the product she sold ('Pan de molde'). It is optional: send the recipe_id instead when the name carries more than one product."`

	// Quantity is how many units of the product the bakery sold.
	Quantity float64 `json:"quantity" jsonschema:"How many units of the product the bakery sold (3). It is required, and PACA never fills it by herself."`

	// Unit is the unit of that quantity.
	Unit string `json:"unit,omitempty" jsonschema:"The unit of the quantity ('unidad'). It is optional: the sale counts the units it sold when it is left out."`

	// Money is the money the client paid, in whole pesos.
	Money int64 `json:"money" jsonschema:"The money the client paid, in whole pesos (5000). It is required, and PACA never infers it."`

	// ToAccountID is the account the money of the sale lands in.
	ToAccountID int64 `json:"to_account_id,omitempty" jsonschema:"The identity of the account the money of the sale lands in. It is optional: the money lands in the cash of the business when it is left out."`

	// Day is the day the sale happened.
	Day string `json:"day,omitempty" jsonschema:"The day of the sale, written as '2026-03-14'. It is optional: it means the day of today when it is empty."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty" jsonschema:"The sentence of the owner, verbatim. It is optional: send it as she said it when she said one, and it is kept with the record."`

	// ConfirmedWith is the word the owner used to confirm the sale.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`
}

// recordSaleUnitOutput is what the answer carries about one unit that left
// the book.
type recordSaleUnitOutput struct {
	// UnitID is the identity of the unit.
	UnitID int64 `json:"unit_id"`

	// ProductionID is the production the unit came out of.
	ProductionID int64 `json:"production_id"`

	// Position is the order the unit was made in.
	Position int `json:"position"`

	// Grams is the weight of the unit.
	Grams float64 `json:"grams"`

	// ExpiresOn is the day the unit expires on.
	ExpiresOn string `json:"expires_on"`
}

// recordSaleOutput is the answer of the tool that records the sale.
//
// The answer carries the sale that was written: the product, the quantity,
// the unit, the money the client paid, the cost the sale released and the
// units that left the book.
type recordSaleOutput struct {
	// ID is the identity of the sale.
	ID int64 `json:"id"`

	// BusinessID is the business the sale belongs to.
	BusinessID int64 `json:"business_id"`

	// RecipeID is the product the sale is of.
	RecipeID int64 `json:"recipe_id"`

	// RecipeName is the name the owner reads of the product.
	RecipeName string `json:"recipe_name"`

	// Day is the day the sale happened.
	Day string `json:"day"`

	// Quantity is how many units of the product the bakery sold.
	Quantity float64 `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`

	// Money is the money the client paid, in whole pesos.
	Money int64 `json:"money"`

	// Cost is the cost the sale released, in whole pesos.
	Cost int64 `json:"cost"`

	// Units are the units that left the book.
	Units []recordSaleUnitOutput `json:"units"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty"`

	// ConfirmedWith is the word the owner used to confirm the sale.
	ConfirmedWith string `json:"confirmed_with"`
}

// registerSaleTools registers the tools of the sale.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerSaleTools(recorder ports.SaleRecorder) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordSale,
		Description: descriptionRecordSale,
	}, s.recordSale(recorder))
}

// recordSale is the handler of the tool that records the sale.
//
// The tool writes the sale the owner confirmed and answers with it.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the sale of nobody.
func (s *Server) recordSale(recorder ports.SaleRecorder) sdk.ToolHandlerFor[recordSaleInput, recordSaleOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordSaleInput) (*sdk.CallToolResult, recordSaleOutput, error) {
		if recorder == nil {
			return nil, recordSaleOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, recordSaleOutput{}, s.toolError(recipe.ErrMissingBusiness)
		}

		if input.Quantity <= 0 {
			return nil, recordSaleOutput{}, s.toolError(recipe.ErrInvalidSaleQuantity)
		}

		if input.Money <= 0 {
			return nil, recordSaleOutput{}, s.toolError(recipe.ErrInvalidSaleMoney)
		}

		if strings.TrimSpace(input.ConfirmedWith) == "" {
			return nil, recordSaleOutput{}, s.toolError(recipe.ErrMissingSaleConfirmation)
		}

		recorded, err := recorder.Record(ctx, ports.RecordSale{
			BusinessID:         business.ID(input.BusinessID),
			RecipeID:           input.RecipeID,
			RecipeName:         input.RecipeName,
			Quantity:           input.Quantity,
			Unit:               input.Unit,
			Money:              input.Money,
			ReceivingAccountID: account.ID(input.ToAccountID),
			Day:                input.Day,
			Utterance:          input.Utterance,
			ConfirmedWith:      input.ConfirmedWith,
		})
		if err != nil {
			return nil, recordSaleOutput{}, s.toolError(err)
		}

		return nil, outputOfSale(recorded), nil
	}
}

// outputOfSale maps the sale of the domain onto the answer of the tool.
//
// The units that left the book travel with the answer: the sale is of
// those units, and the answer says which ones they were.
func outputOfSale(recorded recipe.Sale) recordSaleOutput {
	answer := recordSaleOutput{
		ID:            int64(recorded.ID),
		BusinessID:    int64(recorded.BusinessID),
		RecipeID:      int64(recorded.RecipeID),
		RecipeName:    recorded.RecipeName,
		Day:           recorded.Day.String(),
		Quantity:      recorded.Quantity,
		Unit:          recorded.Unit,
		Money:         int64(recorded.Money),
		Cost:          int64(recorded.Cost),
		Units:         make([]recordSaleUnitOutput, 0, len(recorded.Units)),
		Utterance:     recorded.Utterance,
		ConfirmedWith: recorded.ConfirmedWith,
	}

	for _, one := range recorded.Units {
		answer.Units = append(answer.Units, recordSaleUnitOutput{
			UnitID:       int64(one.UnitID),
			ProductionID: int64(one.ProductionID),
			Position:     one.Position,
			Grams:        one.Grams,
			ExpiresOn:    one.ExpiresOn.String(),
		})
	}

	return answer
}
