package mcpserver_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/commitment"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// stubCommitmentRecorder is the use case that writes the promises of the
// bakery, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tools asked it.
type stubCommitmentRecorder struct {
	written commitment.Commitment
	met     commitment.Commitment
	err     error
	got     ports.RecordCommitment
	gotMeet ports.MeetCommitment
	calls   int
}

// Record answers the promise of the test, and remembers what the tool asked.
func (s *stubCommitmentRecorder) Record(_ context.Context, request ports.RecordCommitment) (commitment.Commitment, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return commitment.Commitment{}, s.err
	}

	return s.written, nil
}

// Meet answers the promise that was kept, and remembers what the tool asked.
func (s *stubCommitmentRecorder) Meet(_ context.Context, request ports.MeetCommitment) (commitment.Commitment, error) {
	s.calls++
	s.gotMeet = request

	if s.err != nil {
		return commitment.Commitment{}, s.err
	}

	return s.met, nil
}

// stubCommitmentReader is the use case that reads the calendar of the
// promises, in memory.
type stubCommitmentReader struct {
	calendar commitment.Calendar
	err      error
	got      ports.ReadCommitments
	calls    int
}

// Read answers the calendar of the test, and remembers what the tool asked.
func (s *stubCommitmentReader) Read(_ context.Context, request ports.ReadCommitments) (commitment.Calendar, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return commitment.Calendar{}, s.err
	}

	return s.calendar, nil
}

// connectWithTheCommitments starts the transport with the stubs of the
// promises.
func connectWithTheCommitments(
	t *testing.T,
	recorder ports.CommitmentRecorder,
	reader ports.CommitmentReader,
) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordCommitment:    recorder,
		ReadCommitments:     reader,
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

// theDayOfThePromises reads a day of the book of the tests.
func theDayOfThePromises(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day %q could not be read: %v", day, err)
	}

	return parsed
}

// thePromiseOfTheTools is the promise the stub of the use case answers.
func thePromiseOfTheTools(t *testing.T) commitment.Commitment {
	t.Helper()

	built, err := commitment.New(business.ID(77), "El pedido de la tienda de la esquina", theDayOfThePromises(t, "2026-03-14"))
	if err != nil {
		t.Fatalf("the promise was refused: %v", err)
	}

	built.ID = commitment.ID(9)

	return built
}

