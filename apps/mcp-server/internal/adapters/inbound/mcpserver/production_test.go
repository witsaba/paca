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
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// stubProductionRecorder is the use case that records the production, in
// memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubProductionRecorder struct {
	recorded recipe.Production
	err      error
	got      ports.RecordProduction
	calls    int
}

// Record answers the production of the test, and remembers what the tool
// asked.
func (s *stubProductionRecorder) Record(_ context.Context, request ports.RecordProduction) (recipe.Production, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Production{}, s.err
	}

	return s.recorded, nil
}

// connectWithTheProduction starts the transport with the stub of the
// production.
func connectWithTheProduction(t *testing.T, recorder ports.ProductionRecorder) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordProduction:    recorder,
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

// theProductionOfTheTool is the production the tool of the test answers.
func theProductionOfTheTool(t *testing.T) recipe.Production {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-14")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	production, err := recipe.NewProduction(business.ID(77), recipe.ProductionFact{
		RecipeID:   recipe.ID(9),
		RecipeName: "Pan de molde",
		Day:        day,
		GramsIn:    500,
		Money:      ledger.Amount(520),
		Takes: []purchase.Take{
			{
				LayerID:    purchase.LayerID(1),
				PurchaseID: purchase.ID(9),
				Day:        day,
				Quantity:   0.2,
				Unit:       "kg",
				Grams:      200,
				Money:      ledger.Amount(400),
			},
		},
		Units: []recipe.ProductionUnit{
			{
				Position:  1,
				Grams:     90,
				ExpiresOn: ledger.NewDate(day.Time().AddDate(0, 0, 3)),
			},
		},
		Utterance:     "Hice pan de molde el sábado",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the production of the test could not be built: %v", err)
	}

	return production
}

// TestTheRecordProductionToolWritesTheRecipeAndTheDay proves that the tool
// writes the production the owner confirmed: the recipe she made, the day
// she made it and the word she used, and it answers with the grams that
// went in, the money and the units that came out.
func TestTheRecordProductionToolWritesTheRecipeAndTheDay(t *testing.T) {
	t.Parallel()

	recorder := &stubProductionRecorder{recorded: theProductionOfTheTool(t)}
	session := connectWithTheProduction(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordProduction,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"day":            "2026-03-14",
			"utterance":      "Hice pan de molde el sábado",
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if recorder.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", recorder.calls)
	}
	if recorder.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", recorder.got.BusinessID)
	}
	if recorder.got.RecipeName != "Pan de molde" {
		t.Errorf("the name of the recipe is %q, want %q", recorder.got.RecipeName, "Pan de molde")
	}
	if recorder.got.Day != "2026-03-14" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-14")
	}
	if recorder.got.Utterance != "Hice pan de molde el sábado" {
		t.Errorf("the sentence of the owner is %q, want the one she said", recorder.got.Utterance)
	}
	if recorder.got.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", recorder.got.ConfirmedWith, "sí")
	}

	answer := textOf(t, result)
	for _, want := range []string{"2026-03-14", "500", "520", "2026-03-17"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheRecordProductionToolRefusesTheBusinessThatIsNotThere proves that
// the business is required: the tool refuses the business that is not
// there, and it never asks the use case for the production of nobody.
func TestTheRecordProductionToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubProductionRecorder{recorded: theProductionOfTheTool(t)}
	session := connectWithTheProduction(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordProduction,
		Arguments: map[string]any{
			"business_id":    0,
			"recipe_name":    "Pan de molde",
			"day":            "2026-03-14",
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a production, want the refusal of the business that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordProductionToolSaysTheStockDoesNotHoldWhatTheRecipeNeeds
// proves that the refusal of the domain reaches the agent with the words
// that tell it what to do next.
func TestTheRecordProductionToolSaysTheStockDoesNotHoldWhatTheRecipeNeeds(t *testing.T) {
	t.Parallel()

	recorder := &stubProductionRecorder{err: recipe.ErrMissingStock}
	session := connectWithTheProduction(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordProduction,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"day":            "2026-03-14",
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a production, want the refusal of the stock")
	}

	answer := textOf(t, result)
	if !strings.Contains(answer, "the stock does not hold what the recipe needs") {
		t.Errorf("the answer does not say what the stock does not hold:\n%s", answer)
	}
}

// TestTheRecordProductionToolWithoutADatabaseSaysSo proves that the tool is
// there even when the service has no database, and that it says so.
func TestTheRecordProductionToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheProduction(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordProduction,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"day":            "2026-03-14",
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a production, want the news that the database is not there")
	}
}

// TestTheDescriptionOfTheProductionToolIsAContractWithTheAgent proves that
// the description of the tool says what the tool does: the recipe, the day,
// the grams that went in and the units that came out, and the fact that
// the book derives them instead of asking the owner.
func TestTheDescriptionOfTheProductionToolIsAContractWithTheAgent(t *testing.T) {
	t.Parallel()

	session := connectWithTheProduction(t, &stubProductionRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordProduction)

	for _, want := range []string{
		"business is REQUIRED",
		"recipe is REQUIRED",
		"day is OPTIONAL",
		"utterance is OPTIONAL",
		"confirmed_with is REQUIRED",
		"FIFO",
		"expires",
		"never asks the owner for them",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}
