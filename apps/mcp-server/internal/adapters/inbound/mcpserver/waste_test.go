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
	"github.com/witsaba/paca/packages/domain/recipe"
)

// stubWasteRecorder is the use case that records the waste, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubWasteRecorder struct {
	answer recipe.Waste
	err    error
	got    ports.RecordWaste
	calls  int
}

// Record answers the waste of the test, and remembers what the tool asked.
func (s *stubWasteRecorder) Record(_ context.Context, request ports.RecordWaste) (recipe.Waste, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Waste{}, s.err
	}

	return s.answer, nil
}

// connectWithTheWaste starts the transport with the stub of the waste.
func connectWithTheWaste(t *testing.T, recorder ports.WasteRecorder) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordWaste:         recorder,
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

// theWasteOfTheTool is the waste the tool of the test answers: three units
// that were burnt, worth three thousand pesos.
func theWasteOfTheTool(t *testing.T) recipe.Waste {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-14")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	return recipe.Waste{
		ID:         recipe.WasteID(4),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(9),
		RecipeName: "Pan de molde",
		Day:        day,
		Quantity:   3,
		Unit:       "unidad",
		Reason:     recipe.WasteBurnt,
		Money:      3000,
		Units: []recipe.WastedUnit{
			{
				UnitID:       recipe.UnitID(1),
				ProductionID: recipe.ProductionID(3),
				Position:     1,
				Grams:        90,
				ExpiresOn:    ledger.NewDate(day.Time().AddDate(0, 0, 3)),
			},
		},
		Utterance:     "se me quemaron tres panes",
		ConfirmedWith: "sí",
	}
}

// TestTheRecordWasteToolWritesTheProductTheQuantityAndTheReason proves that
// the tool writes the waste the owner confirmed: the product, the quantity,
// the unit, the reason and her word.
func TestTheRecordWasteToolWritesTheProductTheQuantityAndTheReason(t *testing.T) {
	t.Parallel()

	recorder := &stubWasteRecorder{answer: theWasteOfTheTool(t)}
	session := connectWithTheWaste(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordWaste,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"quantity":       3,
			"unit":           "unidad",
			"reason":         "burnt",
			"day":            "2026-03-14",
			"utterance":      "se me quemaron tres panes",
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
		t.Errorf("the name of the product is %q, want %q", recorder.got.RecipeName, "Pan de molde")
	}
	if recorder.got.Quantity != 3 {
		t.Errorf("the quantity is %v, want 3", recorder.got.Quantity)
	}
	if recorder.got.Unit != "unidad" {
		t.Errorf("the unit is %q, want %q", recorder.got.Unit, "unidad")
	}
	if recorder.got.Reason != "burnt" {
		t.Errorf("the reason is %q, want %q", recorder.got.Reason, "burnt")
	}
	if recorder.got.Day != "2026-03-14" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-14")
	}
	if recorder.got.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", recorder.got.ConfirmedWith, "sí")
	}

	answer := textOf(t, result)
	for _, want := range []string{"Pan de molde", "burnt", "unidad", "3000"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheRecordWasteToolRefusesTheBusinessThatIsNotThere proves that the
// business is required: the book never invents one.
func TestTheRecordWasteToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubWasteRecorder{answer: theWasteOfTheTool(t)}
	session := connectWithTheWaste(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordWaste,
		Arguments: map[string]any{
			"recipe_name":    "Pan de molde",
			"quantity":       3,
			"reason":         "burnt",
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a waste, want the refusal of the business that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordWasteToolRefusesTheQuantityThatIsNotThere proves that the
// quantity is required: the book never fills it by itself.
func TestTheRecordWasteToolRefusesTheQuantityThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubWasteRecorder{answer: theWasteOfTheTool(t)}
	session := connectWithTheWaste(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordWaste,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"quantity":       0,
			"reason":         "burnt",
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a waste, want the refusal of the quantity that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordWasteToolRefusesTheReasonThatIsNotThere proves that the
// reason is required: the reason is what a later analysis reads, and it is
// never invented.
func TestTheRecordWasteToolRefusesTheReasonThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubWasteRecorder{answer: theWasteOfTheTool(t)}
	session := connectWithTheWaste(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordWaste,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"quantity":       3,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a waste, want the refusal of the reason that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordWasteToolWithoutADatabaseSaysSo proves that the tool is there
// even when the service has no database, and that it says so.
func TestTheRecordWasteToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheWaste(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordWaste,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"quantity":       3,
			"reason":         "burnt",
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a waste, want the answer that the database is not there")
	}
}

// TestTheDescriptionOfTheWasteTellsTheAgentTheFourReasons proves that the
// description of the tool names the reasons of the book: the reason is what
// a later analysis reads, and the agent has to know the four of them.
func TestTheDescriptionOfTheWasteTellsTheAgentTheFourReasons(t *testing.T) {
	t.Parallel()

	session := connectWithTheWaste(t, &stubWasteRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordWaste)

	wants := []string{
		"'burnt'",
		"'spoiled'",
		"'dropped'",
		"'returned'",
		"never converts a weight into a number of units",
		"never lost from the book",
	}
	for _, want := range wants {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not tell the agent %q:\n%s", want, description)
		}
	}
}

// TestTheErrorsOfTheWasteReachTheAgentWithAMessage proves that every error
// the waste can return reaches the agent with a message of its own.
func TestTheErrorsOfTheWasteReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	for _, failure := range recipe.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			recorder := &stubWasteRecorder{err: failure}
			session := connectWithTheWaste(t, recorder)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolRecordWaste,
				Arguments: map[string]any{
					"business_id":    77,
					"recipe_name":    "Pan de molde",
					"quantity":       3,
					"reason":         "burnt",
					"confirmed_with": "sí",
				},
			})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool answered no error, want the failure of the use case")
			}

			answer := textOf(t, result)
			if answer == failure.Error() {
				t.Fatalf("the answer is the raw error of the domain: %s", answer)
			}
			if !strings.Contains(answer, ":") {
				t.Errorf("the answer says nothing about what to do: %s", answer)
			}
		})
	}
}
