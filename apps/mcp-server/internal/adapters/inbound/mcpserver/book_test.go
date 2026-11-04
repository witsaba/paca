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

// stubBook is the use case of the book that the tools call.
type stubBook struct {
	recorded   ledger.Entry
	movements  []ledger.Entry
	balances   []ledger.Balance
	money      ledger.Book
	err        error
	gotFrom    string
	gotTo      string
	gotAsOf    string
	gotFromAcc account.ID
	gotToAcc   account.ID
	gotAmount  ledger.Amount
	gotDate    string
	gotAbout   string
	calls      int
}

// Record answers the movement of the test, or the error of the test.
func (s *stubBook) Record(
	_ context.Context,
	businessID business.ID,
	fromAccountID, toAccountID account.ID,
	amount ledger.Amount,
	date, description string,
) (ledger.Entry, error) {
	s.calls++
	s.gotFromAcc = fromAccountID
	s.gotToAcc = toAccountID
	s.gotAmount = amount
	s.gotDate = date
	s.gotAbout = description

	if s.err != nil {
		return ledger.Entry{}, s.err
	}

	return s.recorded, nil
}

// Movements answers the movements of the test, or the error of the test.
func (s *stubBook) Movements(_ context.Context, _ business.ID, from, to string) ([]ledger.Entry, error) {
	s.calls++
	s.gotFrom = from
	s.gotTo = to

	if s.err != nil {
		return nil, s.err
	}

	return s.movements, nil
}

// Balances answers the money of the test, or the error of the test.
func (s *stubBook) Balances(_ context.Context, _ business.ID, asOf string) (ledger.Book, []ledger.Balance, error) {
	s.calls++
	s.gotAsOf = asOf

	if s.err != nil {
		return ledger.Book{}, nil, s.err
	}

	return s.money, s.balances, nil
}

// connectWithBook starts the transport with the stub of the book.
func connectWithBook(t *testing.T, book ports.BookQuerier) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		Book:                book,
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

func TestTheToolOfTheMovementWritesTheMoneyOfTheBook(t *testing.T) {
	t.Parallel()

	day := ledger.NewDate(mustParseDay(t, "2026-03-01"))
	book := &stubBook{recorded: ledger.Entry{
		ID:              7,
		BusinessID:      77,
		DebitAccountID:  2,
		CreditAccountID: 1,
		Amount:          5000,
		OccurredOn:      day,
		Description:     "Lunch",
	}}
	session := connectWithBook(t, book)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordMovement,
		Arguments: map[string]any{
			"business_id":     77,
			"from_account_id": 1,
			"to_account_id":   2,
			"amount":          5000,
			"date":            "2026-03-01",
			"description":     "Lunch",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if book.gotFromAcc != account.ID(1) {
		t.Errorf("the money comes from the account %d, want 1", book.gotFromAcc)
	}
	if book.gotToAcc != account.ID(2) {
		t.Errorf("the money lands in the account %d, want 2", book.gotToAcc)
	}
	if book.gotAmount != 5000 {
		t.Errorf("the amount is %d, want 5000", book.gotAmount)
	}
	if book.gotDate != "2026-03-01" {
		t.Errorf("the day is %q, want %q", book.gotDate, "2026-03-01")
	}
	if book.gotAbout != "Lunch" {
		t.Errorf("the description is %q, want %q", book.gotAbout, "Lunch")
	}

	answer := textOf(t, result)
	for _, want := range []string{"5000", "2026-03-01", "Lunch"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

func TestTheToolOfTheMovementsReadsThePeriod(t *testing.T) {
	t.Parallel()

	day := ledger.NewDate(mustParseDay(t, "2026-03-01"))
	book := &stubBook{movements: []ledger.Entry{{
		ID:          7,
		Amount:      5000,
		OccurredOn:  day,
		Description: "Lunch",
	}}}
	session := connectWithBook(t, book)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadMovements,
		Arguments: map[string]any{
			"business_id": 77,
			"from_date":   "2026-03-01",
			"to_date":     "2026-03-31",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if book.gotFrom != "2026-03-01" {
		t.Errorf("the first day is %q, want %q", book.gotFrom, "2026-03-01")
	}
	if book.gotTo != "2026-03-31" {
		t.Errorf("the last day is %q, want %q", book.gotTo, "2026-03-31")
	}
	if !strings.Contains(textOf(t, result), "Lunch") {
		t.Errorf("the answer does not carry the movement: %s", textOf(t, result))
	}
}

func TestTheToolOfTheBalancesReadsTheMoneyOfTheBusiness(t *testing.T) {
	t.Parallel()

	book := &stubBook{
		money: ledger.Book{
			BusinessID:  77,
			Assets:      100000,
			Liabilities: 30000,
			Income:      120000,
			Expenses:    20000,
		},
		balances: []ledger.Balance{
			{AccountID: 1, Kind: account.KindAsset, Name: "Efectivo", Debits: 100000},
			{AccountID: 2, Kind: account.KindLiability, Name: "Préstamo: Juan", Credits: 30000},
		},
	}
	session := connectWithBook(t, book)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadBalances,
		Arguments: map[string]any{
			"business_id": 77,
			"as_of":       "2026-03-31",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if book.gotAsOf != "2026-03-31" {
		t.Errorf("the day is %q, want %q", book.gotAsOf, "2026-03-31")
	}

	answer := textOf(t, result)
	for _, want := range []string{"Efectivo", "Préstamo: Juan", "100000", "30000", "70000"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

func TestTheToolsOfTheBookExplainTheirFailureToTheAgent(t *testing.T) {
	t.Parallel()

	book := &stubBook{err: ledger.ErrInvalidAmount}
	session := connectWithBook(t, book)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordMovement,
		Arguments: map[string]any{
			"business_id":     77,
			"from_account_id": 1,
			"to_account_id":   2,
			"amount":          0,
			"description":     "Lunch",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the use case")
	}

	answer := textOf(t, result)
	if !strings.Contains(answer, "positive") {
		t.Errorf("the answer does not say what to do: %s", answer)
	}
	if answer == ledger.ErrInvalidAmount.Error() {
		t.Errorf("the answer is the raw error of the domain: %s", answer)
	}
}

func TestTheToolsOfTheBookAnswerWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithBook(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadBalances,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the service")
	}

	if !strings.Contains(textOf(t, result), "database") {
		t.Errorf("the answer does not say that the database is not there: %s", textOf(t, result))
	}
}

// TestTheDescriptionsOfTheBookCarryTheRulesOfTheDomain proves that the
// contract of the tools is derived from the domain: the kinds of an account
// are read from it, and the description never writes them a second time.
func TestTheDescriptionsOfTheBookCarryTheRulesOfTheDomain(t *testing.T) {
	t.Parallel()

	session := connectWithBook(t, &stubBook{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordMovement)

	for _, one := range account.AllKinds() {
		if !strings.Contains(description, one.String()) {
			t.Errorf("the description does not name the kind %q", one.String())
		}
	}

	for _, want := range []string{
		"FROM one account TO another",
		"positive",
		"The description is REQUIRED",
		"means the day of today",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q", want)
		}
	}
}

// mustParseDay returns the day the tests of the book use.
func mustParseDay(t *testing.T, day string) time.Time {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day could not be parsed: %v", err)
	}

	return parsed.Time()
}
