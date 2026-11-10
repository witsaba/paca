package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/learning"
)

// The names of the tools of the learning log.
//
// The names are the ones the agent reads in the list of the tools, so they say
// what the tool does and not what it writes: a tool that records a lesson is
// not a tool that creates a row.
const (
	// ToolRecordLearning writes an entry of the log: the day, the batch, what
	// happened and what was learned, in the words of the owner.
	ToolRecordLearning = "record_learning"

	// ToolReadLearning reads the log of a business: the entries in the order
	// they were written, and the ones that are the last word of their chain.
	ToolReadLearning = "read_learning"
)

// The descriptions of the tools of the learning log.
//
// The descriptions are a contract with the agent: they say what the tool does,
// what it needs and what it answers, and nothing it does not do.
var (
	descriptionRecordLearning = buildDescriptionRecordLearning()
	descriptionReadLearning   = buildDescriptionReadLearning()
)

// buildDescriptionRecordLearning renders the description of the tool that
// writes an entry of the log.
//
// The description is a contract with the agent: it says what the entry is,
// what the tool needs and what it answers, that the log is never rewritten,
// and that a correction is a new entry that answers the old one.
func buildDescriptionRecordLearning() string {
	return "Record what happened with the bread and what was learned from it: the learning log of the bakery, " +
		"in the words of the owner.\n\n" +
		"The log is the memory of the errors and the wins of the owner — the batch that rose slower than the recipe " +
		"says, the oven that ran hot, the dough that was too stiff — and it is what PACA learns from. It is not a " +
		"sale and it is not a movement of the book: the entry carries NO MONEY, and it never moves a peso.\n\n" +
		"The log is NEVER REWRITTEN: an entry is not updated and it is not deleted, and a correction is a NEW entry " +
		"that answers the old one with 'answers_id'. The entry that was answered keeps its own words for ever, and " +
		"the chain of the answers says which word is the last one — the last word is what the owner already knows.\n\n" +
		"The business ('business_id') is REQUIRED: it is the business of its own, the one that sells the bread. The " +
		"Personal Business of a person holds the money of that person and not the lessons of the bakery, and PACA " +
		"refuses the entry of it.\n\n" +
		"The day ('day') is REQUIRED: the day the entry is about, written as '2026-03-14'. PACA never invents one.\n\n" +
		"The batch ('recipe_id') is REQUIRED: it is the recipe the entry is about, a recipe of the business. A batch " +
		"the business does not hold is refused, and so is the batch of another business.\n\n" +
		"The production ('production_id') is OPTIONAL: it names ONE run of the batch, when the entry is about that " +
		"run and not about the batch as a whole. The production is a production of the business and a run of the " +
		"batch the entry names, and PACA refuses it otherwise. Leave it out when the entry is about the batch alone.\n\n" +
		"What happened ('happened') is REQUIRED: what happened with the bread, in the words of the owner. An entry " +
		"that says nothing is not an entry of the log, and PACA refuses it. PACA keeps the words with no space " +
		"around them, and it never invents them.\n\n" +
		"What was learned ('learned') is REQUIRED: what the owner learned from it, in her words — the lesson she " +
		"will bake with next time. The log is what PACA learns from, and the entry that taught nothing is refused.\n\n" +
		"The entry it answers ('answers_id') is OPTIONAL: the identity the log answered, the 'id' of the entry this " +
		"one corrects. The entry it answers must be an entry of the same business, and it must not have been " +
		"answered already: the log is a chain and not a tree, and two answers of the same entry would leave the " +
		"owner with two truths. Leave it out when the entry corrects nothing.\n\n" +
		"The answer carries the entry that was written: its identity ('id'), the business ('business_id'), the day " +
		"('day'), the batch with the name the owner reads ('recipe_id', 'recipe_name'), the production " +
		"('production_id', absent when the entry is about the batch alone), what happened ('happened'), what was " +
		"learned ('learned') and the entry it answers ('answers_id', absent when it answers none)."
}

