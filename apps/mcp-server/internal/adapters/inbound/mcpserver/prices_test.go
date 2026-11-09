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
	"github.com/witsaba/paca/packages/domain/product"
)

// stubPriceSetter is the use case that sets the price of a product, in
// memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubPriceSetter struct {
	written product.Listed
	err     error
	got     ports.SetPrice
	calls   int
}

// Set answers the price of the test, and remembers what the tool asked.
func (s *stubPriceSetter) Set(_ context.Context, request ports.SetPrice) (product.Listed, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return product.Listed{}, s.err
	}

	return s.written, nil
}

// stubPricesReader is the use case that reads the price list of a business,
// in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubPricesReader struct {
	held  []product.Listed
	err   error
	got   ports.ReadPrices
	calls int
}

// Read answers the price list of the test, and remembers what the tool
// asked.
func (s *stubPricesReader) Read(_ context.Context, request ports.ReadPrices) ([]product.Listed, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return nil, s.err
	}

	return s.held, nil
}

// connectWithThePrices starts the transport with the stubs of the price
// list.
func connectWithThePrices(t *testing.T, setter ports.PriceRecorder, reader ports.PricesReader) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		SetPrice:            setter,
		ReadPrices:          reader,
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

// theProductOfTheTest is the product the stub of the use case answers.
func theProductOfTheTest() product.Product {
	return product.Product{
		ID:           10,
		BusinessID:   business.ID(77),
		RecipeID:     5,
		Name:         "Pan de molde",
		TaxTreatment: "Grammy 19",
	}
}

// thePriceOfTheTest is the price the stub of the use case answers.
func thePriceOfTheTest(t *testing.T) product.Price {
	return product.Price{
		ID:         product.PriceID(100),
		BusinessID: business.ID(77),
		ProductID:  product.ID(10),
		Channel:    product.ChannelWholesale,
		Amount:     ledger.Amount(2500),
		Day:        mustDay(t),
	}
}

// mustDay is the day of the prices of the test.
func mustDay(t *testing.T) (day ledger.Date) {
	t.Helper()

	parsed, err := ledger.ParseDate("2026-03-10")
	if err != nil {
		t.Fatalf("the day of the test could not be read: %v", err)
	}

	return parsed
}

