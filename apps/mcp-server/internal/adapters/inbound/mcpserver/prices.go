package mcpserver

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/product"
)

// The names of the tools of the price list.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that sets a price
// is not a tool that creates a row.
const (
	// ToolSetPrice sets the price of a product for a channel: the retail
	// or the wholesale.
	ToolSetPrice = "set_price"

	// ToolReadPrices reads the price list of the business.
	ToolReadPrices = "read_prices"
)

// The descriptions of the tools of the price list.
//
// The channels of the book belong to the domain: the description reads them
// and never writes them a second time.
var (
	descriptionSetPrice   = buildDescriptionSetPrice()
	descriptionReadPrices = buildDescriptionReadPrices()
)

// buildDescriptionSetPrice renders the description of the tool that sets the
// price of a product.
//
// The channels of the book belong to the domain: the description reads them
// and never writes them a second time.
func buildDescriptionSetPrice() string {
	return "Set the price of a product of the bakery for a channel: the retail or the wholesale.\n\n" +
		"The business is REQUIRED: it is the business that owns the price list — the one that sells the bread. " +
		"The Personal Business of a person holds the money of that person, and not the price list of the bakery, " +
		"and PACA refuses a price list of it.\n\n" +
		"The product is named EITHER by the recipe it comes from ('recipe_id' or 'recipe_name') OR by the name " +
		"the owner reads ('name'). The bakery sells what it bakes: the product carries the recipe it comes from, " +
		"and PACA never invents one. The product is OPENED the first time it is named — that is the only way a " +
		"product comes into being — and PACA never guesses its name.\n\n" +
		"The channel is REQUIRED: it is one of " + theChannelsOfTheBook() + ". The owner says them as " +
		"'minorista' and 'mayorista' too, and PACA reads the two ways of saying them. There is NO default " +
		"channel and no price without one: the channel is already in the price from the first row, so that the " +
		"discount of the wholesale never has to rebuild the history of the prices. For now the two channels " +
		"carry the same price.\n\n" +
		"The amount ('amount') is REQUIRED: it is the money of the product in that channel, in whole pesos, and " +
		"it is a positive number: a price of zero pesos is not a price the owner charges. PACA never infers it.\n\n" +
		"The day ('day') is OPTIONAL: it is the day the price was set, and it is the day of today when the agent " +
		"names none.\n\n" +
		"The tax treatment ('tax_treatment') is OPTIONAL: it is the treatment the owner stated for the product, " +
		"and it is a fact of the product and not of the price. PACA never derives it from the price and never " +
		"assumes a rate: an empty one means the owner has not decided it yet.\n\n" +
		"The price is a FACT and it is never rewritten: every price is kept with the day it was set, and the " +
		"price that holds today for a product and a channel is the last one set on or before the day that is " +
		"asked for.\n\n" +
		"The answer carries the product — its identity, the recipe it comes from, the name the owner reads and " +
		"the tax treatment it carries — and the price: its identity, the channel, the money and the day it was set."
}

// buildDescriptionReadPrices renders the description of the tool that reads
// the price list of a business.
//
// The channels of the book belong to the domain: the description reads them
// and never writes them a second time.
func buildDescriptionReadPrices() string {
	return "Read the price list of the business: the price that holds for every product and every channel.\n\n" +
		"The business is REQUIRED: it is the business whose price list is read, and PACA never invents one.\n\n" +
		"The product is OPTIONAL: the identity of the recipe it comes from ('recipe_id' or 'recipe_name') or the " +
		"name the owner reads ('name') answer the price of that product alone, and neither of them answers the " +
		"whole list.\n\n" +
		"The channel ('channel') is OPTIONAL: it is one of " + theChannelsOfTheBook() + ", and an empty one " +
		"answers both of them.\n\n" +
		"The day ('day') is OPTIONAL: it is the day the list is read at, and it is the day of today when the " +
		"agent names none. The answer carries the price that holds on that day: the last one set on or before " +
		"it, with the day it was set.\n\n" +
		"The answer carries, for every product and every channel:\n\n" +
		"  - the product: its identity, the recipe it comes from, the name the owner reads and the tax treatment " +
		"it carries;\n" +
		"  - the price that holds: the channel, the money in whole pesos and the day it was set.\n\n" +
		"A product the business has not priced yet is not in the answer.\n\n" +
		"It writes nothing."
}