// TestTheCommitmentToolCarriesWhatTheAgentSends proves that the tool carries
// what the agent sends to the use case, and that the answer carries the
// promise that was written.
func TestTheCommitmentToolCarriesWhatTheAgentSends(t *testing.T) {
	t.Parallel()

	recorder := &stubCommitmentRecorder{written: thePromiseOfTheTools(t)}
	session := connectWithTheCommitments(t, recorder, &stubCommitmentReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordCommitment,
		Arguments: map[string]any{
			"business_id": 77,
			"what":        "El pedido de la tienda de la esquina",
			"due_on":      "2026-03-14",
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
	if recorder.got.What != "El pedido de la tienda de la esquina" {
		t.Errorf("the what is %q, want the words of the owner", recorder.got.What)
	}
	if recorder.got.DueOn != "2026-03-14" {
		t.Errorf("the day it is due is %q, want %q", recorder.got.DueOn, "2026-03-14")
	}

	answer := theAnswerOfTheCommitment(t, result)
	if answer.ID != 9 {
		t.Errorf("the identity is %d, want 9", answer.ID)
	}
	if answer.What != "El pedido de la tienda de la esquina" {
		t.Errorf("the what is %q, want the words of the owner", answer.What)
	}
	if answer.DueOn != "2026-03-14" {
		t.Errorf("the day it is due is %q, want %q", answer.DueOn, "2026-03-14")
	}
	if answer.MetOn != "" {
		t.Errorf("the day it was met is %q, want none: the promise is open", answer.MetOn)
	}
	if !answer.Open {
		t.Error("the answer says the promise is not open, and nobody met it")
	}
}

// TestTheMeetToolCarriesWhatTheAgentSends proves that the meeting names the
// promise by its own identity, and that the answer carries the promise that
// was kept with the day it was kept.
func TestTheMeetToolCarriesWhatTheAgentSends(t *testing.T) {
	t.Parallel()

	promise := thePromiseOfTheTools(t)
	kept, err := promise.Met(theDayOfThePromises(t, "2026-03-13"))
	if err != nil {
		t.Fatalf("the meeting of the test was refused: %v", err)
	}

	recorder := &stubCommitmentRecorder{met: kept}
	session := connectWithTheCommitments(t, recorder, &stubCommitmentReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMeetCommitment,
		Arguments: map[string]any{
			"business_id":   77,
			"commitment_id": 9,
			"met_on":        "2026-03-13",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if recorder.gotMeet.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", recorder.gotMeet.BusinessID)
	}
	if recorder.gotMeet.CommitmentID != 9 {
		t.Errorf("the promise is %d, want 9: the promise is named by its own identity", recorder.gotMeet.CommitmentID)
	}
	if recorder.gotMeet.MetOn != "2026-03-13" {
		t.Errorf("the day it was met is %q, want %q", recorder.gotMeet.MetOn, "2026-03-13")
	}

	answer := theAnswerOfTheCommitment(t, result)
	if answer.MetOn != "2026-03-13" {
		t.Errorf("the day it was met is %q, want %q", answer.MetOn, "2026-03-13")
	}
	if answer.Open {
		t.Error("the answer says the promise is open, and it was kept")
	}
}

// TestTheCommitmentToolsRefuseNothingBeforeTheDomain proves the rule of the
// transport: the fields the domain refuses are OPTIONAL in the schema, and
// the call reaches the use case — the agent reads the message of the domain
// and never the refusal of a schema.
//
// The two promises and the meeting are the ones the domain refuses: no
// business, no what, no day, and no day of meeting.
func TestTheCommitmentToolsRefuseNothingBeforeTheDomain(t *testing.T) {
	t.Parallel()

	recorder := &stubCommitmentRecorder{written: thePromiseOfTheTools(t), met: thePromiseOfTheTools(t)}
	session := connectWithTheCommitments(t, recorder, &stubCommitmentReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	for _, name := range []string{mcpserver.ToolRecordCommitment, mcpserver.ToolMeetCommitment, mcpserver.ToolReadCommitments} {
		required := theRequiredFieldsOfTheTool(t, tools.Tools, name)
		if len(required) != 0 {
			t.Errorf("the schema of %q marks %v as required: the schema must not refuse the call before the domain",
				name, required)
		}
	}
}

// TestThePromiseToolRefusesWhatTheDomainRefusesThroughItsOwnMessage proves
// that a promise with no business, with no "what" and with no day is refused
// by the domain, and that the answer of the agent says the rule.
//
// The schema lets the call through: the message is the one of the domain, and
// not the one of a schema that refused the call by itself.
func TestThePromiseToolRefusesWhatTheDomainRefusesThroughItsOwnMessage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    map[string]any
		failure error
		wants   []string
	}{
		{
			name:    "the promise of no business",
			args:    map[string]any{"what": "El pan del domingo", "due_on": "2026-03-14"},
			failure: commitment.ErrMissingBusiness,
			wants:   []string{"belongs to no business", "PACA never invents one"},
		},
		{
			name:    "the promise that says nothing",
			args:    map[string]any{"business_id": 77, "due_on": "2026-03-14"},
			failure: commitment.ErrMissingWhat,
			wants:   []string{"says nothing about what is due", "in her words"},
		},
		{
			name:    "the promise without a day",
			args:    map[string]any{"business_id": 77, "what": "El pan del domingo"},
			failure: commitment.ErrMissingDay,
			wants:   []string{"carries no day", "never invent one"},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			recorder := &stubCommitmentRecorder{err: test.failure}
			session := connectWithTheCommitments(t, recorder, &stubCommitmentReader{})

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      mcpserver.ToolRecordCommitment,
				Arguments: test.args,
			})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool wrote a promise the domain refuses")
			}
			if recorder.calls != 1 {
				t.Fatalf("the use case was called %d times, want 1: the schema refused the call before the domain",
					recorder.calls)
			}

			answer := textOf(t, result)
			for _, want := range test.wants {
				if !strings.Contains(answer, want) {
					t.Errorf("the answer does not name the rule of the promise (%s): %s", want, answer)
				}
			}
		})
	}
}

