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
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// stubBatchCostReader is the use case that reads the cost of a batch, in
// memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubBatchCostReader struct {
	costs []recipe.Cost
	err   error
	got   ports.ReadBatchCost
	calls int
}

// Read answers the cost of the test, and remembers what the tool asked.
func (s *stubBatchCostReader) Read(_ context.Context, request ports.ReadBatchCost) ([]recipe.Cost, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return nil, s.err
	}

	return s.costs, nil
}

// connectWithTheBatchCost starts the transport with the stub of the cost.
func connectWithTheBatchCost(t *testing.T, reader ports.BatchCostReader) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		ReadBatchCost:       reader,
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

// theCostOfTheTest is the cost the tool of the test answers.
func theCostOfTheTest(t *testing.T) []recipe.Cost {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-01")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	return []recipe.Cost{
		{
			RecipeID:    recipe.ID(9),
			Name:        "Pan de molde",
			Direct:      ledger.Amount(240),
			Yield:       recipe.Yield{Units: 10, Grams: 90},
			PerUnit:     24,
			PerGram:     0.27,
			PerKilogram: 266.67,
			Complete:    true,
			Missing:     []string{},
			Lines: []recipe.CostLine{
				{
					IngredientID: ingredient.ID(10),
					Name:         "Harina panadera",
					Grams:        500,
					Available:    500,
					Money:        ledger.Amount(240),
					Layers: []purchase.Take{
						{
							LayerID:    purchase.LayerID(1),
							PurchaseID: purchase.ID(9),
							Day:        day,
							Quantity:   0.5,
							Unit:       "kg",
							Grams:      500,
							Money:      ledger.Amount(240),
						},
					},
				},
			},
		},
	}
}

// TestTheReadBatchCostToolReadsTheCost proves that the tool reads the cost
// of the batch and answers with what the recipe costs right now.
func TestTheReadBatchCostToolReadsTheCost(t *testing.T) {
	t.Parallel()

	reader := &stubBatchCostReader{costs: theCostOfTheTest(t)}
	session := connectWithTheBatchCost(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadBatchCost,
		Arguments: map[string]any{
			"business_id": 77,
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
	if reader.got.RecipeID != 0 {
		t.Errorf("the recipe is %d, want every recipe of the business", reader.got.RecipeID)
	}

	answer := textOf(t, result)
	for _, want := range []string{"Pan de molde", "240", "Harina panadera", "500", "2026-03-01"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheReadBatchCostToolReadsOneRecipe proves that the tool reads the
// cost of the recipe the agent names.
func TestTheReadBatchCostToolReadsOneRecipe(t *testing.T) {
	t.Parallel()

	reader := &stubBatchCostReader{costs: theCostOfTheTest(t)}
	session := connectWithTheBatchCost(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadBatchCost,
		Arguments: map[string]any{
			"business_id": 77,
			"recipe_id":   9,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if reader.got.RecipeID != 9 {
		t.Errorf("the recipe is %d, want 9", reader.got.RecipeID)
	}
}

// TestTheReadBatchCostToolReadsTheNameOfTheRecipe proves that the tool
// reads the cost of the recipe the owner names.
func TestTheReadBatchCostToolReadsTheNameOfTheRecipe(t *testing.T) {
	t.Parallel()

	reader := &stubBatchCostReader{costs: theCostOfTheTest(t)}
	session := connectWithTheBatchCost(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadBatchCost,
		Arguments: map[string]any{
			"business_id": 77,
			"recipe_name": "Pan de molde",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if reader.got.RecipeName != "Pan de molde" {
		t.Errorf("the name of the recipe is %q, want %q", reader.got.RecipeName, "Pan de molde")
	}
}

// TestTheReadBatchCostToolWithoutADatabaseSaysSo proves that the tool is
// there even when the service has no database, and that it says so.
func TestTheReadBatchCostToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheBatchCost(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadBatchCost,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a cost, want the news that the database is not there")
	}
}

// TestTheDescriptionOfTheBatchCostToolSaysThatTheCostIsDerived proves
// that the description of the tool is a contract with the agent: it says
// that the cost is derived from the layers, and it says that the oldest
// layer leaves first.
func TestTheDescriptionOfTheBatchCostToolSaysThatTheCostIsDerived(t *testing.T) {
	t.Parallel()

	session := connectWithTheBatchCost(t, &stubBatchCostReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadBatchCost)

	for _, want := range []string{
		"business is REQUIRED",
		"recipe is OPTIONAL",
		"derived",
		"oldest",
		"FIFO",
		"per unit",
		"per gram",
		"per kilogram",
		"changes nothing",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheReadBatchCostToolRefusesTheBusinessThatIsNotThere proves that
// the business is required: the tool refuses the business that is not
// there, and it never asks the use case for the cost of nobody.
func TestTheReadBatchCostToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	reader := &stubBatchCostReader{costs: theCostOfTheTest(t)}
	session := connectWithTheBatchCost(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadBatchCost,
		Arguments: map[string]any{
			"business_id": 0,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a cost, want the refusal of the business that is not there")
	}
	if reader.calls != 0 {
		t.Errorf("the use case was called %d times, want none", reader.calls)
	}
}
