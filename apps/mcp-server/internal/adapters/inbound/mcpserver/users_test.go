package mcpserver_test

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
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
	if len(tools.Tools) != 5 {
		t.Errorf("the server exposes %d tools, want 5", len(tools.Tools))
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

// lockedBuffer is a buffer that the goroutine of the request and the test
// may write to and read at the same time.
type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

// Write keeps the bytes the server wrote.
func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.buffer.Write(p)
}

// String returns what the server wrote so far.
func (b *lockedBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.buffer.String()
}

// TestTheToolExplainsEveryErrorOfTheDomainToTheAgent proves that every
// error the port of the storage can return reaches the agent with a message
// of its own.
//
// The catalogue of the domain names the errors. An error that no message
// explains would reach the agent as the text of a Go error, or as the
// message of a failure the service does not know: an agent cannot do
// anything with either of them.
func TestTheToolExplainsEveryErrorOfTheDomainToTheAgent(t *testing.T) {
	t.Parallel()

	theUnknown, _ := theMessageOfTheFailure(t, errors.New("a failure the service does not know"))

	for _, failure := range user.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			message, logs := theMessageOfTheFailure(t, failure)

			if message == "" {
				t.Fatal("the tool answered no message")
			}
			if message == failure.Error() {
				t.Fatalf("the message is the raw error: %s", message)
			}
			if message == theUnknown {
				t.Fatalf("the error of the domain has no message of its own: %s", message)
			}
			if strings.Contains(logs, "has no message for the agent") {
				t.Fatalf("the service did not map the error of the domain: %s", logs)
			}
		})
	}
}

// theMessageOfTheFailure calls a tool that fails and returns the message the
// agent reads.
func theMessageOfTheFailure(t *testing.T, failure error) (string, string) {
	t.Helper()

	logs := &lockedBuffer{}
	users := &stubUsers{err: failure}
	session := connectWithLogger(t, users, slog.New(slog.NewTextHandler(logs, nil)))

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

	return textOf(t, result), logs.String()
}

// TestTheToolDoesNotShowTheRawTextOfAnUnknownFailure proves that the text of
// a Go error never reaches the agent, and that the log of the service keeps
// the cause.
//
// The agent reads a message in its own words; the operator reads the cause
// in the log. Neither of them loses what it needs.
func TestTheToolDoesNotShowTheRawTextOfAnUnknownFailure(t *testing.T) {
	t.Parallel()

	logs := &lockedBuffer{}
	logger := slog.New(slog.NewTextHandler(logs, nil))

	users := &stubUsers{err: errors.New("the connection pool is closed")}
	session := connectWithLogger(t, users, logger)

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
	if message == "" {
		t.Fatal("the tool answered no message")
	}
	if strings.Contains(message, "connection pool") {
		t.Errorf("the message carries the raw error: %s", message)
	}
	if !strings.Contains(logs.String(), "connection pool is closed") {
		t.Errorf("the log does not carry the cause: %s", logs.String())
	}
}

// TestTheToolReadsTheDataOfThePerson proves that the tool answers what
// PACA knows about the person, and that it changes nothing.
func TestTheToolReadsTheDataOfThePerson(t *testing.T) {
	t.Parallel()

	users := &stubUsers{found: user.User{
		ID:        1234567,
		FullName:  "Ana María Pérez",
		Phone:     "573212539984",
		Email:     "ana@example.com",
		Status:    user.StatusActive,
		CreatedAt: time.Date(2026, time.October, 29, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.October, 29, 11, 0, 0, 0, time.UTC),
	}}
	session := connect(t, users)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolGetUser,
		Arguments: map[string]any{
			"document_id": "1234567",
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

	answer := textOf(t, result)
	for _, want := range []string{"1234567", "Ana María Pérez", "573212539984", "ana@example.com", "active"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheToolSaysThatNoUserCarriesTheDocument proves that the read of a
// document that nobody carries is an error the agent understands.
func TestTheToolSaysThatNoUserCarriesTheDocument(t *testing.T) {
	t.Parallel()

	users := &stubUsers{err: user.ErrNotFound}
	session := connect(t, users)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolGetUser,
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

	if !strings.Contains(textOf(t, result), "no user with that document ID") {
		t.Errorf("the message does not say that nobody carries the document: %s", textOf(t, result))
	}
}

// TestTheToolAnswersThePeoplePACAKnows proves that the tool answers the
// people PACA knows, and that it asks for as many of them as the agent
// wants.
func TestTheToolAnswersThePeoplePACAKnows(t *testing.T) {
	t.Parallel()

	users := &stubUsers{listed: []user.User{
		{ID: 1234567, FullName: "Ana María Pérez", Status: user.StatusActive},
		{ID: 7654321, FullName: "Luis Gómez", Status: user.StatusOnboarding},
	}}
	session := connect(t, users)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolListUsers,
		Arguments: map[string]any{
			"limit": 2,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if users.listedLimit != 2 {
		t.Errorf("the limit is %d, want %d", users.listedLimit, 2)
	}

	answer := textOf(t, result)
	for _, want := range []string{"1234567", "Ana María Pérez", "7654321", "Luis Gómez"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}
