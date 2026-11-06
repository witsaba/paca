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

// stubLaborRecorder is the use case that records the labor, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubLaborRecorder struct {
	recorded recipe.Labor
	err      error
	got      ports.RecordLabor
	calls    int
}

// Record answers the labor of the test, and remembers what the tool asked.
func (s *stubLaborRecorder) Record(_ context.Context, request ports.RecordLabor) (recipe.Labor, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Labor{}, s.err
	}

	return s.recorded, nil
}

// connectWithTheLabor starts the transport with the stub of the labor.
func connectWithTheLabor(t *testing.T, recorder ports.LaborRecorder) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordLabor:         recorder,
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

// theLaborOfTheTool is the labor the tool of the test answers.
func theLaborOfTheTool(t *testing.T) recipe.Labor {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-15")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	hours, err := recipe.NewHours(2, 1, 0.5)
	if err != nil {
		t.Fatalf("the hours of the test could not be built: %v", err)
	}

	return recipe.Labor{
		ID:            recipe.LaborID(4),
		BusinessID:    business.ID(77),
		Day:           day,
		Hours:         hours,
		HourValue:     ledger.Amount(20000),
		Money:         ledger.Amount(70000),
		Utterance:     "Hoy trabajé tres horas y media en la panadería",
		ConfirmedWith: "sí",
	}
}

// TestTheRecordLaborToolWritesTheHoursAndTheValueOfTheHour proves that
// the tool writes the hours the owner worked and the value of the hour she
// declared, and answers with them.
func TestTheRecordLaborToolWritesTheHoursAndTheValueOfTheHour(t *testing.T) {
	t.Parallel()

	recorder := &stubLaborRecorder{recorded: theLaborOfTheTool(t)}
	session := connectWithTheLabor(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordLabor,
		Arguments: map[string]any{
			"business_id":    77,
			"hands_on_hours": 2,
			"oven_hours":     1,
			"waiting_hours":  0.5,
			"hour_value":     20000,
			"day":            "2026-03-15",
			"utterance":      "Hoy trabajé tres horas y media en la panadería",
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
	if recorder.got.HandsOnHours != 2 {
		t.Errorf("the hours of the hands are %v, want 2", recorder.got.HandsOnHours)
	}
	if recorder.got.OvenHours != 1 {
		t.Errorf("the hours of the oven are %v, want 1", recorder.got.OvenHours)
	}
	if recorder.got.WaitingHours != 0.5 {
		t.Errorf("the hours of the waiting are %v, want 0.5", recorder.got.WaitingHours)
	}
	if recorder.got.HourValue != 20000 {
		t.Errorf("the value of the hour is %d, want 20000", recorder.got.HourValue)
	}
	if recorder.got.Day != "2026-03-15" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-15")
	}
	if recorder.got.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", recorder.got.ConfirmedWith, "sí")
	}

	answer := textOf(t, result)
	for _, want := range []string{"2026-03-15", "70000", "20000"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheRecordLaborToolRefusesTheBusinessThatIsNotThere proves that the
// business is required: the tool refuses the business that is not there,
// and it never asks the use case for the labor of nobody.
func TestTheRecordLaborToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubLaborRecorder{recorded: theLaborOfTheTool(t)}
	session := connectWithTheLabor(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordLabor,
		Arguments: map[string]any{
			"business_id":    0,
			"hands_on_hours": 2,
			"hour_value":     20000,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a labor, want the refusal of the business that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordLaborToolWithoutADatabaseSaysSo proves that the tool is
// there even when the service has no database, and that it says so.
func TestTheRecordLaborToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheLabor(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordLabor,
		Arguments: map[string]any{
			"business_id":    77,
			"hands_on_hours": 2,
			"hour_value":     20000,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a labor, want the news that the database is not there")
	}
}

// TestTheDescriptionOfTheLaborToolIsAContractWithTheAgent proves that the
// description of the tool says what the tool does: the hours in the three
// kinds of the labor, the value of the hour the owner declared, and the
// labor of the owner alone.
func TestTheDescriptionOfTheLaborToolIsAContractWithTheAgent(t *testing.T) {
	t.Parallel()

	session := connectWithTheLabor(t, &stubLaborRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordLabor)

	for _, want := range []string{
		"business is REQUIRED",
		"HOURS",
		"hands_on_hours",
		"oven_hours",
		"waiting_hours",
		"hour_value is REQUIRED",
		"never inferred",
		"day is OPTIONAL",
		"confirmed_with is REQUIRED",
		"payroll of the employees is out of scope",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}
