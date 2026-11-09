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
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// stubInvoiceRecorder is the use case that records an invoice, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubInvoiceRecorder struct {
	recorded purchase.Invoice
	err      error
	got      ports.RecordInvoice
	calls    int
}

// Record answers the invoice of the test, and remembers what the tool
// asked.
func (s *stubInvoiceRecorder) Record(_ context.Context, request ports.RecordInvoice) (purchase.Invoice, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return purchase.Invoice{}, s.err
	}

	return s.recorded, nil
}

// stubInvoicesReader is the use case that reads the invoices of a business,
// in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubInvoicesReader struct {
	held  []purchase.Invoice
	err   error
	got   ports.ReadInvoices
	calls int
}

// Read answers the invoices of the test, and remembers what the tool asked.
func (s *stubInvoicesReader) Read(_ context.Context, request ports.ReadInvoices) ([]purchase.Invoice, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return nil, s.err
	}

	return s.held, nil
}

// connectWithTheInvoices starts the transport with the stubs of the
// invoices.
func connectWithTheInvoices(
	t *testing.T,
	recorder ports.InvoiceRecorder,
	reader ports.InvoicesReader,
) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordInvoice:       recorder,
		ReadInvoices:        reader,
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

// theInvoiceOfTheTest is the invoice the stub of the use case answers.
func theInvoiceOfTheTest() purchase.Invoice {
	rate, err := purchase.NewRate(19)
	if err != nil {
		panic(err)
	}

	return purchase.Invoice{
		ID:         30,
		BusinessID: business.ID(77),
		ProviderID: party.ID(10),
		PurchaseID: purchase.ID(5),
		Number:     "FV-001",
		Day:        theDayOfTheInvoiceOfTheTest(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Tax:        ledger.Amount(1900),
		Total:      ledger.Amount(11900),
		Reference:  "CUFE-1234",
	}
}

// theDayOfTheInvoiceOfTheTest is the day the invoices of the test were
// issued.
func theDayOfTheInvoiceOfTheTest() ledger.Date {
	day, err := ledger.ParseDate("2026-03-02")
	if err != nil {
		panic(err)
	}

	return day
}

// TestTheInvoiceToolCarriesWhatTheAgentSends proves that the tool carries
// what the agent sends to the use case, and that the answer carries the
// invoice with its number, its rate and the tax that travels with it.
func TestTheInvoiceToolCarriesWhatTheAgentSends(t *testing.T) {
	t.Parallel()

	recorder := &stubInvoiceRecorder{recorded: theInvoiceOfTheTest()}
	session := connectWithTheInvoices(t, recorder, &stubInvoicesReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordInvoice,
		Arguments: map[string]any{
			"business_id": 77,
			"provider_id": 10,
			"purchase_id": 5,
			"number":      "FV-001",
			"day":         "2026-03-02",
			"base":        10000,
			"rate":        19,
			"total":       11900,
			"reference":   "CUFE-1234",
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
	if recorder.got.ProviderID != 10 {
		t.Errorf("the provider is %d, want 10", recorder.got.ProviderID)
	}
	if recorder.got.PurchaseID != 5 {
		t.Errorf("the purchase is %d, want 5", recorder.got.PurchaseID)
	}
	if recorder.got.Number != "FV-001" {
		t.Errorf("the number is %q, want %q", recorder.got.Number, "FV-001")
	}
	if recorder.got.Day != "2026-03-02" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-02")
	}
	if recorder.got.Base != 10000 {
		t.Errorf("the base is %d, want 10000", recorder.got.Base)
	}
	if recorder.got.Rate != 19 {
		t.Errorf("the rate is %v, want 19", recorder.got.Rate)
	}
	if recorder.got.Total != 11900 {
		t.Errorf("the total is %d, want 11900", recorder.got.Total)
	}
	if recorder.got.Reference != "CUFE-1234" {
		t.Errorf("the reference is %q, want %q", recorder.got.Reference, "CUFE-1234")
	}

	answer := textOf(t, result)
	for _, want := range []string{"FV-001", "11900", "1900", "19"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheInvoicesToolAnswersEveryInvoiceOfTheBusiness proves that the tool
// reads the invoices of the business and answers all of them.
func TestTheInvoicesToolAnswersEveryInvoiceOfTheBusiness(t *testing.T) {
	t.Parallel()

	reader := &stubInvoicesReader{held: []purchase.Invoice{
		theInvoiceOfTheTest(),
		{ID: 40, BusinessID: 77, Number: "FV-002", Total: ledger.Amount(5000)},
	}}
	session := connectWithTheInvoices(t, &stubInvoiceRecorder{}, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadInvoices,
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
	for _, want := range []string{"FV-001", "FV-002"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheDescriptionOfTheInvoiceTellsTheTruth proves that the description
// of the tool says what the tool does: the tax of an invoice comes from the
// invoice and is never assumed, the invoice that does not add up is
// refused, and the electronic invoice is never assumed.
func TestTheDescriptionOfTheInvoiceTellsTheTruth(t *testing.T) {
	t.Parallel()

	session := connectWithTheInvoices(t, &stubInvoiceRecorder{}, &stubInvoicesReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordInvoice)

	for _, want := range []string{
		"NEVER assumed: there is no default of nineteen percent",
		"the rate the invoice states",
		"the base the invoice states",
		"does not add up",
		"electronic invoice",
		"moves NO money",
		"the role of the provider",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheDescriptionOfTheInvoicesSaysThatItChangesNothing proves that the
// description of the tool that reads the invoices says that it writes
// nothing.
func TestTheDescriptionOfTheInvoicesSaysThatItChangesNothing(t *testing.T) {
	t.Parallel()

	session := connectWithTheInvoices(t, &stubInvoiceRecorder{}, &stubInvoicesReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadInvoices)

	if !strings.Contains(description, "It writes nothing.") {
		t.Errorf("the description does not say that the tool writes nothing:\n%s", description)
	}
}

// TestTheInvoiceToolsExplainTheFailureOfTheUseCase proves that the tools
// answer the error of the use case with the message the agent reads.
func TestTheInvoiceToolsExplainTheFailureOfTheUseCase(t *testing.T) {
	t.Parallel()

	recorder := &stubInvoiceRecorder{err: purchase.ErrInvoiceDoesNotAddUp}
	reader := &stubInvoicesReader{err: purchase.ErrNotFound}
	session := connectWithTheInvoices(t, recorder, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordInvoice,
		Arguments: map[string]any{
			"business_id": 77,
			"provider_id": 10,
			"purchase_id": 5,
			"number":      "FV-001",
			"base":        10000,
			"rate":        19,
			"total":       11900,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}
	if textOf(t, result) == "" {
		t.Error("the tool answered no message")
	}

	read, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadInvoices,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !read.IsError {
		t.Fatal("the tool did not answer an error")
	}
	if textOf(t, read) == "" {
		t.Error("the tool answered no message")
	}
}

// TestTheInvoiceToolsHaveNoStorage proves that the transport exposes the
// tools of the invoices even when the service has no database, and that
// every call answers that the database is not there.
func TestTheInvoiceToolsHaveNoStorage(t *testing.T) {
	t.Parallel()

	session := connectWithTheInvoices(t, nil, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordInvoice,
		Arguments: map[string]any{
			"business_id": 77,
			"provider_id": 10,
			"purchase_id": 5,
			"number":      "FV-001",
			"base":        10000,
			"rate":        19,
			"total":       11900,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	read, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadInvoices,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !read.IsError {
		t.Fatal("the tool did not answer an error")
	}
}