// TestTheMeetingToolRefusesThePromiseThatWasMetThroughItsOwnMessage proves
// that the second meeting of a promise is refused by the domain, and that the
// answer of the agent says the rule.
func TestTheMeetingToolRefusesThePromiseThatWasMetThroughItsOwnMessage(t *testing.T) {
	t.Parallel()

	recorder := &stubCommitmentRecorder{err: commitment.ErrAlreadyMet}
	session := connectWithTheCommitments(t, recorder, &stubCommitmentReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMeetCommitment,
		Arguments: map[string]any{
			"business_id":   77,
			"commitment_id": 9,
			"met_on":        "2026-03-15",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool met a promise that was met already")
	}

	answer := textOf(t, result)
	for _, want := range []string{"met already", "with the day it was kept", "kept once"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not name the rule of the meeting (%s): %s", want, answer)
		}
	}
}

// TestTheCalendarToolCarriesTheDayAndAnswersTheCalendar proves that the tool
// of the calendar carries the day the agent reads, and that the answer
// carries the promises due on it and the ones that are still owed.
func TestTheCalendarToolCarriesTheDayAndAnswersTheCalendar(t *testing.T) {
	t.Parallel()

	promise := thePromiseOfTheTools(t)
	overdue, err := commitment.New(business.ID(77), "El pan del domingo", theDayOfThePromises(t, "2026-03-07"))
	if err != nil {
		t.Fatalf("the promise was refused: %v", err)
	}
	overdue.ID = commitment.ID(8)

	reader := &stubCommitmentReader{calendar: commitment.Calendar{
		Day:  theDayOfThePromises(t, "2026-03-14"),
		Due:  []commitment.Commitment{promise},
		Owed: []commitment.Commitment{promise, overdue},
	}}
	session := connectWithTheCommitments(t, &stubCommitmentRecorder{}, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadCommitments,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-14",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if reader.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", reader.got.BusinessID)
	}
	if reader.got.Day != "2026-03-14" {
		t.Errorf("the day is %q, want %q", reader.got.Day, "2026-03-14")
	}

	answer := theAnswerOfTheCalendar(t, result)
	if answer.Day != "2026-03-14" {
		t.Errorf("the day of the answer is %q, want %q", answer.Day, "2026-03-14")
	}
	if len(answer.Due) != 1 || answer.Due[0].ID != 9 {
		t.Errorf("the calendar carries the promises due %+v, want the promise 9", answer.Due)
	}
	if len(answer.Owed) != 2 {
		t.Fatalf("the calendar carries %d promises owed, want 2", len(answer.Owed))
	}
	if answer.Owed[0].ID != 9 || answer.Owed[1].ID != 8 {
		t.Errorf("the calendar carries the promises owed %+v, want the promises 9 and 8", answer.Owed)
	}
}

