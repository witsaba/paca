package mcpserver

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the labor.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records the
// labor is not a tool that records the overhead.
const (
	// ToolRecordLabor writes the hours the owner worked and the value of
	// the hour she declared.
	ToolRecordLabor = "record_labor"
)

// The description of the tools of the labor.
//
// The kinds of the labor belong to the domain: the description reads them
// and never writes them a second time.
var (
	descriptionRecordLabor = buildDescriptionRecordLabor()
)

// buildDescriptionRecordLabor renders the description of the tool that
// records the labor.
//
// The labor of the bakery is the labor of the owner alone: the description
// says so, because the payroll of the employees is out of scope for now.
func buildDescriptionRecordLabor() string {
	return "Record the labor of the owner: the hours she worked and the value of the hour she declared.\n\n" +
		"The labor is measured in HOURS, and in three kinds (" + theNamesOfTheKindsOfTheLabor() + "):\n\n" +
		"  - hands_on_hours: the hours her hands were on the dough;\n" +
		"  - oven_hours: the hours the batch was in the oven;\n" +
		"  - waiting_hours: the hours the dough waited.\n\n" +
		"At least one of the three kinds carries hours: a labor without hours is nothing, and PACA refuses it.\n\n" +
		"The business is REQUIRED: it is the business the labor belongs to, and PACA never invents one.\n\n" +
		"The hour_value is REQUIRED: what the owner declared the hour of her work is worth, in whole pesos ('" +
		business.DefaultCurrency + "'). The money is never inferred: PACA refuses a labor whose value of the hour " +
		"it filled by itself, and it asks the owner for it instead.\n\n" +
		"The day is OPTIONAL: the day the labor was worked, written the way a person writes it ('2026-03-01'). It " +
		"is the day of today when it is left out.\n\n" +
		"The utterance is OPTIONAL: the sentence of the owner, verbatim. Send it as she said it when she said " +
		"one, and it is kept with the record.\n\n" +
		"confirmed_with is REQUIRED: the word she used to confirm it ('sí', 'confirmado'), and it is kept with " +
		"the record.\n\n" +
		"The labor of the owner alone: the payroll of the employees is out of scope for now, and PACA never asks " +
		"who worked.\n\n" +
		"The answer carries the labor that was written: the hours in the three kinds, the value of the hour, the " +
		"money of the labor and the day it was worked."
}

// recordLaborInput is the input of the tool that records the labor.
type recordLaborInput struct {
	// BusinessID is the business the labor belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the labor belongs to. It is required."`

	// HandsOnHours is the number of hours the hands of the owner were on
	// the dough.
	HandsOnHours float64 `json:"hands_on_hours" jsonschema:"The number of hours the hands of the owner were on the dough. It is required, and at least one of the three kinds of the labor carries hours."`

	// OvenHours is the number of hours the batch was in the oven.
	OvenHours float64 `json:"oven_hours" jsonschema:"The number of hours the batch was in the oven. It is required, and at least one of the three kinds of the labor carries hours."`

	// WaitingHours is the number of hours the dough waited.
	WaitingHours float64 `json:"waiting_hours" jsonschema:"The number of hours the dough waited. It is required, and at least one of the three kinds of the labor carries hours."`

	// HourValue is the value of the hour, in whole pesos.
	HourValue int64 `json:"hour_value" jsonschema:"The value of the hour the owner declared, in whole pesos. It is required, it is positive, and PACA never infers it."`

	// Day is the day the labor was worked.
	Day string `json:"day,omitempty" jsonschema:"The day the labor was worked, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty" jsonschema:"The sentence of the owner, verbatim. It is optional: send it as she said it when she said one, and it is kept with the record."`

	// ConfirmedWith is the word the owner used to confirm the labor.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`
}

