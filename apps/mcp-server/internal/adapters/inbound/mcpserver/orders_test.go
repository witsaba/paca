package mcpserver_test

import (
	"context"
	"encoding/json"
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
	"github.com/witsaba/paca/packages/domain/order"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
)

// stubOrderRecorder is the use case that records an order, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubOrderRecorder struct {
	written order.Order
	err     error
	got     ports.RecordOrder
	calls   int
}

// Record answers the order of the test, and remembers what the tool asked.
func (s *stubOrderRecorder) Record(_ context.Context, request ports.RecordOrder) (order.Order, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return order.Order{}, s.err
	}

	return s.written, nil
}

// stubOrdersReader is the use case that reads the orders of a business, in
// memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubOrdersReader struct {
	held  order.Round
	err   error
	got   ports.ReadOrders
	calls int
}

// Read answers the round of the test, and remembers what the tool asked.
func (s *stubOrdersReader) Read(_ context.Context, request ports.ReadOrders) (order.Round, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return order.Round{}, s.err
	}

	return s.held, nil
}

// connectWithTheOrders starts the transport with the stubs of the orders.
func connectWithTheOrders(t *testing.T, recorder ports.OrderRecorder, reader ports.OrdersReader) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordOrder:         recorder,
		ReadOrders:          reader,
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

// theDayOfTheOrders is the day the order of the test is due on.
func theDayOfTheOrders(t *testing.T) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate("2026-03-14")
	if err != nil {
		t.Fatalf("the day of the test could not be read: %v", err)
	}

	return parsed
}

// theOrderOfTheTest is the order the stub of the use case answers.
func theOrderOfTheTest(t *testing.T) order.Order {
	t.Helper()

	return order.Order{
		ID:         100,
		BusinessID: business.ID(77),
		PartyID:    party.ID(10),
		PartyName:  "Doña Rosalba",
		DueOn:      theDayOfTheOrders(t),
		Lines: []order.Line{
			{ProductID: product.ID(5), ProductName: "Pan de molde", Quantity: 12, Unit: "pan"},
		},
		State: order.StateOpen,
	}
}

// theDeliveredOrderOfTheTest is the order that the round delivered.
func theDeliveredOrderOfTheTest(t *testing.T) order.Order {
	t.Helper()

	delivered := theOrderOfTheTest(t)
	delivered.ID = 101
	delivered.State = order.StateDelivered

	return delivered
}