// buildDescriptionReadLearning renders the description of the tool that reads
// the log.
//
// The description is a contract with the agent: it says what the log carries,
// what the tool needs and what it answers, and nothing it does not do.
func buildDescriptionReadLearning() string {
	return "Read the learning log of the bakery: the memory of the errors and the wins of the owner.\n\n" +
		"The log answers every entry of the business, in the order they were written, each one carrying the entry " +
		"it answers. The log is never rewritten, so the corrections of the past are in it and they are not hidden: " +
		"an entry that was corrected keeps its own words.\n\n" +
		"The log also answers the entries that are the LAST WORD of their chain ('current'): the ones no other " +
		"entry answers. They are what the owner already knows, and what Paca tells her when she asks what she " +
		"learned: the batch that rose slower than the recipe says, the oven that ran hot, the dough that was too " +
		"stiff. An entry belongs to ONE chain: the entry that was never corrected and the correction that nobody " +
		"corrected are the last word of their own chain.\n\n" +
		"The business ('business_id') is REQUIRED: it is the business whose log is read, and PACA never invents one. " +
		"A business that is not there is refused: the answer is never an empty log for a business that does not " +
		"exist. The Personal Business of a person keeps the money of that person and not the lessons of the " +
		"bakery, and its log is refused.\n\n" +
		"The answer carries the entries of the business ('entries') and the ones that are the last word of their " +
		"chain ('current'). Every entry carries its identity ('id'), the business ('business_id'), the day ('day'), " +
		"the batch with the name the owner reads ('recipe_id', 'recipe_name'), the production when it is about one " +
		"run of the batch ('production_id'), what happened ('happened'), what was learned ('learned') and the entry " +
		"it answers ('answers_id', absent when it answers none).\n\n" +
		"It writes nothing."
}

// recordLearningInput is the input of the tool that writes an entry of the log.
//
// Every field is OPTIONAL in the schema: the domain is the one that refuses the
// entry without a business, without a day, without a batch, with nothing that
// happened and with nothing that was learned, and the agent reads the message
// of the domain.
type recordLearningInput struct {
	// BusinessID is the business that owns the memory.
	BusinessID int64 `json:"business_id,omitempty" jsonschema:"The identity of the business that owns the log: the business of its own, the one that sells the bread. Send it always: PACA never invents one, and the domain refuses the entry when it is left out."`

	// Day is the day the entry is about.
	Day string `json:"day,omitempty" jsonschema:"The day the entry is about, written as '2026-03-14'. Send it always: PACA never invents one, and the domain refuses the entry when it is left out."`

	// RecipeID is the batch the entry is about.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the batch the entry is about: the recipe of the business she baked. Send it always: the entry is about the bread, and the domain refuses the entry about no batch."`

	// ProductionID is the run of the batch the entry is about.
	ProductionID int64 `json:"production_id,omitempty" jsonschema:"The identity of the production the entry is about: one run of the batch. It is optional: leave it out when the entry is about the batch as a whole. The production is a production of the business and a run of the batch the entry names, and the domain refuses it otherwise."`

	// Happened is what happened, in the words of the owner.
	Happened string `json:"happened,omitempty" jsonschema:"What happened with the bread, in the words of the owner ('la masa subió más lento que la receta dice'). Send it always: the entry that says nothing is not an entry of the log, and the domain refuses it when it is left out."`

	// Learned is what was learned, in the words of the owner.
	Learned string `json:"learned,omitempty" jsonschema:"What the owner learned from it, in her words ('leudar dos horas más en invierno'). Send it always: the log is what PACA learns from, and the domain refuses the entry that taught nothing."`

	// AnswersID is the entry this one answers.
	AnswersID int64 `json:"answers_id,omitempty" jsonschema:"The identity of the entry this one corrects, the 'id' the log answered. It is optional: leave it out when the entry corrects nothing. The entry it answers must be of the same business and must not have been answered already, because the log is a chain and not a tree."`
}

// readLearningInput is the input of the tool that reads the log.
type readLearningInput struct {
	// BusinessID is the business whose log is read.
	BusinessID int64 `json:"business_id,omitempty" jsonschema:"The identity of the business whose log is read. Send it always: PACA never invents one, and the domain refuses the reading when it is left out."`
}

