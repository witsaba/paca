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
	"github.com/witsaba/paca/packages/domain/party"
)

// stubPartyOpener is the use case that opens a party, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubPartyOpener struct {
	opened party.Party
	err    error
	got    ports.OpenParty
	calls  int
}

// Open answers the party of the test, and remembers what the tool asked.
func (s *stubPartyOpener) Open(_ context.Context, request ports.OpenParty) (party.Party, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return party.Party{}, s.err
	}

	return s.opened, nil
}

// stubPartiesReader is the use case that reads the parties of a business,
// in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubPartiesReader struct {
	held  []party.Party
	err   error
	got   ports.ReadParties
	calls int
}

// Read answers the parties of the test, and remembers what the tool asked.
func (s *stubPartiesReader) Read(_ context.Context, request ports.ReadParties) ([]party.Party, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return nil, s.err
	}

	return s.held, nil
}

// connectWithTheParties starts the transport with the stubs of the parties.
func connectWithTheParties(t *testing.T, opener ports.PartyOpener, reader ports.PartiesReader) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		OpenParty:           opener,
		ReadParties:         reader,
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

// thePartyOfTheTest is the party the stub of the use case answers.
func thePartyOfTheTest() party.Party {
	return party.Party{
		ID:         10,
		BusinessID: business.ID(77),
		Kind:       party.KindStore,
		Name:       "La Panadería de Don Jaime",
		Roles:      []party.Role{party.RoleClient, party.RoleProvider},
		Phone:      "573212539984",
		Email:      "jaime@panaderia.com",
		Address:    "Calle 5 # 12-30",
		LegalHuman: party.LegalHuman{
			Name:   "Jaime Ortiz",
			Figure: "S.A.S.",
		},
	}
}

