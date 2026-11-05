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
	"github.com/witsaba/paca/packages/domain/user"
)

// stubOnboarding is the use case of the onboarding that the tool calls.
type stubOnboarding struct {
	completed   ports.Onboarding
	err         error
	gotDocument string
	gotName     string
	gotPhone    string
	gotEmail    string
	calls       int
}

// Complete answers the onboarding of the test, or the error of the test.
func (s *stubOnboarding) Complete(_ context.Context, documentID, fullName, phone, email string) (ports.Onboarding, error) {
	s.calls++
	s.gotDocument = documentID
	s.gotName = fullName
	s.gotPhone = phone
	s.gotEmail = email

	if s.err != nil {
		return ports.Onboarding{}, s.err
	}

	return s.completed, nil
}

// connectWithOnboarding starts the transport with the stub of the
// onboarding.
func connectWithOnboarding(t *testing.T, onboarding ports.OnboardingCompleter) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		Onboarding:          onboarding,
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

// theAnswerOfTheOnboarding is the answer the stub of the onboarding gives.
func theAnswerOfTheOnboarding() ports.Onboarding {
	accounts := make([]account.Account, 0, len(account.DefaultPersonalAccounts()))
	for index, one := range account.DefaultPersonalAccounts() {
		accounts = append(accounts, account.Account{
			ID:         account.ID(index + 1),
			BusinessID: business.ID(77),
			Kind:       one.Kind,
			Name:       one.Name,
		})
	}

	return ports.Onboarding{
		User: user.User{
			ID:       1234567,
			FullName: "Ana María Pérez",
			Status:   user.StatusActive,
		},
		Business: business.Business{
			ID:       77,
			OwnerID:  1234567,
			Kind:     business.KindPersonal,
			Name:     business.DefaultPersonalName,
			Currency: business.DefaultCurrency,
		},
		Accounts: accounts,
	}
}

func TestTheOnboardingToolCompletesTheOnboardingOfThePerson(t *testing.T) {
	t.Parallel()

	onboarding := &stubOnboarding{completed: theAnswerOfTheOnboarding()}
	session := connectWithOnboarding(t, onboarding)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCompleteOnboarding,
		Arguments: map[string]any{
			"document_id": "1234567",
			"full_name":   "Ana María Pérez",
			"phone":       "+57 321-253-9984",
			"email":       "ana@example.com",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if onboarding.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", onboarding.calls)
	}
	if onboarding.gotDocument != "1234567" {
		t.Errorf("the document is %q, want %q", onboarding.gotDocument, "1234567")
	}
	if onboarding.gotName != "Ana María Pérez" {
		t.Errorf("the name is %q, want %q", onboarding.gotName, "Ana María Pérez")
	}
	if onboarding.gotPhone != "+57 321-253-9984" {
		t.Errorf("the phone is %q, want %q", onboarding.gotPhone, "+57 321-253-9984")
	}

	answer := textOf(t, result)
	for _, want := range []string{"1234567", "Ana María Pérez", "active", "77", "Efectivo", "Otros gastos"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

func TestTheOnboardingToolExplainsTheFailureToTheAgent(t *testing.T) {
	t.Parallel()

	onboarding := &stubOnboarding{err: account.ErrAlreadyExists}
	session := connectWithOnboarding(t, onboarding)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCompleteOnboarding,
		Arguments: map[string]any{
			"document_id": "1234567",
			"full_name":   "Ana María Pérez",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the use case")
	}

	answer := textOf(t, result)
	if !strings.Contains(answer, "already exists") {
		t.Errorf("the answer does not explain the failure: %s", answer)
	}
	if answer == account.ErrAlreadyExists.Error() {
		t.Errorf("the answer is the raw error of the domain: %s", answer)
	}
}

func TestTheOnboardingToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithOnboarding(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCompleteOnboarding,
		Arguments: map[string]any{
			"document_id": "1234567",
			"full_name":   "Ana María Pérez",
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

// TestTheDescriptionOfTheOnboardingToolNamesTheSeedOfTheBook proves that the
// description of the tool is derived from the domain: the seed of the
// personal business belongs to the domain, and the description never writes
// it a second time.
func TestTheDescriptionOfTheOnboardingToolNamesTheSeedOfTheBook(t *testing.T) {
	t.Parallel()

	session := connectWithOnboarding(t, &stubOnboarding{completed: theAnswerOfTheOnboarding()})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolCompleteOnboarding)

	for _, one := range account.DefaultPersonalAccounts() {
		if !strings.Contains(description, one.Name) {
			t.Errorf("the description does not name the account %q", one.Name)
		}
	}

	for _, want := range []string{
		"national document ID is REQUIRED",
		"PACA never invents one",
		"full name is REQUIRED",
		"Ask the person for their name kindly",
		"must carry the country code",
		"+57 321 253 9984",
		"573212539984",
		"email is OPTIONAL",
		"valid address",
		"Try to gather as much information",
		"ONE call",
		"starts empty",
		"safe to call again",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q", want)
		}
	}
}

// TestTheErrorsOfTheBookReachTheAgentWithAMessage proves that every error
// the book of a person can return reaches the agent with a message of its
// own.
func TestTheErrorsOfTheBookReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	failures := append(business.Errors(), account.Errors()...)
	for _, failure := range failures {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			onboarding := &stubOnboarding{err: failure}
			session := connectWithOnboarding(t, onboarding)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolCompleteOnboarding,
				Arguments: map[string]any{
					"document_id": "1234567",
					"full_name":   "Ana María Pérez",
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