// laborOutput is what the tool answers about one labor.
//
// The answer carries the hours in the three kinds of the labor, the value
// of the hour and the money of the labor.
type laborOutput struct {
	// ID is the identity of the labor.
	ID int64 `json:"id"`

	// BusinessID is the business the labor belongs to.
	BusinessID int64 `json:"business_id"`

	// Day is the day the labor was worked.
	Day string `json:"day"`

	// HandsOnHours is the number of hours the hands of the owner were on
	// the dough.
	HandsOnHours float64 `json:"hands_on_hours"`

	// OvenHours is the number of hours the batch was in the oven.
	OvenHours float64 `json:"oven_hours"`

	// WaitingHours is the number of hours the dough waited.
	WaitingHours float64 `json:"waiting_hours"`

	// TotalHours is the number of hours of the labor, in the three kinds
	// together.
	TotalHours float64 `json:"total_hours"`

	// HourValue is the value of the hour, in whole pesos.
	HourValue int64 `json:"hour_value"`

	// Money is the money of the labor, in whole pesos.
	Money int64 `json:"money"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty"`

	// ConfirmedWith is the word the owner used to confirm the labor.
	ConfirmedWith string `json:"confirmed_with,omitempty"`
}

// recordLaborOutput is the answer of the tool that records the labor.
type recordLaborOutput struct {
	// Labor is the labor that was written.
	Labor laborOutput `json:"labor"`
}

// registerLaborTools registers the tools of the labor.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerLaborTools(recorder ports.LaborRecorder) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordLabor,
		Description: descriptionRecordLabor,
	}, s.recordLabor(recorder))
}

// recordLabor is the handler of the tool that records the labor.
//
// The tool writes the hours the owner worked and the value of the hour she
// declared, and answers with them.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the labor of nobody.
func (s *Server) recordLabor(recorder ports.LaborRecorder) sdk.ToolHandlerFor[recordLaborInput, recordLaborOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordLaborInput) (*sdk.CallToolResult, recordLaborOutput, error) {
		if recorder == nil {
			return nil, recordLaborOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, recordLaborOutput{}, s.toolError(recipe.ErrMissingBusiness)
		}

		recorded, err := recorder.Record(ctx, ports.RecordLabor{
			BusinessID:    business.ID(input.BusinessID),
			HandsOnHours:  input.HandsOnHours,
			OvenHours:     input.OvenHours,
			WaitingHours:  input.WaitingHours,
			HourValue:     input.HourValue,
			Day:           input.Day,
			Utterance:     input.Utterance,
			ConfirmedWith: input.ConfirmedWith,
		})
		if err != nil {
			return nil, recordLaborOutput{}, s.toolError(err)
		}

		return nil, recordLaborOutput{Labor: outputOfLabor(recorded)}, nil
	}
}

// outputOfLabor maps the labor of the domain onto the answer of the tool.
//
// The hours travel in the three kinds of the labor, and the money of the
// labor travels with them: the money of the book counts whole pesos.
func outputOfLabor(labour recipe.Labor) laborOutput {
	return laborOutput{
		ID:            int64(labour.ID),
		BusinessID:    int64(labour.BusinessID),
		Day:           labour.Day.String(),
		HandsOnHours:  labour.Hours.HandsOn,
		OvenHours:     labour.Hours.Oven,
		WaitingHours:  labour.Hours.Waiting,
		TotalHours:    labour.Hours.Total(),
		HourValue:     int64(labour.HourValue),
		Money:         int64(labour.Money),
		Utterance:     labour.Utterance,
		ConfirmedWith: labour.ConfirmedWith,
	}
}

// theNamesOfTheKindsOfTheLabor renders the kinds of the labor in a
// sentence the caller reads.
//
// The vocabulary belongs to the domain: the description of the tool names
// the kinds the book knows, and it never writes them a second time.
func theNamesOfTheKindsOfTheLabor() string {
	names := make([]string, 0, len(recipe.LaborKinds()))

	for _, one := range recipe.LaborKinds() {
		names = append(names, one.String())
	}

	return strings.Join(names, ", ")
}

// theSpokenNamesOfTheKindsOfTheLabor renders the ways the owner says the
// kinds of the labor in a sentence the caller reads.
//
// The vocabulary belongs to the domain: the agent reads the words the
// owner uses and sends the kind of the book.
func theSpokenNamesOfTheKindsOfTheLabor() string {
	return strings.Join(recipe.SpokenLaborKinds(), ", ")
}
