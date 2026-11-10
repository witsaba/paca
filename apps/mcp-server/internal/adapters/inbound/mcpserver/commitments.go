package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/commitment"
)

// The names of the tools of the commitments.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records a
// promise is not a tool that creates a row.
const (
	// ToolRecordCommitment writes a promise of the business: what is due,
	// in the words of the owner, and the day it is due.
	ToolRecordCommitment = "record_commitment"

	// ToolMeetCommitment writes the day a promise was met: the promise that
	// was kept stays in the book with the day it was kept.
	ToolMeetCommitment = "meet_commitment"

	// ToolReadCommitments reads the calendar of the promises of a business:
	// what is due and what is still owed.
	ToolReadCommitments = "read_commitments"
)

// The descriptions of the tools of the commitments.
//
// The descriptions are a contract with the agent: they say what the tool
// does, what it needs and what it answers, and nothing it does not do.
var (
	descriptionRecordCommitment = buildDescriptionRecordCommitment()
	descriptionMeetCommitment   = buildDescriptionMeetCommitment()
	descriptionReadCommitments  = buildDescriptionReadCommitments()
)

// buildDescriptionRecordCommitment renders the description of the tool that
// records a promise.
//
// The description is a contract with the agent: it says what the promise is,
// what the tool needs and what it answers, and nothing it does not do.
func buildDescriptionRecordCommitment() string {
	return "Record a promise of the bakery: what is due, in the words of the owner, and the day it is due.\n\n" +
		"A commitment is a PROMISE of the business: what is due and the day it is due. It is not a sale and it is " +
		"not an order — the order of the bakery already carries the client, the products, the money and the round, " +
		"and the promise never copies it. The promise carries NO MONEY and NO PARTY of its own: the 'what' is what " +
		"is due, in the words of the owner, and it may name the party in her own words.\n\n" +
		"The business ('business_id') is REQUIRED: it is the business of its own, the one that sells the bread. The " +
		"Personal Business of a person holds the money of that person and not the promises of the bakery, and PACA " +
		"refuses the promise of it.\n\n" +
		"\"What is due\" ('what') is REQUIRED: the promise in the words of the owner ('el pedido de la tienda de la " +
		"esquina'). A promise that says nothing about what is due is not a promise, and PACA refuses it. PACA keeps " +
		"the words with no space around them, and it never invents them.\n\n" +
		"The day it is due ('due_on') is REQUIRED: the day the promise is due, written as '2026-03-14'. PACA never " +
		"invents one.\n\n" +
		"The promise is written OPEN: it carries no day of meeting, because nobody said it was met. The day a " +
		"promise was kept is written with 'meet_commitment', and the promise that was kept stays in the book with " +
		"the day it was kept.\n\n" +
		"The answer carries the promise that was written: its identity ('id'), the business that owes it " +
		"('business_id'), what is due ('what'), the day it is due ('due_on'), the day it was met ('met_on', absent " +
		"while the promise is open) and whether it is still owed ('open').\n\n" +
		"The alarm is not PACA's: PACA holds the fact of what is due and the day it is due, and the agent is the one " +
		"that rings."
}

// buildDescriptionMeetCommitment renders the description of the tool that
// records the day a promise was met.
//
// The description is a contract with the agent: it says what the promise that
// was kept is, what the tool needs and what it answers, and nothing it does
// not do.
func buildDescriptionMeetCommitment() string {
	return "Record that a promise of the bakery was met: the day it was kept.\n\n" +
		"The promise that was kept is KEPT: it stays in the book with the day it was kept, and the promise is not " +
		"owed any more. A promise is kept once: the second meeting of the same promise is refused, and the book " +
		"never hides the day it already holds.\n\n" +
		"The promise is named by ITS OWN IDENTITY ('commitment_id'), never by its text: two promises that say the " +
		"same thing on the same day are two promises, and PACA never merges them. The identity is the 'id' that " +
		"'record_commitment' or 'read_commitments' answered.\n\n" +
		"The business ('business_id') is REQUIRED: it is the business that owes the promise, and its promises are " +
		"the ones the identity is looked for in. PACA refuses the meeting of a promise the business does not hold.\n\n" +
		"The day it was met ('met_on') is REQUIRED: the day the promise was kept, written as '2026-03-14'. The day " +
		"it was met may be before or after the day it is due — a promise kept early is kept and a promise kept late " +
		"is late — and PACA never invents the day it was kept.\n\n" +
		"The answer carries the promise that was kept: its identity ('id'), the business that owes it " +
		"('business_id'), what is due ('what'), the day it is due ('due_on'), the day it was met ('met_on') and " +
		"whether it is still owed ('open', which is false after the meeting)."
}

