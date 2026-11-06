package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the cost of a batch.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that reads the
// cost of a batch is not a tool that records a production.
const (
	// ToolReadBatchCost reads what the batch of a recipe costs right now.
	ToolReadBatchCost = "read_batch_cost"
)

// The description of the tools of the cost of a batch.
//
// The cost is derived from the layers of the stock, and the description
// reads it: the vocabulary of the answer is a rule, not a promise of the
// transport.
var (
	descriptionReadBatchCost = buildDescriptionReadBatchCost()
)

// buildDescriptionReadBatchCost renders the description of the tool that
// reads the cost of a batch.
//
// The cost is derived from the layers that the recipe consumes, and the
// valuation is FIFO: the description says so in the words the agent reads.
func buildDescriptionReadBatchCost() string {
	return "Read the cost of a batch of a recipe: what the ingredients of the recipe cost right now.\n\n" +
		"The cost of the batch is derived from the layers of the stock, and it is never typed: the recipe " +
		"consumes the layers in the order of the FIFO - the oldest layer is the first one that leaves - and the " +
		"money of what is taken is the money of the layer, pro rata the quantity that was taken.\n\n" +
		"The business is REQUIRED: it is the business whose cost is read, and PACA never invents one.\n\n" +
		"The recipe is OPTIONAL: send the identity the tool " + ToolRecordRecipe + " wrote for the recipe, or " +
		"the name the owner reads ('Pan de molde'), and leave both out to read the cost of every recipe of the " +
		"business.\n\n" +
		"The answer carries, for every recipe:\n\n" +
		"  - the name the owner reads and the yield of the recipe: the units it makes and the weight of each one;\n" +
		"  - the DIRECT cost of the batch, in whole pesos ('" + business.DefaultCurrency + "'): the money of the " +
		"ingredients it consumes, without the overhead;\n" +
		"  - the LABOR of the batch, beside the direct cost and never inside it: the hours of the steps of the " +
		"recipe in the three kinds of the labor - the hands, the oven and the waiting - and the money of them, " +
		"valued with the value of the hour the owner declared;\n" +
		"  - the cost per unit, per gram and per kilogram: the money of the book counts whole pesos, and these " +
		"three are rates;\n" +
		"  - for every ingredient of the recipe: the grams the recipe needs, the grams the stock gave to the " +
		"batch, the money of the batch, and the layers it took from, the oldest one first.\n\n" +
		"When the stock does not hold everything the recipe needs, the answer says what is missing, and the " +
		"cost is not complete: a cost that cannot be baked is not the cost of the batch.\n\n" +
		"This tool changes nothing: it reads the cost of the batch and it writes nothing."
}

// readBatchCostInput is the input of the tool that reads the cost of a
// batch.
type readBatchCostInput struct {
	// BusinessID is the business whose cost is read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose cost is read. It is required."`

	// RecipeID is the recipe the caller asks about.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the recipe whose cost is read. It is optional: leave it out to read the cost of every recipe of the business."`

	// RecipeName is the name the owner reads of the recipe.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name the owner reads of the recipe ('Pan de molde'). It is optional: leave it out to read the cost of every recipe of the business."`
}

// batchCostLayerOutput is what the answer carries about one layer the
// batch took from.
type batchCostLayerOutput struct {
	// PurchaseID is the purchase the layer comes from: the purchase is the
	// fact, and the layer is derived from it.
	PurchaseID int64 `json:"purchase_id"`

	// Day is the day the layer was bought.
	Day string `json:"day"`

	// Quantity is what the batch took, in the unit of the layer.
	Quantity float64 `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`

	// Grams is what the batch took, in grams.
	Grams float64 `json:"grams"`

	// Money is the money of what the batch took, in whole pesos.
	Money int64 `json:"money"`
}

// batchCostLineOutput is what the answer carries about one ingredient of
// the recipe.
type batchCostLineOutput struct {
	// IngredientID is the ingredient of the catalog the line is of.
	IngredientID int64 `json:"ingredient_id"`

	// Name is the name the owner reads of the ingredient.
	Name string `json:"name"`

	// Grams is how many grams of it the recipe needs.
	Grams float64 `json:"grams"`

	// Available is how many grams of it the stock gave to the batch.
	Available float64 `json:"available"`

	// Missing is how many grams of it the stock does not hold.
	Missing float64 `json:"missing"`

	// Money is the money of the grams of the batch, in whole pesos.
	Money int64 `json:"money"`

	// Unknown says that the money of the ingredient is not derivable: the
	// stock holds it in a unit the book never converts into grams.
	Unknown bool `json:"unknown"`

	// Layers are the layers the batch took from, the oldest one first.
	Layers []batchCostLayerOutput `json:"layers"`
}

// batchCostLaborOutput is what the answer carries about the labor of the
// batch.
//
// The labor is a layer of the cost of its own: it travels beside the direct
// cost, and it is never added to it.
type batchCostLaborOutput struct {
	// HandsOnHours is the number of hours the hands of the owner were on
	// the dough.
	HandsOnHours float64 `json:"hands_on_hours"`

	// OvenHours is the number of hours the batch was in the oven.
	OvenHours float64 `json:"oven_hours"`

	// WaitingHours is the number of hours the dough waited.
	WaitingHours float64 `json:"waiting_hours"`

	// TotalHours is the number of hours of the labor, in the three kinds
	// together.
	TotalHours float64 `json:"total_hours"`

	// HourValue is the value of the hour that valued the labor, in whole
	// pesos.
	HourValue int64 `json:"hour_value"`

	// Money is the money of the labor of the batch, in whole pesos.
	Money int64 `json:"money"`

	// Unknown says that the money of the labor is not derivable: the
	// owner has not declared the value of the hour.
	Unknown bool `json:"unknown"`

	// Reason says why the money of the labor is not derivable, in the
	// words of the book.
	Reason string `json:"reason,omitempty"`
}

