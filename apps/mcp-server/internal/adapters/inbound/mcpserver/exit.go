package mcpserver

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the exit.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records an
// exit is not a tool that reads what expires.
const (
	// ToolRecordExit writes the exit of a unit of the book: the
	// consumption of the owner or the donation.
	ToolRecordExit = "record_exit"
)

// The description of the tools of the exit.
//
// The vocabulary of the exit belongs to the domain: the description reads
// it and never writes it a second time.
var (
	descriptionRecordExit = buildDescriptionRecordExit()
)

// buildDescriptionRecordExit renders the description of the tool that
// records the exit.
//
// The exit is what left the book without money in it: the bread the owner
// consumed and the bread the bakery gave away. The description says so in
// the words she reads: the kind of the exit, the product, the quantity and
// the unit.
func buildDescriptionRecordExit() string {
	kinds := make([]string, 0, len(recipe.AllExitKinds()))

	for _, one := range recipe.AllExitKinds() {
		kinds = append(kinds, "'"+one.String()+"'")
	}

	return "Record the exit of a unit of the book: what the owner CONSUMED or what the bakery DONATED.\n\n" +
		"An exit carries:\n\n" +
		"  - the KIND of the exit: " + strings.Join(kinds, ", ") + ";\n" +
		"  - the PRODUCT it took: send the identity of it, or the name the owner reads ('Pan de molde');\n" +
		"  - the QUANTITY it took ('3');\n" +
		"  - the UNIT of that quantity ('unidad');\n" +
		"  - the DAY it happened.\n\n" +
		"The KIND is REQUIRED: the exits of the bakery are the consumption of the owner and the donation, " +
		"and PACA never guesses one of them. Use 'consumption' when the owner ate the bread, and 'donation' " +
		"when the bakery gave it away.\n\n" +
		"The exit moves NO money: nobody paid for the bread the owner consumed or for the bread the bakery " +
		"gave away. What the exit does is RELEASE the cost of the units that left the book: the cost leaves " +
		"the inventory and it lands in the account of the expense of the exit - the cost of the sales for the " +
		"consumption, and the account of the donation for the donation, which is a social expense of the " +
		"business.\n\n" +
		"The COST of the exit is NOT typed by the agent: PACA derives it from the book. The units that leave " +
		"the book are the units of the product that EXPIRE FIRST, and the cost of the exit is the cost of those " +
		"units - the money of the production they came out of, divided by the units that production made.\n\n" +
		"The TAX TREATMENT of the exit is NOT decided by PACA: it waits for the professional of the taxes of " +
		"the owner, and the answer carries the question that waits. Never tell the owner that the book knows " +
		"the tax of a consumption or of a donation.\n\n" +
		"The business is REQUIRED: it is the business the exit belongs to, and PACA never invents one.\n\n" +
		"The product is REQUIRED: it is the product she consumed or gave away. The name of a product is not " +
		"unique inside the business, so send the identity of the product when the name carries more than one: " +
		"PACA refuses to guess which product she meant.\n\n" +
		"The quantity is REQUIRED: it is the number of units the exit took, and PACA never fills it by herself.\n\n" +
		"The day is OPTIONAL: the day the exit happened, written the way a person writes it ('2026-03-14'). It " +
		"is the day of today when it is left out.\n\n" +
		"The utterance is OPTIONAL: the sentence of the owner, verbatim. Send it as she said it when she said " +
		"one, and it is kept with the record.\n\n" +
		"confirmed_with is REQUIRED: the word she used to confirm it ('sí', 'confirmado'), and it is kept with " +
		"the record.\n\n" +
		"The answer carries the exit that was written: the kind, the product, the quantity, the unit, the cost " +
		"the exit released, the tax treatment it waits for, the question of the taxes, and the units that left " +
		"the book."
}

// recordExitInput is the input of the tool that records the exit.
type recordExitInput struct {
	// BusinessID is the business the exit belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the exit belongs to. It is required."`

	// RecipeID is the identity of the product she consumed or gave away.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the product she consumed or gave away. It is optional: send the recipe_name instead when the name carries only one product."`

	// RecipeName is the name the owner reads of the product.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name the owner reads of the product she consumed or gave away ('Pan de molde'). It is optional: send the recipe_id instead when the name carries more than one product."`

	// Kind is the kind of the exit.
	Kind string `json:"kind" jsonschema:"The kind of the exit: 'consumption' (the owner ate the bread) or 'donation' (the bakery gave it away). It is required, and PACA never guesses one."`

	// Quantity is how many units of the product the exit took.
	Quantity float64 `json:"quantity" jsonschema:"How many units of the product the exit took (3). It is required, and PACA never fills it by herself."`

	// Unit is the unit of that quantity.
	Unit string `json:"unit,omitempty" jsonschema:"The unit of the quantity ('unidad'). It is optional: the exit counts the units it took when it is left out."`

	// Day is the day the exit happened.
	Day string `json:"day,omitempty" jsonschema:"The day the exit happened, written as '2026-03-14'. It is optional: it means the day of today when it is empty."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty" jsonschema:"The sentence of the owner, verbatim. It is optional: send it as she said it when she said one, and it is kept with the record."`

	// ConfirmedWith is the word the owner used to confirm the exit.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`
}

