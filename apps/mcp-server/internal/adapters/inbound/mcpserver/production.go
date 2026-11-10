package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the production.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records a
// production is not a tool that reads the expiry of what it made.
const (
	// ToolRecordProduction writes the production of the bakery that the
	// owner confirmed.
	ToolRecordProduction = "record_production"
)

// The description of the tools of the production.
//
// The vocabulary of the production belongs to the domain: the description
// reads it and never writes it a second time.
var (
	descriptionRecordProduction = buildDescriptionRecordProduction()
)

// buildDescriptionRecordProduction renders the description of the tool that
// records the production.
//
// The production of the bakery is what came out of the oven on one day: the
// description says so, and it says that the book derives the grams and the
// units instead of asking the owner for them.
func buildDescriptionRecordProduction() string {
	return "Record the production of the bakery: what came out of the oven on one day.\n\n" +
		"A production carries the recipe it was made of, the day it was made, the grams that went in and the units that came out:\n\n" +
		"  - the RECIPE she made: send the identity of it, or the name the owner reads ('Pan de molde');\n" +
		"  - the DAY she made it, written the way a person writes it ('2026-03-14');\n" +
		"  - the GRAMS that went in and the UNITS that came out.\n\n" +
		"The grams and the units are NOT typed by the agent: PACA derives them from the book and never asks the owner for them. The grams that went in are the grams the layers of the stock gave, taken in the order of the FIFO - the oldest layer is the first one that leaves - and the units that came out are the yield of the recipe, each one with the day it expires: the day of the production plus the shelf life of the recipe.\n\n" +
		"The business is REQUIRED: it is the business the production belongs to, and PACA never invents one.\n\n" +
		"The recipe is REQUIRED: it is the recipe she made. The name of a recipe is not unique inside the business, so send the identity of the recipe when the name carries more than one: PACA refuses to guess which recipe she made.\n\n" +
		"The day is OPTIONAL: the day the production was made, written the way a person writes it ('2026-03-14'). It is the day of today when it is left out.\n\n" +
		"The utterance is OPTIONAL: the sentence of the owner, verbatim. Send it as she said it when she said one, and it is kept with the record.\n\n" +
		"confirmed_with is REQUIRED: the word she used to confirm it ('sí', 'confirmado'), and it is kept with the record.\n\n" +
		"PACA refuses a production the stock cannot cover: the stock does not hold what the recipe needs, and a production of what she made is never guessed. Record the purchase of the missing ingredient first.\n\n" +
		"The answer carries the production that was written: the recipe, the day, the grams that went in, the money of the layers the production consumed, the layers it consumed in the order they left the stock, and the units that came out with the day each one expires."
}

// recordProductionInput is the input of the tool that records a production.
type recordProductionInput struct {
	// BusinessID is the business the production belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the production belongs to. It is required."`

	// RecipeID is the identity of the recipe she made.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the recipe she made. It is optional: send the recipe_name instead when the name carries only one recipe."`

	// RecipeName is the name the owner reads of the recipe she made.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name the owner reads of the recipe she made ('Pan de molde'). It is optional: send the recipe_id instead when the name carries more than one recipe."`

	// Day is the day the production was made.
	Day string `json:"day,omitempty" jsonschema:"The day the production was made, written as '2026-03-14'. It is optional: it means the day of today when it is empty."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty" jsonschema:"The sentence of the owner, verbatim. It is optional: send it as she said it when she said one, and it is kept with the record."`

	// ConfirmedWith is the word the owner used to confirm the production.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`
}

// recordProductionLayerOutput is what the answer carries about one layer the
// production consumed.
type recordProductionLayerOutput struct {
	// PurchaseID is the purchase the layer comes from.
	PurchaseID int64 `json:"purchase_id"`

	// Day is the day the layer was bought.
	Day string `json:"day"`

	// Quantity is what the production took, in the unit of the layer.
	Quantity float64 `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`

	// Grams is what the production took, in grams.
	Grams float64 `json:"grams"`

	// Money is the money of what the production took, in whole pesos.
	Money int64 `json:"money"`
}

