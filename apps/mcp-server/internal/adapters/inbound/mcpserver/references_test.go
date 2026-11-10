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
)

// stubReferenceRecorder is the use case that writes the reference prices, in
// memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubReferenceRecorder struct {
	recorded purchase.Reference
	err      error
	got      ports.RecordReference
	calls    int
}

// Record answers the reference of the test, and remembers what the tool
// asked.
func (s *stubReferenceRecorder) Record(_ context.Context, request ports.RecordReference) (purchase.Reference, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return purchase.Reference{}, s.err
	}

	return s.recorded, nil
}

// stubReferencesReader is the use case that reads the reference prices, in
// memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubReferencesReader struct {
	read  []purchase.Reference
	err   error
	got   ports.ReadReferences
	calls int
}

// Read answers the references of the test, and remembers what the tool
// asked.
func (s *stubReferencesReader) Read(_ context.Context, request ports.ReadReferences) ([]purchase.Reference, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return nil, s.err
	}

	return s.read, nil
}

// connectWithTheReferences starts the transport with the stubs of the
// reference prices.
func connectWithTheReferences(
	t *testing.T,
	recorder ports.ReferenceRecorder,
	reader ports.ReferencesReader,
) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordReference:     recorder,
		ReadReferences:      reader,
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

// theAnswerOfTheReference is the answer the stub of the use case gives.
func theAnswerOfTheReference(t *testing.T) purchase.Reference {
	return purchase.Reference{
		ID:            purchase.ReferenceID(9),
		BusinessID:    business.ID(77),
		IngredientID:  ingredient.ID(10),
		Price:         480,
		Unit:          "kg",
		Market:        "Plaza de Mercado",
		Source:        "Boletín de precios al por mayor, DANE",
		SeenOn:        mustDayOfTheReference(t),
		Utterance:     "La harina está a 480 el kilo en la plaza",
		ConfirmedWith: "sí",
	}
}

// mustDayOfTheReference is the day the reference of the test was seen.
func mustDayOfTheReference(t *testing.T) ledger.Date {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-01")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	return day
}

// theArgumentsOfTheReference are the arguments the agent sends.
func theArgumentsOfTheReference() map[string]any {
	return map[string]any{
		"business_id":    77,
		"ingredient_id":  10,
		"price":          480,
		"unit":           "kg",
		"market":         "Plaza de Mercado",
		"source":         "Boletín de precios al por mayor, DANE",
		"day":            "2026-03-01",
		"utterance":      "La harina está a 480 el kilo en la plaza",
		"confirmed_with": "sí",
	}
}