// buildDescriptionReadCommitments renders the description of the tool that
// reads the calendar of the promises.
//
// The description is a contract with the agent: it says what the calendar
// carries, what the tool needs and what it answers, and nothing it does not
// do.
func buildDescriptionReadCommitments() string {
	return "Read the calendar of the promises of the bakery: what is due and what is still owed.\n\n" +
		"The calendar of a day answers two things. It carries the promises that are due on the day: the bread and " +
		"the commitments the bakery promised for it. It also carries what is still owed: every open promise of the " +
		"business, the ones that are due on the day and the ones that fell due on an earlier day and were not met. " +
		"The promise that was met is not owed any more, and it stays in the book with the day it was kept.\n\n" +
		"The business ('business_id') is REQUIRED: it is the business whose promises are read, and PACA never " +
		"invents one. A business that is not there is refused: the answer is never an empty calendar for a business " +
		"that does not exist.\n\n" +
		"The day ('day') is OPTIONAL: the day of the calendar that is read, written as '2026-03-14'. The calendar of " +
		"that day carries the promises that are due on it, and the whole book of the promises is read when the agent " +
		"names none.\n\n" +
		"The answer carries the day of the calendar ('day') only when the agent names a day, the promises that are " +
		"due on it ('due') and the ones that are still owed ('owed'). Every promise carries its identity ('id'), " +
		"the business that owes it ('business_id'), what is due ('what'), the day it is due ('due_on'), the day it " +
		"was met ('met_on', absent while the promise is open) and whether it is still owed ('open').\n\n" +
		"The alarm is the work of the agent: PACA holds the fact of what is due and the day it is due, and this is " +
		"what the agent reads to ring. It writes nothing."
}

// recordCommitmentInput is the input of the tool that records a promise.
type recordCommitmentInput struct {
	// BusinessID is the business that owes the promise.
	BusinessID int64 `json:"business_id,omitempty" jsonschema:"The identity of the business that owes the promise: the business of its own, the one that sells the bread. Send it always: PACA never invents one, and the domain refuses the promise when it is left out."`

	// What is what is due, in the words of the owner.
	What string `json:"what,omitempty" jsonschema:"What is due, in the words of the owner ('el pedido de la tienda de la esquina'). Send it always: a promise that says nothing about what is due is not a promise, and the domain refuses it when it is left out."`

	// DueOn is the day the promise is due on.
	DueOn string `json:"due_on,omitempty" jsonschema:"The day the promise is due on, written as '2026-03-14'. Send it always: PACA never invents one, and the domain refuses the promise when it is left out."`
}

// meetCommitmentInput is the input of the tool that records the day a
// promise was met.
type meetCommitmentInput struct {
	// BusinessID is the business that owes the promise.
	BusinessID int64 `json:"business_id,omitempty" jsonschema:"The identity of the business that owes the promise. Send it always: PACA never invents one, and the domain refuses the meeting when it is left out."`

	// CommitmentID is the identity of the promise that was met.
	CommitmentID int64 `json:"commitment_id,omitempty" jsonschema:"The identity of the promise that was kept, the 'id' that 'record_commitment' or 'read_commitments' answered. Send it always: the identity of a promise is its own identity and never its text, and two promises that say the same thing are two promises."`

	// MetOn is the day the promise was met.
	MetOn string `json:"met_on,omitempty" jsonschema:"The day the promise was kept, written as '2026-03-14'. Send it always: the promise that was kept stays in the book with the day it was kept, and the domain refuses a meeting with no day."`
}

// readCommitmentsInput is the input of the tool that reads the calendar of
// the promises.
type readCommitmentsInput struct {
	// BusinessID is the business whose promises are read.
	BusinessID int64 `json:"business_id,omitempty" jsonschema:"The identity of the business whose promises are read. Send it always: PACA never invents one, and the domain refuses the reading when it is left out."`

	// Day is the day of the calendar that is read.
	Day string `json:"day,omitempty" jsonschema:"The day of the calendar that is read, written as '2026-03-14'. It is optional: the calendar of that day carries the promises that are due on it, and the whole book of the promises is read when it is left out."`
}

// commitmentOutput is what the tools of the commitments answer about one
// promise.
type commitmentOutput struct {
	// ID is the identity of the promise.
	ID int64 `json:"id"`

	// BusinessID is the business that owes the promise.
	BusinessID int64 `json:"business_id"`

	// What is what is due, in the words of the owner.
	What string `json:"what"`

	// DueOn is the day the promise is due on.
	DueOn string `json:"due_on"`

	// MetOn is the day the promise was met. It is absent while the promise
	// is open: the day of meeting is the state of the promise.
	MetOn string `json:"met_on,omitempty"`

	// Open says whether the promise is still owed.
	Open bool `json:"open"`
}

// recordCommitmentOutput is the answer of the tool that records a promise.
type recordCommitmentOutput struct {
	// Commitment is the promise that was written.
	Commitment commitmentOutput `json:"commitment"`
}

