package mcpserver

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the real price.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that reads the
// report of the real price is not a tool that records a price.
const (
	// ToolReadRealPrice reads the report of the real price of a product.
	ToolReadRealPrice = "read_real_price"
)

// The description of the tools of the real price.
//
// The vocabulary of the ways of charging belongs to the domain: the
// description reads it and never writes it a second time.
var (
	descriptionReadRealPrice = buildDescriptionReadRealPrice()
)

// buildDescriptionReadRealPrice renders the description of the tool that
// reads the report of the real price.
//
// The three layers of the cost travel apart, and the price of the owner
// travels beside them: the description says so in the words the agent reads.
func buildDescriptionReadRealPrice() string {
	return "Read the report of the real price of a product: what the three layers of the cost of a batch are - " +
		"the DIRECT cost, the LABOUR and the OVERHEAD - and the price the owner charges, side by side.\n\n" +
		"The three layers travel APART: the direct cost is the money of the ingredients the batch consumes, the " +
		"labour is the money of the hours of the steps of the recipe, and the overhead is the money of the hours " +
		"of the oven. Their sum travels apart too, beside them. No layer of the cost is ever hidden inside " +
		"another one.\n\n" +
		"Read the two monies apart: the money of the price of the owner is the money in her basis - the price " +
		"of one unit, of one gram or of one kilogram, the way she charges it - and the money of the layers is " +
		"the money of the whole batch: the batch is made of many units, and the rates of each layer tell what " +
		"that money weighs per unit, per gram and per kilogram.\n\n" +
		"Every layer answers per unit, per gram and per kilogram, because the bakery sells by the unit and by " +
		"the weight, and a comparison between two products that knows only one of them is a lie.\n\n" +
		"The business is REQUIRED: it is the business whose report is read, and PACA never invents one.\n\n" +
		"The recipe is OPTIONAL: send the identity the tool " + ToolRecordRecipe + " wrote for the recipe, or " +
		"the name the owner reads ('Pan de molde'), and leave both out to read the report of every recipe of the " +
		"business.\n\n" +
		"The price is OPTIONAL: it is the money the owner charges for the product, in whole pesos ('" +
		business.DefaultCurrency + "'), and the money of what she sells is never inferred. When she has not " +
		"declared it, the answer says that the price is not known, and the three layers of the cost still " +
		"answer.\n\n" +
		"The price_basis is REQUIRED when the price is sent: it is the way she charges, and it is one of " +
		priceBasesOfTheBook() + ". The way she sells is hers, and the book never moves her price: the money " +
		"she declared is the money of the report.\n\n" +
		"The answer carries, for every recipe:\n\n" +
		"  - the name the owner reads and the yield of the recipe: the units it makes and the weight of each one;\n" +
		"  - the DIRECT cost, the LABOUR and the OVERHEAD, each one with its money and its rates per unit, per " +
		"gram and per kilogram;\n" +
		"  - the TOTAL of the three layers, shown apart;\n" +
		"  - the PRICE of the owner, with the way she charges and the same three rates;\n" +
		"  - what the stock does not hold, and what the report still needs.\n\n" +
		"This tool changes nothing: it reads the report of the real price and it writes nothing."
}

// priceBasesOfTheBook renders the ways the owner may charge, in the words
// the agent reads.
//
// The vocabulary belongs to the domain: the description of the tool reads
// it, so the agent sends the way she charges the way the book knows it.
func priceBasesOfTheBook() string {
	names := make([]string, 0, len(purchase.PriceBases()))

	for _, one := range purchase.PriceBases() {
		names = append(names, "'"+one.String()+"'")
	}

	return strings.Join(names, ", ")
}

// readRealPriceInput is the input of the tool that reads the report of the
// real price.
type readRealPriceInput struct {
	// BusinessID is the business whose report is read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose report is read. It is required."`

	// RecipeID is the recipe the caller asks about.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the recipe whose real price is read. It is optional: leave it out to read the report of every recipe of the business."`

	// RecipeName is the name the owner reads of the recipe.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name the owner reads of the recipe ('Pan de molde'). It is optional: leave it out to read the report of every recipe of the business."`

	// Price is the money the owner charges for the product.
	Price int64 `json:"price,omitempty" jsonschema:"The money the owner charges for the product, in whole pesos. It is optional: PACA never infers it, and the answer says that the price is not known when the owner declared none."`

	// PriceBasis is the way the owner charges.
	PriceBasis string `json:"price_basis,omitempty" jsonschema:"The way the owner charges: 'unit', 'gram' or 'kilogram'. It is required when the price is sent, and it is optional when the owner declared no price."`
}

