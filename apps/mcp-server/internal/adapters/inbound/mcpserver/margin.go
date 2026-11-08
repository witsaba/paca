package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the margin.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that reads the
// margin of a product is not a tool that sets the price of it.
const (
	// ToolReadMargin reads the margin of a product.
	ToolReadMargin = "read_margin"
)

// The description of the tools of the margin.
var (
	descriptionReadMargin = buildDescriptionReadMargin()
)

// buildDescriptionReadMargin renders the description of the tool that reads
// the margin of a product.
//
// The margin compares the price of the owner with the direct cost of the
// batch and with the reference of the market, and the description says so
// in the words the agent reads.
func buildDescriptionReadMargin() string {
	return "Read the margin of a product: what the owner charges above the DIRECT cost of the batch, and what " +
		"the MARKET said about the same ingredients, side by side.\n\n" +
		"The margin compares the price of the owner with the DIRECT cost of the batch - the money of the " +
		"ingredients it consumes - and with the REFERENCE of the market: the price PACA researched, with the " +
		"place it was read in and the day it was seen.\n\n" +
		"Read the two monies apart: the money of the price of the owner is the money in her basis - the price " +
		"of one unit, of one gram or of one kilogram, the way she charges it - and the money of the layers is " +
		"the money of the whole batch: the direct cost is the money of the whole batch of the recipe, and the " +
		"rates tell what that money weighs per unit, per gram and per kilogram.\n\n" +
		"A reference is never the price of the ingredient: the price the owner pays comes from her own " +
		"purchases, and the reference is only what the market was seen at. It never moves the money of the " +
		"book.\n\n" +
		"Every reference shows its age: the number of days between the day it was seen and the day of the " +
		"reading. When the owner expects the references to be fresh, send max_age_days, and every reference " +
		"older than that is marked as old, with its age.\n\n" +
		"The business is REQUIRED: it is the business whose margin is read, and PACA never invents one.\n\n" +
		"The recipe is OPTIONAL: send the identity the tool " + ToolRecordRecipe + " wrote for the recipe, or " +
		"the name the owner reads ('Pan de molde'), and leave both out to read the margin of every recipe of the " +
		"business.\n\n" +
		"The price is OPTIONAL: it is the money the owner charges for the product, in whole pesos ('" +
		business.DefaultCurrency + "'), and the money of what she sells is never inferred. When she has not " +
		"declared it, the answer says that the price is not known, and the cost and the market still answer.\n\n" +
		"The price_basis is REQUIRED when the price is sent: it is the way she charges, and it is one of " +
		priceBasesOfTheBook() + ". The way she sells is hers, and the book never moves her price: the money " +
		"she declared is the money of the margin.\n\n" +
		"The answer carries, for every recipe:\n\n" +
		"  - the name the owner reads and the yield of the recipe: the units it makes and the weight of each one;\n" +
		"  - the PRICE of the owner and the DIRECT cost of the batch, each one with its rates per unit, per " +
		"gram and per kilogram;\n" +
		"  - the PROFIT: what the price leaves above the direct cost, per unit, per gram and per kilogram, and " +
		"the percent it is of the direct cost;\n" +
		"  - the MARKET: what the market said the ingredients of the batch cost, what the owner paid for them, " +
		"and the difference, with one line for every ingredient: the reference it compares with, the place it " +
		"was read in, the day it was seen, its age and whether it is old;\n" +
		"  - the PROFIT OVER THE MARKET: what the price leaves above what the market said;\n" +
		"  - what the margin still needs.\n\n" +
		"This tool changes nothing: it reads the margin of the products and it writes nothing."
}

// readMarginInput is the input of the tool that reads the margin of a
// product.
type readMarginInput struct {
	// BusinessID is the business whose margin is read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose margin is read. It is required."`

	// RecipeID is the recipe the caller asks about.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the recipe whose margin is read. It is optional: leave it out to read the margin of every recipe of the business."`

	// RecipeName is the name the owner reads of the recipe.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name the owner reads of the recipe ('Pan de molde'). It is optional: leave it out to read the margin of every recipe of the business."`

	// Price is the money the owner charges for the product.
	Price int64 `json:"price,omitempty" jsonschema:"The money the owner charges for the product, in whole pesos. It is optional: PACA never infers it, and the answer says that the price is not known when the owner declared none."`

	// PriceBasis is the way the owner charges.
	PriceBasis string `json:"price_basis,omitempty" jsonschema:"The way the owner charges: 'unit', 'gram' or 'kilogram'. It is required when the price is sent, and it is optional when the owner declared no price."`

	// MaxAgeDays is the number of days a reference of the market may be
	// old.
	MaxAgeDays int `json:"max_age_days,omitempty" jsonschema:"The number of days a reference of the market may be old. It is optional: leave it out when the owner expects no age, and every reference shows its age anyway."`
}