// TestTheOrderToolCarriesWhatTheAgentSends proves that the tool carries
// what the agent sends to the use case, and that the answer carries the
// order with its lines.
func TestTheOrderToolCarriesWhatTheAgentSends(t *testing.T) {
	t.Parallel()

	recorder := &stubOrderRecorder{written: theOrderOfTheTest(t)}
	session := connectWithTheOrders(t, recorder, &stubOrdersReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordOrder,
		Arguments: map[string]any{
			"business_id": 77,
			"party_id":    10,
			"due_on":      "2026-03-14",
			"lines": []any{
				map[string]any{
					"product_id":   5,
					"product_name": "Pan de molde",
					"quantity":     12,
					"unit":         "pan",
				},
			},
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
	if recorder.got.PartyID != 10 {
		t.Errorf("the party is %d, want 10", recorder.got.PartyID)
	}
	if recorder.got.DueOn != "2026-03-14" {
		t.Errorf("the day it is due is %q, want %q", recorder.got.DueOn, "2026-03-14")
	}
	if len(recorder.got.Lines) != 1 {
		t.Fatalf("the call carries %d lines, want 1", len(recorder.got.Lines))
	}
	if recorder.got.Lines[0].ProductID != 5 {
		t.Errorf("the product of the line is %d, want 5", recorder.got.Lines[0].ProductID)
	}
	if recorder.got.Lines[0].ProductName != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", recorder.got.Lines[0].ProductName, "Pan de molde")
	}
	if recorder.got.Lines[0].Quantity != 12 {
		t.Errorf("the quantity is %d, want 12", recorder.got.Lines[0].Quantity)
	}
	if recorder.got.Lines[0].Unit != "pan" {
		t.Errorf("the unit is %q, want %q", recorder.got.Lines[0].Unit, "pan")
	}

	answer := textOf(t, result)
	for _, want := range []string{"Doña Rosalba", "Pan de molde", "12", "pan", "2026-03-14", "open"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheOrdersToolAnswersWhatIsStillOwed proves that the tool reads the
// round of the day and answers the orders it carries and the ones that are
// still owed.
func TestTheOrdersToolAnswersWhatIsStillOwed(t *testing.T) {
	t.Parallel()

	reader := &stubOrdersReader{held: order.Round{
		Day:    theDayOfTheOrders(t),
		Orders: []order.Order{theOrderOfTheTest(t), theDeliveredOrderOfTheTest(t)},
	}}
	session := connectWithTheOrders(t, &stubOrderRecorder{}, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadOrders,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-14",
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
	if reader.got.Day != "2026-03-14" {
		t.Errorf("the day is %q, want %q", reader.got.Day, "2026-03-14")
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	answer := string(raw)
	if !strings.Contains(answer, `"owed"`) {
		t.Errorf("the answer does not carry the orders that are still owed: %s", answer)
	}
	if !strings.Contains(answer, `"orders"`) {
		t.Errorf("the answer does not carry the orders of the round: %s", answer)
	}
}

// TestTheDescriptionOfTheOrderEnumeratesWhatTheAnswerCarries proves that
// the description of the tool that records an order says exactly what the
// answer carries.
//
// The fields are read from the answer itself: a field added to the answer
// and not to the description breaks this test, and the description never
// promises a field the answer does not carry.
func TestTheDescriptionOfTheOrderEnumeratesWhatTheAnswerCarries(t *testing.T) {
	t.Parallel()

	recorder := &stubOrderRecorder{written: theOrderOfTheTest(t)}
	session := connectWithTheOrders(t, recorder, &stubOrdersReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordOrder,
		Arguments: map[string]any{
			"business_id": 77,
			"party_id":    10,
			"due_on":      "2026-03-14",
			"lines": []any{
				map[string]any{
					"product_id": 5,
					"quantity":   12,
					"unit":       "pan",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordOrder)

	for _, field := range theFieldsOfTheAnswer(t, result, "order") {
		if !strings.Contains(description, "'"+field+"'") {
			t.Errorf("the description does not name the field %q of the answer:\n%s", field, description)
		}
	}

	for _, field := range theFieldsOfTheAnswer(t, result, "order", "lines", "0") {
		if !strings.Contains(description, "'"+field+"'") {
			t.Errorf("the description does not name the field %q of the answer:\n%s", field, description)
		}
	}
}

// TestTheDescriptionOfTheOrdersEnumeratesWhatTheAnswerCarries proves that
// the description of the tool that reads the orders says exactly what the
// answer carries.
func TestTheDescriptionOfTheOrdersEnumeratesWhatTheAnswerCarries(t *testing.T) {
	t.Parallel()

	reader := &stubOrdersReader{held: order.Round{
		Day:    theDayOfTheOrders(t),
		Orders: []order.Order{theOrderOfTheTest(t)},
	}}
	session := connectWithTheOrders(t, &stubOrderRecorder{}, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadOrders,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-14",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadOrders)

	for _, field := range theFieldsOfTheAnswer(t, result) {
		if !strings.Contains(description, "'"+field+"'") {
			t.Errorf("the description does not name the field %q of the answer:\n%s", field, description)
		}
	}

	for _, path := range [][]string{{"orders", "0"}, {"owed", "0"}} {
		for _, field := range theFieldsOfTheAnswer(t, result, path...) {
			if !strings.Contains(description, "'"+field+"'") {
				t.Errorf("the description does not name the field %q of the answer:\n%s", field, description)
			}
		}
	}
}

// TestTheDescriptionOfTheOrderTellsTheTruth proves that the description of
// the tool says what the tool does: the party that ordered, the products
// with their quantities, the day it is due and the state of the order.
func TestTheDescriptionOfTheOrderTellsTheTruth(t *testing.T) {
	t.Parallel()

	session := connectWithTheOrders(t, &stubOrderRecorder{}, &stubOrdersReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordOrder)

	for _, want := range []string{
		"the day it is due",
		"written OPEN",
		"an order that was not delivered stays open",
		"delivery of an order is written with the route of the round",
		"whole number of units",
		"never splits a unit",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheDescriptionOfTheOrdersSaysThatItChangesNothing proves that the
// description of the tool that reads the orders says that it writes
// nothing, and that it names the states of the book.
func TestTheDescriptionOfTheOrdersSaysThatItChangesNothing(t *testing.T) {
	t.Parallel()

	session := connectWithTheOrders(t, &stubOrderRecorder{}, &stubOrdersReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadOrders)

	for _, want := range []string{"It writes nothing.", "open, delivered", "what is still owed"} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheSchemaOfTheOrderToolsRefusesNothingBeforeTheDomain proves that
// the schema of the tools of the orders does not refuse the call before the
// domain can answer it: a field that the domain refuses is not marked as
// required.
func TestTheSchemaOfTheOrderToolsRefusesNothingBeforeTheDomain(t *testing.T) {
	t.Parallel()

	session := connectWithTheOrders(t, &stubOrderRecorder{}, &stubOrdersReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	for _, name := range []string{mcpserver.ToolRecordOrder, mcpserver.ToolReadOrders} {
		required := theRequiredFieldsOfTheTool(t, tools.Tools, name)

		for _, field := range []string{"business_id", "party_id", "party_name", "due_on", "lines", "product_id", "quantity", "unit", "day"} {
			for _, one := range required {
				if one == field {
					t.Errorf("the schema of the tool %q marks %q as required: the schema must not refuse the "+
						"call before the domain can answer it", name, field)
				}
			}
		}
	}
}

// TestTheOrderToolsExplainTheFailureOfTheUseCase proves that the tools
// answer the error of the use case with the message the agent reads.
func TestTheOrderToolsExplainTheFailureOfTheUseCase(t *testing.T) {
	t.Parallel()

	recorder := &stubOrderRecorder{err: order.ErrNotAClient}
	reader := &stubOrdersReader{err: order.ErrMissingDay}
	session := connectWithTheOrders(t, recorder, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordOrder,
		Arguments: map[string]any{
			"business_id": 77,
			"party_id":    10,
			"due_on":      "2026-03-14",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	answer := textOf(t, result)
	if !strings.Contains(answer, "the role of the client") {
		t.Errorf("the answer does not carry the refusal of the domain: %s", answer)
	}

	result, err = session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadOrders,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	answer = textOf(t, result)
	if !strings.Contains(answer, "the order carries no day") {
		t.Errorf("the answer does not carry the refusal of the domain: %s", answer)
	}
}

// TestTheOrderToolsWithoutADatabaseSaySo proves that the tools are there
// even when the service has no database, and that they say so.
func TestTheOrderToolsWithoutADatabaseSaySo(t *testing.T) {
	t.Parallel()

	session := connectWithTheOrders(t, nil, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordOrder,
		Arguments: map[string]any{
			"business_id": 77,
			"party_id":    10,
			"due_on":      "2026-03-14",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	answer := textOf(t, result)
	if !strings.Contains(answer, "the service has no database configured") {
		t.Errorf("the answer does not say that the database is not there: %s", answer)
	}
}
