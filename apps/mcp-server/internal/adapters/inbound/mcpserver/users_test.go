package mcpserver_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/packages/domain/user"
)

func TestTheOnboardingToolRegistersThePerson(t *testing.T) {
	t.Parallel()

	users := &stubUsers{created: user.User{
		ID:        1234567,
		FullName:  "Ana María Pérez",
		Phone:     "573212539984",
		Email:     "ana@example.com",
		Status:    user.StatusOnboarding,
		CreatedAt: time.Date(2026, time.October, 29, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.October, 29, 10, 0, 0, 0, time.UTC),
	}}
	session := connect(t, users)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolOnboardUser,
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

	if users.gotDocument != "1234567" {
		t.Errorf("the document is %q, want %q", users.gotDocument, "1234567")
	}
	if users.gotPhone != "+57 321-253-9984" {
		t.Errorf("the phone is %q, want %q", users.gotPhone, "+57 321-253-9984")
	}
	if !strings.Contains(textOf(t, result), "1234567") {
		t.Errorf("the answer does not carry the document: %s", textOf(t, result))
	}
}

func TestTheStatusToolMovesTheUser(t *testing.T) {
	t.Parallel()

	users := &stubUsers{moved: user.User{
		ID:        1234567,
		Status:    user.StatusInactive,
		CreatedAt: time.Date(2026, time.October, 29, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.October, 29, 11, 0, 0, 0, time.UTC),
	}}
	session := connect(t, users)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolChangeUserStatus,
		Arguments: map[string]any{
			"document_id":         "1234567",
			"status":              "inactive",
			"status_observations": "the person asked to leave",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if users.gotStatus != "inactive" {
		t.Errorf("the status is %q, want %q", users.gotStatus, "inactive")
	}
	if users.gotReason != "the person asked to leave" {
		t.Errorf("the reason is %q, want %q", users.gotReason, "the person asked to leave")
	}
}

func TestTheDataToolWritesOnlyTheFieldsItIsGiven(t *testing.T) {
	t.Parallel()

	users := &stubUsers{updated: user.User{
		ID:        1234567,
		FullName:  "Ana María Pérez",
		Email:     "ana@example.com",
		Status:    user.StatusOnboarding,
		CreatedAt: time.Date(2026, time.October, 29, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.October, 29, 12, 0, 0, 0, time.UTC),
	}}
	session := connect(t, users)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolUpdateUserContactData,
		Arguments: map[string]any{
			"document_id": "1234567",
			"full_name":   "Ana María Pérez",
			"email":       "ana@example.com",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if users.gotName != "Ana María Pérez" {
		t.Errorf("the name is %q, want %q", users.gotName, "Ana María Pérez")
	}
	if users.gotPhone != "" {
		t.Errorf("the phone is %q, want it empty", users.gotPhone)
	}
}

func TestTheToolExplainsItsFailureToTheAgent(t *testing.T) {
	t.Parallel()

	users := &stubUsers{err: user.ErrObservationRequired}
	session := connect(t, users)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolChangeUserStatus,
		Arguments: map[string]any{
			"document_id": "1234567",
			"status":      "blocked",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	message := textOf(t, result)
	if !strings.Contains(message, "reason") {
		t.Errorf("the message does not tell the agent that the status needs a reason: %s", message)
	}
}

func TestTheToolExplainsThatTheDocumentIsRequired(t *testing.T) {
	t.Parallel()

	users := &stubUsers{err: user.ErrInvalidID}
	session := connect(t, users)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolOnboardUser,
		Arguments: map[string]any{
			"document_id": "CC",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	message := textOf(t, result)
	if !strings.Contains(message, "document ID") {
		t.Errorf("the message does not talk about the document: %s", message)
	}
}

func TestTheToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connect(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolOnboardUser,
		Arguments: map[string]any{
			"document_id": "1234567",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	if !strings.Contains(textOf(t, result), "database") {
		t.Errorf("the message does not talk about the database: %s", textOf(t, result))
	}
}

func TestTheTransportAnswersTheToolsEvenWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connect(t, nil)

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}
	if len(tools.Tools) != 3 {
		t.Errorf("the server exposes %d tools, want 3", len(tools.Tools))
	}
}

// statusesOfTheSchema returns the statuses the schema of the status lists.
func statusesOfTheSchema(t *testing.T, schema any) []string {
	t.Helper()

	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("the schema is %T, want a map", schema)
	}
	properties, ok := root["properties"].(map[string]any)
	if !ok {
		t.Fatalf("the schema has no properties: %v", root)
	}
	status, ok := properties["status"].(map[string]any)
	if !ok {
		t.Fatalf("the schema has no property for the status: %v", properties)
	}
	values, ok := status["enum"].([]any)
	if !ok {
		t.Fatalf("the property of the status has no list: %v", status)
	}

	statuses := make([]string, 0, len(values))
	for _, value := range values {
		statuses = append(statuses, fmt.Sprint(value))
	}

	return statuses
}

// schemaOf returns the input schema of the tool of the given name.
func schemaOf(t *testing.T, tools []*sdk.Tool, name string) any {
	t.Helper()

	for _, tool := range tools {
		if tool.Name == name {
			return tool.InputSchema
		}
	}

	t.Fatalf("the server exposes no tool named %q", name)
	return nil
}

// descriptionOf returns the description of the tool of the given name.
func descriptionOf(t *testing.T, tools []*sdk.Tool, name string) string {
	t.Helper()

	for _, tool := range tools {
		if tool.Name == name {
			return tool.Description
		}
	}

	t.Fatalf("the server exposes no tool named %q", name)
	return ""
}

// textOf returns the text of the answer of a tool.
func textOf(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()

	var text strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(*sdk.TextContent); ok {
			text.WriteString(textContent.Text)
		}
	}

	return text.String()
}

// TestTheSchemaOfTheStatusToolListsTheStatesOfTheDomain proves that the
// contract of the tool agrees with the domain: the states the agent may send
// are the states the domain allows.
//
// The list is read from the domain, so the day the domain adds a state the
// schema follows it. A list written by hand in the adapter would agree with
// the domain today and would fail this test from that day on.
func TestTheSchemaOfTheStatusToolListsTheStatesOfTheDomain(t *testing.T) {
	t.Parallel()

	session := connect(t, &stubUsers{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	got := statusesOfTheSchema(t, schemaOf(t, tools.Tools, mcpserver.ToolChangeUserStatus))

	want := make([]string, 0, len(user.AllTargets()))
	for _, target := range user.AllTargets() {
		want = append(want, target.String())
	}

	if len(got) != len(want) {
		t.Fatalf("the schema lists %v, want %v", got, want)
	}
	for i, status := range want {
		if got[i] != status {
			t.Fatalf("the schema lists %v, want %v", got, want)
		}
	}
}

// TestTheDescriptionOfTheStatusToolListsTheMovesOfTheDomain proves that the
// instruction the agent reads agrees with the map of the domain: every state
// is a line, and the line holds the destinations the domain allows.
//
// The lines are rendered from the map, so the day the domain changes the map
// the instruction changes with it.
func TestTheDescriptionOfTheStatusToolListsTheMovesOfTheDomain(t *testing.T) {
	t.Parallel()

	session := connect(t, &stubUsers{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolChangeUserStatus)

	lines := make(map[string]string)
	for _, line := range strings.Split(description, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[1] != "->" {
			continue
		}
		lines[fields[0]] = strings.Join(fields[2:], " ")
	}

	for _, status := range user.AllStatuses() {
		targets := status.AllowedTargets()
		if len(targets) == 0 {
			continue
		}

		line, ok := lines[status.String()]
		if !ok {
			t.Errorf("the description does not say what %q may move to:\n%s", status, description)
			continue
		}

		for _, target := range targets {
			if !strings.Contains(line, target.String()) {
				t.Errorf("the line of %q does not hold %q: %q", status, target, line)
			}
		}
	}
}

// TestTheToolExplainsAConcurrentChangeToTheAgent proves that every error of
// the domain that the agent can see has a message in the words of the
// protocol.
func TestTheToolExplainsAConcurrentChangeToTheAgent(t *testing.T) {
	t.Parallel()

	users := &stubUsers{err: user.ErrConcurrentChange}
	session := connect(t, users)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolChangeUserStatus,
		Arguments: map[string]any{
			"document_id": "1234567",
			"status":      "active",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool did not answer an error")
	}

	message := textOf(t, result)
	if !strings.Contains(message, "at the same time") {
		t.Errorf("the message does not tell the agent that the user changed: %s", message)
	}
}
