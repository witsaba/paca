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

// stubStockReader is the use case that reads the stock, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubStockReader struct {
	read  purchase.Stock
	err   error
	got   ports.ReadStock
	calls int
}

// Read answers the stock of the test, and remembers what the tool asked.
func (s *stubStockReader) Read(_ context.Context, request ports.ReadStock) (purchase.Stock, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return purchase.Stock{}, s.err
	}

	return s.read, nil
}

// connectWithTheStock starts the transport with the stub of the stock.
func connectWithTheStock(t *testing.T, reader ports.StockReader) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		ReadStock:           reader,
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

// theStockOfTheTest is the stock the test of the tool answers.
func theStockOfTheTest(t *testing.T) purchase.Stock {
	t.Helper()

	older, err := ledger.ParseDate("2026-02-28")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	return purchase.Stock{
		Holdings: []purchase.Holding{
			{
				IngredientID: ingredient.ID(10),
				Name:         "Harina panadera",
				Unit:         "kg",
				Quantity:     25,
				Cost:         ledger.Amount(12000),
				Layers: []purchase.Layer{
					{
						ID:           purchase.LayerID(1),
						BusinessID:   business.ID(77),
						IngredientID: ingredient.ID(10),
						Name:         "Harina panadera",
						PurchaseID:   purchase.ID(9),
						Day:          older,
						Bought:       purchase.Measure{Quantity: 25, Unit: "kg"},
						Left:         purchase.Measure{Quantity: 25, Unit: "kg"},
						Cost:         ledger.Amount(12000),
					},
				},
			},
		},
		Cost: ledger.Amount(12000),
	}
}

// TestTheReadStockToolReadsTheStock proves that the tool reads the stock of
// the business and answers with what it holds and what it cost.
func TestTheReadStockToolReadsTheStock(t *testing.T) {
	t.Parallel()

	reader := &stubStockReader{read: theStockOfTheTest(t)}
	session := connectWithTheStock(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadStock,
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
	if reader.got.IngredientID != 0 {
		t.Errorf("the ingredient is %d, want the whole stock of the business", reader.got.IngredientID)
	}

	answer := textOf(t, result)
	for _, want := range []string{"Harina panadera", "25", "kg", "12000", "2026-02-28"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheReadStockToolReadsOneIngredient proves that the tool reads the
// stock of one ingredient when the agent names it.
func TestTheReadStockToolReadsOneIngredient(t *testing.T) {
	t.Parallel()

	reader := &stubStockReader{read: theStockOfTheTest(t)}
	session := connectWithTheStock(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadStock,
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

// TestTheReadStockToolWithoutADatabaseSaysSo proves that the tool is there
// even when the service has no database, and that it says so.
func TestTheReadStockToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheStock(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadStock,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a stock, want the news that the database is not there")
	}
}

// TestTheDescriptionOfTheStockToolSaysThatTheValuationIsFIFO proves that
// the description of the tool is a contract with the agent: it says what
// the tool reads, and it says that the oldest layer leaves first.
func TestTheDescriptionOfTheStockToolSaysThatTheValuationIsFIFO(t *testing.T) {
	t.Parallel()

	session := connectWithTheStock(t, &stubStockReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadStock)

	for _, want := range []string{
		"business is REQUIRED",
		"ingredient is OPTIONAL",
		"oldest",
		"FIFO",
		"what it holds and what it cost",
		"changes nothing",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}