// setPriceInput is the input of the tool that sets the price of a product.
type setPriceInput struct {
	// BusinessID is the business that owns the price list.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business that owns the price list: the business of its own, the one that sells the bread. It is required, and PACA never invents one."`

	// RecipeID is the recipe the product comes from.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the recipe the product comes from. It names the product, and the agent sends it OR the name the owner reads, never neither of them."`

	// RecipeName is the name of the recipe the product comes from.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name of the recipe the product comes from ('Pan de molde'). It names the product, and the agent sends it OR the name the owner reads, never neither of them."`

	// Name is the name the owner reads of the product.
	Name string `json:"name,omitempty" jsonschema:"The name the owner reads of the product ('Pan de molde'). It names the product, and the agent sends it OR the recipe it comes from, never neither of them."`

	// Channel is the channel the price was set for.
	Channel string `json:"channel" jsonschema:"The channel the price was set for: 'retail' (the client that buys at the door) or 'wholesale' (the store that buys many of them). The owner says them as 'minorista' and 'mayorista' too. It is required, and there is no default channel."`

	// Amount is the money of the product in that channel.
	Amount int64 `json:"amount" jsonschema:"The money of the product in that channel, in whole pesos. It is required, it is a positive number, and PACA never infers it."`

	// Day is the day the price was set.
	Day string `json:"day,omitempty" jsonschema:"The day the price was set ('2026-03-01'). It is optional, and it is the day of today when the agent names none."`

	// TaxTreatment is the tax treatment the owner stated for the product.
	TaxTreatment string `json:"tax_treatment,omitempty" jsonschema:"The tax treatment the owner stated for the product. It is optional: PACA never derives it from the price, and it never assumes a rate."`
}

// readPricesInput is the input of the tool that reads the price list of a
// business.
type readPricesInput struct {
	// BusinessID is the business whose price list is read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose price list is read. It is required, and PACA never invents one."`

	// RecipeID is the recipe the product comes from.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the recipe the product comes from. It is optional, and it answers the price of that product alone."`

	// RecipeName is the name of the recipe the product comes from.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name of the recipe the product comes from ('Pan de molde'). It is optional, and it answers the price of that product alone."`

	// Name is the name the owner reads of the product.
	Name string `json:"name,omitempty" jsonschema:"The name the owner reads of the product ('Pan de molde'). It is optional, and it answers the price of that product alone."`

	// Channel is the channel the price list is read for.
	Channel string `json:"channel,omitempty" jsonschema:"The channel the price list is read for: 'retail' or 'wholesale'. It is optional, and an empty one answers both of them."`

	// Day is the day the price list is read at.
	Day string `json:"day,omitempty" jsonschema:"The day the price list is read at ('2026-03-01'). It is optional, and it is the day of today when the agent names none."`
}

// productOutput is what the tools of the price list answer about one
// product.
type productOutput struct {
	// ID is the identity of the product.
	ID int64 `json:"id"`

	// BusinessID is the business that owns the product.
	BusinessID int64 `json:"business_id"`

	// RecipeID is the recipe the product comes from.
	RecipeID int64 `json:"recipe_id"`

	// Name is the name the owner reads.
	Name string `json:"name"`

	// TaxTreatment is the tax treatment the product carries.
	TaxTreatment string `json:"tax_treatment,omitempty"`
}

// priceOutput is what the tools of the price list answer about one price.
type priceOutput struct {
	// ID is the identity of the price.
	ID int64 `json:"id"`

	// BusinessID is the business that owns the price.
	BusinessID int64 `json:"business_id"`

	// ProductID is the product the price is of.
	ProductID int64 `json:"product_id"`

	// Channel is the channel the price was set for.
	Channel string `json:"channel"`

	// Amount is the money of the product in that channel, in whole pesos.
	Amount int64 `json:"amount"`

	// Day is the day the price was set.
	Day string `json:"day"`
}

// setPriceOutput is the answer of the tool that sets the price of a
// product.
type setPriceOutput struct {
	// Product is the product the price is of.
	Product productOutput `json:"product"`

	// Price is the price that was set.
	Price priceOutput `json:"price"`
}