// marginProfitOutput is what the answer carries about a profit: what a
// price leaves above a cost, per unit, per gram and per kilogram.
//
// The profit is a rate and never a money: the price and the cost are two
// money facts of their own.
type marginProfitOutput struct {
	// PerUnit is the profit per unit of what the recipe makes.
	PerUnit float64 `json:"per_unit"`

	// PerGram is the profit per gram of what the recipe makes.
	PerGram float64 `json:"per_gram"`

	// PerKilogram is the profit per kilogram of what the recipe makes.
	PerKilogram float64 `json:"per_kilogram"`

	// Unknown says that the profit is not derivable.
	Unknown bool `json:"unknown"`

	// Reason says why the profit is not derivable, in the words of the
	// book.
	Reason string `json:"reason,omitempty"`
}

// marginReferenceOutput is what the answer carries about a reference of the
// market.
//
// The answer carries the place the price was read in and the day it was
// seen, and the age the reference has on the day of the reading.
type marginReferenceOutput struct {
	// ID is the identity of the reference.
	ID int64 `json:"id"`

	// IngredientID is the ingredient of the catalog the reference is about.
	IngredientID int64 `json:"ingredient_id"`

	// Price is the price that was seen, in whole pesos.
	Price int64 `json:"price"`

	// Unit is the unit of that price.
	Unit string `json:"unit"`

	// Market is the market where the price was seen.
	Market string `json:"market,omitempty"`

	// Source is the place the price was read.
	Source string `json:"source"`

	// Day is the day the price was seen.
	Day string `json:"day"`

	// AgeDays is how many days old the reference is.
	AgeDays int `json:"age_days"`

	// Old says that the reference is older than the owner expects.
	Old bool `json:"old"`
}

// marginLineOutput is what the answer carries about one ingredient of the
// batch: what the owner paid and what the market says.
type marginLineOutput struct {
	// IngredientID is the ingredient of the catalog the line is of.
	IngredientID int64 `json:"ingredient_id"`

	// Name is the name the owner reads of the ingredient.
	Name string `json:"name"`

	// Grams is how many grams of it the recipe needs.
	Grams float64 `json:"grams"`

	// Paid is what the owner paid for those grams, in whole pesos.
	Paid int64 `json:"paid"`

	// Money is what the market says those grams cost, in whole pesos.
	Money int64 `json:"money"`

	// Unknown says that the money of the market is not derivable.
	Unknown bool `json:"unknown"`

	// Reason says why the money of the market is not derivable, in the
	// words of the book.
	Reason string `json:"reason,omitempty"`

	// Reference is the reference that compares: the newest one.
	Reference *marginReferenceOutput `json:"reference,omitempty"`

	// References are every reference of the ingredient, the newest one
	// first.
	References []marginReferenceOutput `json:"references"`
}

// marginMarketOutput is what the answer carries about what the market said
// about the ingredients of the batch.
type marginMarketOutput struct {
	// Money is what the market says the ingredients of the batch cost, in
	// whole pesos.
	Money int64 `json:"money"`

	// Paid is what the owner paid for the ingredients of the batch, in
	// whole pesos.
	Paid int64 `json:"paid"`

	// Difference is what the owner paid above the market.
	Difference int64 `json:"difference"`

	// PerUnit is the money of the market per unit of what the recipe
	// makes.
	PerUnit float64 `json:"per_unit"`

	// PerGram is the money of the market per gram of what the recipe
	// makes.
	PerGram float64 `json:"per_gram"`

	// PerKilogram is the money of the market per kilogram of what the
	// recipe makes.
	PerKilogram float64 `json:"per_kilogram"`

	// Unknown says that the money of the market is not derivable.
	Unknown bool `json:"unknown"`

	// Reason says why the money of the market is not derivable, in the
	// words of the book.
	Reason string `json:"reason,omitempty"`

	// Lines are the lines of the market: one for every ingredient of the
	// recipe.
	Lines []marginLineOutput `json:"lines"`
}

// marginOutput is what the answer carries about the margin of one product.
type marginOutput struct {
	// RecipeID is the recipe the margin is of.
	RecipeID int64 `json:"recipe_id"`

	// Name is the name the owner reads of the recipe.
	Name string `json:"name"`

	// YieldUnits is the number of units the recipe makes.
	YieldUnits int `json:"yield_units"`

	// YieldGrams is the weight of one unit, in grams.
	YieldGrams float64 `json:"yield_grams"`

	// Price is the price of the owner: what she charges, as she declared
	// it.
	Price realPriceRateOutput `json:"price"`

	// PriceBasis is the way she charges.
	PriceBasis string `json:"price_basis,omitempty"`

	// Direct is the direct cost of the batch.
	Direct realPriceRateOutput `json:"direct"`

	// Profit is what the price leaves above the direct cost.
	Profit marginProfitOutput `json:"profit"`

	// Percent is the profit over the direct cost, in percent.
	Percent float64 `json:"percent"`

	// Market is what the market said about the ingredients of the batch.
	Market marginMarketOutput `json:"market"`

	// MarketProfit is what the price leaves above what the market said.
	MarketProfit marginProfitOutput `json:"market_profit"`

	// Missing is what the margin still needs.
	Missing []string `json:"missing"`
}

// readMarginOutput is the answer of the tool that reads the margin of a
// product.
type readMarginOutput struct {
	// Margins are the margins of the products, one for every recipe.
	Margins []marginOutput `json:"margins"`
}

