package mcpserver

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
)

// The names of the tools of the catalog.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that matches a
// name is not a tool that writes an ingredient.
const (
	// ToolMatchItems matches the name the owner said against the catalog
	// of the names the business buys.
	ToolMatchItems = "match_items"

	// ToolReadIngredients reads the catalog of the names the business
	// buys, with what is known of every ingredient and what is still
	// missing of it.
	ToolReadIngredients = "read_ingredients"

	// ToolCompleteIngredient writes what the owner declared about an
	// ingredient of the catalog.
	ToolCompleteIngredient = "complete_ingredient"
)

// The description of the tools of the catalog.
//
// The statuses of the answer and the units of the book belong to the
// domain: the description reads them and never writes them a second time.
var (
	descriptionMatchItems         = buildDescriptionMatchItems()
	descriptionReadIngredients    = buildDescriptionReadIngredients()
	descriptionCompleteIngredient = buildDescriptionCompleteIngredient()
)

// buildDescriptionMatchItems renders the description of the tool that
// matches a name against the catalog of a business.
//
// The statuses of the answer belong to the domain, and the description
// reads them: the vocabulary of the answer is a rule, not a promise of the
// transport.
func buildDescriptionMatchItems() string {
	return "Match the name the owner just said against the catalog of the names the business buys.\n\n" +
		"The business is REQUIRED: it is the business whose catalog answers, and PACA never invents one. " +
		"Ask the person for it, and do not call this tool without it.\n\n" +
		"The name is REQUIRED: it is the name as the owner said it ('harina', 'Harina panadera'). Send it as " +
		"she said it, and not a summary of it.\n\n" +
		"The answer carries the status of the name:\n\n" +
		"  - 'exact' when one ingredient answers the name: the answer carries the ingredient and its identity.\n" +
		"  - 'ambiguous' when several answer it: the answer carries the options, so the person chooses with one " +
		"word.\n" +
		"  - 'unknown' when none of them does.\n\n" +
		"A name the catalog does not know is NOT an error: the answer says that it is new ('new': true), and the " +
		"name is recorded as she said it when she buys it.\n\n" +
		"The catalog begins empty and grows with what the business buys, so the first name is the one of the " +
		"first purchase.\n\n" +
		"Send the identity of the ingredient the tool answered to the tool of the draft of the purchase, so the " +
		"line of the purchase carries the ingredient of the catalog.\n\n" +
		"This tool changes nothing: it reads the catalog of the business and it writes nothing."
}

// matchItemsInput is the input of the tool that matches a name against the
// catalog of a business.
type matchItemsInput struct {
	// BusinessID is the business whose catalog answers.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose catalog answers. It is required."`

	// Name is the name as the owner said it.
	Name string `json:"name" jsonschema:"The name as the owner said it ('harina', 'Harina panadera'). It is required."`
}

// ingredientOutput is what the answer carries about one ingredient of the
// catalog.
type ingredientOutput struct {
	// ID is the identity of the ingredient.
	ID int64 `json:"id"`

	// Name is the name the owner reads.
	Name string `json:"name"`
}

// ingredientDetailOutput is what the answer carries about one ingredient
// of the catalog when the agent asks what is known of it.
//
// The answer carries what the owner declared and what is still missing of
// it: the owner asks for what she does not know, and the answer names it.
type ingredientDetailOutput struct {
	// ID is the identity of the ingredient.
	ID int64 `json:"id"`

	// BusinessID is the business that buys it.
	BusinessID int64 `json:"business_id"`

	// Name is the name the owner reads.
	Name string `json:"name"`

	// Aliases are the names the owner uses for it.
	Aliases []string `json:"aliases"`

	// Unit is the unit the ingredient is bought in. It is empty when the
	// owner has not declared it yet.
	Unit string `json:"unit,omitempty"`

	// PackageGrams is the weight of a package of the ingredient, in
	// grams. It is zero when the owner has not declared it yet.
	PackageGrams float64 `json:"package_grams,omitempty"`

	// Allergens are the allergens the ingredient carries.
	Allergens []string `json:"allergens"`

	// CertificationName is the claim the ingredient carries.
	CertificationName string `json:"certification,omitempty"`

	// CertificationEvidence is the document that proves the claim.
	CertificationEvidence string `json:"certification_evidence,omitempty"`

	// Conversion is the grams of the raw that one gram of the prepared
	// ingredient takes.
	Conversion float64 `json:"conversion"`

	// GramsPerUnit is how many grams one unit of the ingredient weighs.
	// It is empty when the book does not know: the piece is one unit of
	// the ingredient, and its weight is never assumed.
	GramsPerUnit float64 `json:"grams_per_unit,omitempty"`

	// Missing is what the ingredient still needs from its owner.
	Missing []string `json:"missing"`
}

