package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the expiry.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that reads what
// expires is not a tool that corrects the shelf life.
const (
	// ToolCorrectShelfLife writes the correction of the shelf life the
	// owner declared.
	ToolCorrectShelfLife = "correct_shelf_life"

	// ToolReadExpiring reads what expires, and when.
	ToolReadExpiring = "read_expiring"
)

// The description of the tools of the expiry.
//
// The vocabulary of the expiry belongs to the domain: the description reads
// it and never writes it a second time.
var (
	descriptionCorrectShelfLife = buildDescriptionCorrectShelfLife()
	descriptionReadExpiring     = buildDescriptionReadExpiring()
)

// buildDescriptionCorrectShelfLife renders the description of the tool that
// corrects the shelf life of a recipe.
//
// The shelf life of a recipe is approximate until the owner corrects it, and
// the description says so in the words she reads.
func buildDescriptionCorrectShelfLife() string {
	return "Correct the shelf life of a recipe: the number of days what it makes keeps.\n\n" +
		"The shelf life of a recipe is an APPROXIMATION until the owner corrects it: she wrote it down " +
		"because the book asks for it, and she corrects it when she knows it better. The correction is KEPT: " +
		"every correction she declares is written down, and the last one is the shelf life that counts.\n\n" +
		"A correction carries:\n\n" +
		"  - the RECIPE she corrected: send the identity of it, or the name the owner reads ('Pan de molde');\n" +
		"  - the DAYS what it makes keeps, the way she corrected it ('5');\n" +
		"  - the DAY she corrected it.\n\n" +
		"The days the correction moves are NEVER invented by PACA: the number of days is what she said, and " +
		"PACA never fills it by itself. The day of the expiry of a unit is derived by the book: the day the " +
		"unit was made plus the days of the shelf life.\n\n" +
		"The correction moves the units that were ALREADY made: every unit of the recipe that is in the book " +
		"expires again, the day it was made plus the days she corrected. The units of the other recipes do " +
		"not move.\n\n" +
		"The business is REQUIRED: it is the business the recipe belongs to, and PACA never invents one.\n\n" +
		"The recipe is REQUIRED: it is the recipe she corrected. The name of a recipe is not unique inside " +
		"the business, so send the identity of the recipe when the name carries more than one: PACA refuses " +
		"to guess which recipe she corrected.\n\n" +
		"The day is OPTIONAL: the day she corrected it, written the way a person writes it ('2026-03-14'). It " +
		"is the day of today when it is left out.\n\n" +
		"The utterance is OPTIONAL: the sentence of the owner, verbatim. Send it as she said it when she said " +
		"one, and it is kept with the record.\n\n" +
		"confirmed_with is REQUIRED: the word she used to confirm it ('sí', 'confirmado'), and it is kept with " +
		"the record.\n\n" +
		"The answer carries the correction that was written: the recipe, the number of days, the day it was " +
		"declared, and the units whose day of expiry moved with the day each one expires now."
}

// buildDescriptionReadExpiring renders the description of the tool that
// reads what expires.
//
// The expiry is what the owner asks about before she sells, and the
// description says so in the words she reads.
func buildDescriptionReadExpiring() string {
	return "Read what expires, and when: the units of the bakery and the day each one expires.\n\n" +
		"The answer carries, for every unit the business made:\n\n" +
		"  - the name the owner reads of the recipe it was made of, the day it was made and its weight;\n" +
		"  - the DAY it expires ('2026-03-17');\n" +
		"  - the DAYS that are left of it, and whether it is ALREADY expired.\n\n" +
		"The units are ordered by the day they expire: the one that expires first is the first one.\n\n" +
		"The answer carries the units the bakery still holds: the units that already left the book are not " +
		"offered any more, and the ones that were sold and the ones that were wasted are gone.\n\n" +
		"A unit is expired on the day it expires and on every day after it: the shelf life is the number of " +
		"days the unit keeps, and a unit that expired is NEVER sold.\n\n" +
		"The business is REQUIRED: it is the business whose units are read, and PACA never invents one.\n\n" +
		"This tool changes nothing: it reads the units of the business and it writes nothing."
}