// learningOutput is what the tools of the log answer about one entry.
type learningOutput struct {
	// ID is the identity of the entry.
	ID int64 `json:"id"`

	// BusinessID is the business that owns the memory.
	BusinessID int64 `json:"business_id"`

	// Day is the day the entry is about.
	Day string `json:"day"`

	// RecipeID is the batch the entry is about.
	RecipeID int64 `json:"recipe_id"`

	// RecipeName is the name the owner reads of the batch.
	RecipeName string `json:"recipe_name"`

	// ProductionID is the run of the batch the entry is about. It is absent
	// when the entry is about the batch alone.
	ProductionID int64 `json:"production_id,omitempty"`

	// Happened is what happened, in the words of the owner.
	Happened string `json:"happened"`

	// Learned is what was learned, in the words of the owner.
	Learned string `json:"learned"`

	// AnswersID is the entry this one answers. It is absent when the entry
	// answers no other one.
	AnswersID int64 `json:"answers_id,omitempty"`
}

// recordLearningOutput is the answer of the tool that writes an entry.
type recordLearningOutput struct {
	// Learning is the entry that was written.
	Learning learningOutput `json:"learning"`
}

// readLearningOutput is the answer of the tool that reads the log.
type readLearningOutput struct {
	// Entries are the entries of the business, in the order they were
	// written.
	Entries []learningOutput `json:"entries"`

	// Current are the entries that are the last word of their chain: the
	// ones the owner already knows.
	Current []learningOutput `json:"current"`
}

// registerLearningTools registers the tools of the learning log.
//
// The tools are registered even when the service has no database: the list of
// the tools is a property of the server, and a call without a database answers
// that the database is not there.
func (s *Server) registerLearningTools(recorder ports.LearningRecorder, reader ports.LearningReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordLearning,
		Description: descriptionRecordLearning,
	}, s.recordLearning(recorder))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadLearning,
		Description: descriptionReadLearning,
	}, s.readLearning(reader))
}

// recordLearning is the handler of the tool that writes an entry of the log.
//
// The tool writes the entry of the business and answers with it. The log is
// never rewritten: the entry answers another one when the agent names it, and
// no entry is ever updated.
func (s *Server) recordLearning(recorder ports.LearningRecorder) sdk.ToolHandlerFor[recordLearningInput, recordLearningOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordLearningInput) (*sdk.CallToolResult, recordLearningOutput, error) {
		if recorder == nil {
			return nil, recordLearningOutput{}, ports.ErrNoDatabase
		}

		written, err := recorder.Record(ctx, ports.RecordLearning{
			BusinessID:   businessIDOf(input.BusinessID),
			Day:          input.Day,
			RecipeID:     input.RecipeID,
			ProductionID: input.ProductionID,
			Happened:     input.Happened,
			Learned:      input.Learned,
			AnswersID:    input.AnswersID,
		})
		if err != nil {
			return nil, recordLearningOutput{}, s.toolError(err)
		}

		return nil, recordLearningOutput{Learning: outputOfLearning(written)}, nil
	}
}

// readLearning is the handler of the tool that reads the log.
//
// The tool reads the log of the business: it changes nothing, and it answers
// with the entries in the order they were written and with the ones that are
// the last word of their chain.
func (s *Server) readLearning(reader ports.LearningReader) sdk.ToolHandlerFor[readLearningInput, readLearningOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readLearningInput) (*sdk.CallToolResult, readLearningOutput, error) {
		if reader == nil {
			return nil, readLearningOutput{}, ports.ErrNoDatabase
		}

		log, err := reader.Read(ctx, ports.ReadLearnings{BusinessID: businessIDOf(input.BusinessID)})
		if err != nil {
			return nil, readLearningOutput{}, s.toolError(err)
		}

		entries := make([]learningOutput, 0, len(log.Entries))
		for _, one := range log.Entries {
			entries = append(entries, outputOfLearning(one))
		}

		current := make([]learningOutput, 0, len(log.Entries))
		for _, one := range log.Current() {
			current = append(current, outputOfLearning(one))
		}

		return nil, readLearningOutput{Entries: entries, Current: current}, nil
	}
}

// outputOfLearning maps the model of the domain onto the answer of the tools
// of the log.
func outputOfLearning(model learning.Entry) learningOutput {
	return learningOutput{
		ID:           int64(model.ID),
		BusinessID:   int64(model.BusinessID),
		Day:          model.Day.String(),
		RecipeID:     int64(model.RecipeID),
		RecipeName:   model.RecipeName,
		ProductionID: int64(model.ProductionID),
		Happened:     model.Happened,
		Learned:      model.Learned,
		AnswersID:    int64(model.Answers),
	}
}
