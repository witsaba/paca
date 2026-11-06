package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the recipes.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records a
// recipe is not a tool that bakes it.
const (
	// ToolRecordRecipe writes the recipe of the bakery that the owner
	// confirmed.
	ToolRecordRecipe = "record_recipe"
)

// The description of the tool of the recipes.
//
// The shape of the recipe belongs to the domain: the description reads it
// and never writes it a second time.
var descriptionRecordRecipe = buildDescriptionRecordRecipe()

// buildDescriptionRecordRecipe renders the description of the tool that
// records a recipe.
//
// A recipe is a guide to learn and to teach: the description says what the
// tool writes, so the agent offers the owner a guide and not a list.
func buildDescriptionRecordRecipe() string {
	return "Record the recipe the owner just confirmed, and write it in the book of the business.\n\n" +
		"A recipe is a guide to learn and to teach, not a list of ingredients: it carries the name, the " +
		"ingredients with their grams, the steps, the yield and the shelf life of what it makes.\n\n" +
		"The business is REQUIRED: it is the business the recipe belongs to, and PACA never invents one.\n\n" +
		"The name is REQUIRED: it is the name the owner reads ('Pan de molde').\n\n" +
		"Every ingredient carries the identity of the ingredient of the catalog and the grams of it. At least " +
		"one ingredient is REQUIRED:\n\n" +
		"  - ingredient_id is REQUIRED: send the identity the tool " + ToolMatchItems + " answered for the " +
		"name. The catalog is the one owner of the ingredient, and PACA refuses an ingredient the catalog of the " +
		"business does not hold.\n" +
		"  - grams is REQUIRED: how many grams of it the recipe needs. The grams are never inferred, and PACA " +
		"refuses an ingredient without them.\n\n" +
		"Every step carries what the owner does. At least one step is REQUIRED:\n\n" +
		"  - instruction is REQUIRED: what the owner does in the step ('Amasar 20 minutos').\n" +
		"  - cue is OPTIONAL: the sign the owner looks for ('la masa que holds the forma'). Leave it out when " +
		"she named none.\n" +
		"  - minutes is OPTIONAL: the time of the step, in minutes. Leave it out when she named none.\n" +
		"  - temperature is OPTIONAL: the temperature of the step, in degrees Celsius. Leave it out when she " +
		"named none.\n" +
		"  - labor_kind is OPTIONAL: the kind of the labor the step is ('" + theNamesOfTheKindsOfTheLabor() +
		"'). The owner says it in many ways ('" + theSpokenNamesOfTheKindsOfTheLabor() + "'), and PACA keeps " +
		"the kind of the book. Leave it out when she named none, and the step is then the labor of the hands.\n\n" +
		"The steps keep the order the owner said them in: send them in that order, and PACA writes them in that " +
		"order.\n\n" +
		"yield_units is REQUIRED: the number of units the recipe makes. yield_grams is REQUIRED: the weight of " +
		"one unit, in grams. The yield is a declared constant of the recipe, and it is never measured per bake.\n\n" +
		"shelf_life_days is REQUIRED: the number of days what the recipe makes keeps. It is approximate until " +
		"the owner corrects it.\n\n" +
		"The utterance is OPTIONAL: the sentence of the owner, verbatim. Send it as she said it when she said " +
		"one, and it is kept with the record.\n\n" +
		"confirmed_with is REQUIRED: the word she used to confirm it ('sí', 'confirmado'), and it is kept with " +
		"the record.\n\n" +
		"The answer carries the recipe that was written: its name, its ingredients with their grams, its steps " +
		"in order, its yield and its shelf life."
}

// recordRecipeIngredientInput is one ingredient of the recipe.
type recordRecipeIngredientInput struct {
	// IngredientID is the identity of the ingredient of the catalog.
	IngredientID int64 `json:"ingredient_id" jsonschema:"The identity of the ingredient of the catalog the recipe names. It is required: send the identity the tool match_items answered."`

	// Grams is how many grams of it the recipe needs.
	Grams float64 `json:"grams" jsonschema:"How many grams of the ingredient the recipe needs. It is required and it is positive."`
}