// recordProductionUnitOutput is what the answer carries about one unit that
// came out of the production.
type recordProductionUnitOutput struct {
	// Position is the order the unit was made in.
	Position int `json:"position"`

	// Grams is the weight of the unit.
	Grams float64 `json:"grams"`

	// ExpiresOn is the day the unit expires.
	ExpiresOn string `json:"expires_on"`
}

// recordProductionOutput is the answer of the tool that records a production.
//
// The answer carries the production that was written: the recipe, the day,
// the grams that went in, the money, the layers the production consumed and
// the units that came out.
type recordProductionOutput struct {
	// ID is the identity of the production.
	ID int64 `json:"id"`

	// BusinessID is the business the production belongs to.
	BusinessID int64 `json:"business_id"`

	// RecipeID is the recipe the production was made of.
	RecipeID int64 `json:"recipe_id"`

	// RecipeName is the name the owner reads of the recipe.
	RecipeName string `json:"recipe_name"`

	// Day is the day the production was made.
	Day string `json:"day"`

	// GramsIn is the grams that went into the production.
	GramsIn float64 `json:"grams_in"`

	// Money is the money of what the layers gave, in whole pesos.
	Money int64 `json:"money"`

	// Layers are the layers the production consumed, in the order they
	// left the stock: the oldest one is the first.
	Layers []recordProductionLayerOutput `json:"layers"`

	// Units are the units that came out of the production.
	Units []recordProductionUnitOutput `json:"units"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty"`

	// ConfirmedWith is the word the owner used to confirm the production.
	ConfirmedWith string `json:"confirmed_with"`
}

// registerProductionTools registers the tools of the production.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerProductionTools(recorder ports.ProductionRecorder) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordProduction,
		Description: descriptionRecordProduction,
	}, s.recordProduction(recorder))
}

// recordProduction is the handler of the tool that records a production.
//
// The tool writes the production the owner confirmed and answers with it.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the production of nobody.
func (s *Server) recordProduction(recorder ports.ProductionRecorder) sdk.ToolHandlerFor[recordProductionInput, recordProductionOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordProductionInput) (*sdk.CallToolResult, recordProductionOutput, error) {
		if recorder == nil {
			return nil, recordProductionOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, recordProductionOutput{}, s.toolError(recipe.ErrMissingBusiness)
		}

		recorded, err := recorder.Record(ctx, ports.RecordProduction{
			BusinessID:    business.ID(input.BusinessID),
			RecipeID:      input.RecipeID,
			RecipeName:    input.RecipeName,
			Day:           input.Day,
			Utterance:     input.Utterance,
			ConfirmedWith: input.ConfirmedWith,
		})
		if err != nil {
			return nil, recordProductionOutput{}, s.toolError(err)
		}

		return nil, outputOfProduction(recorded), nil
	}
}

// outputOfProduction maps the production of the domain onto the answer of
// the tool.
//
// The layers travel in the order they left the stock: the oldest one is the
// first, because the valuation is FIFO.
func outputOfProduction(recorded recipe.Production) recordProductionOutput {
	answer := recordProductionOutput{
		ID:            int64(recorded.ID),
		BusinessID:    int64(recorded.BusinessID),
		RecipeID:      int64(recorded.RecipeID),
		RecipeName:    recorded.RecipeName,
		Day:           recorded.Day.String(),
		GramsIn:       recorded.GramsIn,
		Money:         int64(recorded.Money),
		Layers:        make([]recordProductionLayerOutput, 0, len(recorded.Takes)),
		Units:         make([]recordProductionUnitOutput, 0, len(recorded.Units)),
		Utterance:     recorded.Utterance,
		ConfirmedWith: recorded.ConfirmedWith,
	}

	for _, one := range recorded.Takes {
		answer.Layers = append(answer.Layers, recordProductionLayerOutput{
			PurchaseID: int64(one.PurchaseID),
			Day:        one.Day.String(),
			Quantity:   one.Quantity,
			Unit:       one.Unit,
			Grams:      one.Grams,
			Money:      int64(one.Money),
		})
	}

	for _, one := range recorded.Units {
		answer.Units = append(answer.Units, recordProductionUnitOutput{
			Position:  one.Position,
			Grams:     one.Grams,
			ExpiresOn: one.ExpiresOn.String(),
		})
	}

	return answer
}