// matchItemsOutput is the answer of the tool that matches a name against
// the catalog of a business.
type matchItemsOutput struct {
	// Name is the name as the owner said it.
	Name string `json:"name"`

	// Status is the answer of the catalog: exact, ambiguous or unknown.
	Status string `json:"status"`

	// New says that the catalog does not know the name.
	New bool `json:"new"`

	// Ingredient is the ingredient of an exact answer.
	Ingredient *ingredientOutput `json:"ingredient,omitempty"`

	// Options are the candidates of an ambiguous answer.
	Options []ingredientOutput `json:"options"`
}

// buildDescriptionReadIngredients renders the description of the tool that
// reads the catalog of a business.
//
// The units of the book belong to the domain, and the description reads
// them: the vocabulary is a rule, not a promise of the transport.
func buildDescriptionReadIngredients() string {
	return "Read the catalog of the names the business buys: what is known of every ingredient, and what is still " +
		"missing of each one.\n\n" +
		"The business is REQUIRED: it is the business whose catalog is read, and PACA never invents one.\n\n" +
		"The answer carries every ingredient of the business, in the order of their identity, with:\n\n" +
		"  - the name the owner reads and the names she uses for it;\n" +
		"  - the unit it is bought in (" + namesOfTheUnits() + "), and the weight of a package of it in grams;\n" +
		"  - the allergens it carries;\n" +
		"  - the certification it has and the evidence of that certification: a claim without the document that " +
		"proves it is a wish, and PACA never writes one;\n" +
		"  - the conversion of its raw grams into its prepared ones, and the weight of its unit in grams: the " +
		"piece is one unit of the ingredient, and its weight is never assumed.\n\n" +
		"The answer carries also what each ingredient is still missing ('missing'), so the agent asks the owner " +
		"for exactly what she has not said yet.\n\n" +
		"This tool changes nothing: it reads the catalog of the business and it writes nothing."
}

// buildDescriptionCompleteIngredient renders the description of the tool
// that completes an ingredient.
//
// The units of the book belong to the domain, and the description reads
// them: the vocabulary is a rule, not a promise of the transport.
func buildDescriptionCompleteIngredient() string {
	return "Write what the owner just declared about an ingredient of the catalog, and answer with the ingredient " +
		"as it is now.\n\n" +
		"The business is REQUIRED: it is the business the ingredient belongs to.\n\n" +
		"The ingredient is REQUIRED: send its identity ('ingredient_id'), the one the tool " + ToolMatchItems + " answered for the name. " +
		"PACA refuses an ingredient the catalog of the business does not hold.\n\n" +
		"Every field below is OPTIONAL, and the ingredient keeps what it had when the owner declared none:\n\n" +
		"  - the unit it is bought in (" + namesOfTheUnits() + "). The unit is a rule of the book, and PACA " +
		"refuses a unit it does not know;\n" +
		"  - the weight of a package of it, in grams ('package_grams'). The weight of a package is never " +
		"assumed: send it when the owner says it;\n" +
		"  - the allergens it carries ('allergens'). Send the list the owner declared, and an empty list when " +
		"she says it carries none;\n" +
		"  - the certification it has ('certification') and the document that proves it " +
		"('certification_evidence'). The evidence is REQUIRED when there is a certification: 'organic' is a " +
		"claim, and a claim carries its evidence. PACA refuses a certification without it;\n" +
		"  - the names the owner uses for it ('aliases'). They are the names she says, and they are written " +
		"here;\n" +
		"  - the conversion of its raw grams into its prepared ones ('conversion'): the grams of the raw that " +
		"one gram of the prepared ingredient takes. For now the owner decided that the grams of a recipe are " +
		"the raw ones, and the ingredient keeps that one to one until she declares another factor.\n\n" +
		"The answer carries the ingredient as it is now and what it is still missing ('missing')."
}

// namesOfTheUnits renders the units of the book in a sentence the agent
// reads.
//
// The vocabulary belongs to the domain: the description reads it and never
// writes it a second time.
func namesOfTheUnits() string {
	names := make([]string, 0, len(ingredient.Units()))

	for _, one := range ingredient.Units() {
		names = append(names, "'"+one.String()+"'")
	}

	return strings.Join(names, ", ")
}

