package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The name of the tool of the capacity.
//
// The name is the one the agent reads in the list of the tools, so it says
// what the tool does and not what it writes: a tool that reads the capacity
// of a weekend is not a tool that writes a production.
const (
	// ToolReadCapacity reads what the hours the owner declared carry of the
	// recipes of the business.
	ToolReadCapacity = "read_capacity"
)

// The description of the tool of the capacity.
//
// The description is a contract with the agent: it says what the tool does,
// what it needs and what it answers, and nothing it does not do.
var (
	descriptionReadCapacity = buildDescriptionReadCapacity()
)

// buildDescriptionReadCapacity renders the description of the tool that reads
// the capacity of a weekend.
//
// The description says whose hours they are, what the capacity counts and
// what it never promises: the capacity is not a promise of the calendar.
func buildDescriptionReadCapacity() string {
	return "Read the capacity of the weekend: what the hours the owner has carry of the recipes of the bakery.\n\n" +
		"The hours ('hours') are the hours of the OWNER's OWN HANDS for the weekend: the time she has to amass, to " +
		"shape and to bake. PACA NEVER INVENTS THEM: they are hers, she declares them, and the tool refuses the " +
		"weekend that carries no hours.\n\n" +
		"The capacity counts the labor of the HANDS of the owner, and nothing else: the OVEN and the WAITING are " +
		"NOT counted, because the batch bakes and rests while she does the next thing. The minutes of a batch that " +
		"travel in the answer are the minutes of her hands alone.\n\n" +
		"The capacity is NEVER a promise of the calendar: it is what the weekend can carry with those hours, and " +
		"the stock of the ingredients may carry LESS. A weekend of ten hours that the book says carries three " +
		"batches carries three batches only if the flour, the fat and the starter are in the house.\n\n" +
		"The BATCHES are WHOLE: the book never splits a batch, so the hours carry the whole number of batches they " +
		"carry and never a part of one. The UNITS are the units of those batches: the batches times the yield of " +
		"the recipe, the yield the owner declared when she wrote it.\n\n" +
		"The business ('business_id') is REQUIRED: it is the business whose capacity is read, the one that sells " +
		"the bread, and PACA never invents one. A business that is not there is refused, and so is the recipe of " +
		"another business.\n\n" +
		"The answer carries the hours the owner declared ('hours'), and for every recipe of the business " +
		"('recipes'): the identity and the name the owner reads ('recipe_id', 'recipe_name'), the minutes of a " +
		"batch that need her hands ('minutes'), the whole batches those hours carry ('batches'), the units of those " +
		"batches ('units') and whether the hours bound the recipe ('limited').\n\n" +
		"A recipe whose labor of the hands is zero is NOT LIMITED by the hours: it answers 'limited' false, with no " +
		"batch and no unit, because the hours do not bound it and a zero there would be a lie.\n\n" +
		"This tool changes nothing: it reads the capacity of the weekend and it writes nothing."
}

// readCapacityInput is the input of the tool that reads the capacity.
//
// Every field is OPTIONAL in the schema: the domain is the one that refuses
// the weekend without hours and without a business, and the agent reads the
// message of the domain.
type readCapacityInput struct {
	// BusinessID is the business whose capacity is read.
	BusinessID int64 `json:"business_id,omitempty" jsonschema:"The identity of the business whose capacity is read: the business of its own, the one that sells the bread. Send it always: PACA never invents one, and the domain refuses the reading when it is left out."`

	// Hours is the hours of the hands of the owner for the weekend.
	Hours float64 `json:"hours,omitempty" jsonschema:"The hours of the OWNER's own hands for the weekend: the time she has to amass, to shape and to bake. Send it always: PACA never invents them, and the domain refuses the weekend that carries no hours."`
}

// capacityOfARecipeOutput is what the answer carries about one recipe.
type capacityOfARecipeOutput struct {
	// RecipeID is the recipe the capacity is of.
	RecipeID int64 `json:"recipe_id"`

	// RecipeName is the name the owner reads of the recipe.
	RecipeName string `json:"recipe_name"`

	// Minutes is the labor of a batch that needs the owner, in minutes of
	// her hands.
	Minutes int `json:"minutes"`

	// Batches is the whole number of batches the hours carry.
	Batches int `json:"batches"`

	// Units is the units those batches make.
	Units int `json:"units"`

	// Limited says that the hours bound the recipe: it is false when the
	// recipe carries no labor of the hands.
	Limited bool `json:"limited"`
}

// readCapacityOutput is the answer of the tool that reads the capacity.
type readCapacityOutput struct {
	// Hours is the hours the owner declared.
	Hours float64 `json:"hours"`

	// Recipes is the capacity of every recipe of the business.
	Recipes []capacityOfARecipeOutput `json:"recipes"`
}

// registerCapacityTools registers the tool of the capacity.
//
// The tool is registered even when the service has no database: the list of
// the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerCapacityTools(reader ports.CapacityReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadCapacity,
		Description: descriptionReadCapacity,
	}, s.readCapacity(reader))
}

// readCapacity is the handler of the tool that reads the capacity.
//
// The tool reads the capacity of the weekend and answers it: it writes
// nothing.
//
// The tool refuses nothing before the use case and the domain: the hours that
// are not positive and the business that is not there are refused by them,
// and the agent reads the message of the domain.
func (s *Server) readCapacity(reader ports.CapacityReader) sdk.ToolHandlerFor[readCapacityInput, readCapacityOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readCapacityInput) (*sdk.CallToolResult, readCapacityOutput, error) {
		if reader == nil {
			return nil, readCapacityOutput{}, ports.ErrNoDatabase
		}

		capacity, err := reader.Read(ctx, ports.ReadCapacity{
			BusinessID: businessIDOf(input.BusinessID),
			Hours:      input.Hours,
		})
		if err != nil {
			return nil, readCapacityOutput{}, s.toolError(err)
		}

		return nil, outputOfTheCapacity(capacity), nil
	}
}

// outputOfTheCapacity maps the capacity of the domain onto the answer of the
// tool.
func outputOfTheCapacity(capacity recipe.Capacity) readCapacityOutput {
	answer := readCapacityOutput{
		Hours:   capacity.Hours,
		Recipes: make([]capacityOfARecipeOutput, 0, len(capacity.Recipes)),
	}

	for _, one := range capacity.Recipes {
		answer.Recipes = append(answer.Recipes, capacityOfARecipeOutput{
			RecipeID:   int64(one.RecipeID),
			RecipeName: one.RecipeName,
			Minutes:    one.Minutes,
			Batches:    one.Batches,
			Units:      one.Units,
			Limited:    one.Limited,
		})
	}

	return answer
}
