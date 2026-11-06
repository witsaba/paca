package mcpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// timeOfTheTest is the day the reference of the market was seen.
func timeOfTheTest() time.Time {
	return time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
}

// stubMarginReader is the use case that reads the margin, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubMarginReader struct {
	margins []recipe.Margin
	err     error
	got     ports.ReadMargin
	calls   int
}

// Read answers the margin of the test, and remembers what the tool asked.
func (s *stubMarginReader) Read(_ context.Context, request ports.ReadMargin) ([]recipe.Margin, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return nil, s.err
	}

	return s.margins, nil
}

// connectWithTheMargin starts the transport with the stub of the margin.
func connectWithTheMargin(t *testing.T, reader ports.MarginReader) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		ReadMargin:          reader,
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

// theMarginOfTheTest is the margin the tool of the test answers: the
// price of the owner, the direct cost of the batch, the profit and what
// the market said.
func theMarginOfTheTest(t *testing.T) []recipe.Margin {
	t.Helper()

	return []recipe.Margin{
		{
			RecipeID:   recipe.ID(9),
			Name:       "Pan de molde",
			Yield:      recipe.Yield{Units: 10, Grams: 90},
			Price:      recipe.Rate{Money: ledger.Amount(8000), PerUnit: 8000, PerGram: 88.89},
			PriceBasis: purchase.PricePerUnit,
			Direct:     recipe.Rate{Money: ledger.Amount(240), PerUnit: 24},
			Profit:     recipe.Profit{PerUnit: 7976, PerGram: 88.62},
			Percent:    33233.33,
			Market: recipe.Market{
				Money:      ledger.Amount(250),
				Paid:       ledger.Amount(240),
				Difference: ledger.Amount(-10),
				PerUnit:    25,
				Lines: []recipe.MarketLine{
					{
						IngredientID: ingredient.ID(10),
						Name:         "Harina panadera",
						Grams:        500,
						Paid:         ledger.Amount(240),
						Money:        ledger.Amount(250),
						Reference: recipe.ReferenceView{
							ID:           purchase.ReferenceID(3),
							IngredientID: ingredient.ID(10),
							Price:        ledger.Amount(5000),
							Unit:         "kg",
							Market:       "Plaza de Mercado",
							Source:       "Boletín de precios al por mayor, DANE",
							SeenOn:       ledger.NewDate(timeOfTheTest()),
							AgeDays:      10,
							Old:          true,
						},
						References: []recipe.ReferenceView{
							{
								ID:           purchase.ReferenceID(3),
								IngredientID: ingredient.ID(10),
								Price:        ledger.Amount(5000),
								Unit:         "kg",
								Market:       "Plaza de Mercado",
								Source:       "Boletín de precios al por mayor, DANE",
								SeenOn:       ledger.NewDate(timeOfTheTest()),
								AgeDays:      10,
								Old:          true,
							},
						},
					},
				},
			},
			MarketProfit: recipe.Profit{PerUnit: 7975},
			Missing:      []string{},
		},
	}
}

// TestTheReadMarginToolAnswersTheMargin proves that the tool reads the
// margin and answers with the price of the owner, the direct cost of the
// batch, the profit and what the market said.
func TestTheReadMarginToolAnswersTheMargin(t *testing.T) {
	t.Parallel()

	reader := &stubMarginReader{margins: theMarginOfTheTest(t)}
	session := connectWithTheMargin(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadMargin,
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
		"price",
		"direct",
		"profit",
		"market",
		"market_profit",
		"8000",
		"240",
		"7976",
		"Harina panadera",
		"Boletín de precios al por mayor, DANE",
		"age_days",
		"per_unit",
		"per_gram",
		"per_kilogram",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheReadMarginToolCarriesThePriceAndTheExpectationOfTheAge proves
// that the tool carries the price the owner declared, the way she charges
// and how old a reference may be.
func TestTheReadMarginToolCarriesThePriceAndTheExpectationOfTheAge(t *testing.T) {
	t.Parallel()

	reader := &stubMarginReader{margins: theMarginOfTheTest(t)}
	session := connectWithTheMargin(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadMargin,
		Arguments: map[string]any{
			"business_id":  77,
			"recipe_id":    9,
			"price":        8000,
			"price_basis":  "unidad",
			"max_age_days": 7,
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
	if reader.got.MaxAgeDays != 7 {
		t.Errorf("the expectation of the age is %d days, want 7", reader.got.MaxAgeDays)
	}
}

// TestTheReadMarginToolRefusesTheBusinessThatIsNotThere proves that the
// business is required: the tool refuses the business that is not there,
// and it never asks the use case for the margin of nobody.
func TestTheReadMarginToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	reader := &stubMarginReader{margins: theMarginOfTheTest(t)}
	session := connectWithTheMargin(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadMargin,
		Arguments: map[string]any{
			"business_id": 0,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a margin, want the refusal of the business that is not there")
	}
	if reader.calls != 0 {
		t.Errorf("the use case was called %d times, want none", reader.calls)
	}
}

// TestTheReadMarginToolWithoutADatabaseSaysSo proves that the tool is
// there even when the service has no database, and that it says so.
func TestTheReadMarginToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheMargin(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadMargin,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a margin, want the news that the database is not there")
	}
}

// TestTheDescriptionOfTheMarginToolSaysThatTheReferenceIsAReference proves
// that the description of the tool is a contract with the agent: it says
// that the reference is a reference and never the price of the ingredient,
// and it says that the age of the reference is shown.
func TestTheDescriptionOfTheMarginToolSaysThatTheReferenceIsAReference(t *testing.T) {
	t.Parallel()

	session := connectWithTheMargin(t, &stubMarginReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadMargin)

	for _, want := range []string{
		"business is REQUIRED",
		"recipe is OPTIONAL",
		"price is OPTIONAL",
		"price_basis is REQUIRED",
		"max_age_days",
		"REFERENCE",
		"never the price of the ingredient",
		"its age",
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