// meetCommitmentOutput is the answer of the tool that records the day a
// promise was met.
type meetCommitmentOutput struct {
	// Commitment is the promise that was kept.
	Commitment commitmentOutput `json:"commitment"`
}

// readCommitmentsOutput is the answer of the tool that reads the calendar of
// the promises of a business.
type readCommitmentsOutput struct {
	// Day is the day of the calendar that was read.
	Day string `json:"day,omitempty"`

	// Due are the promises that are due on the day.
	Due []commitmentOutput `json:"due"`

	// Owed are the promises that are still owed: every open promise of the
	// business, the ones that are due on the day and the ones that fell
	// due on an earlier day.
	Owed []commitmentOutput `json:"owed"`
}

// registerCommitmentTools registers the tools of the commitments.
//
// The tools are registered even when the service has no database: the list of
// the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerCommitmentTools(recorder ports.CommitmentRecorder, reader ports.CommitmentReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordCommitment,
		Description: descriptionRecordCommitment,
	}, s.recordCommitment(recorder))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolMeetCommitment,
		Description: descriptionMeetCommitment,
	}, s.meetCommitment(recorder))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadCommitments,
		Description: descriptionReadCommitments,
	}, s.readCommitments(reader))
}

// recordCommitment is the handler of the tool that records a promise.
//
// The tool writes the promise of the business and answers with it. The
// promise is written open: nobody said it was met.
func (s *Server) recordCommitment(recorder ports.CommitmentRecorder) sdk.ToolHandlerFor[recordCommitmentInput, recordCommitmentOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordCommitmentInput) (*sdk.CallToolResult, recordCommitmentOutput, error) {
		if recorder == nil {
			return nil, recordCommitmentOutput{}, ports.ErrNoDatabase
		}

		written, err := recorder.Record(ctx, ports.RecordCommitment{
			BusinessID: businessIDOf(input.BusinessID),
			What:       input.What,
			DueOn:      input.DueOn,
		})
		if err != nil {
			return nil, recordCommitmentOutput{}, s.toolError(err)
		}

		return nil, recordCommitmentOutput{Commitment: outputOfCommitment(written)}, nil
	}
}

// meetCommitment is the handler of the tool that records the day a promise
// was met.
//
// The promise is named by its own identity, and the tool answers with the
// promise that was kept.
func (s *Server) meetCommitment(recorder ports.CommitmentRecorder) sdk.ToolHandlerFor[meetCommitmentInput, meetCommitmentOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input meetCommitmentInput) (*sdk.CallToolResult, meetCommitmentOutput, error) {
		if recorder == nil {
			return nil, meetCommitmentOutput{}, ports.ErrNoDatabase
		}

		kept, err := recorder.Meet(ctx, ports.MeetCommitment{
			BusinessID:   businessIDOf(input.BusinessID),
			CommitmentID: input.CommitmentID,
			MetOn:        input.MetOn,
		})
		if err != nil {
			return nil, meetCommitmentOutput{}, s.toolError(err)
		}

		return nil, meetCommitmentOutput{Commitment: outputOfCommitment(kept)}, nil
	}
}

// readCommitments is the handler of the tool that reads the calendar of the
// promises of a business.
//
// The tool reads the calendar of the day: it changes nothing, and it answers
// with the promises that are due on it and with the ones that are still owed.
func (s *Server) readCommitments(reader ports.CommitmentReader) sdk.ToolHandlerFor[readCommitmentsInput, readCommitmentsOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readCommitmentsInput) (*sdk.CallToolResult, readCommitmentsOutput, error) {
		if reader == nil {
			return nil, readCommitmentsOutput{}, ports.ErrNoDatabase
		}

		calendar, err := reader.Read(ctx, ports.ReadCommitments{
			BusinessID: businessIDOf(input.BusinessID),
			Day:        input.Day,
		})
		if err != nil {
			return nil, readCommitmentsOutput{}, s.toolError(err)
		}

		due := make([]commitmentOutput, 0, len(calendar.Due))
		for _, one := range calendar.Due {
			due = append(due, outputOfCommitment(one))
		}

		owed := make([]commitmentOutput, 0, len(calendar.Owed))
		for _, one := range calendar.Owed {
			owed = append(owed, outputOfCommitment(one))
		}

		answer := readCommitmentsOutput{Due: due, Owed: owed}
		if !calendar.Day.Time().IsZero() {
			answer.Day = calendar.Day.String()
		}

		return nil, answer, nil
	}
}

// outputOfCommitment maps the model of the domain onto the answer of the
// tools of the commitments.
func outputOfCommitment(model commitment.Commitment) commitmentOutput {
	answer := commitmentOutput{
		ID:         int64(model.ID),
		BusinessID: int64(model.BusinessID),
		What:       model.What,
		DueOn:      model.DueOn.String(),
		Open:       model.IsOpen(),
	}

	if !model.MetOn.Time().IsZero() {
		answer.MetOn = model.MetOn.String()
	}

	return answer
}