// namesOfTheSpokenUnits renders the ways the owner says the units of the
// book in a sentence the agent reads.
//
// The vocabulary belongs to the domain: the description reads it and never
// writes it a second time. The agent sends the unit the way the owner said
// it, and the book keeps the unit of the book.
func namesOfTheSpokenUnits() string {
	names := make([]string, 0, len(ingredient.SpokenUnits()))

	for _, one := range ingredient.SpokenUnits() {
		names = append(names, "'"+one+"'")
	}

	return strings.Join(names, ", ")
}

// readIngredientsInput is the input of the tool that reads the catalog of
// a business.
type readIngredientsInput struct {
	// BusinessID is the business whose catalog is read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose catalog is read. It is required."`
}

// readIngredientsOutput is the answer of the tool that reads the catalog of
// a business.
type readIngredientsOutput struct {
	// Ingredients are the ingredients of the business, with what is known
	// of each one and what is still missing of it.
	Ingredients []ingredientDetailOutput `json:"ingredients"`
}

// completeIngredientInput is the input of the tool that completes an
// ingredient of the catalog.
type completeIngredientInput struct {
	// BusinessID is the business the ingredient belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the ingredient belongs to. It is required."`

	// IngredientID is the identity of the ingredient of the catalog.
	IngredientID int64 `json:"ingredient_id" jsonschema:"The identity of the ingredient of the catalog. It is required: send the identity the tool match_items answered."`

	// Unit is the unit the ingredient is bought in.
	Unit string `json:"unit,omitempty" jsonschema:"The unit the ingredient is bought in ('g', 'kg', 'paquete', 'unidad'). It is optional: leave it out when the owner declared none."`

	// PackageGrams is the weight of a package of the ingredient.
	PackageGrams float64 `json:"package_grams,omitempty" jsonschema:"The weight of a package of the ingredient, in grams. It is optional: the weight of a package is never assumed."`

	// Allergens are the allergens the ingredient carries.
	Allergens []string `json:"allergens,omitempty" jsonschema:"The allergens the ingredient carries. It is optional: leave it out when the owner declared none."`

	// CertificationName is the claim the ingredient carries.
	CertificationName string `json:"certification,omitempty" jsonschema:"The claim the ingredient carries ('Orgánico'). It is optional: leave it out when the owner declared none."`

	// CertificationEvidence is the document that proves the claim.
	CertificationEvidence string `json:"certification_evidence,omitempty" jsonschema:"The document that proves the certification ('Certificado de la finca, 2026-02-01'). It is required when there is a certification."`

	// Aliases are the names the owner uses for the ingredient.
	Aliases []string `json:"aliases,omitempty" jsonschema:"The names the owner uses for the ingredient. It is optional: leave it out when she declared none."`

	// Conversion is the grams of the raw that one gram of the prepared
	// ingredient takes.
	Conversion float64 `json:"conversion,omitempty" jsonschema:"The grams of the raw that one gram of the prepared ingredient takes. It is optional: leave it out when the owner declared none, and the ingredient keeps the one to one of the raw."`
}

// completeIngredientOutput is the answer of the tool that completes an
// ingredient of the catalog.
type completeIngredientOutput struct {
	// Ingredient is the ingredient as it is now.
	Ingredient ingredientDetailOutput `json:"ingredient"`
}

// registerIngredientTools registers the tools of the catalog.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerIngredientTools(
	matcher ports.ItemMatcher,
	reader ports.IngredientsReader,
	completer ports.IngredientCompleter,
) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolMatchItems,
		Description: descriptionMatchItems,
	}, s.matchItems(matcher))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadIngredients,
		Description: descriptionReadIngredients,
	}, s.readIngredients(reader))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolCompleteIngredient,
		Description: descriptionCompleteIngredient,
	}, s.completeIngredient(completer))
}

// matchItems is the handler of the tool that matches a name against the
// catalog of a business.
//
// The tool reads the catalog and answers with the candidates: it writes
// nothing.
func (s *Server) matchItems(matcher ports.ItemMatcher) sdk.ToolHandlerFor[matchItemsInput, matchItemsOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input matchItemsInput) (*sdk.CallToolResult, matchItemsOutput, error) {
		if matcher == nil {
			return nil, matchItemsOutput{}, ports.ErrNoDatabase
		}

		matched, err := matcher.Match(ctx, ports.MatchItems{
			BusinessID: business.ID(input.BusinessID),
			Name:       input.Name,
		})
		if err != nil {
			return nil, matchItemsOutput{}, s.toolError(err)
		}

		return nil, outputOfMatch(matched), nil
	}
}

