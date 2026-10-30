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
	"github.com/witsaba/paca/packages/domain/user"
)

// stubUsers is the use case of the users, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tools asked it.
type stubUsers struct {
	created     user.User
	found       user.User
	updated     user.User
	moved       user.User
	err         error
	gotDocument string
	gotName     string
	gotPhone    string
	gotEmail    string
	gotStatus   string
	gotReason   string
	calls       int
}

// Onboard answers the person that was registered.
func (s *stubUsers) Onboard(_ context.Context, documentID, fullName, phone, email string) (user.User, error) {
	s.calls++
	s.gotDocument = documentID
	s.gotName = fullName
	s.gotPhone = phone
	s.gotEmail = email

	if s.err != nil {
		return user.User{}, s.err
	}
	return s.created, nil
}

// Find answers the person that was read.
func (s *stubUsers) Find(_ context.Context, documentID string) (user.User, error) {
	s.calls++
	s.gotDocument = documentID

	if s.err != nil {
		return user.User{}, s.err
	}
	return s.found, nil
}

// ChangeStatus answers the move.
func (s *stubUsers) ChangeStatus(_ context.Context, documentID, status, observation string) (user.User, error) {
	s.calls++
	s.gotDocument = documentID
	s.gotStatus = status
	s.gotReason = observation

	if s.err != nil {
		return user.User{}, s.err
	}
	return s.moved, nil
}

// UpdateContactData answers the data that was written.
func (s *stubUsers) UpdateContactData(_ context.Context, documentID, fullName, phone, email string) (user.User, error) {
	s.calls++
	s.gotDocument = documentID
	s.gotName = fullName
	s.gotPhone = phone
	s.gotEmail = email

	if s.err != nil {
		return user.User{}, s.err
	}
	return s.updated, nil
}

// connect starts the transport and returns a client session on it.
//
// The use case may be nil: the transport exposes the tools even when the
// service has no database.
func connect(t *testing.T, users ports.UsersQuerier) *sdk.ClientSession {
	t.Helper()

	return connectWithLogger(t, users, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// connectWithLogger starts the transport with a logger the test reads.
func connectWithLogger(t *testing.T, users ports.UsersQuerier, logger *slog.Logger) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              logger,
		MaxRequestBodyBytes: 1 << 20,
		Users:               users,
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

func TestTheTransportExposesTheThreeToolsOfTheUsers(t *testing.T) {
	t.Parallel()

	session := connect(t, &stubUsers{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	want := []string{
		mcpserver.ToolOnboardUser,
		mcpserver.ToolGetUser,
		mcpserver.ToolChangeUserStatus,
		mcpserver.ToolUpdateUserContactData,
	}
	if len(tools.Tools) != len(want) {
		t.Fatalf("the server exposes %d tools, want %d", len(tools.Tools), len(want))
	}

	exposed := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		exposed[tool.Name] = true
	}
	for _, name := range want {
		if !exposed[name] {
			t.Errorf("the server exposes no tool named %q", name)
		}
	}
}

func TestTheDescriptionOfTheOnboardingTellsTheAgentWhatItNeeds(t *testing.T) {
	t.Parallel()

	session := connect(t, &stubUsers{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolOnboardUser)

	wants := []string{
		"national document ID is REQUIRED",
		"PACA never invents one",
		"full name is OPTIONAL",
		"Ask kindly for it",
		"must carry the country code",
		"+57 321 253 9984",
		"573212539984",
		"email is OPTIONAL",
		"valid address",
		"Try to gather as much information",
		"onboarding",
	}
	for _, want := range wants {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not tell the agent %q:\n%s", want, description)
		}
	}
}

func TestTheDescriptionOfTheStatusTellsTheAgentAboutTheObservation(t *testing.T) {
	t.Parallel()

	session := connect(t, &stubUsers{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolChangeUserStatus)

	wants := []string{
		"observation is REQUIRED",
		"'inactive' or 'blocked'",
		"left out when the new status is 'active'",
		"never returns to 'onboarding'",
		"onboarding -> active, inactive or blocked",
	}
	for _, want := range wants {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not tell the agent %q:\n%s", want, description)
		}
	}
}

func TestTheDescriptionOfTheDataTellsTheAgentThatAnEmptyFieldDoesNotChange(t *testing.T) {
	t.Parallel()

	session := connect(t, &stubUsers{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolUpdateUserContactData)

	wants := []string{
		"Only the fields you send are updated",
		"empty field is a field that stays as it is",
		"country code",
		"never changes the status",
	}
	for _, want := range wants {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not tell the agent %q:\n%s", want, description)
		}
	}
}

func TestTheSchemaOfTheStatusListsTheStatusesTheUserMayMoveTo(t *testing.T) {
	t.Parallel()

	session := connect(t, &stubUsers{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	for _, tool := range tools.Tools {
		if tool.Name != mcpserver.ToolChangeUserStatus {
			continue
		}

		enum := statusesOfTheSchema(t, tool.InputSchema)
		if len(enum) != 3 {
			t.Fatalf("the schema lists %v, want the three statuses", enum)
		}
		for _, want := range []string{"active", "inactive", "blocked"} {
			found := false
			for _, value := range enum {
				if value == want {
					found = true
				}
			}
			if !found {
				t.Errorf("the schema does not list %q in %v", want, enum)
			}
		}
		return
	}

	t.Fatalf("the server exposes no tool named %q", mcpserver.ToolChangeUserStatus)
}