// realPriceRateOutput is what the answer carries about one money of the
// report and the rates it weighs against.
//
// The money is whole pesos, and the rates are not money in the book: they
// are what the money weighs per unit, per gram and per kilogram.
type realPriceRateOutput struct {
	// Money is the money of the layer, in whole pesos.
	Money int64 `json:"money"`

	// PerUnit is the money per unit of what the recipe makes.
	PerUnit float64 `json:"per_unit"`

	// PerGram is the money per gram of what the recipe makes.
	PerGram float64 `json:"per_gram"`

	// PerKilogram is the money per kilogram of what the recipe makes.
	PerKilogram float64 `json:"per_kilogram"`

	// Unknown says that the money is not derivable: the owner declared
	// none of it, and the money of the book is never inferred.
	Unknown bool `json:"unknown"`

	// Reason says why the money is not derivable, in the words of the
	// book.
	Reason string `json:"reason,omitempty"`
}

// realPriceOutput is what the answer carries about one product.
//
// The three layers of the cost travel apart, and the price of the owner
// travels beside them.
type realPriceOutput struct {
	// RecipeID is the recipe the report is of.
	RecipeID int64 `json:"recipe_id"`

	// Name is the name the owner reads of the recipe.
	Name string `json:"name"`

	// YieldUnits is the number of units the recipe makes.
	YieldUnits int `json:"yield_units"`

	// YieldGrams is the weight of one unit, in grams.
	YieldGrams float64 `json:"yield_grams"`

	// Direct is the direct cost of the batch.
	Direct realPriceRateOutput `json:"direct"`

	// Labor is the labor of the batch, beside the direct cost and never
	// inside it.
	Labor realPriceRateOutput `json:"labor"`

	// Overhead is the overhead of the batch, beside the direct cost and
	// beside the labor and never inside either one.
	Overhead realPriceRateOutput `json:"overhead"`

	// Total is the sum of the three layers, shown apart.
	Total realPriceRateOutput `json:"total"`

	// Price is the price of the owner: what she charges, as she declared
	// it.
	Price realPriceRateOutput `json:"price"`

	// PriceBasis is the way she charges.
	PriceBasis string `json:"price_basis,omitempty"`

	// Complete says that the stock holds everything the recipe needs.
	Complete bool `json:"complete"`

	// Missing is what the report still needs.
	Missing []string `json:"missing"`
}

// readRealPriceOutput is the answer of the tool that reads the report of
// the real price.
type readRealPriceOutput struct {
	// Reports are the reports of the real price, one for every recipe.
	Reports []realPriceOutput `json:"reports"`
}

// registerRealPriceTools registers the tools of the real price.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerRealPriceTools(reader ports.RealPriceReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadRealPrice,
		Description: descriptionReadRealPrice,
	}, s.readRealPrice(reader))
}

// readRealPrice is the handler of the tool that reads the report of the
// real price.
//
// The tool reads the report and answers with the three layers of the cost
// apart and the price of the owner beside them: it writes nothing.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the report of nobody.
func (s *Server) readRealPrice(reader ports.RealPriceReader) sdk.ToolHandlerFor[readRealPriceInput, readRealPriceOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readRealPriceInput) (*sdk.CallToolResult, readRealPriceOutput, error) {
		if reader == nil {
			return nil, readRealPriceOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, readRealPriceOutput{}, s.toolError(purchase.ErrMissingBusiness)
		}

		read, err := reader.Read(ctx, ports.ReadRealPrice{
			BusinessID: business.ID(input.BusinessID),
			RecipeID:   input.RecipeID,
			RecipeName: input.RecipeName,
			Price:      input.Price,
			PriceBasis: input.PriceBasis,
		})
		if err != nil {
			return nil, readRealPriceOutput{}, s.toolError(err)
		}

		answer := readRealPriceOutput{
			Reports: make([]realPriceOutput, 0, len(read)),
		}

		for _, one := range read {
			answer.Reports = append(answer.Reports, outputOfRealPrice(one))
		}

		return nil, answer, nil
	}
}

// outputOfRealPrice maps the report of the domain onto the answer of the
// tool.
//
// The three layers of the cost travel apart, and the price of the owner
// travels beside them: the answer never folds one into another.
func outputOfRealPrice(report recipe.RealPrice) realPriceOutput {
	answer := realPriceOutput{
		RecipeID:   int64(report.RecipeID),
		Name:       report.Name,
		YieldUnits: report.Yield.Units,
		YieldGrams: report.Yield.Grams,
		Direct:     outputOfRate(report.Direct),
		Labor:      outputOfRate(report.Labor),
		Overhead:   outputOfRate(report.Overhead),
		Total:      outputOfRate(report.Total),
		Price:      outputOfRate(report.Price),
		PriceBasis: report.PriceBasis.String(),
		Complete:   report.Complete,
		Missing:    report.Missing,
	}

	if answer.Missing == nil {
		answer.Missing = []string{}
	}

	return answer
}

// outputOfRate maps the money of the domain and its rates onto the answer
// of the tool.
func outputOfRate(rate recipe.Rate) realPriceRateOutput {
	return realPriceRateOutput{
		Money:       int64(rate.Money),
		PerUnit:     rate.PerUnit,
		PerGram:     rate.PerGram,
		PerKilogram: rate.PerKilogram,
		Unknown:     rate.Unknown,
		Reason:      rate.Reason,
	}
}