// outputOfMatch maps the answer of the domain onto the answer of the tool.
//
// The answer carries the status of the name, the ingredient of an exact
// answer and the options of an ambiguous one.
func outputOfMatch(matched ingredient.Match) matchItemsOutput {
	options := make([]ingredientOutput, 0, len(matched.Options))
	for _, one := range matched.Options {
		options = append(options, ingredientOutput{ID: int64(one.ID), Name: one.Name})
	}

	answer := matchItemsOutput{
		Name:    matched.Name,
		Status:  matched.Status.String(),
		New:     matched.New,
		Options: options,
	}

	if matched.Status == ingredient.StatusExact {
		answer.Ingredient = &ingredientOutput{
			ID:   int64(matched.Ingredient.ID),
			Name: matched.Ingredient.Name,
		}
	}

	return answer
}

// readIngredients is the handler of the tool that reads the catalog of a
// business.
//
// The tool reads the catalog and answers with what is known of every
// ingredient and what is still missing of it: it writes nothing.
func (s *Server) readIngredients(reader ports.IngredientsReader) sdk.ToolHandlerFor[readIngredientsInput, readIngredientsOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readIngredientsInput) (*sdk.CallToolResult, readIngredientsOutput, error) {
		if reader == nil {
			return nil, readIngredientsOutput{}, ports.ErrNoDatabase
		}

		read, err := reader.Read(ctx, ports.ReadIngredients{
			BusinessID: business.ID(input.BusinessID),
		})
		if err != nil {
			return nil, readIngredientsOutput{}, s.toolError(err)
		}

		answer := readIngredientsOutput{Ingredients: make([]ingredientDetailOutput, 0, len(read))}
		for _, one := range read {
			answer.Ingredients = append(answer.Ingredients, outputOfIngredient(one))
		}

		return nil, answer, nil
	}
}

// completeIngredient is the handler of the tool that completes an
// ingredient of the catalog.
//
// The tool writes what the owner declared and answers with the ingredient
// as it is now.
func (s *Server) completeIngredient(completer ports.IngredientCompleter) sdk.ToolHandlerFor[completeIngredientInput, completeIngredientOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input completeIngredientInput) (*sdk.CallToolResult, completeIngredientOutput, error) {
		if completer == nil {
			return nil, completeIngredientOutput{}, ports.ErrNoDatabase
		}

		completed, err := completer.Complete(ctx, ports.CompleteIngredient{
			BusinessID:            business.ID(input.BusinessID),
			IngredientID:          input.IngredientID,
			Unit:                  input.Unit,
			PackageGrams:          input.PackageGrams,
			Allergens:             input.Allergens,
			CertificationName:     input.CertificationName,
			CertificationEvidence: input.CertificationEvidence,
			Aliases:               input.Aliases,
			Conversion:            input.Conversion,
		})
		if err != nil {
			return nil, completeIngredientOutput{}, s.toolError(err)
		}

		return nil, completeIngredientOutput{Ingredient: outputOfIngredient(completed)}, nil
	}
}

// outputOfIngredient maps the ingredient of the domain onto the answer of
// the tool.
//
// The answer carries what is known of the ingredient and what is still
// missing of it: the owner asks for what she does not know.
func outputOfIngredient(item ingredient.Ingredient) ingredientDetailOutput {
	return ingredientDetailOutput{
		ID:                    int64(item.ID),
		BusinessID:            int64(item.BusinessID),
		Name:                  item.Name,
		Aliases:               emptyIfNil(item.Aliases),
		Unit:                  item.Unit.String(),
		PackageGrams:          item.PackageGrams,
		Allergens:             emptyIfNil(item.Allergens),
		CertificationName:     item.Certification.Name,
		CertificationEvidence: item.Certification.Evidence,
		Conversion:            item.Conversion.RawGrams,
		GramsPerUnit:          gramsOfTheUnit(item),
		Missing:               ingredient.Missing(item),
	}
}

// gramsOfTheUnit answers how many grams one unit of the ingredient weighs.
//
// The conversion of a unit into the gram is declared, never assumed: the
// gram is one, the kilogram is a thousand, the package is the weight the
// owner declares, and the piece is one unit of the ingredient, whose weight
// the book never assumes. The answer of a piece is nothing.
func gramsOfTheUnit(item ingredient.Ingredient) float64 {
	grams, err := item.Grams(1)
	if err != nil {
		return 0
	}

	return grams
}