// TestThePriceToolCarriesWhatTheAgentSends proves that the tool carries
// what the agent sends to the use case, and that the answer carries the
// product and the price with their channel.
func TestThePriceToolCarriesWhatTheAgentSends(t *testing.T) {
	t.Parallel()

	setter := &stubPriceSetter{written: product.Listed{
		Product: theProductOfTheTest(),
		Price:   thePriceOfTheTest(t),
	}}
	session := connectWithThePrices(t, setter, &stubPricesReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolSetPrice,
		Arguments: map[string]any{
			"business_id":   77,
			"recipe_id":     5,
			"name":          "Pan de molde",
			"channel":       "mayorista",
			"amount":        2500,
			"day":           "2026-03-10",
			"tax_treatment": "Grammy 19",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if setter.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", setter.calls)
	}
	if setter.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", setter.got.BusinessID)
	}
	if setter.got.RecipeID != 5 {
		t.Errorf("the recipe is %d, want 5", setter.got.RecipeID)
	}
	if setter.got.Name != "Pan de molde" {
		t.Errorf("the name is %q, want %q", setter.got.Name, "Pan de molde")
	}
	if setter.got.Channel != "mayorista" {
		t.Errorf("the channel is %q, want %q", setter.got.Channel, "mayorista")
	}
	if setter.got.Amount != 2500 {
		t.Errorf("the money is %d, want 2500", setter.got.Amount)
	}
	if setter.got.Day != "2026-03-10" {
		t.Errorf("the day is %q, want %q", setter.got.Day, "2026-03-10")
	}
	if setter.got.TaxTreatment != "Grammy 19" {
		t.Errorf("the tax treatment is %q, want %q", setter.got.TaxTreatment, "Grammy 19")
	}

	answer := textOf(t, result)
	for _, want := range []string{
		"Pan de molde", "wholesale", "2500", "2026-03-10", "Grammy 19",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestThePriceListToolAnswersEveryPriceThatHolds proves that the tool reads
// the price list of the business and answers every line of it.
func TestThePriceListToolAnswersEveryPriceThatHolds(t *testing.T) {
	t.Parallel()

	reader := &stubPricesReader{held: []product.Listed{
		{Product: theProductOfTheTest(), Price: thePriceOfTheTest(t)},
		{
			Product: product.Product{ID: 20, BusinessID: business.ID(77), RecipeID: 6, Name: "Biznaga"},
			Price: product.Price{
				ID:         product.PriceID(101),
				BusinessID: business.ID(77),
				ProductID:  product.ID(20),
				Channel:    product.ChannelRetail,
				Amount:     ledger.Amount(3000),
				Day:        mustDay(t),
			},
		},
	}}
	session := connectWithThePrices(t, &stubPriceSetter{}, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadPrices,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-10",
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
	if reader.got.Day != "2026-03-10" {
		t.Errorf("the day is %q, want %q", reader.got.Day, "2026-03-10")
	}

	answer := textOf(t, result)
	for _, want := range []string{"Pan de molde", "Biznaga", "wholesale", "retail", "2500", "3000"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheDescriptionOfThePriceTellsTheTruth proves that the description of
// the tool says what the tool does: the two channels of the book, the
// product that comes into being the first time it is named, the money in
// whole pesos and the fact that the price is never rewritten.
func TestTheDescriptionOfThePriceTellsTheTruth(t *testing.T) {
	t.Parallel()

	session := connectWithThePrices(t, &stubPriceSetter{}, &stubPricesReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolSetPrice)

	for _, want := range []string{
		"retail, wholesale",
		"minorista",
		"mayorista",
		"OPENED the first time it is named",
		"whole pesos",
		"never rewritten",
		"NO default channel",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheDescriptionOfThePriceListSaysThatItChangesNothing proves that the
// description of the tool that reads the price list says that it writes
// nothing.
func TestTheDescriptionOfThePriceListSaysThatItChangesNothing(t *testing.T) {
	t.Parallel()

	session := connectWithThePrices(t, &stubPriceSetter{}, &stubPricesReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadPrices)

	if !strings.Contains(description, "It writes nothing.") {
		t.Errorf("the description does not say that the tool writes nothing:\n%s", description)
	}
}

// TestThePriceToolsExplainTheFailureOfTheUseCase proves that the tools
// answer the error of the use case with the message the agent reads.
func TestThePriceToolsExplainTheFailureOfTheUseCase(t *testing.T) {
	t.Parallel()

	setter := &stubPriceSetter{err: product.ErrInvalidPrice}
	reader := &stubPricesReader{err: product.ErrNotFound}
	session := connectWithThePrices(t, setter, reader)

	calls := map[string]map[string]any{
		mcpserver.ToolSetPrice:   {"business_id": 77, "channel": "retail", "amount": 3000},
		mcpserver.ToolReadPrices: {"business_id": 77},
	}

	for name, arguments := range calls {
		result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name:      name,
			Arguments: arguments,
		})
		if err != nil {
			t.Fatalf("the tool failed: %v", err)
		}
		if !result.IsError {
			t.Fatalf("the tool %q did not answer an error", name)
		}
		if textOf(t, result) == "" {
			t.Errorf("the tool %q answered no message", name)
		}
	}
}

// TestThePriceToolsHaveNoStorage proves that the transport exposes the
// tools of the price list even when the service has no database, and that
// every call answers that the database is not there.
func TestThePriceToolsHaveNoStorage(t *testing.T) {
	t.Parallel()

	session := connectWithThePrices(t, nil, nil)

	calls := map[string]map[string]any{
		mcpserver.ToolSetPrice:   {"business_id": 77, "channel": "retail", "amount": 3000},
		mcpserver.ToolReadPrices: {"business_id": 77},
	}

	for name, arguments := range calls {
		result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name:      name,
			Arguments: arguments,
		})
		if err != nil {
			t.Fatalf("the tool failed: %v", err)
		}
		if !result.IsError {
			t.Fatalf("the tool %q did not answer an error", name)
		}
		if !strings.Contains(textOf(t, result), "database") {
			t.Errorf("the message of the tool %q does not talk about the database: %s", name, textOf(t, result))
		}
	}
}
