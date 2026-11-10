package mcpserver_test

import (
	"context"
	"encoding/json"
	"errors"
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
)

// stubRouteRecorder is the use case that records the route, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubRouteRecorder struct {
	written order.Route
	err     error
	got     ports.RecordRoute
	calls   int
}

// Record answers the route of the test, and remembers what the tool asked.
func (s *stubRouteRecorder) Record(_ context.Context, request ports.RecordRoute) (order.Route, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return order.Route{}, s.err
	}

	return s.written, nil
}

// connectWithTheRoute starts the transport with the stub of the route.
func connectWithTheRoute(t *testing.T, recorder ports.RouteRecorder) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordRoute:         recorder,
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

// theRouteOfTheTest is the route the stub of the use case answers.
func theRouteOfTheTest(t *testing.T) order.Route {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-14")
	if err != nil {
		t.Fatalf("the day of the test could not be read: %v", err)
	}

	return order.Route{
		ID:         order.RouteID(7),
		BusinessID: business.ID(77),
		Day:        day,
		Stops: []order.Stop{
			{OrderID: order.ID(10), Cost: ledger.Amount(3000)},
			{OrderID: order.ID(11), Cost: ledger.Amount(1500)},
		},
	}
}

// TestTheRouteToolCarriesWhatTheAgentSends proves that the tool carries
// what the agent sends to the use case, and that the answer carries the
// route with its stops and the total PACA derives from them.
func TestTheRouteToolCarriesWhatTheAgentSends(t *testing.T) {
	t.Parallel()

	recorder := &stubRouteRecorder{written: theRouteOfTheTest(t)}
	session := connectWithTheRoute(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordRoute,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-14",
			"stops": []any{
				map[string]any{"order_id": 10, "cost": 3000},
				map[string]any{"order_id": 11, "cost": 1500},
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
	if recorder.got.Day != "2026-03-14" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-14")
	}
	if len(recorder.got.Stops) != 2 {
		t.Fatalf("the call carries %d stops, want 2", len(recorder.got.Stops))
	}
	if recorder.got.Stops[0].OrderID != 10 {
		t.Errorf("the order of the first stop is %d, want 10", recorder.got.Stops[0].OrderID)
	}
	if recorder.got.Stops[0].Cost != 3000 {
		t.Errorf("the cost of the first stop is %d, want 3000", recorder.got.Stops[0].Cost)
	}

	answer := textOf(t, result)
	for _, want := range []string{"2026-03-14", "4500", "3000", "1500"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheDescriptionOfTheRouteEnumeratesWhatTheAnswerCarries proves that
// the description of the tool says exactly what the answer carries.
//
// The fields are read from the answer itself: a field added to the answer
// and not to the description breaks this test, and the description never
// promises a field the answer does not carry.
func TestTheDescriptionOfTheRouteEnumeratesWhatTheAnswerCarries(t *testing.T) {
	t.Parallel()

	recorder := &stubRouteRecorder{written: theRouteOfTheTest(t)}
	session := connectWithTheRoute(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordRoute,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-14",
			"stops": []any{
				map[string]any{"order_id": 10, "cost": 3000},
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

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordRoute)

	for _, field := range theFieldsOfTheAnswer(t, result, "route") {
		if !strings.Contains(description, "'"+field+"'") {
			t.Errorf("the description does not name the field %q of the answer:\n%s", field, description)
		}
	}

	for _, field := range theFieldsOfTheAnswer(t, result, "route", "stops", "0") {
		if !strings.Contains(description, "'"+field+"'") {
			t.Errorf("the description does not name the field %q of the answer:\n%s", field, description)
		}
	}
}

// TestTheDescriptionOfTheRouteTellsTheTruth proves that the description
// of the tool says what the tool does: the route is computed by the agent
// and it is written back into the round, and writing it delivers the orders
// of its stops.
func TestTheDescriptionOfTheRouteTellsTheTruth(t *testing.T) {
	t.Parallel()

	session := connectWithTheRoute(t, &stubRouteRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordRoute)

	for _, want := range []string{
		"the order you will visit the clients in",
		"written back INTO the round of that day",
		"DELIVERS the orders of its stops",
		"WHOLE PESOS",
		"never negative",
		"written ONCE",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheSchemaOfTheRouteToolRefusesNothingBeforeTheDomain proves that
// the schema of the tool does not refuse the call before the domain can
// answer it: a field that the domain refuses is not marked as required.
func TestTheSchemaOfTheRouteToolRefusesNothingBeforeTheDomain(t *testing.T) {
	t.Parallel()

	session := connectWithTheRoute(t, &stubRouteRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	required := theRequiredFieldsOfTheTool(t, tools.Tools, mcpserver.ToolRecordRoute)

	for _, field := range []string{"business_id", "day", "stops", "order_id", "cost"} {
		for _, one := range required {
			if one == field {
				t.Errorf("the schema marks %q as required: the schema must not refuse the call before the "+
					"domain can answer it", field)
			}
		}
	}
}

// TestTheRouteToolExplainsTheFailureOfTheUseCase proves that the tool
// answers the error of the use case with the message the agent reads.
func TestTheRouteToolExplainsTheFailureOfTheUseCase(t *testing.T) {
	t.Parallel()

	recorder := &stubRouteRecorder{err: order.ErrRouteAlreadyWritten}
	session := connectWithTheRoute(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordRoute,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-14",
			"stops": []any{
				map[string]any{"order_id": 10, "cost": 3000},
			},
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	answer := textOf(t, result)
	if !strings.Contains(answer, "already written") {
		t.Errorf("the answer does not carry the refusal of the domain: %s", answer)
	}
}

// TestTheRouteToolWithoutADatabaseSaysSo proves that the tool is there
// even when the service has no database, and that it says so.
func TestTheRouteToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheRoute(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordRoute,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-14",
			"stops": []any{
				map[string]any{"order_id": 10, "cost": 3000},
			},
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

// TestTheRoundOfTheDayCarriesTheRouteItWasPlannedWith proves that the
// reading of the round carries the route of the day, and that a round
// without a route says that it was never planned.
func TestTheRoundOfTheDayCarriesTheRouteItWasPlannedWith(t *testing.T) {
	t.Parallel()

	reader := &stubOrdersReader{held: order.Round{
		Day:    theDayOfTheOrders(t),
		Orders: []order.Order{theOrderOfTheTest(t)},
		Owed:   []order.Order{theOrderOfTheTest(t)},
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

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	answer := string(raw)
	if !strings.Contains(answer, `"planned":false`) {
		t.Errorf("the answer does not say that the round was never planned: %s", answer)
	}
	if strings.Contains(answer, `"route"`) {
		t.Errorf("the answer of a round that was never planned carries a route: %s", answer)
	}
}

// TestTheRoundThatWasPlannedCarriesTheRouteOfTheDay proves that the
// reading of the round carries the route that was written for the day, with
// its stops and the total PACA derives from them.
func TestTheRoundThatWasPlannedCarriesTheRouteOfTheDay(t *testing.T) {
	t.Parallel()

	route := theRouteOfTheTest(t)

	reader := &stubOrdersReader{held: order.Round{
		Day:    theDayOfTheOrders(t),
		Orders: []order.Order{theOrderOfTheTest(t)},
		Owed:   []order.Order{theOrderOfTheTest(t)},
		Route:  &route,
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

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	answer := string(raw)
	if !strings.Contains(answer, `"planned":true`) {
		t.Errorf("the answer does not say that the round was planned: %s", answer)
	}
	if !strings.Contains(answer, `"route"`) {
		t.Errorf("the answer does not carry the route of the day: %s", answer)
	}
	for _, want := range []string{`"total":4500`, `"order_id":10`, `"cost":3000`} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %s: %s", want, answer)
		}
	}
}

// TestTheDescriptionOfTheOrdersEnumeratesTheRouteOfTheDay proves that
// the description of the tool that reads the orders says what the answer
// carries about the route of the day.
func TestTheDescriptionOfTheOrdersEnumeratesTheRouteOfTheDay(t *testing.T) {
	t.Parallel()

	session := connectWithTheOrders(t, &stubOrderRecorder{}, &stubOrdersReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadOrders)

	for _, want := range []string{"'planned'", "'route'", "'total'", "'stops'", "'order_id'", "'cost'"} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not name %s:\n%s", want, description)
		}
	}
}

// TestTheRouteToolCarriesTheErrorOfTheDomain proves that the errors of
// the domain of the route reach the agent with a message of their own.
func TestTheRouteToolCarriesTheErrorOfTheDomain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		failure error
		want    string
	}{
		{
			name:    "the route of a day that was already written",
			failure: order.ErrRouteAlreadyWritten,
			want:    "already written",
		},
		{
			name:    "the stop that is not an order of the round",
			failure: order.ErrStopNotOfTheRound,
			want:    "not one of the orders of that round",
		},
		{
			name:    "the cost that is not a number of pesos",
			failure: order.ErrInvalidCost,
			want:    "not a number of pesos",
		},
	}

	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			t.Parallel()

			session := connectWithTheRoute(t, &stubRouteRecorder{err: one.failure})

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolRecordRoute,
				Arguments: map[string]any{
					"business_id": 77,
					"day":         "2026-03-14",
					"stops": []any{
						map[string]any{"order_id": 10, "cost": 3000},
					},
				},
			})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool did not answer an error")
			}

			answer := textOf(t, result)
			if !strings.Contains(answer, one.want) {
				t.Errorf("the answer does not carry the message of the domain: %s", answer)
			}
		})
	}
}

// TestTheRouteToolAnswersTheUnknownError proves that an error the service
// does not know reaches the agent with the message of the service.
func TestTheRouteToolAnswersTheUnknownError(t *testing.T) {
	t.Parallel()

	session := connectWithTheRoute(t, &stubRouteRecorder{err: errors.New("a failure the service does not know")})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordRoute,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-14",
			"stops": []any{
				map[string]any{"order_id": 10, "cost": 3000},
			},
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	answer := textOf(t, result)
	if !strings.Contains(answer, "the tool failed for a reason the service does not know") {
		t.Errorf("the answer does not carry the failure of the service: %s", answer)
	}
}