// readPricesOutput is the answer of the tool that reads the price list of a
// business.
type readPricesOutput struct {
	// Prices are the prices that hold, one line for every product and
	// every channel.
	Prices []listedPriceOutput `json:"prices"`
}

// listedPriceOutput is one line of the price list.
type listedPriceOutput struct {
	// Product is the product the line is of.
	Product productOutput `json:"product"`

	// Price is the price that holds for the product and the channel of
	// the line.
	Price priceOutput `json:"price"`
}

// registerPriceTools registers the tools of the price list.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerPriceTools(setter ports.PriceRecorder, reader ports.PricesReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolSetPrice,
		Description: descriptionSetPrice,
	}, s.setPrice(setter))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadPrices,
		Description: descriptionReadPrices,
	}, s.readPrices(reader))
}

// setPrice is the handler of the tool that sets the price of a product.
//
// The tool writes the price of the product for the channel and answers with
// it. The price is a fact: it is written once, and it is never rewritten.
func (s *Server) setPrice(setter ports.PriceRecorder) sdk.ToolHandlerFor[setPriceInput, setPriceOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input setPriceInput) (*sdk.CallToolResult, setPriceOutput, error) {
		if setter == nil {
			return nil, setPriceOutput{}, ports.ErrNoDatabase
		}

		written, err := setter.Set(ctx, ports.SetPrice{
			BusinessID:   businessIDOf(input.BusinessID),
			RecipeID:     input.RecipeID,
			RecipeName:   input.RecipeName,
			Name:         input.Name,
			Channel:      input.Channel,
			Amount:       input.Amount,
			Day:          input.Day,
			TaxTreatment: input.TaxTreatment,
		})
		if err != nil {
			return nil, setPriceOutput{}, s.toolError(err)
		}

		return nil, setPriceOutput{
			Product: outputOfProduct(written.Product),
			Price:   outputOfPrice(written.Price),
		}, nil
	}
}

// readPrices is the handler of the tool that reads the price list of a
// business.
//
// The tool reads the prices that hold: it changes nothing, and it answers
// with every product and every channel of the list.
func (s *Server) readPrices(reader ports.PricesReader) sdk.ToolHandlerFor[readPricesInput, readPricesOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readPricesInput) (*sdk.CallToolResult, readPricesOutput, error) {
		if reader == nil {
			return nil, readPricesOutput{}, ports.ErrNoDatabase
		}

		held, err := reader.Read(ctx, ports.ReadPrices{
			BusinessID: businessIDOf(input.BusinessID),
			RecipeID:   input.RecipeID,
			RecipeName: input.RecipeName,
			Name:       input.Name,
			Channel:    input.Channel,
			Day:        input.Day,
		})
		if err != nil {
			return nil, readPricesOutput{}, s.toolError(err)
		}

		prices := make([]listedPriceOutput, 0, len(held))
		for _, one := range held {
			prices = append(prices, listedPriceOutput{
				Product: outputOfProduct(one.Product),
				Price:   outputOfPrice(one.Price),
			})
		}

		return nil, readPricesOutput{Prices: prices}, nil
	}
}

// outputOfProduct maps the model of the domain onto the answer of the tools
// of the price list.
func outputOfProduct(model product.Product) productOutput {
	return productOutput{
		ID:           int64(model.ID),
		BusinessID:   int64(model.BusinessID),
		RecipeID:     int64(model.RecipeID),
		Name:         model.Name,
		TaxTreatment: model.TaxTreatment,
	}
}

// outputOfPrice maps the model of the domain onto the answer of the tools of
// the price list.
func outputOfPrice(model product.Price) priceOutput {
	return priceOutput{
		ID:         int64(model.ID),
		BusinessID: int64(model.BusinessID),
		ProductID:  int64(model.ProductID),
		Channel:    model.Channel.String(),
		Amount:     int64(model.Amount),
		Day:        model.Day.String(),
	}
}

// theChannelsOfTheBook renders the channels of the price list for the
// description of the tool.
//
// The channels belong to the domain: the description reads them and never
// writes them a second time.
func theChannelsOfTheBook() string {
	channels := product.AllChannels()

	names := make([]string, 0, len(channels))
	for _, one := range channels {
		names = append(names, one.String())
	}

	return strings.Join(names, ", ")
}
