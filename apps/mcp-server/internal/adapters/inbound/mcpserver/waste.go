package mcpserver

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the waste.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records a waste
// is not a tool that reads what expires.
const (
	// ToolRecordWaste writes the waste of the bakery that the owner
	// confirmed.
	ToolRecordWaste = "record_waste"
)

// The description of the tools of the waste.
//
// The vocabulary of the waste belongs to the domain: the description reads
// it and never writes it a second time.
var (
	descriptionRecordWaste = buildDescriptionRecordWaste()
)

// buildDescriptionRecordWaste renders the description of the tool that
// records the waste.
//
// The waste is what the bakery lost, and the description says so in the
// words she reads: the product, the quantity, the unit and the reason.
func buildDescriptionRecordWaste() string {
	reasons := make([]string, 0, len(recipe.AllWasteReasons()))

	for _, one := range recipe.AllWasteReasons() {
		reasons = append(reasons, "'"+one.String()+"'")
	}

	return "Record the waste of the bakery: what she lost, and what it was worth.\n\n" +
		"A waste carries:\n\n" +
		"  - the PRODUCT she lost: send the identity of it, or the name the owner reads ('Pan de molde');\n" +
		"  - the QUANTITY she lost ('3');\n" +
		"  - the UNIT of that quantity ('unidad');\n" +
		"  - the REASON she lost them: " + strings.Join(reasons, ", ") + ";\n" +
		"  - the DAY it happened.\n\n" +
		"The waste of the bakery counts the UNITS it lost: the unit is the piece ('unidad'), and PACA never " +
		"converts a weight into a number of units. When the owner names a weight, ask her how many units she " +
		"lost.\n\n" +
		"The REASON is what a later analysis reads, and PACA never invents one: ask the owner why she lost them. " +
		"The reason is one of the four of the book: the one that was burnt, the one that spoiled, the one that " +
		"fell and the one that came back.\n\n" +
		"The money of the waste is NOT typed by the agent: PACA derives it from the book. The units that leave " +
		"the book are the units of the product that EXPIRE FIRST, and the money of the waste is the money of " +
		"those units - the cost the waste releases. The money is never lost from the book: it leaves the " +
		"inventory and it lands in the account of the waste.\n\n" +
		"The business is REQUIRED: it is the business the waste belongs to, and PACA never invents one.\n\n" +
		"The product is REQUIRED: it is the product she lost. The name of a product is not unique inside the " +
		"business, so send the identity of the product when the name carries more than one: PACA refuses to guess " +
		"which product she lost.\n\n" +
		"The quantity is REQUIRED: it is the number of units she lost, and PACA never fills it by herself.\n\n" +
		"The day is OPTIONAL: the day the waste happened, written the way a person writes it ('2026-03-14'). It " +
		"is the day of today when it is left out.\n\n" +
		"The utterance is OPTIONAL: the sentence of the owner, verbatim. Send it as she said it when she said " +
		"one, and it is kept with the record.\n\n" +
		"confirmed_with is REQUIRED: the word she used to confirm it ('sí', 'confirmado'), and it is kept with " +
		"the record.\n\n" +
		"The answer carries the waste that was written: the product, the quantity, the unit, the reason, the day, " +
		"the money of what it was worth, and the units that left the book."
}

// recordWasteInput is the input of the tool that records the waste.
type recordWasteInput struct {
	// BusinessID is the business the waste belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the waste belongs to. It is required."`

	// RecipeID is the identity of the product she lost.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the product she lost. It is optional: send the recipe_name instead when the name carries only one product."`

	// RecipeName is the name the owner reads of the product she lost.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name the owner reads of the product she lost ('Pan de molde'). It is optional: send the recipe_id instead when the name carries more than one product."`

	// Quantity is how many units of the product the bakery lost.
	Quantity float64 `json:"quantity" jsonschema:"How many units of the product the bakery lost (3). It is required, and PACA never fills it by herself."`

	// Unit is the unit of that quantity.
	Unit string `json:"unit,omitempty" jsonschema:"The unit of the quantity ('unidad'). It is optional: the waste counts the units it lost when it is left out."`

	// Reason is why the bakery lost them.
	Reason string `json:"reason" jsonschema:"Why the bakery lost them: 'burnt', 'spoiled', 'dropped' or 'returned'. It is required, and PACA never invents one."`

	// Day is the day the waste happened.
	Day string `json:"day,omitempty" jsonschema:"The day the waste happened, written as '2026-03-14'. It is optional: it means the day of today when it is empty."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty" jsonschema:"The sentence of the owner, verbatim. It is optional: send it as she said it when she said one, and it is kept with the record."`

	// ConfirmedWith is the word the owner used to confirm the waste.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`
}

