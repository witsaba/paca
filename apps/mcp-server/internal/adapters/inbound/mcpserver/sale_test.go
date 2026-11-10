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

// stubSaleRecorder is the use case that records the sale, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubSaleRecorder struct {
	answer recipe.Sale
	err    error
	got    ports.RecordSale
	calls  int
}

// Record answers the sale of the test, and remembers what the tool asked.
func (s *stubSaleRecorder) Record(_ context.Context, request ports.RecordSale) (recipe.Sale, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Sale{}, s.err
	}

	return s.answer, nil
}

// connectWithTheSale starts the transport with the stub of the sale.
func connectWithTheSale(t *testing.T, recorder ports.SaleRecorder) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordSale:          recorder,
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

// theSaleOfTheTool is the sale the tool of the test answers: three units of
// bread for five thousand pesos, worth three thousand pesos.
func theSaleOfTheTool(t *testing.T) recipe.Sale {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-14")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	return recipe.Sale{
		ID:         recipe.SaleID(5),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(9),
		RecipeName: "Pan de molde",
		Day:        day,
		Quantity:   3,
		Unit:       "unidad",
		Money:      5000,
		Cost:       3000,
		Units: []recipe.SoldUnit{
			{
				UnitID:       recipe.UnitID(1),
				ProductionID: recipe.ProductionID(3),
				Position:     1,
				Grams:        90,
				ExpiresOn:    ledger.NewDate(day.Time().AddDate(0, 0, 3)),
			},
		},
		Utterance:     "vendí tres panes",
		ConfirmedWith: "sí",
	}
}

// TestTheRecordSaleToolWritesTheProductTheQuantityAndTheMoney proves that
// the tool writes the sale the owner confirmed: the product, the quantity,
// the unit, the money the client paid and her word.
func TestTheRecordSaleToolWritesTheProductTheQuantityAndTheMoney(t *testing.T) {
	t.Parallel()

	recorder := &stubSaleRecorder{answer: theSaleOfTheTool(t)}
	session := connectWithTheSale(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordSale,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"quantity":       3,
			"unit":           "unidad",
			"money":          5000,
			"day":            "2026-03-14",
			"utterance":      "vendí tres panes",
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
	if recorder.got.Money != 5000 {
		t.Errorf("the money is %d, want 5000", recorder.got.Money)
	}
	if recorder.got.Day != "2026-03-14" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-14")
	}
	if recorder.got.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", recorder.got.ConfirmedWith, "sí")
	}

	answer := textOf(t, result)
	for _, want := range []string{"Pan de molde", "unidad", "5000", "3000"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheRecordSaleToolRefusesTheBusinessThatIsNotThere proves that the
// business is required: the book never invents one.
//
// The call reaches the use case and the domain is the one that refuses it:
// the schema of the tool does not refuse the call before the domain can
// answer it.
func TestTheRecordSaleToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubSaleRecorder{err: recipe.ErrMissingBusiness}
	session := connectWithTheSale(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordSale,
		Arguments: map[string]any{
			"recipe_name":    "Pan de molde",
			"quantity":       3,
			"money":          5000,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a sale, want the refusal of the business that is not there")
	}
	if recorder.calls != 1 {
		t.Errorf("the use case was called %d times, want 1: the call has to reach the use case",
			recorder.calls)
	}
	if recorder.got.BusinessID != business.ID(0) {
		t.Errorf("the business of the call is %d, want the zero of the field that was left out",
			recorder.got.BusinessID)
	}

	answer := textOf(t, result)
	if strings.Contains(answer, "validating") {
		t.Errorf("the schema of the tool refused the call before the use case:\n%s", answer)
	}
	if !strings.Contains(answer, "the business is required") {
		t.Errorf("the answer does not carry the refusal of the domain:\n%s", answer)
	}
}

// TestTheRecordSaleToolRefusesTheMoneyThatIsNotThere proves that the money
// is required: the money of a sale is never inferred.
//
// The call reaches the use case and the domain is the one that refuses it:
// the schema of the tool does not refuse the call before the domain can
// answer it.
func TestTheRecordSaleToolRefusesTheMoneyThatIsNotThere(t *testing.T) {
	t.Parallel()

	recorder := &stubSaleRecorder{err: recipe.ErrInvalidSaleMoney}
	session := connectWithTheSale(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordSale,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"quantity":       3,
			"money":          0,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a sale, want the refusal of the money that is not there")
	}
	if recorder.calls != 1 {
		t.Errorf("the use case was called %d times, want 1: the call has to reach the use case",
			recorder.calls)
	}
	if recorder.got.Money != 0 {
		t.Errorf("the money of the call is %d, want the zero of the field that was left out",
			recorder.got.Money)
	}

	answer := textOf(t, result)
	if strings.Contains(answer, "validating") {
		t.Errorf("the schema of the tool refused the call before the use case:\n%s", answer)
	}
	if !strings.Contains(answer, "the money of the sale is not a positive number of pesos") {
		t.Errorf("the answer does not carry the refusal of the domain:\n%s", answer)
	}
}

// TestTheSchemaOfTheRecordSaleToolDoesNotRefuseTheCall proves that the
// schema of the tool does not refuse the call before the domain can answer
// it: the business, the quantity, the money and the word of the owner are
// optional in the call, and the domain is the one that refuses the sale
// that carries none of them.
func TestTheSchemaOfTheRecordSaleToolDoesNotRefuseTheCall(t *testing.T) {
	t.Parallel()

	session := connectWithTheSale(t, &stubSaleRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	required := theRequiredFieldsOfTheTool(t, tools.Tools, mcpserver.ToolRecordSale)

	for _, field := range []string{"business_id", "quantity", "money", "confirmed_with"} {
		for _, one := range required {
			if one == field {
				t.Errorf("the schema of the tool marks %q as required: the schema must not refuse the "+
					"call before the domain can answer it", field)
			}
		}
	}
}

// TestTheRecordSaleToolWithoutADatabaseSaysSo proves that the tool is there
// even when the service has no database, and that it says so.
func TestTheRecordSaleToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheSale(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordSale,
		Arguments: map[string]any{
			"business_id":    77,
			"recipe_name":    "Pan de molde",
			"quantity":       3,
			"money":          5000,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a sale, want the answer that the database is not there")
	}
}

// TestTheDescriptionOfTheSaleTellsTheAgentTheRules proves that the
// description of the tool names the rules the agent has to know: the money
// the client paid, the rotation by the expiry and the refusal of the
// expired unit.
func TestTheDescriptionOfTheSaleTellsTheAgentTheRules(t *testing.T) {
	t.Parallel()

	session := connectWithTheSale(t, &stubSaleRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordSale)

	wants := []string{
		"MONEY",
		"EXPIRE FIRST",
		"never sold",
		"whole pesos",
	}
	for _, want := range wants {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not tell the agent %q:\n%s", want, description)
		}
	}
}

// TestTheErrorsOfTheSaleReachTheAgentWithAMessage proves that every error
// the sale can return reaches the agent with a message of its own.
func TestTheErrorsOfTheSaleReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	for _, failure := range recipe.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			recorder := &stubSaleRecorder{err: failure}
			session := connectWithTheSale(t, recorder)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolRecordSale,
				Arguments: map[string]any{
					"business_id":    77,
					"recipe_name":    "Pan de molde",
					"quantity":       3,
					"money":          5000,
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
