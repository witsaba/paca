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
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
)

// stubBusinessOpener is the use case that opens a business, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubBusinessOpener struct {
	opened   ports.RegisteredBusiness
	err      error
	gotOwner string
	gotName  string
	gotMoney string
	calls    int
}

// Open answers the business that was opened, or the error of the test.
func (s *stubBusinessOpener) Open(_ context.Context, documentID, name, currency string) (ports.RegisteredBusiness, error) {
	s.calls++
	s.gotOwner = documentID
	s.gotName = name
	s.gotMoney = currency

	if s.err != nil {
		return ports.RegisteredBusiness{}, s.err
	}

	return s.opened, nil
}

// stubBusinessQuerier is the use case that reads the businesses of an owner,
// in memory.
type stubBusinessQuerier struct {
	owned    []business.Business
	err      error
	gotOwner string
	calls    int
}

// List answers the businesses of the owner, or the error of the test.
func (s *stubBusinessQuerier) List(_ context.Context, documentID string) ([]business.Business, error) {
	s.calls++
	s.gotOwner = documentID

	if s.err != nil {
		return nil, s.err
	}

	return s.owned, nil
}

// connectWithBusinesses starts the transport with the stubs of the
// businesses.
func connectWithBusinesses(t *testing.T, opener ports.BusinessOpener, querier ports.BusinessQuerier) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		OpenBusiness:        opener,
		ReadBusinesses:      querier,
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

// theAnswerOfTheBusiness is the answer the stub of the use case gives.
func theAnswerOfTheBusiness() ports.RegisteredBusiness {
	accounts := make([]account.Account, 0, len(account.DefaultBusinessAccounts()))
	for index, one := range account.DefaultBusinessAccounts() {
		accounts = append(accounts, account.Account{
			ID:         account.ID(index + 1),
			BusinessID: business.ID(88),
			Kind:       one.Kind,
			Name:       one.Name,
		})
	}

	return ports.RegisteredBusiness{
		Business: business.Business{
			ID:       88,
			OwnerID:  1234567,
			Kind:     business.KindBusiness,
			Name:     "Panadería del barrio",
			Currency: business.DefaultCurrency,
		},
		Accounts: accounts,
	}
}

