package mcpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// stubCapacityReader is the use case that reads the capacity, in memory.
//
// It answers what the test tells it to answer, and it remembers what the tool
// asked it.
type stubCapacityReader struct {
	answer recipe.Capacity
	err    error
	got    ports.ReadCapacity
	calls  int
}

// Read answers the capacity of the test, and remembers what the tool asked.
func (s *stubCapacityReader) Read(_ context.Context, request ports.ReadCapacity) (recipe.Capacity, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Capacity{}, s.err
	}

	return s.answer, nil
}

// theCapacityOfTheTool is the answer of the use case of the test: ten hours
// of the owner, a batch of three hours of her hands that makes ten units, and
// a recipe the hours do not bound.
func theCapacityOfTheTool() recipe.Capacity {
	return recipe.Capacity{
		Hours: 10,
		Recipes: []recipe.CapacityOfARecipe{
			{
				RecipeID:   recipe.ID(9),
				RecipeName: "Pan de molde",
				Minutes:    180,
				Batches:    3,
				Units:      30,
				Limited:    true,
			},
			{
				RecipeID:   recipe.ID(10),
				RecipeName: "Pan integral",
				Minutes:    0,
				Batches:    0,
				Units:      0,
				Limited:    false,
			},
		},
	}
}

// theCapacityAnswer is what the tool answers, read back by the test.
type theCapacityAnswer struct {
	Hours   float64 `json:"hours"`
	Recipes []struct {
		RecipeID   int64  `json:"recipe_id"`
		RecipeName string `json:"recipe_name"`
		Minutes    int    `json:"minutes"`
		Batches    int    `json:"batches"`
		Units      int    `json:"units"`
		Limited    bool   `json:"limited"`
	} `json:"recipes"`
}

// connectWithTheCapacity starts the transport with the stub of the capacity.
func connectWithTheCapacity(t *testing.T, reader ports.CapacityReader) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		ReadCapacity:        reader,
	})

	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)

	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: server.URL}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the transport: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Logf("the session did not close: %v", err)
		}
	})

	return session
}

// TestTheReadCapacityToolAnswersWhatTheWeekendCarries proves that the tool
// carries the business and the hours of the owner to the use case, and that
// the answer carries the hours and the capacity of every recipe.
func TestTheReadCapacityToolAnswersWhatTheWeekendCarries(t *testing.T) {
	t.Parallel()

	reader := &stubCapacityReader{answer: theCapacityOfTheTool()}
	session := connectWithTheCapacity(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadCapacity,
		Arguments: map[string]any{
			"business_id": 77,
			"hours":       10,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if reader.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", reader.calls)
	}
	if reader.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", reader.got.BusinessID)
	}
	if reader.got.Hours != 10 {
		t.Errorf("the hours are %v, want the 10 the owner declared", reader.got.Hours)
	}

	var answer theCapacityAnswer

	jsonOf(t, result, &answer)

	if answer.Hours != 10 {
		t.Errorf("the answer carries %v hours, want 10", answer.Hours)
	}
	if len(answer.Recipes) != 2 {
		t.Fatalf("the answer carries %d recipes, want 2", len(answer.Recipes))
	}

	first := answer.Recipes[0]
	if first.RecipeID != 9 || first.RecipeName != "Pan de molde" {
		t.Errorf("the first recipe is %d with the name %q, want 9 with the name %q",
			first.RecipeID, first.RecipeName, "Pan de molde")
	}
	if first.Minutes != 180 {
		t.Errorf("the minutes of the batch are %d, want the 180 of the hands of the owner", first.Minutes)
	}
	if first.Batches != 3 {
		t.Errorf("the batches are %d, want the 3 whole batches of the weekend", first.Batches)
	}
	if first.Units != 30 {
		t.Errorf("the units are %d, want the 30 of the 3 batches of 10 units", first.Units)
	}
	if !first.Limited {
		t.Error("the recipe of the hands of the owner is not limited by the hours")
	}

	second := answer.Recipes[1]
	if second.Limited {
		t.Error("the recipe without labor of the hands says that the hours limit it")
	}
	if second.Batches != 0 || second.Units != 0 {
		t.Errorf("the recipe the hours do not bound carries %d batches and %d units, want none",
			second.Batches, second.Units)
	}
}