// recordWasteUnitOutput is what the answer carries about one unit that left
// the book.
type recordWasteUnitOutput struct {
	// UnitID is the identity of the unit.
	UnitID int64 `json:"unit_id"`

	// ProductionID is the production the unit came out of.
	ProductionID int64 `json:"production_id"`

	// Position is the order the unit was made in.
	Position int `json:"position"`

	// Grams is the weight of the unit.
	Grams float64 `json:"grams"`

	// ExpiresOn is the day the unit expired on.
	ExpiresOn string `json:"expires_on"`
}

// recordWasteOutput is the answer of the tool that records the waste.
//
// The answer carries the waste that was written: the product, the quantity,
// the unit, the reason, the money of what it was worth and the units that
// left the book.
type recordWasteOutput struct {
	// ID is the identity of the waste.
	ID int64 `json:"id"`

	// BusinessID is the business the waste belongs to.
	BusinessID int64 `json:"business_id"`

	// RecipeID is the product the waste is of.
	RecipeID int64 `json:"recipe_id"`

	// RecipeName is the name the owner reads of the product.
	RecipeName string `json:"recipe_name"`

	// Day is the day the waste happened.
	Day string `json:"day"`

	// Quantity is how many units of the product the bakery lost.
	Quantity float64 `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`

	// Reason is why the bakery lost them.
	Reason string `json:"reason"`

	// Money is the money of what was lost, in whole pesos.
	Money int64 `json:"money"`

	// Units are the units that left the book.
	Units []recordWasteUnitOutput `json:"units"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty"`

	// ConfirmedWith is the word the owner used to confirm the waste.
	ConfirmedWith string `json:"confirmed_with"`
}

// registerWasteTools registers the tools of the waste.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerWasteTools(recorder ports.WasteRecorder) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordWaste,
		Description: descriptionRecordWaste,
	}, s.recordWaste(recorder))
}

// recordWaste is the handler of the tool that records the waste.
//
// The tool writes the waste the owner confirmed and answers with it.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the waste of nobody.
func (s *Server) recordWaste(recorder ports.WasteRecorder) sdk.ToolHandlerFor[recordWasteInput, recordWasteOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordWasteInput) (*sdk.CallToolResult, recordWasteOutput, error) {
		if recorder == nil {
			return nil, recordWasteOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, recordWasteOutput{}, s.toolError(recipe.ErrMissingBusiness)
		}

		if input.Quantity <= 0 {
			return nil, recordWasteOutput{}, s.toolError(recipe.ErrInvalidWasteQuantity)
		}

		if strings.TrimSpace(input.Reason) == "" {
			return nil, recordWasteOutput{}, s.toolError(recipe.ErrUnknownWasteReason)
		}

		if strings.TrimSpace(input.ConfirmedWith) == "" {
			return nil, recordWasteOutput{}, s.toolError(recipe.ErrMissingWasteConfirmation)
		}

		recorded, err := recorder.Record(ctx, ports.RecordWaste{
			BusinessID:    business.ID(input.BusinessID),
			RecipeID:      input.RecipeID,
			RecipeName:    input.RecipeName,
			Quantity:      input.Quantity,
			Unit:          input.Unit,
			Reason:        input.Reason,
			Day:           input.Day,
			Utterance:     input.Utterance,
			ConfirmedWith: input.ConfirmedWith,
		})
		if err != nil {
			return nil, recordWasteOutput{}, s.toolError(err)
		}

		return nil, outputOfWaste(recorded), nil
	}
}

// outputOfWaste maps the waste of the domain onto the answer of the tool.
//
// The units that left the book travel with the answer: the waste is of
// those units, and the answer says which ones they were.
func outputOfWaste(recorded recipe.Waste) recordWasteOutput {
	answer := recordWasteOutput{
		ID:            int64(recorded.ID),
		BusinessID:    int64(recorded.BusinessID),
		RecipeID:      int64(recorded.RecipeID),
		RecipeName:    recorded.RecipeName,
		Day:           recorded.Day.String(),
		Quantity:      recorded.Quantity,
		Unit:          recorded.Unit,
		Reason:        recorded.Reason.String(),
		Money:         int64(recorded.Money),
		Units:         make([]recordWasteUnitOutput, 0, len(recorded.Units)),
		Utterance:     recorded.Utterance,
		ConfirmedWith: recorded.ConfirmedWith,
	}

	for _, one := range recorded.Units {
		answer.Units = append(answer.Units, recordWasteUnitOutput{
			UnitID:       int64(one.UnitID),
			ProductionID: int64(one.ProductionID),
			Position:     one.Position,
			Grams:        one.Grams,
			ExpiresOn:    one.ExpiresOn.String(),
		})
	}

	return answer
}
