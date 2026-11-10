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
	"github.com/witsaba/paca/packages/domain/learning"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// stubLearningRecorder is the use case that writes the log, in memory.
//
// It answers what the test tells it to answer, and it remembers what the tool
// asked it.
type stubLearningRecorder struct {
	written learning.Entry
	err     error
	got     ports.RecordLearning
	calls   int
}

// Record answers the entry of the test, and remembers what the tool asked.
func (s *stubLearningRecorder) Record(_ context.Context, request ports.RecordLearning) (learning.Entry, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return learning.Entry{}, s.err
	}

	return s.written, nil
}

// stubLearningReader is the use case that reads the log, in memory.
type stubLearningReader struct {
	log   learning.Log
	err   error
	got   ports.ReadLearnings
	calls int
}

// Read answers the log of the test, and remembers what the tool asked.
func (s *stubLearningReader) Read(_ context.Context, request ports.ReadLearnings) (learning.Log, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return learning.Log{}, s.err
	}

	return s.log, nil
}

// connectWithTheLearnings starts the transport with the stubs of the log.
func connectWithTheLearnings(
	t *testing.T,
	recorder ports.LearningRecorder,
	reader ports.LearningReader,
) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordLearning:      recorder,
		ReadLearning:        reader,
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

// theDayOfTheLearnings reads a day of the book of the tests.
func theDayOfTheLearnings(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day %q could not be read: %v", day, err)
	}

	return parsed
}

// theEntryOfTheTools is the entry the stub of the use case answers.
func theEntryOfTheTools(t *testing.T, id learning.ID, learned string, answers learning.ID) learning.Entry {
	t.Helper()

	built, err := learning.New(business.ID(77), theDayOfTheLearnings(t, "2026-03-14"), recipe.Recipe{
		ID:         recipe.ID(9),
		BusinessID: business.ID(77),
		Name:       "Pan de molde",
	}, nil, "La masa subió más lento que la receta dice", learned, nil)
	if err != nil {
		t.Fatalf("the entry was refused: %v", err)
	}

	built.ID = id
	built.Answers = answers

	return built
}