// TestTheReadCapacityToolRefusesTheCallWithoutHours proves the rule of the
// domain through the transport: the hours are the rule of the domain, and the
// tool answers the refusal with the message of the domain.
func TestTheReadCapacityToolRefusesTheCallWithoutHours(t *testing.T) {
	t.Parallel()

	reader := &stubCapacityReader{err: recipe.ErrInvalidCapacityHours}
	session := connectWithTheCapacity(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadCapacity,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a capacity of no hours")
	}

	message := textOf(t, result)
	if message == recipe.ErrInvalidCapacityHours.Error() {
		t.Errorf("the message is the raw error of the domain: %s", message)
	}
	if !strings.Contains(message, "hours") || !strings.Contains(message, "weekend") {
		t.Errorf("the message does not name the rule of the hours of the weekend: %s", message)
	}
	if !strings.Contains(message, "positive") {
		t.Errorf("the message does not say that the hours are positive: %s", message)
	}
}

// TestTheReadCapacityToolRefusesTheBusinessThatIsNotThere proves that the
// capacity belongs to a business: the agent reads the refusal of the domain
// in its own words.
func TestTheReadCapacityToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	reader := &stubCapacityReader{err: purchase.ErrMissingBusiness}
	session := connectWithTheCapacity(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolReadCapacity,
		Arguments: map[string]any{"hours": 10},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered the capacity of no business")
	}

	message := textOf(t, result)
	if message == purchase.ErrMissingBusiness.Error() {
		t.Errorf("the message is the raw error of the domain: %s", message)
	}
	if !strings.Contains(message, "business") {
		t.Errorf("the message does not name the business: %s", message)
	}
}

// TestTheReadCapacityToolOfARecipeOfAnotherBusinessSaysTheRule proves that
// the refusal of the recipe of another business is the rule of every tool
// that reads a recipe: the message names the rule and the way out, and it
// never names one single tool.
func TestTheReadCapacityToolOfARecipeOfAnotherBusinessSaysTheRule(t *testing.T) {
	t.Parallel()

	reader := &stubCapacityReader{err: recipe.ErrWrongBusiness}
	session := connectWithTheCapacity(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolReadCapacity,
		Arguments: map[string]any{"business_id": 77, "hours": 10},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered the capacity of a recipe of another business")
	}

	message := textOf(t, result)
	if !strings.Contains(message, "other business") && !strings.Contains(message, "another business") {
		t.Errorf("the message does not name the rule of the recipe of another business: %s", message)
	}
	if strings.Contains(message, mcpserver.ToolReadBatchCost) {
		t.Errorf("the message names one single tool: %s", message)
	}
}

// TestTheReadCapacityToolWithoutADatabaseSaysSo proves that the tool of a
// server without a database says that the database is not there.
func TestTheReadCapacityToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheCapacity(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolReadCapacity,
		Arguments: map[string]any{"business_id": 77, "hours": 10},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a capacity without a database")
	}
	if !strings.Contains(textOf(t, result), "database") {
		t.Errorf("the message does not talk about the database: %s", textOf(t, result))
	}
}

// TestTheSchemaOfTheReadCapacityToolRefusesNothingBeforeTheDomain proves the
// rule of the transport: the fields the domain refuses are optional in the
// schema, because the schema of a tool never refuses the call before the
// domain answers it.
func TestTheSchemaOfTheReadCapacityToolRefusesNothingBeforeTheDomain(t *testing.T) {
	t.Parallel()

	session := connectWithTheCapacity(t, &stubCapacityReader{answer: theCapacityOfTheTool()})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	required := theRequiredFieldsOfTheTool(t, tools.Tools, mcpserver.ToolReadCapacity)

	for _, field := range []string{"business_id", "hours"} {
		for _, one := range required {
			if one == field {
				t.Errorf("the schema marks %q as required: the schema must not refuse the call before the "+
					"domain answers it", field)
			}
		}
	}
}

// TestTheDescriptionOfTheCapacityIsAContractWithTheAgent proves that the
// description of the tool says what the capacity is, what it needs and what
// it never promises.
func TestTheDescriptionOfTheCapacityIsAContractWithTheAgent(t *testing.T) {
	t.Parallel()

	session := connectWithTheCapacity(t, &stubCapacityReader{answer: theCapacityOfTheTool()})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadCapacity)

	for _, want := range []string{
		"HANDS",
		"OVEN",
		"WAITING",
		"NEVER a promise of the calendar",
		"never invents",
		"whole",
		"stock",
		"changes nothing",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheToolListHoldsTheCapacityTool proves that the transport exposes the
// tool under the name the documentation of the service promises.
func TestTheToolListHoldsTheCapacityTool(t *testing.T) {
	t.Parallel()

	session := connectWithTheCapacity(t, &stubCapacityReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	for _, tool := range tools.Tools {
		if tool.Name == mcpserver.ToolReadCapacity {
			return
		}
	}

	t.Errorf("the server exposes no tool named %q", mcpserver.ToolReadCapacity)
}