// recordRecipeStepInput is one step of the process of the recipe.
type recordRecipeStepInput struct {
	// Instruction is what the owner does in this step.
	Instruction string `json:"instruction" jsonschema:"What the owner does in this step ('Amasar 20 minutos'). It is required."`

	// Cue is the sign the owner looks for.
	Cue string `json:"cue,omitempty" jsonschema:"The sign the owner looks for ('la masa que holds la forma'). It is optional: leave it out when she named none."`

	// Minutes is the time of the step, in minutes.
	Minutes int `json:"minutes,omitempty" jsonschema:"The time of the step, in minutes. It is optional: leave it out when the owner named none."`

	// Temperature is the temperature of the step, in degrees Celsius.
	Temperature float64 `json:"temperature,omitempty" jsonschema:"The temperature of the step, in degrees Celsius. It is optional: leave it out when the owner named none."`

	// LaborKind is the kind of the labor the step is.
	LaborKind string `json:"labor_kind,omitempty" jsonschema:"The kind of the labor the step is ('hands_on', 'oven', 'waiting'). It is optional: leave it out when the owner named none, and the step is then the labor of the hands."`
}

// recordRecipeInput is the input of the tool that records a recipe.
type recordRecipeInput struct {
	// BusinessID is the business the recipe belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the recipe belongs to. It is required."`

	// Name is the name the owner reads.
	Name string `json:"name" jsonschema:"The name the owner reads ('Pan de molde'). It is required."`

	// Ingredients are the ingredients of the recipe, with their grams.
	Ingredients []recordRecipeIngredientInput `json:"ingredients" jsonschema:"The ingredients of the recipe, each one with the identity of the ingredient of the catalog and its grams. At least one ingredient is required."`

	// Steps are the steps of the process, in the order the owner said them.
	Steps []recordRecipeStepInput `json:"steps" jsonschema:"The steps of the recipe, in the order the owner said them. At least one step is required."`

	// YieldUnits is the number of units the recipe makes.
	YieldUnits int `json:"yield_units" jsonschema:"The number of units the recipe makes. It is required and it is positive."`

	// YieldGrams is the weight of one unit, in grams.
	YieldGrams float64 `json:"yield_grams" jsonschema:"The weight of one unit of the recipe, in grams. It is required and it is positive."`

	// ShelfLifeDays is the number of days what the recipe makes keeps.
	ShelfLifeDays int `json:"shelf_life_days" jsonschema:"The number of days what the recipe makes keeps. It is required and it is positive."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty" jsonschema:"The sentence of the owner, verbatim. It is optional: send it as she said it when she said one, and it is kept with the record."`

	// ConfirmedWith is the word the owner used to confirm the recipe.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`
}

// recipeIngredientOutput is what the answer carries about one ingredient of
// the recipe.
type recipeIngredientOutput struct {
	// IngredientID is the identity of the ingredient of the catalog.
	IngredientID int64 `json:"ingredient_id"`

	// Grams is how many grams of it the recipe needs.
	Grams float64 `json:"grams"`
}

// recipeStepOutput is what the answer carries about one step of the recipe.
type recipeStepOutput struct {
	// Instruction is what the owner does in this step.
	Instruction string `json:"instruction"`

	// Cue is the sign the owner looks for.
	Cue string `json:"cue,omitempty"`

	// Minutes is the time of the step, in minutes.
	Minutes int `json:"minutes,omitempty"`

	// Temperature is the temperature of the step, in degrees Celsius.
	Temperature float64 `json:"temperature,omitempty"`

	// LaborKind is the kind of the labor the step is.
	LaborKind string `json:"labor_kind,omitempty"`
}

// recordRecipeOutput is the answer of the tool that records a recipe.
//
// The answer carries the recipe that was written: the guide the owner reads
// in her book.
type recordRecipeOutput struct {
	// ID is the identity of the recipe.
	ID int64 `json:"id"`

	// BusinessID is the business the recipe belongs to.
	BusinessID int64 `json:"business_id"`

	// Name is the name the owner reads.
	Name string `json:"name"`

	// Ingredients are the ingredients of the recipe, with their grams.
	Ingredients []recipeIngredientOutput `json:"ingredients"`

	// Steps are the steps of the process, in the order the owner said them.
	Steps []recipeStepOutput `json:"steps"`

	// YieldUnits is the number of units the recipe makes.
	YieldUnits int `json:"yield_units"`

	// YieldGrams is the weight of one unit, in grams.
	YieldGrams float64 `json:"yield_grams"`

	// ShelfLifeDays is the number of days what the recipe makes keeps.
	ShelfLifeDays int `json:"shelf_life_days"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty"`

	// ConfirmedWith is the word the owner used to confirm the recipe.
	ConfirmedWith string `json:"confirmed_with"`
}

