package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The names of the tools of the stock.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that reads the
// stock is not a tool that writes a layer.
const (
	// ToolReadStock reads what the business holds and what it cost.
	ToolReadStock = "read_stock"
)

// The description of the tools of the stock.
//
// The valuation of the stock is FIFO, and the description reads it: the
// vocabulary of the answer is a rule, not a promise of the transport.
var (
	descriptionReadStock = buildDescriptionReadStock()
)

// buildDescriptionReadStock renders the description of the tool that reads
// the stock of a business.
//
// The valuation is FIFO, and the description says so in the words the
// agent reads: the oldest layer is the first one that leaves, and the
// order of the output by expiry is a different rule that does not live
// here.
func buildDescriptionReadStock() string {
	return "Read the stock of the business: what it holds and what it cost.\n\n" +
		"The business is REQUIRED: it is the business whose stock is read, and PACA never invents one.\n\n" +
		"The ingredient is OPTIONAL: send the identity the tool " + ToolMatchItems + " answered for the name to " +
		"read the stock of that ingredient alone, and leave it out to read the whole stock of the business.\n\n" +
		"The answer carries, for every ingredient the business holds:\n\n" +
		"  - the name the owner reads, the quantity it holds and the unit of that quantity;\n" +
		"  - the layers that are still open, the OLDEST one first: the valuation is FIFO, and the oldest layer " +
		"is the first one that leaves;\n" +
		"  - the money the stock cost, in whole pesos ('" + business.DefaultCurrency + "').\n\n" +
		"The stock of an ingredient is the sum of the layers that are still open, and the layers bought in " +
		"different units are held apart: the quantity and its unit travel together.\n\n" +
		"This tool changes nothing: it reads the stock of the business and it writes nothing."
}

// readStockInput is the input of the tool that reads the stock of a
// business.
type readStockInput struct {
	// BusinessID is the business whose stock is read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose stock is read. It is required."`

	// IngredientID is the ingredient the caller asks about.
	IngredientID int64 `json:"ingredient_id,omitempty" jsonschema:"The identity of the ingredient whose stock is read. It is optional: leave it out to read the whole stock of the business."`
}

// layerOutput is what the answer carries about one layer of the stock.
type layerOutput struct {
	// PurchaseID is the purchase the layer comes from: the purchase is
	// the fact, and the layer is derived from it.
	PurchaseID int64 `json:"purchase_id"`

	// Day is the day the layer was bought.
	Day string `json:"day"`

	// Quantity is what the layer bought.
	Quantity float64 `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`

	// Remaining is what is left of the layer.
	Remaining float64 `json:"remaining"`

	// Cost is the money the layer cost, in whole pesos.
	Cost int64 `json:"cost"`
}

// holdingOutput is what the answer carries about one ingredient the
// business holds.
type holdingOutput struct {
	// IngredientID is the ingredient of the catalog the holding is of.
	IngredientID int64 `json:"ingredient_id"`

	// Name is the name the owner reads of the ingredient.
	Name string `json:"name"`

	// Unit is the unit of the quantity of the holding.
	Unit string `json:"unit"`

	// Quantity is how much of the ingredient the business holds.
	Quantity float64 `json:"quantity"`

	// Cost is the money of the holding, in whole pesos.
	Cost int64 `json:"cost"`

	// Layers are the layers that are still open, the oldest one first.
	Layers []layerOutput `json:"layers"`
}

// readStockOutput is the answer of the tool that reads the stock of a
// business.
type readStockOutput struct {
	// Holdings are the holdings of the business: what it holds of every
	// ingredient, in the order the layers leave the stock.
	Holdings []holdingOutput `json:"holdings"`

	// Cost is the money of the whole stock, in whole pesos.
	Cost int64 `json:"cost"`
}

// registerStockTools registers the tools of the stock.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerStockTools(reader ports.StockReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadStock,
		Description: descriptionReadStock,
	}, s.readStock(reader))
}

// readStock is the handler of the tool that reads the stock of a business.
//
// The tool reads the stock and answers with what the business holds and
// what it cost: it writes nothing.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the stock of nobody.
func (s *Server) readStock(reader ports.StockReader) sdk.ToolHandlerFor[readStockInput, readStockOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readStockInput) (*sdk.CallToolResult, readStockOutput, error) {
		if reader == nil {
			return nil, readStockOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, readStockOutput{}, s.toolError(purchase.ErrMissingBusiness)
		}

		read, err := reader.Read(ctx, ports.ReadStock{
			BusinessID:   business.ID(input.BusinessID),
			IngredientID: input.IngredientID,
		})
		if err != nil {
			return nil, readStockOutput{}, s.toolError(err)
		}

		answer := readStockOutput{
			Holdings: make([]holdingOutput, 0, len(read.Holdings)),
			Cost:     int64(read.Cost),
		}

		for _, one := range read.Holdings {
			answer.Holdings = append(answer.Holdings, outputOfHolding(one))
		}

		return nil, answer, nil
	}
}

// outputOfHolding maps the holding of the domain onto the answer of the
// tool.
//
// The layers of the holding travel in the order they leave the stock: the
// oldest one is the first, because the valuation is FIFO.
func outputOfHolding(holding purchase.Holding) holdingOutput {
	answer := holdingOutput{
		IngredientID: int64(holding.IngredientID),
		Name:         holding.Name,
		Unit:         holding.Unit,
		Quantity:     holding.Quantity,
		Cost:         int64(holding.Cost),
		Layers:       make([]layerOutput, 0, len(holding.Layers)),
	}

	for _, one := range holding.Layers {
		answer.Layers = append(answer.Layers, layerOutput{
			PurchaseID: int64(one.PurchaseID),
			Day:        one.Day.String(),
			Quantity:   one.Bought.Quantity,
			Unit:       one.Bought.Unit,
			Remaining:  one.Left.Quantity,
			Cost:       int64(one.Cost),
		})
	}

	return answer
}
