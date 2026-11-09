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

// stubExitRecorder is the use case that records the exit, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubExitRecorder struct {
	answer recipe.Exit
	err    error
	got    ports.RecordExit
	calls  int
}

// Record answers the exit of the test, and remembers what the tool asked.
func (s *stubExitRecorder) Record(_ context.Context, request ports.RecordExit) (recipe.Exit, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Exit{}, s.err
	}

	return s.answer, nil
}

// connectWithTheExit starts the transport with the stub of the exit.
func connectWithTheExit(t *testing.T, recorder ports.ExitRecorder) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordExit:          recorder,
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

// theExitOfTheTool is the exit the tool of the test answers: three units of
// bread the owner consumed, worth three thousand pesos.
func theExitOfTheTool(t *testing.T) recipe.Exit {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-14")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	return recipe.Exit{
		ID:           recipe.ExitID(6),
		BusinessID:   business.ID(77),
		Kind:         recipe.ExitConsumption,
		RecipeID:     recipe.ID(9),
		RecipeName:   "Pan de molde",
		Day:          day,
		Quantity:     3,
		Unit:         "unidad",
		Cost:         3000,
		TaxTreatment: recipe.TaxPendingTheProfessional,
		TaxQuestion:  "El tratamiento tributario del consumo del dueño está pendiente del profesional de los impuestos.",
		Units: []recipe.ExitedUnit{
			{
				UnitID:       recipe.UnitID(1),
				ProductionID: recipe.ProductionID(3),
				Position:     1,
				Grams:        90,
				ExpiresOn:    ledger.NewDate(day.Time().AddDate(0, 0, 3)),
			},
		},
		Utterance:     "me comí tres panes",
		ConfirmedWith: "sí",
	}
}

// TestTheRecordExitToolWritesTheKindTheProductAndTheQuantity proves that
// the tool writes the exit the owner confirmed: the kind of the exit, the
// product, the quantity, the unit, the day and her word.
func TestTheRecordExitToolWritesTheKindTheProductAndTheQuantity(t *testing.T) {
	t.Parallel()

	recorder := &stubExitRecorder{answer: theExitOfTheTool(t)}
	session := connectWithTheExit(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordExit,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"kind":           "consumption",
			"quantity":       3,
			"unit":           "unidad",
			"day":            "2026-03-14",
			"utterance":      "me comí tres panes",
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
	if recorder.got.Kind != "consumption" {
		t.Errorf("the kind of the exit is %q, want %q", recorder.got.Kind, "consumption")
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
	if recorder.got.Day != "2026-03-14" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-14")
	}
	if recorder.got.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", recorder.got.ConfirmedWith, "sí")
	}

	answer := textOf(t, result)
	for _, want := range []string{
		"Pan de molde",
		"unidad",
		"consumption",
		"3000",
		"pending_the_professional",
		"profesional de los impuestos",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheRecordExitToolRefusesTheBusinessThatIsNotThere proves that the
// business is required: the book never invents one.
func TestTheRecordExitToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubExitRecorder{answer: theExitOfTheTool(t)}
	session := connectWithTheExit(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordExit,
		Arguments: map[string]any{
			"recipe_name":    "Pan de molde",
			"kind":           "consumption",
			"quantity":       3,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered an exit, want the refusal of the business that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordExitToolRefusesTheKindThatIsNotThere proves that the kind
// of the exit is required: the exits of the bakery are the consumption of
// the owner and the donation, and PACA never guesses one.
func TestTheRecordExitToolRefusesTheKindThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubExitRecorder{answer: theExitOfTheTool(t)}
	session := connectWithTheExit(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordExit,
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
		t.Fatal("the tool answered an exit, want the refusal of the kind that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordExitToolRefusesTheQuantityThatIsNotThere proves that the
// quantity is required: PACA never fills it by herself.
func TestTheRecordExitToolRefusesTheQuantityThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubExitRecorder{answer: theExitOfTheTool(t)}
	session := connectWithTheExit(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordExit,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"kind":           "consumption",
			"quantity":       0,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered an exit, want the refusal of the quantity that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordExitToolRefusesTheWordThatIsNotThere proves that the word of
// the owner is required: the exit is written with her confirmation.
func TestTheRecordExitToolRefusesTheWordThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubExitRecorder{answer: theExitOfTheTool(t)}
	session := connectWithTheExit(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordExit,
		Arguments: map[string]any{
			"business_id": 77,
			"recipe_name": "Pan de molde",
			"kind":        "consumption",
			"quantity":    3,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered an exit, want the refusal of the word that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordExitToolWithoutADatabaseSaysSo proves that the tool is there
// even when the service has no database, and that it says so.
func TestTheRecordExitToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheExit(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordExit,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"kind":           "consumption",
			"quantity":       3,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered an exit, want the answer that the database is not there")
	}
}

// TestTheDescriptionOfTheExitTellsTheAgentTheRules proves that the
// description of the tool names the rules the agent has to know: the two
// kinds of the exit, the rotation by the expiry and the question of the
// taxes.
func TestTheDescriptionOfTheExitTellsTheAgentTheRules(t *testing.T) {
	t.Parallel()

	session := connectWithTheExit(t, &stubExitRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordExit)

	wants := []string{
		"consumption",
		"donation",
		"EXPIRE FIRST",
		"moves NO money",
		"professional of the taxes",
	}
	for _, want := range wants {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not tell the agent %q:\n%s", want, description)
		}
	}
}

// TestTheErrorsOfTheExitReachTheAgentWithAMessage proves that every error
// the exit can return reaches the agent with a message of its own.
func TestTheErrorsOfTheExitReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	for _, failure := range recipe.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			recorder := &stubExitRecorder{err: failure}
			session := connectWithTheExit(t, recorder)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolRecordExit,
				Arguments: map[string]any{
					"business_id":    77,
					"recipe_name":    "Pan de molde",
					"kind":           "consumption",
					"quantity":       3,
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
