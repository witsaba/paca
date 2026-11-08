package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the overhead.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records the
// overhead is not a tool that records the labor.
const (
	// ToolRecordOverhead writes the electricity of the house, the hours of
	// the oven and the allocation the owner declared.
	ToolRecordOverhead = "record_overhead"
)

// The description of the tools of the overhead.
//
// The vocabulary of the overhead belongs to the domain: the description
// reads it and never writes it a second time.
var (
	descriptionRecordOverhead = buildDescriptionRecordOverhead()
)

// buildDescriptionRecordOverhead renders the description of the tool that
// records the overhead.
//
// The overhead of the bakery is the electricity of the house and the hours
// of the oven: the description says so in the words the owner reads, and
// it says that the allocation is hers.
func buildDescriptionRecordOverhead() string {
	return "Record the overhead of the house: the electricity the owner paid, the hours the oven ran and the share of the electricity that belongs to the oven.\n\n" +
		"The overhead is the electricity of the house and the hours of the oven:\n\n" +
		"  - electricity: the money of the electricity of the house, in whole pesos ('" +
		business.DefaultCurrency + "');\n" +
		"  - oven_hours: the hours the oven ran;\n" +
		"  - allocation: the SHARE of that electricity that belongs to the oven (0.6 is six of every ten pesos).\n\n" +
		"The business is REQUIRED: it is the business the overhead belongs to, and PACA never invents one.\n\n" +
		"The electricity is REQUIRED: the money of the house is what the owner paid, and PACA never infers it. " +
		"When PACA filled it by itself, say so with electricity_inferred, and the book refuses the overhead. " +
		"The oven_hours are REQUIRED: without them the money of the oven has no rate, and a rate nobody declared " +
		"is a rate PACA invented. The allocation is REQUIRED, and it is a share: it is greater than zero and not " +
		"greater than the electricity itself.\n\n" +
		"The day is OPTIONAL: the day the overhead was declared, written the way a person writes it ('2026-03-01'). " +
		"It is the day of today when it is left out.\n\n" +
		"The utterance is OPTIONAL: the sentence of the owner, verbatim. Send it as she said it when she said " +
		"one, and it is kept with the record.\n\n" +
		"confirmed_with is REQUIRED: the word she used to confirm it ('sí', 'confirmado'), and it is kept with " +
		"the record.\n\n" +
		"The answer carries the overhead that was written: the electricity of the house, the hours of the oven, " +
		"the allocation and the money of the overhead, which is the electricity times the allocation.\n\n" +
		"The overhead of a batch is derived from the hours of the oven of the steps of the recipe: it is never " +
		"typed, and it travels beside the direct cost and beside the labor of the batch."
}

// recordOverheadInput is the input of the tool that records the overhead.
type recordOverheadInput struct {
	// BusinessID is the business the overhead belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the overhead belongs to. It is required."`

	// Electricity is the money of the electricity of the house, in whole
	// pesos.
	Electricity int64 `json:"electricity" jsonschema:"The money of the electricity of the house, in whole pesos. It is required, it is positive, and PACA never infers it."`

	// ElectricityInferred says that PACA filled the electricity by
	// itself.
	ElectricityInferred bool `json:"electricity_inferred,omitempty" jsonschema:"True when PACA filled the electricity of the house by itself and the owner did not say it. The money of the house is never inferred, and PACA refuses an overhead that carries it."`

	// OvenHours is the number of hours the oven ran.
	OvenHours float64 `json:"oven_hours" jsonschema:"The number of hours the oven ran. It is required, and it is positive: without it the money of the oven has no rate."`

	// Allocation is the share of the electricity of the house that belongs
	// to the oven.
	Allocation float64 `json:"allocation" jsonschema:"The share of the electricity of the house that belongs to the oven (0.6 is six of every ten pesos). It is required, it is greater than zero and it is not greater than one."`

	// Day is the day the overhead was declared.
	Day string `json:"day,omitempty" jsonschema:"The day the overhead was declared, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty" jsonschema:"The sentence of the owner, verbatim. It is optional: send it as she said it when she said one, and it is kept with the record."`

	// ConfirmedWith is the word the owner used to confirm the overhead.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`
}

// overheadOutput is what the tool answers about one overhead.
//
// The answer carries the electricity of the house, the hours of the oven,
// the allocation and the money of the overhead.
type overheadOutput struct {
	// ID is the identity of the overhead.
	ID int64 `json:"id"`

	// BusinessID is the business the overhead belongs to.
	BusinessID int64 `json:"business_id"`

	// Day is the day the overhead was declared.
	Day string `json:"day"`

	// Electricity is the money of the electricity of the house, in whole
	// pesos.
	Electricity int64 `json:"electricity"`

	// OvenHours is the number of hours the oven ran.
	OvenHours float64 `json:"oven_hours"`

	// Allocation is the share of the electricity of the house that belongs
	// to the oven.
	Allocation float64 `json:"allocation"`

	// Money is the money of the overhead, in whole pesos: the electricity
	// of the house times the allocation.
	Money int64 `json:"money"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty"`

	// ConfirmedWith is the word the owner used to confirm the overhead.
	ConfirmedWith string `json:"confirmed_with,omitempty"`
}

// recordOverheadOutput is the answer of the tool that records the
// overhead.
type recordOverheadOutput struct {
	// Overhead is the overhead that was written.
	Overhead overheadOutput `json:"overhead"`
}

// registerOverheadTools registers the tools of the overhead.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerOverheadTools(recorder ports.OverheadRecorder) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordOverhead,
		Description: descriptionRecordOverhead,
	}, s.recordOverhead(recorder))
}

// recordOverhead is the handler of the tool that records the overhead.
//
// The tool writes the electricity of the house, the hours of the oven and
// the allocation the owner declared, and answers with them.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the overhead of nobody.
func (s *Server) recordOverhead(recorder ports.OverheadRecorder) sdk.ToolHandlerFor[recordOverheadInput, recordOverheadOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordOverheadInput) (*sdk.CallToolResult, recordOverheadOutput, error) {
		if recorder == nil {
			return nil, recordOverheadOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, recordOverheadOutput{}, s.toolError(recipe.ErrMissingBusiness)
		}

		recorded, err := recorder.Record(ctx, ports.RecordOverhead{
			BusinessID:          business.ID(input.BusinessID),
			Electricity:         input.Electricity,
			ElectricityInferred: input.ElectricityInferred,
			OvenHours:           input.OvenHours,
			Allocation:          input.Allocation,
			Day:                 input.Day,
			Utterance:           input.Utterance,
			ConfirmedWith:       input.ConfirmedWith,
		})
		if err != nil {
			return nil, recordOverheadOutput{}, s.toolError(err)
		}

		return nil, recordOverheadOutput{Overhead: outputOfOverhead(recorded)}, nil
	}
}

// outputOfOverhead maps the overhead of the domain onto the answer of the
// tool.
//
// The electricity, the hours of the oven, the allocation and the money of
// the overhead travel together: the money of the book counts whole pesos.
func outputOfOverhead(overhead recipe.Overhead) overheadOutput {
	return overheadOutput{
		ID:            int64(overhead.ID),
		BusinessID:    int64(overhead.BusinessID),
		Day:           overhead.Day.String(),
		Electricity:   int64(overhead.Electricity),
		OvenHours:     overhead.OvenHours,
		Allocation:    overhead.Allocation,
		Money:         int64(overhead.Money),
		Utterance:     overhead.Utterance,
		ConfirmedWith: overhead.ConfirmedWith,
	}
}
