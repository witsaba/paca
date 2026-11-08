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

// stubRealPriceReader is the use case that reads the report of the real
// price, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubRealPriceReader struct {
	reports []recipe.RealPrice
	err     error
	got     ports.ReadRealPrice
	calls   int
}

// Read answers the report of the test, and remembers what the tool asked.
func (s *stubRealPriceReader) Read(_ context.Context, request ports.ReadRealPrice) ([]recipe.RealPrice, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return nil, s.err
	}

	return s.reports, nil
}

// connectWithTheRealPrice starts the transport with the stub of the report.
func connectWithTheRealPrice(t *testing.T, reader ports.RealPriceReader) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		ReadRealPrice:       reader,
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

// theReportOfTheTest is the report the tool of the test answers: the three
// layers of the cost apart, and the price of the owner beside them.
func theReportOfTheTest(t *testing.T) []recipe.RealPrice {
	t.Helper()

	return []recipe.RealPrice{
		{
			RecipeID:   recipe.ID(9),
			Name:       "Pan de molde",
			Yield:      recipe.Yield{Units: 10, Grams: 90},
			Direct:     recipe.Rate{Money: ledger.Amount(240), PerUnit: 24, PerGram: 0.27, PerKilogram: 266.67},
			Labor:      recipe.Rate{Money: ledger.Amount(25000), PerUnit: 2500},
			Overhead:   recipe.Rate{Money: ledger.Amount(1125), PerUnit: 112.5},
			Total:      recipe.Rate{Money: ledger.Amount(26365), PerUnit: 2636.5},
			Price:      recipe.Rate{Money: ledger.Amount(8000), PerUnit: 8000, PerGram: 88.89},
			PriceBasis: purchase.PricePerUnit,
			Complete:   true,
			Missing:    []string{},
		},
	}
}

// TestTheReadRealPriceToolAnswersTheReport proves that the tool reads the
// report of the real price and answers with the three layers of the cost
// apart and the price of the owner beside them.
func TestTheReadRealPriceToolAnswersTheReport(t *testing.T) {
	t.Parallel()

	reader := &stubRealPriceReader{reports: theReportOfTheTest(t)}
	session := connectWithTheRealPrice(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadRealPrice,
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

	answer := textOf(t, result)
	for _, want := range []string{
		"Pan de molde",
		"direct",
		"labor",
		"overhead",
		"total",
		"price",
		"240",
		"25000",
		"1125",
		"8000",
		"per_unit",
		"per_gram",
		"per_kilogram",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// theReportAnswer is the answer of the tool of the real price, decoded.
//
// Every money of the report travels under its own key: the answer is a
// document, and the test binds every money to the layer it belongs to.
type theReportAnswer struct {
	Reports []struct {
		RecipeID   int64                 `json:"recipe_id"`
		Name       string                `json:"name"`
		Direct     struct{ Money int64 } `json:"direct"`
		Labor      struct{ Money int64 } `json:"labor"`
		Overhead   struct{ Money int64 } `json:"overhead"`
		Total      struct{ Money int64 } `json:"total"`
		Price      struct{ Money int64 } `json:"price"`
		PriceBasis string                `json:"price_basis"`
	} `json:"reports"`
}

// TestTheReadRealPriceToolBindsEveryMoneyToItsLayer proves that every
// money of the answer travels under the key of its own layer: the direct
// cost, the labor, the overhead, the total and the price of the owner are
// five money facts of their own, and the price carries the way she charges
// it by.
//
// The answer is decoded and not read as text: a money that travels under
// the key of another one cannot hide in a document.
func TestTheReadRealPriceToolBindsEveryMoneyToItsLayer(t *testing.T) {
	t.Parallel()

	reader := &stubRealPriceReader{reports: theReportOfTheTest(t)}
	session := connectWithTheRealPrice(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadRealPrice,
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

	var answer theReportAnswer

	jsonOf(t, result, &answer)

	if len(answer.Reports) != 1 {
		t.Fatalf("the answer carries %d reports, want 1", len(answer.Reports))
	}

	report := answer.Reports[0]

	if report.Direct.Money != 240 {
		t.Errorf("the direct cost is %d, want 240", report.Direct.Money)
	}
	if report.Labor.Money != 25000 {
		t.Errorf("the labor is %d, want 25000", report.Labor.Money)
	}
	if report.Overhead.Money != 1125 {
		t.Errorf("the overhead is %d, want 1125", report.Overhead.Money)
	}
	if report.Total.Money != 26365 {
		t.Errorf("the total is %d, want 26365", report.Total.Money)
	}
	if report.Price.Money != 8000 {
		t.Errorf("the price of the owner is %d, want 8000", report.Price.Money)
	}
	if report.PriceBasis != "unit" {
		t.Errorf("the way the owner charges is %q, want %q", report.PriceBasis, "unit")
	}
}

// TestTheReadRealPriceToolCarriesThePriceOfTheOwner proves that the tool
// carries the price the owner declared and the way she charges it: the
// price of the owner is hers, and PACA never infers it.
func TestTheReadRealPriceToolCarriesThePriceOfTheOwner(t *testing.T) {
	t.Parallel()

	reader := &stubRealPriceReader{reports: theReportOfTheTest(t)}
	session := connectWithTheRealPrice(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadRealPrice,
		Arguments: map[string]any{
			"business_id": 77,
			"recipe_id":   9,
			"price":       8000,
			"price_basis": "unidad",
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
	if reader.got.Price != 8000 {
		t.Errorf("the price of the owner is %d, want 8000", reader.got.Price)
	}
	if reader.got.PriceBasis != "unidad" {
		t.Errorf("the way she charges is %q, want %q", reader.got.PriceBasis, "unidad")
	}
}

// TestTheReadRealPriceToolRefusesTheBusinessThatIsNotThere proves that
// the business is required: the tool refuses the business that is not
// there, and it never asks the use case for the report of nobody.
func TestTheReadRealPriceToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	reader := &stubRealPriceReader{reports: theReportOfTheTest(t)}
	session := connectWithTheRealPrice(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadRealPrice,
		Arguments: map[string]any{
			"business_id": 0,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a report, want the refusal of the business that is not there")
	}
	if reader.calls != 0 {
		t.Errorf("the use case was called %d times, want none", reader.calls)
	}
}

// TestTheReadRealPriceToolWithoutADatabaseSaysSo proves that the tool is
// there even when the service has no database, and that it says so.
func TestTheReadRealPriceToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheRealPrice(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadRealPrice,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a report, want the news that the database is not there")
	}
}

// TestTheDescriptionOfTheRealPriceToolSaysThatTheLayersTravelApart proves
// that the description of the tool is a contract with the agent: it says
// that the three layers of the cost travel apart, and it says that the
// price of the owner is hers.
func TestTheDescriptionOfTheRealPriceToolSaysThatTheLayersTravelApart(t *testing.T) {
	t.Parallel()

	session := connectWithTheRealPrice(t, &stubRealPriceReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadRealPrice)

	for _, want := range []string{
		"business is REQUIRED",
		"recipe is OPTIONAL",
		"price is OPTIONAL",
		"price_basis is REQUIRED",
		"APART",
		"per unit",
		"per gram",
		"per kilogram",
		"never inferred",
		"changes nothing",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}
