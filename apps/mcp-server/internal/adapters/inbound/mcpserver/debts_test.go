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
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// stubDebts is the use case of the debts that the tools call.
type stubDebts struct {
	opened    ledger.Debt
	debts     []ledger.Debt
	paid      ports.DebtPayment
	err       error
	calls     int
	gotWho    string
	gotAmount ledger.Amount
	gotToAcc  account.ID
	gotDebt   account.ID
	gotFrom   account.ID
	gotDate   string
	gotAbout  string
	gotDue    string
	gotName   string
}

// Record answers the debt of the test, and remembers what the tool asked.
func (s *stubDebts) Record(
	_ context.Context,
	_ business.ID,
	counterparty string,
	amount ledger.Amount,
	toAccountID account.ID,
	date, description, dueDate, name string,
) (ledger.Debt, error) {
	s.calls++
	s.gotWho = counterparty
	s.gotAmount = amount
	s.gotToAcc = toAccountID
	s.gotDate = date
	s.gotAbout = description
	s.gotDue = dueDate
	s.gotName = name

	if s.err != nil {
		return ledger.Debt{}, s.err
	}

	return s.opened, nil
}

// Debts answers the debts of the test.
func (s *stubDebts) Debts(_ context.Context, _ business.ID) ([]ledger.Debt, error) {
	s.calls++

	if s.err != nil {
		return nil, s.err
	}

	return s.debts, nil
}

// Pay answers the payment of the test, and remembers what the tool asked.
func (s *stubDebts) Pay(
	_ context.Context,
	_ business.ID,
	debtAccountID account.ID,
	amount ledger.Amount,
	fromAccountID account.ID,
	date, description string,
) (ports.DebtPayment, error) {
	s.calls++
	s.gotDebt = debtAccountID
	s.gotAmount = amount
	s.gotFrom = fromAccountID
	s.gotDate = date
	s.gotAbout = description

	if s.err != nil {
		return ports.DebtPayment{}, s.err
	}

	return s.paid, nil
}

// stubSummary is the use case of the closing of a month that the tool calls.
type stubSummary struct {
	closed   ports.MonthSummary
	err      error
	calls    int
	gotYear  int
	gotMonth int
}

// Month answers the closing of the test, and remembers the month it asked.
func (s *stubSummary) Month(_ context.Context, _ business.ID, year, month int) (ports.MonthSummary, error) {
	s.calls++
	s.gotYear = year
	s.gotMonth = month

	if s.err != nil {
		return ports.MonthSummary{}, s.err
	}

	return s.closed, nil
}

// connectWithDebts starts the transport with the stubs of the debts and of
// the closing.
func connectWithDebts(t *testing.T, debts ports.DebtQuerier, summary ports.SummaryQuerier) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		Debts:               debts,
		Summary:             summary,
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

// theDebtOfTheTools is the debt the tools answer with.
func theDebtOfTheTools() ledger.Debt {
	due := time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC)

	return ledger.Debt{
		Account: account.Account{
			ID:           9,
			BusinessID:   77,
			Kind:         account.KindLiability,
			Name:         "Deuda con Ana",
			Counterparty: "Ana",
			DueDate:      &due,
		},
		Amount: 500000,
	}
}