// TestTheLearningToolCarriesWhatTheAgentSends proves that the tool carries
// what the agent sends to the use case, and that the answer carries the entry
// that was written.
func TestTheLearningToolCarriesWhatTheAgentSends(t *testing.T) {
	t.Parallel()

	recorder := &stubLearningRecorder{written: theEntryOfTheTools(t, learning.ID(3), "Leudar dos horas más", 0)}
	session := connectWithTheLearnings(t, recorder, &stubLearningReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordLearning,
		Arguments: map[string]any{
			"business_id": 77,
			"day":         "2026-03-14",
			"recipe_id":   9,
			"happened":    "La masa subió más lento que la receta dice",
			"learned":     "Leudar dos horas más",
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
	if recorder.got.Day != "2026-03-14" {
		t.Errorf("the day is %q, want %q", recorder.got.Day, "2026-03-14")
	}
	if recorder.got.RecipeID != 9 {
		t.Errorf("the batch is %d, want 9", recorder.got.RecipeID)
	}
	if recorder.got.ProductionID != 0 {
		t.Errorf("the production is %d, want none: the agent named none", recorder.got.ProductionID)
	}
	if recorder.got.Happened != "La masa subió más lento que la receta dice" {
		t.Errorf("what happened is %q, want the words of the owner", recorder.got.Happened)
	}
	if recorder.got.Learned != "Leudar dos horas más" {
		t.Errorf("what was learned is %q, want the words of the owner", recorder.got.Learned)
	}
	if recorder.got.AnswersID != 0 {
		t.Errorf("the entry it answers is %d, want none", recorder.got.AnswersID)
	}

	answer := theAnswerOfTheLearning(t, result)
	if answer.ID != 3 {
		t.Errorf("the identity is %d, want 3", answer.ID)
	}
	if answer.RecipeID != 9 || answer.RecipeName != "Pan de molde" {
		t.Errorf("the batch is %d %q, want 9 %q", answer.RecipeID, answer.RecipeName, "Pan de molde")
	}
	if answer.Learned != "Leudar dos horas más" {
		t.Errorf("what was learned is %q, want the words of the owner", answer.Learned)
	}
	if answer.ProductionID != 0 {
		t.Errorf("the production is %d, want none", answer.ProductionID)
	}
	if answer.AnswersID != 0 {
		t.Errorf("the entry it answers is %d, want none", answer.AnswersID)
	}
}

// TestTheLearningToolCarriesTheProductionAndTheAnswer proves that the tool
// carries the run of the batch and the entry the new one answers: a
// correction is a NEW entry that answers the old one.
func TestTheLearningToolCarriesTheProductionAndTheAnswer(t *testing.T) {
	t.Parallel()

	recorder := &stubLearningRecorder{written: theEntryOfTheTools(t, learning.ID(4), "Leudar tres horas más", learning.ID(3))}
	session := connectWithTheLearnings(t, recorder, &stubLearningReader{})

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolRecordLearning,
		Arguments: map[string]any{
			"business_id":   77,
			"day":           "2026-03-15",
			"recipe_id":     9,
			"production_id": 4,
			"happened":      "La masa quedó dura",
			"learned":       "Leudar tres horas más",
			"answers_id":    3,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if recorder.got.ProductionID != 4 {
		t.Errorf("the production is %d, want 4", recorder.got.ProductionID)
	}
	if recorder.got.AnswersID != 3 {
		t.Errorf("the entry it answers is %d, want 3", recorder.got.AnswersID)
	}

	answer := theAnswerOfTheLearning(t, result)
	if answer.AnswersID != 3 {
		t.Errorf("the entry it answers is %d, want 3", answer.AnswersID)
	}
}

// TestTheLearningToolsRefuseNothingBeforeTheDomain proves the rule of the
// transport: every field of the two tools is OPTIONAL in the schema, and the
// call reaches the use case — the agent reads the message of the domain and
// never the refusal of a schema.
func TestTheLearningToolsRefuseNothingBeforeTheDomain(t *testing.T) {
	t.Parallel()

	recorder := &stubLearningRecorder{written: theEntryOfTheTools(t, learning.ID(3), "Leudar dos horas más", 0)}
	session := connectWithTheLearnings(t, recorder, &stubLearningReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	for _, name := range []string{mcpserver.ToolRecordLearning, mcpserver.ToolReadLearning} {
		required := theRequiredFieldsOfTheTool(t, tools.Tools, name)
		if len(required) != 0 {
			t.Errorf("the schema of %q marks %v as required: the schema must not refuse the call before the domain",
				name, required)
		}
	}
}

// TestTheLearningToolRefusesWhatTheDomainRefusesThroughItsOwnMessage proves
// that the entry with no business, with no day, with no batch, with nothing
// that happened and with nothing that was learned is refused by the domain,
// and that the answer of the agent says the rule.
//
// The schema lets the call through: the message is the one of the domain, and
// not the one of a schema that refused the call by itself.
func TestTheLearningToolRefusesWhatTheDomainRefusesThroughItsOwnMessage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    map[string]any
		failure error
		wants   []string
	}{
		{
			name: "the entry of no business",
			args: map[string]any{
				"day": "2026-03-14", "recipe_id": 9,
				"happened": "La masa subió lento", "learned": "Leudar más",
			},
			failure: learning.ErrMissingBusiness,
			wants:   []string{"belongs to no business", "PACA never invents one"},
		},
		{
			name: "the entry of no day",
			args: map[string]any{
				"business_id": 77, "recipe_id": 9,
				"happened": "La masa subió lento", "learned": "Leudar más",
			},
			failure: learning.ErrMissingDay,
			wants:   []string{"carries no day", "never invent one"},
		},
		{
			name: "the entry about no batch",
			args: map[string]any{
				"business_id": 77, "day": "2026-03-14",
				"happened": "La masa subió lento", "learned": "Leudar más",
			},
			failure: learning.ErrMissingRecipe,
			wants:   []string{"names no batch", "read the recipes of the business"},
		},
		{
			name: "the entry with nothing that happened",
			args: map[string]any{
				"business_id": 77, "day": "2026-03-14", "recipe_id": 9,
				"learned": "Leudar más",
			},
			failure: learning.ErrMissingHappened,
			wants:   []string{"says nothing happened", "in her words"},
		},
		{
			name: "the entry with nothing that was learned",
			args: map[string]any{
				"business_id": 77, "day": "2026-03-14", "recipe_id": 9,
				"happened": "La masa subió lento",
			},
			failure: learning.ErrMissingLearned,
			wants:   []string{"says nothing that was learned", "how she will bake it next time"},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			recorder := &stubLearningRecorder{err: test.failure}
			session := connectWithTheLearnings(t, recorder, &stubLearningReader{})

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      mcpserver.ToolRecordLearning,
				Arguments: test.args,
			})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool wrote an entry the domain refuses")
			}
			if recorder.calls != 1 {
				t.Fatalf("the use case was called %d times, want 1: the schema refused the call before the domain",
					recorder.calls)
			}

			answer := textOf(t, result)
			for _, want := range test.wants {
				if !strings.Contains(answer, want) {
					t.Errorf("the answer does not name the rule of the entry (%s): %s", want, answer)
				}
			}
		})
	}
}