// TestTheCalendarToolReadsTheWholeBookWhenTheAgentNamesNoDay proves that the
// day of the calendar is optional: the whole book of the promises is read
// when the agent names none, and the answer carries no day.
func TestTheCalendarToolReadsTheWholeBookWhenTheAgentNamesNoDay(t *testing.T) {
	t.Parallel()

	reader := &stubCommitmentReader{calendar: commitment.Calendar{
		Due:  []commitment.Commitment{thePromiseOfTheTools(t)},
		Owed: []commitment.Commitment{thePromiseOfTheTools(t)},
	}}
	session := connectWithTheCommitments(t, &stubCommitmentRecorder{}, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolReadCommitments,
		Arguments: map[string]any{"business_id": 77},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if reader.got.Day != "" {
		t.Errorf("the day is %q, want none", reader.got.Day)
	}

	answer := theAnswerOfTheCalendar(t, result)
	if answer.Day != "" {
		t.Errorf("the day of the answer is %q, want none: the whole book was read", answer.Day)
	}
}

// TestTheCommitmentToolsExplainTheFailureOfTheDomain proves that every error
// of the domain of the commitments reaches the agent with a message of its
// own, in the words of the caller.
func TestTheCommitmentToolsExplainTheFailureOfTheDomain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		tool    string
		args    map[string]any
		failure error
		want    string
	}{
		{
			name:    "the promise that is not there",
			tool:    mcpserver.ToolMeetCommitment,
			args:    map[string]any{"business_id": 77, "commitment_id": 9, "met_on": "2026-03-14"},
			failure: commitment.ErrNotFound,
			want:    "there is no commitment with that identity",
		},
		{
			name:    "the meeting without its day",
			tool:    mcpserver.ToolMeetCommitment,
			args:    map[string]any{"business_id": 77, "commitment_id": 9},
			failure: commitment.ErrMissingMetDay,
			want:    "carries no day of meeting",
		},
		{
			name:    "the business that does not hold promises",
			tool:    mcpserver.ToolRecordCommitment,
			args:    map[string]any{"business_id": 77, "what": "El pan", "due_on": "2026-03-14"},
			failure: commitment.ErrWrongBusiness,
			want:    "the commitments belong to a business of its own",
		},
		{
			name:    "the business of the promises that is not there",
			tool:    mcpserver.ToolReadCommitments,
			args:    map[string]any{"business_id": 77},
			failure: commitment.ErrBusinessNotFound,
			want:    "there is no business with that identity",
		},
		{
			name:    "the row the database refused",
			tool:    mcpserver.ToolRecordCommitment,
			args:    map[string]any{"business_id": 77, "what": "El pan", "due_on": "2026-03-14"},
			failure: commitment.ErrInvalidValue,
			want:    "the database refused the promise",
		},
		{
			name:    "the storage that is not open",
			tool:    mcpserver.ToolReadCommitments,
			args:    map[string]any{"business_id": 77},
			failure: commitment.ErrNoStorage,
			want:    "the service has no storage of the commitments",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			session := connectWithTheCommitments(
				t,
				&stubCommitmentRecorder{err: test.failure},
				&stubCommitmentReader{err: test.failure},
			)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      test.tool,
				Arguments: test.args,
			})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool did not answer an error")
			}

			answer := textOf(t, result)
			if !strings.Contains(answer, test.want) {
				t.Errorf("the answer does not hold %q: %s", test.want, answer)
			}
			if answer == test.failure.Error() {
				t.Errorf("the message is the raw error: %s", answer)
			}
		})
	}
}

// TestTheCommitmentToolsAnswerWithoutADatabase proves that the transport
// exposes the tools even when the service has no database, and that every call
// says so.
func TestTheCommitmentToolsAnswerWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithTheCommitments(t, nil, nil)

	cases := map[string]map[string]any{
		mcpserver.ToolRecordCommitment: {"business_id": 77, "what": "El pan", "due_on": "2026-03-14"},
		mcpserver.ToolMeetCommitment:   {"business_id": 77, "commitment_id": 9, "met_on": "2026-03-14"},
		mcpserver.ToolReadCommitments:  {"business_id": 77},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: args})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool did not answer an error")
			}
			if !strings.Contains(textOf(t, result), "database") {
				t.Errorf("the message does not talk about the database: %s", textOf(t, result))
			}
		})
	}
}