func TestTheToolOfTheDebtOpensTheDebtAndTheMoney(t *testing.T) {
	t.Parallel()

	debts := &stubDebts{opened: theDebtOfTheTools()}
	session := connectWithDebts(t, debts, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordDebt,
		Arguments: map[string]any{
			"business_id":   77,
			"counterparty":  "Ana",
			"amount":        500000,
			"to_account_id": 1,
			"date":          "2026-03-01",
			"due_date":      "2026-04-30",
			"name":          "Deuda con Ana",
			"description":   "Loan of Ana",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if debts.gotWho != "Ana" {
		t.Errorf("the counterparty is %q, want %q", debts.gotWho, "Ana")
	}
	if debts.gotAmount != 500000 {
		t.Errorf("the amount is %d, want 500000", debts.gotAmount)
	}
	if debts.gotToAcc != account.ID(1) {
		t.Errorf("the money lands in the account %d, want 1", debts.gotToAcc)
	}
	if debts.gotDate != "2026-03-01" {
		t.Errorf("the day is %q, want %q", debts.gotDate, "2026-03-01")
	}
	if debts.gotDue != "2026-04-30" {
		t.Errorf("the due date is %q, want %q", debts.gotDue, "2026-04-30")
	}
	if debts.gotName != "Deuda con Ana" {
		t.Errorf("the name is %q, want %q", debts.gotName, "Deuda con Ana")
	}
	if debts.gotAbout != "Loan of Ana" {
		t.Errorf("the description is %q, want %q", debts.gotAbout, "Loan of Ana")
	}

	answer := textOf(t, result)
	for _, want := range []string{"Ana", "500000", "2026-04-30"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

func TestTheToolOfTheDebtsReadsWhatThePersonOwes(t *testing.T) {
	t.Parallel()

	debts := &stubDebts{debts: []ledger.Debt{theDebtOfTheTools()}}
	session := connectWithDebts(t, debts, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolListDebts,
		Arguments: map[string]any{"business_id": 77},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	answer := textOf(t, result)
	for _, want := range []string{"Ana", "500000", "2026-04-30"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

func TestTheToolOfThePaymentOfADebtPaysIt(t *testing.T) {
	t.Parallel()

	debts := &stubDebts{paid: ports.DebtPayment{
		Movement: ledger.Entry{ID: 7, Amount: 100000, Description: "Payment to Ana"},
		Debt:     ledger.Debt{Account: theDebtOfTheTools().Account, Amount: 400000},
	}}
	session := connectWithDebts(t, debts, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordDebtPayment,
		Arguments: map[string]any{
			"business_id":     77,
			"debt_account_id": 9,
			"amount":          100000,
			"from_account_id": 1,
			"date":            "2026-03-02",
			"description":     "Payment to Ana",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if debts.gotDebt != account.ID(9) {
		t.Errorf("the debt is the account %d, want 9", debts.gotDebt)
	}
	if debts.gotAmount != 100000 {
		t.Errorf("the amount is %d, want 100000", debts.gotAmount)
	}
	if debts.gotFrom != account.ID(1) {
		t.Errorf("the money comes from the account %d, want 1", debts.gotFrom)
	}

	answer := textOf(t, result)
	for _, want := range []string{"400000", "100000"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

func TestTheToolOfTheClosingOfAMonthReadsTheMonth(t *testing.T) {
	t.Parallel()

	summary := &stubSummary{closed: ports.MonthSummary{
		From:    ledger.NewDate(mustParseDay(t, "2026-03-01")),
		To:      ledger.NewDate(mustParseDay(t, "2026-03-31")),
		Summary: ledger.NewSummary(1000000, 400000),
		Debts:   []ledger.Debt{theDebtOfTheTools()},
	}}
	session := connectWithDebts(t, nil, summary)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadMonthSummary,
		Arguments: map[string]any{
			"business_id": 77,
			"year":        2026,
			"month":       3,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if summary.gotYear != 2026 {
		t.Errorf("the year is %d, want 2026", summary.gotYear)
	}
	if summary.gotMonth != 3 {
		t.Errorf("the month is %d, want 3", summary.gotMonth)
	}

	answer := textOf(t, result)
	for _, want := range []string{"1000000", "400000", "600000", "2026-03-01", "2026-03-31"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

func TestTheToolsOfTheDebtsWithoutADatabaseSaySo(t *testing.T) {
	t.Parallel()

	session := connectWithDebts(t, nil, nil)

	calls := []struct {
		name string
		args map[string]any
	}{
		{name: mcpserver.ToolRecordDebt, args: map[string]any{"business_id": 77, "counterparty": "Ana", "amount": 500000, "to_account_id": 1}},
		{name: mcpserver.ToolListDebts, args: map[string]any{"business_id": 77}},
		{name: mcpserver.ToolRecordDebtPayment, args: map[string]any{"business_id": 77, "debt_account_id": 9, "amount": 1000, "from_account_id": 1, "description": "Payment"}},
		{name: mcpserver.ToolReadMonthSummary, args: map[string]any{"business_id": 77}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      call.name,
				Arguments: call.args,
			})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool did not answer an error")
			}
			if !strings.Contains(textOf(t, result), "database") {
				t.Errorf("the answer does not say that the database is not there: %s", textOf(t, result))
			}
		})
	}
}