// recordExitUnitOutput is what the answer carries about one unit that left
// the book.
type recordExitUnitOutput struct {
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

// recordExitOutput is the answer of the tool that records the exit.
//
// The answer carries the exit that was written: the kind, the product, the
// quantity, the unit, the cost it released, the tax treatment it waits for
// and the units that left the book.
type recordExitOutput struct {
	// ID is the identity of the exit.
	ID int64 `json:"id"`

	// BusinessID is the business the exit belongs to.
	BusinessID int64 `json:"business_id"`

	// Kind is the kind of the exit.
	Kind string `json:"kind"`

	// RecipeID is the product the exit is of.
	RecipeID int64 `json:"recipe_id"`

	// RecipeName is the name the owner reads of the product.
	RecipeName string `json:"recipe_name"`

	// Day is the day the exit happened.
	Day string `json:"day"`

	// Quantity is how many units of the product the exit took.
	Quantity float64 `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`

	// Cost is the cost the exit released, in whole pesos.
	Cost int64 `json:"cost"`

	// TaxTreatment is the tax treatment the exit waits for.
	TaxTreatment string `json:"tax_treatment"`

	// TaxQuestion is the question of the taxes the exit waits for.
	TaxQuestion string `json:"tax_question"`

	// Units are the units that left the book.
	Units []recordExitUnitOutput `json:"units"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty"`

	// ConfirmedWith is the word the owner used to confirm the exit.
	ConfirmedWith string `json:"confirmed_with"`
}

// registerExitTools registers the tools of the exit.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerExitTools(recorder ports.ExitRecorder) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordExit,
		Description: descriptionRecordExit,
	}, s.recordExit(recorder))
}

// recordExit is the handler of the tool that records the exit.
//
// The tool writes the exit the owner confirmed and answers with it.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the exit of nobody.
func (s *Server) recordExit(recorder ports.ExitRecorder) sdk.ToolHandlerFor[recordExitInput, recordExitOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordExitInput) (*sdk.CallToolResult, recordExitOutput, error) {
		if recorder == nil {
			return nil, recordExitOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, recordExitOutput{}, s.toolError(recipe.ErrMissingBusiness)
		}

		if strings.TrimSpace(input.Kind) == "" {
			return nil, recordExitOutput{}, s.toolError(recipe.ErrUnknownExitKind)
		}

		if input.Quantity <= 0 {
			return nil, recordExitOutput{}, s.toolError(recipe.ErrInvalidExitQuantity)
		}

		if strings.TrimSpace(input.ConfirmedWith) == "" {
			return nil, recordExitOutput{}, s.toolError(recipe.ErrMissingExitConfirmation)
		}

		recorded, err := recorder.Record(ctx, ports.RecordExit{
			BusinessID:    business.ID(input.BusinessID),
			RecipeID:      input.RecipeID,
			RecipeName:    input.RecipeName,
			Kind:          input.Kind,
			Quantity:      input.Quantity,
			Unit:          input.Unit,
			Day:           input.Day,
			Utterance:     input.Utterance,
			ConfirmedWith: input.ConfirmedWith,
		})
		if err != nil {
			return nil, recordExitOutput{}, s.toolError(err)
		}

		return nil, outputOfExit(recorded), nil
	}
}

// outputOfExit maps the exit of the domain onto the answer of the tool.
//
// The units that left the book travel with the answer: the exit is of
// those units, and the answer says which ones they were. The tax
// treatment and the question of the taxes travel with it too: the question
// waits for the professional of the taxes of the owner, and the answer is
// what tells the agent about it.
func outputOfExit(recorded recipe.Exit) recordExitOutput {
	answer := recordExitOutput{
		ID:            int64(recorded.ID),
		BusinessID:    int64(recorded.BusinessID),
		Kind:          recorded.Kind.String(),
		RecipeID:      int64(recorded.RecipeID),
		RecipeName:    recorded.RecipeName,
		Day:           recorded.Day.String(),
		Quantity:      recorded.Quantity,
		Unit:          recorded.Unit,
		Cost:          int64(recorded.Cost),
		TaxTreatment:  recorded.TaxTreatment.String(),
		TaxQuestion:   recorded.TaxQuestion,
		Units:         make([]recordExitUnitOutput, 0, len(recorded.Units)),
		Utterance:     recorded.Utterance,
		ConfirmedWith: recorded.ConfirmedWith,
	}

	for _, one := range recorded.Units {
		answer.Units = append(answer.Units, recordExitUnitOutput{
			UnitID:       int64(one.UnitID),
			ProductionID: int64(one.ProductionID),
			Position:     one.Position,
			Grams:        one.Grams,
			ExpiresOn:    one.ExpiresOn.String(),
		})
	}

	return answer
}