// correctShelfLifeInput is the input of the tool that corrects the shelf
// life of a recipe.
type correctShelfLifeInput struct {
	// BusinessID is the business the recipe belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the recipe belongs to. It is required."`

	// RecipeID is the identity of the recipe she corrected.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the recipe she corrected. It is optional: send the recipe_name instead when the name carries only one recipe."`

	// RecipeName is the name the owner reads of the recipe she corrected.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name the owner reads of the recipe she corrected ('Pan de molde'). It is optional: send the recipe_id instead when the name carries more than one recipe."`

	// ShelfLifeDays is the number of days what the recipe makes keeps.
	ShelfLifeDays int `json:"shelf_life_days" jsonschema:"The number of days what the recipe makes keeps, the way the owner corrected it (5). It is required, and PACA never fills it by itself."`

	// Day is the day the correction was declared.
	Day string `json:"day,omitempty" jsonschema:"The day she corrected it, written as '2026-03-14'. It is optional: it means the day of today when it is empty."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty" jsonschema:"The sentence of the owner, verbatim. It is optional: send it as she said it when she said one, and it is kept with the record."`

	// ConfirmedWith is the word the owner used to confirm the correction.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`
}

// correctedUnitOutput is what the answer carries about one unit whose day
// of expiry moved.
type correctedUnitOutput struct {
	// UnitID is the identity of the unit.
	UnitID int64 `json:"unit_id"`

	// Was is the day the unit expired on before the correction.
	Was string `json:"was"`

	// Now is the day the unit expires on, the one the correction gives it.
	Now string `json:"now"`
}

// correctShelfLifeOutput is the answer of the tool that corrects the shelf
// life of a recipe.
type correctShelfLifeOutput struct {
	// ID is the identity of the correction.
	ID int64 `json:"id"`

	// BusinessID is the business the recipe belongs to.
	BusinessID int64 `json:"business_id"`

	// RecipeID is the recipe the correction is of.
	RecipeID int64 `json:"recipe_id"`

	// RecipeName is the name the owner reads of the recipe.
	RecipeName string `json:"recipe_name"`

	// ShelfLifeDays is the number of days what the recipe makes keeps.
	ShelfLifeDays int `json:"shelf_life_days"`

	// Day is the day the correction was declared.
	Day string `json:"day"`

	// Units are the units whose day of expiry moved.
	Units []correctedUnitOutput `json:"units"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance,omitempty"`

	// ConfirmedWith is the word the owner used to confirm the correction.
	ConfirmedWith string `json:"confirmed_with"`
}

// readExpiringInput is the input of the tool that reads what expires.
type readExpiringInput struct {
	// BusinessID is the business whose units are read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose units are read. It is required."`
}

// expiringUnitOutput is what the answer carries about one unit of the
// business.
type expiringUnitOutput struct {
	// UnitID is the identity of the unit.
	UnitID int64 `json:"unit_id"`

	// ProductionID is the production the unit came out of.
	ProductionID int64 `json:"production_id"`

	// RecipeID is the recipe the unit was made of.
	RecipeID int64 `json:"recipe_id"`

	// RecipeName is the name the owner reads of the recipe.
	RecipeName string `json:"recipe_name"`

	// ProducedOn is the day the production was made.
	ProducedOn string `json:"produced_on"`

	// Grams is the weight of the unit.
	Grams float64 `json:"grams"`

	// ExpiresOn is the day the unit expires.
	ExpiresOn string `json:"expires_on"`

	// DaysLeft are the days that are left of the unit.
	DaysLeft int `json:"days_left"`

	// Expired says whether the unit is already expired: a unit that
	// expired is never sold.
	Expired bool `json:"expired"`
}

// readExpiringOutput is the answer of the tool that reads what expires.
type readExpiringOutput struct {
	// Day is the day the answer was read on.
	Day string `json:"day"`

	// Units are the units of the business, in the order they expire.
	Units []expiringUnitOutput `json:"units"`
}

// registerExpiryTools registers the tools of the expiry.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerExpiryTools(corrector ports.ShelfLifeCorrector, reader ports.ExpiringReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolCorrectShelfLife,
		Description: descriptionCorrectShelfLife,
	}, s.correctShelfLife(corrector))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadExpiring,
		Description: descriptionReadExpiring,
	}, s.readExpiring(reader))
}