// TestTheBusinessToolOpensTheBusiness proves that the tool carries what the
// agent sends to the use case, and that the answer carries the business and
// the accounts of its book.
func TestTheBusinessToolOpensTheBusiness(t *testing.T) {
	t.Parallel()

	opener := &stubBusinessOpener{opened: theAnswerOfTheBusiness()}
	session := connectWithBusinesses(t, opener, &stubBusinessQuerier{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolOpenBusiness,
		Arguments: map[string]any{
			"owner_id": "1234567",
			"name":     "Panadería del barrio",
			"currency": "COP",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if opener.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", opener.calls)
	}
	if opener.gotOwner != "1234567" {
		t.Errorf("the owner is %q, want %q", opener.gotOwner, "1234567")
	}
	if opener.gotName != "Panadería del barrio" {
		t.Errorf("the name is %q, want %q", opener.gotName, "Panadería del barrio")
	}
	if opener.gotMoney != "COP" {
		t.Errorf("the currency is %q, want %q", opener.gotMoney, "COP")
	}

	answer := textOf(t, result)
	for _, want := range []string{"88", "Panadería del barrio", "business", "COP", "Efectivo", "Capital"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheBusinessToolAnswersWithoutADatabase proves that a server without a
// database still exposes the tool, and that the call says so.
func TestTheBusinessToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithBusinesses(t, nil, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolOpenBusiness,
		Arguments: map[string]any{
			"owner_id": "1234567",
			"name":     "Panadería del barrio",
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

// TestTheBusinessToolExplainsTheFailureToTheAgent proves that the failure
// of the use case reaches the agent with a message of its own, and never as
// the raw error of the domain.
func TestTheBusinessToolExplainsTheFailureToTheAgent(t *testing.T) {
	t.Parallel()

	opener := &stubBusinessOpener{err: business.ErrAlreadyExists}
	session := connectWithBusinesses(t, opener, &stubBusinessQuerier{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolOpenBusiness,
		Arguments: map[string]any{
			"owner_id": "1234567",
			"name":     "Panadería del barrio",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the use case")
	}

	answer := textOf(t, result)
	if answer == business.ErrAlreadyExists.Error() {
		t.Errorf("the answer is the raw error of the domain: %s", answer)
	}
	if !strings.Contains(answer, "already exists") {
		t.Errorf("the answer does not explain the failure: %s", answer)
	}
	if !strings.Contains(answer, "taken inside the businesses of its owner") {
		t.Errorf("the answer does not say that the name is taken: %s", answer)
	}
}

// TestTheBusinessesToolReadsTheBusinessesOfTheOwner proves that the tool
// carries the document of the owner and answers every business of it.
func TestTheBusinessesToolReadsTheBusinessesOfTheOwner(t *testing.T) {
	t.Parallel()

	querier := &stubBusinessQuerier{owned: []business.Business{
		{ID: 1, OwnerID: 1234567, Kind: business.KindPersonal, Name: business.DefaultPersonalName, Currency: business.DefaultCurrency},
		{ID: 2, OwnerID: 1234567, Kind: business.KindBusiness, Name: "Panadería del barrio", Currency: business.DefaultCurrency},
	}}
	session := connectWithBusinesses(t, &stubBusinessOpener{}, querier)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadBusinesses,
		Arguments: map[string]any{
			"owner_id": "1234567",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if querier.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", querier.calls)
	}
	if querier.gotOwner != "1234567" {
		t.Errorf("the owner is %q, want %q", querier.gotOwner, "1234567")
	}

	answer := textOf(t, result)
	for _, want := range []string{
		"Negocio Personal", "personal",
		"Panadería del barrio", "business",
		"COP",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheBusinessesToolAnswersWithoutADatabase proves that a server without a
// database still exposes the tool of the list, and that the call says so.
func TestTheBusinessesToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithBusinesses(t, &stubBusinessOpener{}, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadBusinesses,
		Arguments: map[string]any{
			"owner_id": "1234567",
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

// TestTheDescriptionOfTheBusinessToolSaysWhatItNeeds proves that the
// description of the tool is a contract with the agent: it says what is
// required, what the empty field means, and what the tool writes.
func TestTheDescriptionOfTheBusinessToolSaysWhatItNeeds(t *testing.T) {
	t.Parallel()

	session := connectWithBusinesses(t, &stubBusinessOpener{}, &stubBusinessQuerier{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolOpenBusiness)

	for _, want := range []string{
		"document ID of the owner is REQUIRED",
		"name is REQUIRED",
		"currency is OPTIONAL",
		"peso",
		"one transaction",
		"taken inside the businesses of its owner",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheDescriptionOfTheBusinessesToolNamesTheKindsOfABusiness proves that
// the list of the kinds of a business is derived from the domain: the
// description never writes it a second time.
func TestTheDescriptionOfTheBusinessesToolNamesTheKindsOfABusiness(t *testing.T) {
	t.Parallel()

	session := connectWithBusinesses(t, &stubBusinessOpener{}, &stubBusinessQuerier{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadBusinesses)

	for _, one := range business.AllKinds() {
		if !strings.Contains(description, one.String()) {
			t.Errorf("the description does not name the kind %q:\n%s", one.String(), description)
		}
	}
	if !strings.Contains(description, "changes nothing") {
		t.Errorf("the description does not say that the tool changes nothing:\n%s", description)
	}
}

// TestTheDescriptionOfTheBusinessToolNamesTheBookOfTheBusiness proves that
// the accounts the book starts with are derived from the domain.
func TestTheDescriptionOfTheBusinessToolNamesTheBookOfTheBusiness(t *testing.T) {
	t.Parallel()

	session := connectWithBusinesses(t, &stubBusinessOpener{}, &stubBusinessQuerier{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolOpenBusiness)

	for _, one := range account.DefaultBusinessAccounts() {
		if !strings.Contains(description, one.Name) {
			t.Errorf("the description does not name the account %q", one.Name)
		}
	}
}

// TestTheErrorsOfTheBusinessesReachTheAgentWithAMessage proves that every
// error the businesses can return reaches the agent with a message of its
// own.
func TestTheErrorsOfTheBusinessesReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	for _, failure := range business.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			opener := &stubBusinessOpener{err: failure}
			session := connectWithBusinesses(t, opener, &stubBusinessQuerier{})

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolOpenBusiness,
				Arguments: map[string]any{
					"owner_id": "1234567",
					"name":     "Panadería del barrio",
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
