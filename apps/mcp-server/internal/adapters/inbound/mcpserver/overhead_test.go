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

// stubOverheadRecorder is the use case that records the overhead, in
// memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubOverheadRecorder struct {
	recorded recipe.Overhead
	err      error
	got      ports.RecordOverhead
	calls    int
}

// Record answers the overhead of the test, and remembers what the tool
// asked.
func (s *stubOverheadRecorder) Record(_ context.Context, request ports.RecordOverhead) (recipe.Overhead, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Overhead{}, s.err
	}

	return s.recorded, nil
}

// connectWithTheOverhead starts the transport with the stub of the
// overhead.
func connectWithTheOverhead(t *testing.T, recorder ports.OverheadRecorder) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordOverhead:      recorder,
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

// theOverheadOfTheTool is the overhead the tool of the test answers.
func theOverheadOfTheTool(t *testing.T) recipe.Overhead {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-15")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	overhead, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
		Electricity:   ledger.Amount(120000),
		OvenHours:     40,
		Allocation:    0.5,
		Day:           day,
		Utterance:     "Pagué la luz del mes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the overhead of the test could not be built: %v", err)
	}

	return overhead
}

// TestTheRecordOverheadToolWritesTheElectricityAndTheAllocation proves that
// the tool writes the electricity of the house, the hours of the oven and
// the allocation the owner declared, and answers with them.
func TestTheRecordOverheadToolWritesTheElectricityAndTheAllocation(t *testing.T) {
	t.Parallel()

	recorder := &stubOverheadRecorder{recorded: theOverheadOfTheTool(t)}
	session := connectWithTheOverhead(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordOverhead,
		Arguments: map[string]any{
			"business_id":    77,
			"electricity":    120000,
			"oven_hours":     40,
			"allocation":     0.5,
			"day":            "2026-03-15",
			"utterance":      "Pagué la luz del mes",
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
	if recorder.got.Electricity != 120000 {
		t.Errorf("the electricity of the house is %d, want 120000", recorder.got.Electricity)
	}
	if recorder.got.OvenHours != 40 {
		t.Errorf("the hours of the oven are %v, want 40", recorder.got.OvenHours)
	}
	if recorder.got.Allocation != 0.5 {
		t.Errorf("the allocation is %v, want 0.5", recorder.got.Allocation)
	}
	if recorder.got.Day != "2026-03-15" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-15")
	}
	if recorder.got.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", recorder.got.ConfirmedWith, "sí")
	}

	answer := textOf(t, result)
	for _, want := range []string{"2026-03-15", "120000", "60000"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheRecordOverheadToolCarriesTheInferredElectricity proves that the
// tool carries the fact that PACA filled the electricity by itself: the
// money of the house is never inferred, and the domain refuses the
// overhead that carries it.
func TestTheRecordOverheadToolCarriesTheInferredElectricity(t *testing.T) {
	t.Parallel()

	recorder := &stubOverheadRecorder{recorded: theOverheadOfTheTool(t)}
	session := connectWithTheOverhead(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordOverhead,
		Arguments: map[string]any{
			"business_id":          77,
			"electricity":          120000,
			"electricity_inferred": true,
			"oven_hours":           40,
			"allocation":           0.5,
			"confirmed_with":       "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if !recorder.got.ElectricityInferred {
		t.Error("the tool did not carry that the electricity was inferred")
	}
}

// TestTheRecordOverheadToolRefusesTheBusinessThatIsNotThere proves that
// the business is required: the tool refuses the business that is not
// there, and it never asks the use case for the overhead of nobody.
func TestTheRecordOverheadToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubOverheadRecorder{recorded: theOverheadOfTheTool(t)}
	session := connectWithTheOverhead(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordOverhead,
		Arguments: map[string]any{
			"business_id":    0,
			"electricity":    120000,
			"oven_hours":     40,
			"allocation":     0.5,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered an overhead, want the refusal of the business that is not there")
	}
	if recorder.calls != 0 {
		t.Errorf("the use case was called %d times, want none", recorder.calls)
	}
}

// TestTheRecordOverheadToolWithoutADatabaseSaysSo proves that the tool is
// there even when the service has no database, and that it says so.
func TestTheRecordOverheadToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheOverhead(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordOverhead,
		Arguments: map[string]any{
			"business_id":    77,
			"electricity":    120000,
			"oven_hours":     40,
			"allocation":     0.5,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered an overhead, want the news that the database is not there")
	}
}

// TestTheDescriptionOfTheOverheadToolIsAContractWithTheAgent proves that
// the description of the tool says what the tool does: the electricity of
// the house, the hours of the oven, the allocation the owner declared and
// the fact that the money of the house is never inferred.
func TestTheDescriptionOfTheOverheadToolIsAContractWithTheAgent(t *testing.T) {
	t.Parallel()

	session := connectWithTheOverhead(t, &stubOverheadRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordOverhead)

	for _, want := range []string{
		"business is REQUIRED",
		"electricity",
		"oven_hours",
		"allocation",
		"SHARE",
		"never infers",
		"day is OPTIONAL",
		"confirmed_with is REQUIRED",
		"derived from the hours of the oven",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}
