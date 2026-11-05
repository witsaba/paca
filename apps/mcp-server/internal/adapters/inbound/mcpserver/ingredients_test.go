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
)

// stubItemMatcher is the use case that matches a name against the catalog,
// in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubItemMatcher struct {
	match ingredient.Match
	err   error
	got   ports.MatchItems
	calls int
}

// Match answers the match of the test, and remembers what the tool asked.
func (s *stubItemMatcher) Match(_ context.Context, request ports.MatchItems) (ingredient.Match, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return ingredient.Match{}, s.err
	}

	return s.match, nil
}

// connectWithTheCatalog starts the transport with the stub of the catalog.
func connectWithTheCatalog(t *testing.T, matcher ports.ItemMatcher) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		MatchItems:          matcher,
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

// theAnswerOfTheMatch is the answer the stub of the use case gives.
func theAnswerOfTheMatch() ingredient.Match {
	return ingredient.Match{
		Name:   "harina",
		Status: ingredient.StatusAmbiguous,
		Options: []ingredient.Option{
			{ID: 10, Name: "Harina panadera"},
			{ID: 20, Name: "Harina de trigo"},
		},
	}
}

// TestTheMatchToolAnswersTheCandidates proves that the tool carries what
// the agent sends to the use case, and that the answer carries the status
// of the name and the options it offers.
func TestTheMatchToolAnswersTheCandidates(t *testing.T) {
	t.Parallel()

	matcher := &stubItemMatcher{match: theAnswerOfTheMatch()}
	session := connectWithTheCatalog(t, matcher)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMatchItems,
		Arguments: map[string]any{
			"business_id": 77,
			"name":        "harina",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if matcher.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", matcher.calls)
	}
	if matcher.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", matcher.got.BusinessID)
	}
	if matcher.got.Name != "harina" {
		t.Errorf("the name is %q, want %q", matcher.got.Name, "harina")
	}

	answer := textOf(t, result)
	for _, want := range []string{
		"harina", "ambiguous", "Harina panadera", "Harina de trigo", "10", "20",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheMatchToolAnswersTheNameItDoesNotKnow proves that a name the
// catalog does not know is not an error: the answer says that it is new.
func TestTheMatchToolAnswersTheNameItDoesNotKnow(t *testing.T) {
	t.Parallel()

	matcher := &stubItemMatcher{match: ingredient.Match{
		Name:   "polvasa",
		Status: ingredient.StatusUnknown,
		New:    true,
	}}
	session := connectWithTheCatalog(t, matcher)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMatchItems,
		Arguments: map[string]any{
			"business_id": 77,
			"name":        "polvasa",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	answer := textOf(t, result)
	for _, want := range []string{"polvasa", "unknown", "true"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheMatchToolAnswersWithoutADatabase proves that a server without a
// database still exposes the tool, and that the call says so.
func TestTheMatchToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithTheCatalog(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMatchItems,
		Arguments: map[string]any{
			"business_id": 77,
			"name":        "harina",
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

// TestTheMatchToolExplainsTheFailureToTheAgent proves that the failure of
// the use case reaches the agent with a message of its own, and never as
// the raw error of the domain.
func TestTheMatchToolExplainsTheFailureToTheAgent(t *testing.T) {
	t.Parallel()

	matcher := &stubItemMatcher{err: ingredient.ErrDuplicateName}
	session := connectWithTheCatalog(t, matcher)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMatchItems,
		Arguments: map[string]any{
			"business_id": 77,
			"name":        "harina",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the use case")
	}

	answer := textOf(t, result)
	if answer == ingredient.ErrDuplicateName.Error() {
		t.Errorf("the answer is the raw error of the domain: %s", answer)
	}
	if !strings.Contains(answer, ":") {
		t.Errorf("the answer says nothing about what to do: %s", answer)
	}
}

// TestTheDescriptionOfTheMatchToolSaysWhatItNeeds proves that the
// description of the tool is a contract with the agent: it says what is
// required, what the answer carries, and that it changes nothing.
func TestTheDescriptionOfTheMatchToolSaysWhatItNeeds(t *testing.T) {
	t.Parallel()

	session := connectWithTheCatalog(t, &stubItemMatcher{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolMatchItems)

	for _, want := range []string{
		"business is REQUIRED",
		"name is REQUIRED",
		"exact",
		"ambiguous",
		"unknown",
		"new",
		"changes nothing",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheErrorsOfTheCatalogReachTheAgentWithAMessage proves that every
// error the catalog can return reaches the agent with a message of its own.
func TestTheErrorsOfTheCatalogReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	for _, failure := range ingredient.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			matcher := &stubItemMatcher{err: failure}
			session := connectWithTheCatalog(t, matcher)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolMatchItems,
				Arguments: map[string]any{
					"business_id": 77,
					"name":        "harina",
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