// TestTheDescriptionsOfTheCommitmentToolsSayWhatTheyDo proves that the
// description of every tool of the commitments is a contract with the agent:
// it says what the tool does, what it needs and what it answers, and it names
// the alarm as the work of the agent.
func TestTheDescriptionsOfTheCommitmentToolsSayWhatTheyDo(t *testing.T) {
	t.Parallel()

	session := connectWithTheCommitments(t, &stubCommitmentRecorder{}, &stubCommitmentReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	descriptions := map[string][]string{
		mcpserver.ToolRecordCommitment: {
			"What is due",
			"in the words of the owner",
			"the day it is due",
			"written OPEN",
			"carries NO MONEY and NO PARTY",
			"The alarm is not PACA's",
		},
		mcpserver.ToolMeetCommitment: {
			"stays in the book with the day it was kept",
			"the second meeting of the same promise is refused",
			"ITS OWN IDENTITY",
			"never merges them",
		},
		mcpserver.ToolReadCommitments: {
			"the promises that are due on the day",
			"every open promise of the business",
			"The alarm is the work of the agent",
			"It writes nothing",
		},
	}

	for name, wants := range descriptions {
		description := descriptionOf(t, tools.Tools, name)
		for _, want := range wants {
			if !strings.Contains(description, want) {
				t.Errorf("the description of %q does not say %q:\n%s", name, want, description)
			}
		}
	}
}

// commitmentAnswer is the promise the tools of the commitments answer.
type commitmentAnswer struct {
	ID         int64  `json:"id"`
	BusinessID int64  `json:"business_id"`
	What       string `json:"what"`
	DueOn      string `json:"due_on"`
	MetOn      string `json:"met_on"`
	Open       bool   `json:"open"`
}

// recordCommitmentAnswer is the answer of the tool that records a promise.
type recordCommitmentAnswer struct {
	Commitment commitmentAnswer `json:"commitment"`
}

// calendarAnswer is the answer of the tool that reads the calendar of the
// promises.
type calendarAnswer struct {
	Day  string             `json:"day"`
	Due  []commitmentAnswer `json:"due"`
	Owed []commitmentAnswer `json:"owed"`
}

// theAnswerOfTheCommitment reads the promise of the answer of a tool of the
// commitments.
func theAnswerOfTheCommitment(t *testing.T, result *sdk.CallToolResult) commitmentAnswer {
	t.Helper()

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	answer := recordCommitmentAnswer{}
	if err := json.Unmarshal(raw, &answer); err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	return answer.Commitment
}

// theAnswerOfTheCalendar reads the calendar of the answer of the tool of the
// calendar.
func theAnswerOfTheCalendar(t *testing.T, result *sdk.CallToolResult) calendarAnswer {
	t.Helper()

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	answer := calendarAnswer{}
	if err := json.Unmarshal(raw, &answer); err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	return answer
}

// theNamesOfTheCommitmentTools is the list of the tools of the commitments,
// in the order they are presented.
func theNamesOfTheCommitmentTools() []string {
	return []string{
		mcpserver.ToolRecordCommitment,
		mcpserver.ToolMeetCommitment,
		mcpserver.ToolReadCommitments,
	}
}

// TestTheToolListHoldsTheToolsOfTheCommitments proves that the transport
// exposes the three tools of the promises with the names the agent reads.
func TestTheToolListHoldsTheToolsOfTheCommitments(t *testing.T) {
	t.Parallel()

	session := connectWithTheCommitments(t, &stubCommitmentRecorder{}, &stubCommitmentReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	exposed := make([]string, 0, len(theNamesOfTheCommitmentTools()))
	for _, tool := range tools.Tools {
		if slices.Contains(theNamesOfTheCommitmentTools(), tool.Name) {
			exposed = append(exposed, tool.Name)
		}
	}

	slices.Sort(exposed)
	want := theNamesOfTheCommitmentTools()
	slices.Sort(want)

	if !slices.Equal(exposed, want) {
		t.Fatalf("the server exposes the tools of the commitments %v, want %v", exposed, want)
	}
}