// TestTheReadingToolAnswersTheLogAndItsCurrentEntries proves that the tool of
// the log answers the entries of the business in the order they were written,
// each one carrying the entry it answers, and the ones that are the last word
// of their chain.
func TestTheReadingToolAnswersTheLogAndItsCurrentEntries(t *testing.T) {
	t.Parallel()

	first := theEntryOfTheTools(t, learning.ID(3), "Leudar una hora más", 0)
	correction := theEntryOfTheTools(t, learning.ID(4), "Leudar dos horas más", learning.ID(3))
	alone := theEntryOfTheTools(t, learning.ID(5), "Bajar el horno a 200 grados", 0)

	reader := &stubLearningReader{log: learning.TheLog([]learning.Entry{first, correction, alone})}
	session := connectWithTheLearnings(t, &stubLearningRecorder{}, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolReadLearning,
		Arguments: map[string]any{"business_id": 77},
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

	answer := theAnswerOfTheLog(t, result)
	if len(answer.Entries) != 3 {
		t.Fatalf("the log carries %d entries, want 3", len(answer.Entries))
	}
	if answer.Entries[1].ID != 4 || answer.Entries[1].AnswersID != 3 {
		t.Errorf("the entry %+v answers no entry, want the entry 4 answering 3", answer.Entries[1])
	}
	if len(answer.Current) != 2 {
		t.Fatalf("the log carries %d current entries, want 2: %+v", len(answer.Current), answer.Current)
	}
	if answer.Current[0].ID != 4 || answer.Current[1].ID != 5 {
		t.Errorf("the current entries are %+v, want the entries 4 and 5", answer.Current)
	}
}

// TestTheReadingToolExplainsTheFailureOfTheDomain proves that every failure of
// the log reaches the agent with a message of its own, in the words of the
// caller.
func TestTheReadingToolExplainsTheFailureOfTheDomain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		failure error
		want    string
	}{
		{
			name:    "the business that is not there",
			failure: learning.ErrBusinessNotFound,
			want:    "there is no business with that identity",
		},
		{
			name:    "the business that does not keep its lessons",
			failure: learning.ErrWrongBusiness,
			want:    "the log belongs to a business of its own",
		},
		{
			name:    "the entry that was answered already",
			failure: learning.ErrAlreadyAnswered,
			want:    "was answered already",
		},
		{
			name:    "the answer of an entry that is not there",
			failure: learning.ErrEntryNotFound,
			want:    "there is no entry of the log with that identity",
		},
		{
			name:    "the row the database refused",
			failure: learning.ErrInvalidValue,
			want:    "the database refused the entry of the log",
		},
		{
			name:    "the storage that is not open",
			failure: learning.ErrNoStorage,
			want:    "the service has no storage of the log",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			session := connectWithTheLearnings(
				t,
				&stubLearningRecorder{err: test.failure},
				&stubLearningReader{err: test.failure},
			)

			for _, name := range []string{mcpserver.ToolReadLearning, mcpserver.ToolRecordLearning} {
				args := map[string]any{"business_id": 77}
				if name == mcpserver.ToolRecordLearning {
					args = map[string]any{
						"business_id": 77, "day": "2026-03-14", "recipe_id": 9,
						"happened": "La masa subió lento", "learned": "Leudar más",
					}
				}

				result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: args})
				if err != nil {
					t.Fatalf("the tool failed: %v", err)
				}
				if !result.IsError {
					t.Fatalf("the tool %q did not answer an error", name)
				}

				answer := textOf(t, result)
				if !strings.Contains(answer, test.want) {
					t.Errorf("the answer of %q does not hold %q: %s", name, test.want, answer)
				}
				if answer == test.failure.Error() {
					t.Errorf("the message of %q is the raw error: %s", name, answer)
				}
			}
		})
	}
}