// TestTheReferenceToolWritesTheReference proves that the tool carries what
// the agent sends to the use case, and that the answer carries the reference
// with its source and its day.
func TestTheReferenceToolWritesTheReference(t *testing.T) {
	t.Parallel()

	recorder := &stubReferenceRecorder{recorded: theAnswerOfTheReference(t)}
	session := connectWithTheReferences(t, recorder, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolRecordReference,
		Arguments: theArgumentsOfTheReference(),
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
	if recorder.got.IngredientID != 10 {
		t.Errorf("the ingredient is %d, want 10", recorder.got.IngredientID)
	}
	if recorder.got.Price != 480 {
		t.Errorf("the price is %d, want 480", recorder.got.Price)
	}
	if recorder.got.Unit != "kg" {
		t.Errorf("the unit is %q, want %q", recorder.got.Unit, "kg")
	}
	if recorder.got.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", recorder.got.Market, "Plaza de Mercado")
	}
	if recorder.got.Source != "Boletín de precios al por mayor, DANE" {
		t.Errorf("the source is %q, want the place the price was read", recorder.got.Source)
	}
	if recorder.got.Day != "2026-03-01" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-01")
	}
	if recorder.got.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", recorder.got.ConfirmedWith, "sí")
	}

	answer := textOf(t, result)
	for _, want := range []string{"reference", "480", "kg", "DANE", "2026-03-01", "Plaza de Mercado"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheDescriptionOfTheReferenceToolSaysThatItIsAReference proves that the
// description of the tool is a contract with the agent: it says what the
// tool writes, and it says that a reference is never the price of the
// ingredient.
func TestTheDescriptionOfTheReferenceToolSaysThatItIsAReference(t *testing.T) {
	t.Parallel()

	session := connectWithTheReferences(t, &stubReferenceRecorder{}, nil)

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordReference)

	for _, want := range []string{
		"business is REQUIRED",
		"ingredient_id is REQUIRED",
		"price is REQUIRED",
		"unit is REQUIRED",
		"source is REQUIRED",
		"day is OPTIONAL",
		"market is OPTIONAL",
		"confirmed_with is REQUIRED",
		"never the price of the ingredient",
		"her own purchases",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheReadReferenceToolReadsTheReferences proves that the tool reads the
// references of the business, for one ingredient or for every one of them,
// and that the answer carries their source and their day.
func TestTheReadReferenceToolReadsTheReferences(t *testing.T) {
	t.Parallel()

	reader := &stubReferencesReader{read: []purchase.Reference{
		{
			ID:            purchase.ReferenceID(1),
			BusinessID:    business.ID(77),
			IngredientID:  ingredient.ID(10),
			Price:         480,
			Unit:          "kg",
			Market:        "Plaza de Mercado",
			Source:        "Boletín de precios al por mayor, DANE",
			SeenOn:        mustDayOfTheReference(t),
			ConfirmedWith: "sí",
		},
	}}
	session := connectWithTheReferences(t, nil, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolReadReferences,
		Arguments: map[string]any{"business_id": 77},
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
	if reader.got.IngredientID != 0 {
		t.Errorf("the ingredient is %d, want none", reader.got.IngredientID)
	}

	answer := textOf(t, result)
	for _, want := range []string{"reference", "480", "DANE", "2026-03-01"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheReadReferenceToolReadsOneIngredient proves that the tool reads the
// references of one ingredient when the agent names one.
func TestTheReadReferenceToolReadsOneIngredient(t *testing.T) {
	t.Parallel()

	reader := &stubReferencesReader{}
	session := connectWithTheReferences(t, nil, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadReferences,
		Arguments: map[string]any{
			"business_id":   77,
			"ingredient_id": 10,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if reader.got.IngredientID != 10 {
		t.Errorf("the ingredient is %d, want 10", reader.got.IngredientID)
	}
}

// TestTheErrorsOfTheReferencesReachTheAgentWithAMessage proves that every
// error the reference can return reaches the agent with a message of its
// own.
func TestTheErrorsOfTheReferencesReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	failures := []error{
		purchase.ErrMissingIngredient,
		purchase.ErrInvalidReferencePrice,
		purchase.ErrMissingReferenceUnit,
		purchase.ErrMissingSource,
		purchase.ErrMissingDay,
		purchase.ErrMissingConfirmation,
		purchase.ErrNotFound,
		purchase.ErrInvalidValue,
		purchase.ErrNoStorage,
	}

	for _, failure := range failures {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			recorder := &stubReferenceRecorder{err: failure}
			session := connectWithTheReferences(t, recorder, nil)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      mcpserver.ToolRecordReference,
				Arguments: theArgumentsOfTheReference(),
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

// TestTheReferenceToolWithoutADatabaseIsRefused proves that the tools are
// there even when the service has no database, and that they say so.
func TestTheReferenceToolWithoutADatabaseIsRefused(t *testing.T) {
	t.Parallel()

	session := connectWithTheReferences(t, nil, nil)

	for _, name := range []string{mcpserver.ToolRecordReference, mcpserver.ToolReadReferences} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			arguments := map[string]any{"business_id": 77}
			if name == mcpserver.ToolRecordReference {
				arguments = theArgumentsOfTheReference()
			}

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      name,
				Arguments: arguments,
			})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool answered no error, want the refusal of the service")
			}

			answer := textOf(t, result)
			if !strings.Contains(answer, "database") {
				t.Errorf("the answer does not say that the database is not there: %s", answer)
			}
		})
	}
}