// registerMarginTools registers the tools of the margin.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerMarginTools(reader ports.MarginReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadMargin,
		Description: descriptionReadMargin,
	}, s.readMargin(reader))
}

// readMargin is the handler of the tool that reads the margin of a
// product.
//
// The tool reads the margin and answers with the price of the owner, the
// direct cost of the batch, the profit and what the market said: it
// writes nothing.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the margin of nobody.
func (s *Server) readMargin(reader ports.MarginReader) sdk.ToolHandlerFor[readMarginInput, readMarginOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readMarginInput) (*sdk.CallToolResult, readMarginOutput, error) {
		if reader == nil {
			return nil, readMarginOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, readMarginOutput{}, s.toolError(purchase.ErrMissingBusiness)
		}

		read, err := reader.Read(ctx, ports.ReadMargin{
			BusinessID: business.ID(input.BusinessID),
			RecipeID:   input.RecipeID,
			RecipeName: input.RecipeName,
			Price:      input.Price,
			PriceBasis: input.PriceBasis,
			MaxAgeDays: input.MaxAgeDays,
		})
		if err != nil {
			return nil, readMarginOutput{}, s.toolError(err)
		}

		answer := readMarginOutput{Margins: make([]marginOutput, 0, len(read))}

		for _, one := range read {
			answer.Margins = append(answer.Margins, outputOfMargin(one))
		}

		return nil, answer, nil
	}
}

// outputOfMargin maps the margin of the domain onto the answer of the
// tool.
//
// The price of the owner travels beside the direct cost, and the market
// travels beside them: the answer never folds one into another.
func outputOfMargin(margin recipe.Margin) marginOutput {
	answer := marginOutput{
		RecipeID:     int64(margin.RecipeID),
		Name:         margin.Name,
		YieldUnits:   margin.Yield.Units,
		YieldGrams:   margin.Yield.Grams,
		Price:        outputOfRate(margin.Price),
		PriceBasis:   margin.PriceBasis.String(),
		Direct:       outputOfRate(margin.Direct),
		Profit:       outputOfProfit(margin.Profit),
		Percent:      margin.Percent,
		Market:       outputOfMarket(margin.Market),
		MarketProfit: outputOfProfit(margin.MarketProfit),
		Missing:      margin.Missing,
	}

	if answer.Missing == nil {
		answer.Missing = []string{}
	}

	return answer
}

// outputOfProfit maps the profit of the domain onto the answer of the
// tool.
func outputOfProfit(profit recipe.Profit) marginProfitOutput {
	return marginProfitOutput{
		PerUnit:     profit.PerUnit,
		PerGram:     profit.PerGram,
		PerKilogram: profit.PerKilogram,
		Unknown:     profit.Unknown,
		Reason:      profit.Reason,
	}
}

// outputOfMarket maps what the market said onto the answer of the tool.
func outputOfMarket(market recipe.Market) marginMarketOutput {
	answer := marginMarketOutput{
		Money:       int64(market.Money),
		Paid:        int64(market.Paid),
		Difference:  int64(market.Difference),
		PerUnit:     market.PerUnit,
		PerGram:     market.PerGram,
		PerKilogram: market.PerKilogram,
		Unknown:     market.Unknown,
		Reason:      market.Reason,
		Lines:       make([]marginLineOutput, 0, len(market.Lines)),
	}

	for _, one := range market.Lines {
		answer.Lines = append(answer.Lines, outputOfMarketLine(one))
	}

	return answer
}

// outputOfMarketLine maps one line of the market onto the answer of the
// tool.
//
// The reference that compares is the one the domain chose: the domain
// decides which reference of the ingredient compares, and the adapter never
// decides it again.
func outputOfMarketLine(line recipe.MarketLine) marginLineOutput {
	answer := marginLineOutput{
		IngredientID: int64(line.IngredientID),
		Name:         line.Name,
		Grams:        line.Grams,
		Paid:         int64(line.Paid),
		Money:        int64(line.Money),
		Unknown:      line.Unknown,
		Reason:       line.Reason,
		References:   make([]marginReferenceOutput, 0, len(line.References)),
	}

	if line.Reference.ID != 0 {
		chosen := outputOfReferenceView(line.Reference)
		answer.Reference = &chosen
	}

	for _, one := range line.References {
		answer.References = append(answer.References, outputOfReferenceView(one))
	}

	return answer
}

// outputOfReferenceView maps a reference of the market onto the answer of
// the tool.
//
// The answer carries the place the price was read in and the day it was
// seen, and the age the reference has: a reference older than the owner
// expects is shown with its age.
func outputOfReferenceView(reference recipe.ReferenceView) marginReferenceOutput {
	return marginReferenceOutput{
		ID:           int64(reference.ID),
		IngredientID: int64(reference.IngredientID),
		Price:        int64(reference.Price),
		Unit:         reference.Unit,
		Market:       reference.Market,
		Source:       reference.Source,
		Day:          reference.SeenOn.String(),
		AgeDays:      reference.AgeDays,
		Old:          reference.Old,
	}
}