// batchCostOutput is what the answer carries about one recipe.
type batchCostOutput struct {
	// RecipeID is the recipe the cost is of.
	RecipeID int64 `json:"recipe_id"`

	// Name is the name the owner reads of the recipe.
	Name string `json:"name"`

	// YieldUnits is the number of units the recipe makes.
	YieldUnits int `json:"yield_units"`

	// YieldGrams is the weight of one unit, in grams.
	YieldGrams float64 `json:"yield_grams"`

	// Direct is the money of the batch, in whole pesos.
	Direct int64 `json:"direct"`

	// Labor is the labor of the batch: the hours of the steps of the
	// recipe and the money of them.
	Labor batchCostLaborOutput `json:"labor"`

	// PerUnit is the money of the batch per unit it makes.
	PerUnit float64 `json:"per_unit"`

	// PerGram is the money of the batch per gram it makes.
	PerGram float64 `json:"per_gram"`

	// PerKilogram is the money of the batch per kilogram it makes.
	PerKilogram float64 `json:"per_kilogram"`

	// Complete says that the stock holds everything the recipe needs.
	Complete bool `json:"complete"`

	// Missing is what the cost still needs.
	Missing []string `json:"missing"`

	// Lines are the lines of the cost: one for every ingredient of the
	// recipe.
	Lines []batchCostLineOutput `json:"lines"`
}

// readBatchCostOutput is the answer of the tool that reads the cost of a
// batch.
type readBatchCostOutput struct {
	// Costs are the costs of the batches of the recipes.
	Costs []batchCostOutput `json:"costs"`
}

// registerBatchCostTools registers the tools of the cost of a batch.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerBatchCostTools(reader ports.BatchCostReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadBatchCost,
		Description: descriptionReadBatchCost,
	}, s.readBatchCost(reader))
}

// readBatchCost is the handler of the tool that reads the cost of a batch.
//
// The tool reads the cost and answers with what the recipe costs right
// now: it writes nothing.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the cost of nobody.
func (s *Server) readBatchCost(reader ports.BatchCostReader) sdk.ToolHandlerFor[readBatchCostInput, readBatchCostOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readBatchCostInput) (*sdk.CallToolResult, readBatchCostOutput, error) {
		if reader == nil {
			return nil, readBatchCostOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, readBatchCostOutput{}, s.toolError(purchase.ErrMissingBusiness)
		}

		read, err := reader.Read(ctx, ports.ReadBatchCost{
			BusinessID: business.ID(input.BusinessID),
			RecipeID:   input.RecipeID,
			RecipeName: input.RecipeName,
		})
		if err != nil {
			return nil, readBatchCostOutput{}, s.toolError(err)
		}

		answer := readBatchCostOutput{
			Costs: make([]batchCostOutput, 0, len(read)),
		}

		for _, one := range read {
			answer.Costs = append(answer.Costs, outputOfCost(one))
		}

		return nil, answer, nil
	}
}

// outputOfCost maps the cost of the domain onto the answer of the tool.
//
// The layers of every line travel in the order they leave the stock: the
// oldest one is the first, because the valuation is FIFO.
func outputOfCost(cost recipe.Cost) batchCostOutput {
	answer := batchCostOutput{
		RecipeID:    int64(cost.RecipeID),
		Name:        cost.Name,
		YieldUnits:  cost.Yield.Units,
		YieldGrams:  cost.Yield.Grams,
		Direct:      int64(cost.Direct),
		PerUnit:     cost.PerUnit,
		PerGram:     cost.PerGram,
		PerKilogram: cost.PerKilogram,
		Complete:    cost.Complete,
		Missing:     cost.Missing,
		Labor: batchCostLaborOutput{
			HandsOnHours: cost.Labor.Hours.HandsOn,
			OvenHours:    cost.Labor.Hours.Oven,
			WaitingHours: cost.Labor.Hours.Waiting,
			TotalHours:   cost.Labor.Hours.Total(),
			HourValue:    int64(cost.Labor.Value),
			Money:        int64(cost.Labor.Money),
			Unknown:      cost.Labor.Unknown,
			Reason:       cost.Labor.Reason,
		},
		Lines: make([]batchCostLineOutput, 0, len(cost.Lines)),
	}

	if answer.Missing == nil {
		answer.Missing = []string{}
	}

	for _, one := range cost.Lines {
		line := batchCostLineOutput{
			IngredientID: int64(one.IngredientID),
			Name:         one.Name,
			Grams:        one.Grams,
			Available:    one.Available,
			Missing:      one.Missing,
			Money:        int64(one.Money),
			Unknown:      one.Unknown,
			Layers:       make([]batchCostLayerOutput, 0, len(one.Layers)),
		}

		for _, layer := range one.Layers {
			line.Layers = append(line.Layers, batchCostLayerOutput{
				PurchaseID: int64(layer.PurchaseID),
				Day:        layer.Day.String(),
				Quantity:   layer.Quantity,
				Unit:       layer.Unit,
				Grams:      layer.Grams,
				Money:      int64(layer.Money),
			})
		}

		answer.Lines = append(answer.Lines, line)
	}

	return answer
}
