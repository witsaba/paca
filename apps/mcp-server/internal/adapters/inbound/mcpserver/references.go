package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The names of the tools of the reference prices.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records a
// reference is not a tool that sets the price of the ingredient.
const (
	// ToolRecordReference writes the price of the market that PACA
	// researched, as a reference of the business.
	ToolRecordReference = "record_reference"

	// ToolReadReferences reads the reference prices of a business, with
	// their source and their day.
	ToolReadReferences = "read_references"
)

// The description of the tools of the reference prices.
//
// The kind of what is written belongs to the domain: the description reads
// it and never writes it a second time.
var (
	descriptionRecordReference = buildDescriptionRecordReference()
	descriptionReadReferences  = buildDescriptionReadReferences()
)

// buildDescriptionRecordReference renders the description of the tool that
// records a reference.
//
// A reference is never the price of the ingredient: the description says so
// in the words the agent reads, because the price the owner pays comes from
// her own purchases.
func buildDescriptionRecordReference() string {
	return "Record the price of the market that PACA researched, and write it in the book of the business as a " +
		"REFERENCE.\n\n" +
		"A reference is never the price of the ingredient: the price the owner pays comes from her own " +
		"purchases, and the reference is what the market was seen at. Tell the owner that, every time.\n\n" +
		"The business is REQUIRED: it is the business the reference belongs to, and PACA never invents one.\n\n" +
		"The ingredient is REQUIRED:\n\n" +
		"  - ingredient_id is REQUIRED: send the identity the tool " + ToolMatchItems + " answered for the name. " +
		"The catalog is the one owner of the ingredient, and PACA refuses an ingredient the catalog of the " +
		"business does not hold.\n\n" +
		"The price is REQUIRED: the price that was seen, in whole pesos ('" + business.DefaultCurrency + "'). The " +
		"money of the book counts whole pesos, like everywhere else.\n\n" +
		"The unit is REQUIRED: the unit of that price, the way the owner says it (" + namesOfTheSpokenUnits() + "). " +
		"The book stores the unit of the book (" + namesOfTheUnits() + "), and PACA refuses a unit it does not " +
		"know. A price without a unit is a number nobody can compare with anything, and PACA refuses it.\n\n" +
		"The source is REQUIRED: the place the price was read ('Boletín de precios al por mayor, DANE', the " +
		"listing of the store). A price without a source is a price nobody can check, and PACA refuses it.\n\n" +
		"The day is OPTIONAL: the day the price was seen, written the way a person writes it ('2026-03-01'). It is " +
		"the day of today when it is left out.\n\n" +
		"The market is OPTIONAL: the market where the price was seen ('Plaza de Mercado'). Leave it out when the " +
		"research named none.\n\n" +
		"The utterance is OPTIONAL: the sentence of the owner, verbatim. Send it as she said it when she said " +
		"one, and it is kept with the record.\n\n" +
		"confirmed_with is REQUIRED: the word she used to confirm it ('sí', 'confirmado'), and it is kept with " +
		"the record.\n\n" +
		"The answer carries the reference that was written: the ingredient, the price, the unit, the market, the " +
		"source, the day it was seen and the kind of what it is."
}

// buildDescriptionReadReferences renders the description of the tool that
// reads the reference prices.
func buildDescriptionReadReferences() string {
	return "Read the reference prices of the business: what the market was seen at, where it was read and the day " +
		"it was read.\n\n" +
		"A reference is never the price of the ingredient: the price the owner pays comes from her own purchases.\n\n" +
		"The business is REQUIRED: it is the business whose references are read.\n\n" +
		"The ingredient is OPTIONAL: send the identity the tool " + ToolMatchItems + " answered for the name to " +
		"read the references of that ingredient alone, and leave it out to read every reference of the business.\n\n" +
		"This tool changes nothing: it reads the references of the business and it writes nothing."
}

// recordReferenceInput is the input of the tool that records a reference.
type recordReferenceInput struct {
	// BusinessID is the business the reference belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the reference belongs to. It is required."`

	// IngredientID is the identity of the ingredient of the catalog the
	// reference is about.
	IngredientID int64 `json:"ingredient_id" jsonschema:"The identity of the ingredient of the catalog the reference is about. It is required: send the identity the tool match_items answered."`

	// Price is the price that was seen.
	Price int64 `json:"price" jsonschema:"The price that was seen, in whole pesos. It is required and it is positive."`

	// Unit is the unit of that price.
	Unit string `json:"unit" jsonschema:"The unit of the price, the way the owner says it ('kilos', 'gramos', 'paquetes', 'piezas'). It is required. PACA keeps the unit of the book ('g', 'kg', 'paquete', 'unidad')."`

	// Market is the market where the price was seen.
	Market string `json:"market,omitempty" jsonschema:"The market where the price was seen ('Plaza de Mercado'). It is optional: leave it out when the research named none."`

	// Source is the place the price was read.
	Source string `json:"source" jsonschema:"The place the price was read ('Boletín de precios al por mayor, DANE'). It is required: a price without a source is a price nobody can check."`

	// Day is the day the price was seen.
	Day string `json:"day,omitempty" jsonschema:"The day the price was seen, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty" jsonschema:"The sentence of the owner, verbatim. It is optional: send it as she said it when she said one, and it is kept with the record."`

	// ConfirmedWith is the word the owner used to confirm the reference.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`
}

