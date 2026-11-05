package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
)

// The names of the tools of the catalog.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that matches a
// name is not a tool that writes an ingredient.
const (
	// ToolMatchItems matches the name the owner said against the catalog
	// of the names the business buys.
	ToolMatchItems = "match_items"
)

// The description of the tool of the catalog.
//
// The statuses of the answer belong to the domain: the description reads
// them and never writes them a second time.
var descriptionMatchItems = buildDescriptionMatchItems()

// buildDescriptionMatchItems renders the description of the tool that
// matches a name against the catalog of a business.
//
// The statuses of the answer belong to the domain, and the description
// reads them: the vocabulary of the answer is a rule, not a promise of the
// transport.
func buildDescriptionMatchItems() string {
	return "Match the name the owner just said against the catalog of the names the business buys.\n\n" +
		"The business is REQUIRED: it is the business whose catalog answers, and PACA never invents one. " +
		"Ask the person for it, and do not call this tool without it.\n\n" +
		"The name is REQUIRED: it is the name as the owner said it ('harina', 'Harina panadera'). Send it as " +
		"she said it, and not a summary of it.\n\n" +
		"The answer carries the status of the name:\n\n" +
		"  - 'exact' when one ingredient answers the name: the answer carries the ingredient and its identity.\n" +
		"  - 'ambiguous' when several answer it: the answer carries the options, so the person chooses with one " +
		"word.\n" +
		"  - 'unknown' when none of them does.\n\n" +
		"A name the catalog does not know is NOT an error: the answer says that it is new ('new': true), and the " +
		"name is recorded as she said it when she buys it.\n\n" +
		"The catalog begins empty and grows with what the business buys, so the first name is the one of the " +
		"first purchase.\n\n" +
		"Send the identity of the ingredient the tool answered to the tool of the draft of the purchase, so the " +
		"line of the purchase carries the ingredient of the catalog.\n\n" +
		"This tool changes nothing: it reads the catalog of the business and it writes nothing."
}

// matchItemsInput is the input of the tool that matches a name against the
// catalog of a business.
type matchItemsInput struct {
	// BusinessID is the business whose catalog answers.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose catalog answers. It is required."`

	// Name is the name as the owner said it.
	Name string `json:"name" jsonschema:"The name as the owner said it ('harina', 'Harina panadera'). It is required."`
}

// ingredientOutput is what the answer carries about one ingredient of the
// catalog.
type ingredientOutput struct {
	// ID is the identity of the ingredient.
	ID int64 `json:"id"`

	// Name is the name the owner reads.
	Name string `json:"name"`
}

// matchItemsOutput is the answer of the tool that matches a name against
// the catalog of a business.
type matchItemsOutput struct {
	// Name is the name as the owner said it.
	Name string `json:"name"`

	// Status is the answer of the catalog: exact, ambiguous or unknown.
	Status string `json:"status"`

	// New says that the catalog does not know the name.
	New bool `json:"new"`

	// Ingredient is the ingredient of an exact answer.
	Ingredient *ingredientOutput `json:"ingredient,omitempty"`

	// Options are the candidates of an ambiguous answer.
	Options []ingredientOutput `json:"options"`
}

// registerIngredientTools registers the tools of the catalog.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerIngredientTools(matcher ports.ItemMatcher) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolMatchItems,
		Description: descriptionMatchItems,
	}, s.matchItems(matcher))
}

// matchItems is the handler of the tool that matches a name against the
// catalog of a business.
//
// The tool reads the catalog and answers with the candidates: it writes
// nothing.
func (s *Server) matchItems(matcher ports.ItemMatcher) sdk.ToolHandlerFor[matchItemsInput, matchItemsOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input matchItemsInput) (*sdk.CallToolResult, matchItemsOutput, error) {
		if matcher == nil {
			return nil, matchItemsOutput{}, ports.ErrNoDatabase
		}

		matched, err := matcher.Match(ctx, ports.MatchItems{
			BusinessID: business.ID(input.BusinessID),
			Name:       input.Name,
		})
		if err != nil {
			return nil, matchItemsOutput{}, s.toolError(err)
		}

		return nil, outputOfMatch(matched), nil
	}
}

// outputOfMatch maps the answer of the domain onto the answer of the tool.
//
// The answer carries the status of the name, the ingredient of an exact
// answer and the options of an ambiguous one.
func outputOfMatch(matched ingredient.Match) matchItemsOutput {
	options := make([]ingredientOutput, 0, len(matched.Options))
	for _, one := range matched.Options {
		options = append(options, ingredientOutput{ID: int64(one.ID), Name: one.Name})
	}

	answer := matchItemsOutput{
		Name:    matched.Name,
		Status:  matched.Status.String(),
		New:     matched.New,
		Options: options,
	}

	if matched.Status == ingredient.StatusExact {
		answer.Ingredient = &ingredientOutput{
			ID:   int64(matched.Ingredient.ID),
			Name: matched.Ingredient.Name,
		}
	}

	return answer
}