// correctShelfLife is the handler of the tool that corrects the shelf life
// of a recipe.
//
// The tool writes the correction the owner confirmed and answers with it.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the correction of nobody.
func (s *Server) correctShelfLife(corrector ports.ShelfLifeCorrector) sdk.ToolHandlerFor[correctShelfLifeInput, correctShelfLifeOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input correctShelfLifeInput) (*sdk.CallToolResult, correctShelfLifeOutput, error) {
		if corrector == nil {
			return nil, correctShelfLifeOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, correctShelfLifeOutput{}, s.toolError(recipe.ErrMissingBusiness)
		}

		if input.ShelfLifeDays <= 0 {
			return nil, correctShelfLifeOutput{}, s.toolError(recipe.ErrMissingShelfLife)
		}

		corrected, err := corrector.Correct(ctx, ports.CorrectShelfLife{
			BusinessID:    business.ID(input.BusinessID),
			RecipeID:      input.RecipeID,
			RecipeName:    input.RecipeName,
			ShelfLifeDays: input.ShelfLifeDays,
			Day:           input.Day,
			Utterance:     input.Utterance,
			ConfirmedWith: input.ConfirmedWith,
		})
		if err != nil {
			return nil, correctShelfLifeOutput{}, s.toolError(err)
		}

		return nil, outputOfCorrection(corrected), nil
	}
}

// outputOfCorrection maps the correction of the domain onto the answer of
// the tool.
//
// The units that moved travel with the day they expired on before the
// correction and the day they expire on now: the correction is what moved
// them, and the answer says so.
func outputOfCorrection(corrected ports.CorrectedShelfLife) correctShelfLifeOutput {
	answer := correctShelfLifeOutput{
		ID:            int64(corrected.Correction.ID),
		BusinessID:    int64(corrected.Correction.BusinessID),
		RecipeID:      int64(corrected.Correction.RecipeID),
		RecipeName:    corrected.Correction.RecipeName,
		ShelfLifeDays: corrected.Correction.ShelfLifeDays,
		Day:           corrected.Correction.Day.String(),
		Units:         make([]correctedUnitOutput, 0, len(corrected.Units)),
		Utterance:     corrected.Correction.Utterance,
		ConfirmedWith: corrected.Correction.ConfirmedWith,
	}

	for _, one := range corrected.Units {
		answer.Units = append(answer.Units, correctedUnitOutput{
			UnitID: int64(one.UnitID),
			Was:    one.Was.String(),
			Now:    one.Now.String(),
		})
	}

	return answer
}

// readExpiring is the handler of the tool that reads what expires.
//
// The tool reads the units of the business and answers with the day each one
// expires: it writes nothing.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the expiry of nobody.
func (s *Server) readExpiring(reader ports.ExpiringReader) sdk.ToolHandlerFor[readExpiringInput, readExpiringOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readExpiringInput) (*sdk.CallToolResult, readExpiringOutput, error) {
		if reader == nil {
			return nil, readExpiringOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, readExpiringOutput{}, s.toolError(recipe.ErrMissingBusiness)
		}

		read, err := reader.Read(ctx, ports.ReadExpiring{
			BusinessID: business.ID(input.BusinessID),
		})
		if err != nil {
			return nil, readExpiringOutput{}, s.toolError(err)
		}

		answer := readExpiringOutput{
			Day:   read.Day.String(),
			Units: make([]expiringUnitOutput, 0, len(read.Units)),
		}

		for _, one := range read.Units {
			answer.Units = append(answer.Units, outputOfExpiring(one, read.Day))
		}

		return nil, answer, nil
	}
}

// outputOfExpiring maps the unit of the domain onto the answer of the tool.
//
// The days that are left are counted from the day the answer was read, and
// the domain is the one who says whether the unit expired: the rule of the
// expiry is the rule of the expiry, and not the rule of the sale. A read
// with no day expires nothing.
func outputOfExpiring(unit recipe.ExpiringUnit, day ledger.Date) expiringUnitOutput {
	return expiringUnitOutput{
		UnitID:       int64(unit.ID),
		ProductionID: int64(unit.ProductionID),
		RecipeID:     int64(unit.RecipeID),
		RecipeName:   unit.RecipeName,
		ProducedOn:   unit.ProducedOn.String(),
		Grams:        unit.Grams,
		ExpiresOn:    unit.ExpiresOn.String(),
		DaysLeft:     unit.DaysLeft(day),
		Expired:      unit.Expired(day),
	}
}