// registerRecipeTools registers the tools of the recipes.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerRecipeTools(recorder ports.RecipeRecorder) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordRecipe,
		Description: descriptionRecordRecipe,
	}, s.recordRecipe(recorder))
}

// recordRecipe is the handler of the tool that records a recipe.
//
// The tool writes the recipe the owner confirmed and answers with it.
func (s *Server) recordRecipe(recorder ports.RecipeRecorder) sdk.ToolHandlerFor[recordRecipeInput, recordRecipeOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordRecipeInput) (*sdk.CallToolResult, recordRecipeOutput, error) {
		if recorder == nil {
			return nil, recordRecipeOutput{}, ports.ErrNoDatabase
		}

		recorded, err := recorder.Record(ctx, requestOfRecipe(input))
		if err != nil {
			return nil, recordRecipeOutput{}, s.toolError(err)
		}

		return nil, outputOfRecipe(recorded), nil
	}
}

// requestOfRecipe turns what the agent sent into the request of the use
// case.
//
// The port speaks in the words of the agent and in numbers: the use case
// turns them into the model of the domain, and the domain decides what may
// enter the book.
func requestOfRecipe(input recordRecipeInput) ports.RecordRecipe {
	ingredients := make([]ports.RecipeIngredient, 0, len(input.Ingredients))
	for _, one := range input.Ingredients {
		ingredients = append(ingredients, ports.RecipeIngredient{
			IngredientID: one.IngredientID,
			Grams:        one.Grams,
		})
	}

	steps := make([]ports.RecipeStep, 0, len(input.Steps))
	for _, one := range input.Steps {
		steps = append(steps, ports.RecipeStep{
			Instruction: one.Instruction,
			Cue:         one.Cue,
			Minutes:     one.Minutes,
			Temperature: one.Temperature,
			LaborKind:   one.LaborKind,
		})
	}

	return ports.RecordRecipe{
		BusinessID:    business.ID(input.BusinessID),
		Name:          input.Name,
		Ingredients:   ingredients,
		Steps:         steps,
		YieldUnits:    input.YieldUnits,
		YieldGrams:    input.YieldGrams,
		ShelfLifeDays: input.ShelfLifeDays,
		Utterance:     input.Utterance,
		ConfirmedWith: input.ConfirmedWith,
	}
}

// outputOfRecipe maps the recipe of the domain onto the answer of the tool.
//
// The answer carries the guide the owner reads: its ingredients with their
// grams, its steps in order, its yield and its shelf life.
func outputOfRecipe(recorded recipe.Recipe) recordRecipeOutput {
	ingredients := make([]recipeIngredientOutput, 0, len(recorded.Ingredients))
	for _, one := range recorded.Ingredients {
		ingredients = append(ingredients, recipeIngredientOutput{
			IngredientID: int64(one.IngredientID),
			Grams:        one.Grams,
		})
	}

	steps := make([]recipeStepOutput, 0, len(recorded.Steps))
	for _, one := range recorded.Steps {
		steps = append(steps, recipeStepOutput{
			Instruction: one.Instruction,
			Cue:         one.Cue,
			Minutes:     one.Minutes,
			Temperature: one.Temperature,
			LaborKind:   one.Kind.String(),
		})
	}

	return recordRecipeOutput{
		ID:            int64(recorded.ID),
		BusinessID:    int64(recorded.BusinessID),
		Name:          recorded.Name,
		Ingredients:   ingredients,
		Steps:         steps,
		YieldUnits:    recorded.Yield.Units,
		YieldGrams:    recorded.Yield.Grams,
		ShelfLifeDays: recorded.ShelfLifeDays,
		Utterance:     recorded.Utterance,
		ConfirmedWith: recorded.ConfirmedWith,
	}
}