// TestTheLearningToolsAnswerWithoutADatabase proves that the transport exposes
// the tools even when the service has no database, and that every call says so.
func TestTheLearningToolsAnswerWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithTheLearnings(t, nil, nil)

	cases := map[string]map[string]any{
		mcpserver.ToolRecordLearning: {
			"business_id": 77, "day": "2026-03-14", "recipe_id": 9,
			"happened": "La masa subió lento", "learned": "Leudar más",
		},
		mcpserver.ToolReadLearning: {"business_id": 77},
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

// TestTheDescriptionsOfTheLearningToolsSayWhatTheyDo proves that the
// description of every tool of the log is a contract with the agent: it says
// what the tool does, what it needs and what it answers, that the log is never
// rewritten and that a correction is a new entry that answers the old one.
func TestTheDescriptionsOfTheLearningToolsSayWhatTheyDo(t *testing.T) {
	t.Parallel()

	session := connectWithTheLearnings(t, &stubLearningRecorder{}, &stubLearningReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	descriptions := map[string][]string{
		mcpserver.ToolRecordLearning: {
			"NEVER REWRITTEN",
			"a correction is a NEW entry that answers the old one",
			"the day",
			"the batch",
			"what happened",
			"what was learned",
			"carries NO MONEY",
			"('production_id') is OPTIONAL",
			"('answers_id') is OPTIONAL",
		},
		mcpserver.ToolReadLearning: {
			"the order they were written",
			"the entry it answers",
			"the last word of their chain",
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

// TestTheToolListHoldsTheToolsOfTheLearnings proves that the transport exposes
// the two tools of the log with the names the agent reads.
func TestTheToolListHoldsTheToolsOfTheLearnings(t *testing.T) {
	t.Parallel()

	session := connectWithTheLearnings(t, &stubLearningRecorder{}, &stubLearningReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	want := []string{mcpserver.ToolReadLearning, mcpserver.ToolRecordLearning}

	exposed := make([]string, 0, len(want))
	for _, tool := range tools.Tools {
		if slices.Contains(want, tool.Name) {
			exposed = append(exposed, tool.Name)
		}
	}

	slices.Sort(exposed)
	if !slices.Equal(exposed, want) {
		t.Fatalf("the server exposes the tools of the log %v, want %v", exposed, want)
	}
}

// learningAnswer is the entry the tools of the log answer.
type learningAnswer struct {
	ID           int64  `json:"id"`
	BusinessID   int64  `json:"business_id"`
	Day          string `json:"day"`
	RecipeID     int64  `json:"recipe_id"`
	RecipeName   string `json:"recipe_name"`
	ProductionID int64  `json:"production_id"`
	Happened     string `json:"happened"`
	Learned      string `json:"learned"`
	AnswersID    int64  `json:"answers_id"`
}

// recordLearningAnswer is the answer of the tool that writes an entry.
type recordLearningAnswer struct {
	Learning learningAnswer `json:"learning"`
}

// logAnswer is the answer of the tool that reads the log.
type logAnswer struct {
	Entries []learningAnswer `json:"entries"`
	Current []learningAnswer `json:"current"`
}

// theAnswerOfTheLearning reads the entry of the answer of a tool of the log.
func theAnswerOfTheLearning(t *testing.T, result *sdk.CallToolResult) learningAnswer {
	t.Helper()

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	answer := recordLearningAnswer{}
	if err := json.Unmarshal(raw, &answer); err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	return answer.Learning
}

// theAnswerOfTheLog reads the log of the answer of the tool of the log.
func theAnswerOfTheLog(t *testing.T, result *sdk.CallToolResult) logAnswer {
	t.Helper()

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	answer := logAnswer{}
	if err := json.Unmarshal(raw, &answer); err != nil {
		t.Fatalf("the answer of the tool could not be read: %v", err)
	}

	return answer
}