// referenceOutput is what the tools answer about one reference.
//
// The answer carries the kind of what it is: a reference is a reference, and
// never the price of the ingredient.
type referenceOutput struct {
	// ID is the identity of the reference.
	ID int64 `json:"id"`

	// BusinessID is the business the reference belongs to.
	BusinessID int64 `json:"business_id"`

	// IngredientID is the ingredient of the catalog the reference is about.
	IngredientID int64 `json:"ingredient_id"`

	// Kind is the kind of what the answer carries: a reference.
	Kind string `json:"kind"`

	// Price is the price that was seen, in whole pesos.
	Price int64 `json:"price"`

	// Unit is the unit of that price.
	Unit string `json:"unit"`

	// Market is the market where the price was seen.
	Market string `json:"market,omitempty"`

	// Source is the place the price was read.
	Source string `json:"source"`

	// Day is the day the price was seen.
	Day string `json:"day"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty"`

	// ConfirmedWith is the word the owner used to confirm the reference.
	ConfirmedWith string `json:"confirmed_with,omitempty"`
}

// recordReferenceOutput is the answer of the tool that records a reference.
type recordReferenceOutput struct {
	// Reference is the reference that was written.
	Reference referenceOutput `json:"reference"`
}

// readReferencesInput is the input of the tool that reads the references.
type readReferencesInput struct {
	// BusinessID is the business whose references are read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose references are read. It is required."`

	// IngredientID is the ingredient the caller asks about.
	IngredientID int64 `json:"ingredient_id,omitempty" jsonschema:"The identity of the ingredient whose references are read. It is optional: leave it out to read every reference of the business."`
}

// readReferencesOutput is the answer of the tool that reads the references.
type readReferencesOutput struct {
	// References are the references of the business, with their source and
	// their day.
	References []referenceOutput `json:"references"`
}

// registerReferenceTools registers the tools of the reference prices.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerReferenceTools(recorder ports.ReferenceRecorder, reader ports.ReferencesReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordReference,
		Description: descriptionRecordReference,
	}, s.recordReference(recorder))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadReferences,
		Description: descriptionReadReferences,
	}, s.readReferences(reader))
}

// recordReference is the handler of the tool that records a reference.
//
// The tool writes the reference the owner confirmed and answers with it.
func (s *Server) recordReference(recorder ports.ReferenceRecorder) sdk.ToolHandlerFor[recordReferenceInput, recordReferenceOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordReferenceInput) (*sdk.CallToolResult, recordReferenceOutput, error) {
		if recorder == nil {
			return nil, recordReferenceOutput{}, ports.ErrNoDatabase
		}

		recorded, err := recorder.Record(ctx, ports.RecordReference{
			BusinessID:    business.ID(input.BusinessID),
			IngredientID:  input.IngredientID,
			Price:         input.Price,
			Unit:          input.Unit,
			Market:        input.Market,
			Source:        input.Source,
			Day:           input.Day,
			Utterance:     input.Utterance,
			ConfirmedWith: input.ConfirmedWith,
		})
		if err != nil {
			return nil, recordReferenceOutput{}, s.toolError(err)
		}

		return nil, recordReferenceOutput{Reference: outputOfReference(recorded)}, nil
	}
}

// readReferences is the handler of the tool that reads the references.
//
// The tool reads the references and answers with them: it writes nothing.
func (s *Server) readReferences(reader ports.ReferencesReader) sdk.ToolHandlerFor[readReferencesInput, readReferencesOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readReferencesInput) (*sdk.CallToolResult, readReferencesOutput, error) {
		if reader == nil {
			return nil, readReferencesOutput{}, ports.ErrNoDatabase
		}

		read, err := reader.Read(ctx, ports.ReadReferences{
			BusinessID:   business.ID(input.BusinessID),
			IngredientID: input.IngredientID,
		})
		if err != nil {
			return nil, readReferencesOutput{}, s.toolError(err)
		}

		answer := readReferencesOutput{References: make([]referenceOutput, 0, len(read))}
		for _, one := range read {
			answer.References = append(answer.References, outputOfReference(one))
		}

		return nil, answer, nil
	}
}

// outputOfReference maps the reference of the domain onto the answer of the
// tool.
//
// The answer carries the kind of what it is: a reference is a reference, and
// never the price of the ingredient. The kind comes from the domain, and it
// is never written a second time here.
func outputOfReference(reference purchase.Reference) referenceOutput {
	return referenceOutput{
		ID:            int64(reference.ID),
		BusinessID:    int64(reference.BusinessID),
		IngredientID:  int64(reference.IngredientID),
		Kind:          reference.Label(),
		Price:         int64(reference.Price),
		Unit:          reference.Unit,
		Market:        reference.Market,
		Source:        reference.Source,
		Day:           reference.SeenOn.String(),
		Utterance:     reference.Utterance,
		ConfirmedWith: reference.ConfirmedWith,
	}
}