// TestThePartyToolCarriesWhatTheAgentSends proves that the tool carries
// what the agent sends to the use case, and that the answer carries the
// party with its roles and its facts.
func TestThePartyToolCarriesWhatTheAgentSends(t *testing.T) {
	t.Parallel()

	opener := &stubPartyOpener{opened: thePartyOfTheTest()}
	session := connectWithTheParties(t, opener, &stubPartiesReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolOpenParty,
		Arguments: map[string]any{
			"business_id":      77,
			"kind":             "store",
			"name":             "La Panadería de Don Jaime",
			"roles":            []any{"client", "provider"},
			"phone":            "+57 321 253 9984",
			"email":            "jaime@panaderia.com",
			"address":          "Calle 5 # 12-30",
			"legal_human_name": "Jaime Ortiz",
			"legal_figure":     "S.A.S.",
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
	if opener.got.BusinessID != 77 {
		t.Errorf("the business is %d, want 77", opener.got.BusinessID)
	}
	if opener.got.Kind != "store" {
		t.Errorf("the kind is %q, want %q", opener.got.Kind, "store")
	}
	if opener.got.Name != "La Panadería de Don Jaime" {
		t.Errorf("the name is %q, want %q", opener.got.Name, "La Panadería de Don Jaime")
	}
	if len(opener.got.Roles) != 2 || opener.got.Roles[0] != "client" || opener.got.Roles[1] != "provider" {
		t.Errorf("the roles are %v, want the client and the provider", opener.got.Roles)
	}
	if opener.got.Phone != "+57 321 253 9984" {
		t.Errorf("the phone is %q, want %q", opener.got.Phone, "+57 321 253 9984")
	}
	if opener.got.Email != "jaime@panaderia.com" {
		t.Errorf("the email is %q, want %q", opener.got.Email, "jaime@panaderia.com")
	}
	if opener.got.Address != "Calle 5 # 12-30" {
		t.Errorf("the address is %q, want %q", opener.got.Address, "Calle 5 # 12-30")
	}
	if opener.got.LegalHumanName != "Jaime Ortiz" {
		t.Errorf("the legal human is %q, want %q", opener.got.LegalHumanName, "Jaime Ortiz")
	}
	if opener.got.LegalFigure != "S.A.S." {
		t.Errorf("the legal figure is %q, want %q", opener.got.LegalFigure, "S.A.S.")
	}

	answer := textOf(t, result)
	for _, want := range []string{
		"La Panadería de Don Jaime", "store", "client", "provider",
		"Jaime Ortiz", "S.A.S.", "573212539984", "jaime@panaderia.com", "Calle 5 # 12-30",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestThePartiesToolAnswersEveryPartyOfTheBusiness proves that the tool
// reads the parties of the business and answers all of them: the ones that
// share a name among them.
func TestThePartiesToolAnswersEveryPartyOfTheBusiness(t *testing.T) {
	t.Parallel()

	reader := &stubPartiesReader{held: []party.Party{
		{ID: 10, BusinessID: 77, Kind: party.KindPerson, Name: "La Panadería", Roles: []party.Role{party.RoleClient}},
		{ID: 20, BusinessID: 77, Kind: party.KindStore, Name: "La Panadería", Roles: []party.Role{party.RoleProvider}},
	}}
	session := connectWithTheParties(t, &stubPartyOpener{}, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadParties,
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
	if reader.got.BusinessID != 77 {
		t.Errorf("the business is %d, want 77", reader.got.BusinessID)
	}

	answer := textOf(t, result)
	if strings.Count(answer, "La Panadería") != 2 {
		t.Errorf("the answer does not carry the two parties that share the name: %s", answer)
	}
}

// TestTheDescriptionOfThePartyTellsTheTruth proves that the description of
// the tool says what the tool does: the kinds of a party, the roles it
// holds, the legal human that represents it and the name that is not
// unique.
func TestTheDescriptionOfThePartyTellsTheTruth(t *testing.T) {
	t.Parallel()

	session := connectWithTheParties(t, &stubPartyOpener{}, &stubPartiesReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolOpenParty)

	for _, want := range []string{
		"person, store, farm",
		"client, provider",
		"a store REQUIRES",
		"a farm CARRIES",
		"a person REFUSES",
		"is not unique",
		"business of its own",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheDescriptionOfThePartiesSaysThatItChangesNothing proves that the
// description of the tool that reads the parties says that it writes
// nothing.
func TestTheDescriptionOfThePartiesSaysThatItChangesNothing(t *testing.T) {
	t.Parallel()

	session := connectWithTheParties(t, &stubPartyOpener{}, &stubPartiesReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadParties)

	if !strings.Contains(description, "It writes nothing.") {
		t.Errorf("the description does not say that the tool writes nothing:\n%s", description)
	}
}

// TestThePartyToolsExplainTheFailureOfTheUseCase proves that the tools
// answer the error of the use case with the message the agent reads.
func TestThePartyToolsExplainTheFailureOfTheUseCase(t *testing.T) {
	t.Parallel()

	opener := &stubPartyOpener{err: party.ErrMissingLegalHuman}
	reader := &stubPartiesReader{err: party.ErrNotFound}
	session := connectWithTheParties(t, opener, reader)

	for _, name := range []string{mcpserver.ToolOpenParty, mcpserver.ToolReadParties} {
		result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name:      name,
			Arguments: map[string]any{"business_id": 77},
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

// TestThePartyToolNamesTheBusinessOfTheParty proves that the tool that
// opens a party never tells the agent that there is no personal business:
// the business of a party is a business of its own, and the agent is not
// the onboarding.
//
// The message the agent reads names the business of the party, and it never
// names the personal business nor the onboarding.
func TestThePartyToolNamesTheBusinessOfTheParty(t *testing.T) {
	t.Parallel()

	opener := &stubPartyOpener{err: business.ErrNotFound}
	session := connectWithTheParties(t, opener, &stubPartiesReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolOpenParty,
		Arguments: map[string]any{
			"business_id": 77,
			"kind":        "person",
			"name":        "Doña Rosalba",
			"roles":       []any{"client"},
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	message := textOf(t, result)
	if !strings.Contains(message, "business of the party") {
		t.Errorf("the message does not name the business of the party: %s", message)
	}
	if strings.Contains(message, "personal business") {
		t.Errorf("the message names the personal business: %s", message)
	}
	if strings.Contains(message, "onboarding") {
		t.Errorf("the message names the onboarding: %s", message)
	}
}

// TestThePartyToolsHaveNoStorage proves that the transport exposes the
// tools of the parties even when the service has no database, and that
// every call answers that the database is not there.
func TestThePartyToolsHaveNoStorage(t *testing.T) {
	t.Parallel()

	session := connectWithTheParties(t, nil, nil)

	for _, name := range []string{mcpserver.ToolOpenParty, mcpserver.ToolReadParties} {
		result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name:      name,
			Arguments: map[string]any{"business_id": 77},
		})
		if err != nil {
			t.Fatalf("the tool failed: %v", err)
		}
		if !result.IsError {
			t.Fatalf("the tool %q did not answer an error", name)
		}
	}
}
