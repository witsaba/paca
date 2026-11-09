package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names of the tools of the rotation.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that reads the
// rotation is not a tool that writes a sale.
const (
	// ToolReadRotation reads the order in which the units of the book
	// leave it.
	ToolReadRotation = "read_rotation"
)

// The description of the tools of the rotation.
//
// The vocabulary of the rotation belongs to the domain: the description
// reads it and never writes it a second time.
var (
	descriptionReadRotation = buildDescriptionReadRotation()
)

// buildDescriptionReadRotation renders the description of the tool that
// reads the rotation of the book.
//
// The rotation is the ORDER in which the units leave the book, and the
// description says so in the words the agent reads: the earliest expiry
// first, and no money at all.
func buildDescriptionReadRotation() string {
	return "Read the rotation of the book: the order in which the units of the bakery leave it, and " +
		"when each one expires.\n\n" +
		"The answer carries, for every unit the bakery still holds:\n\n" +
		"  - the name the owner reads of the product it was made of, the day it was made and its weight;\n" +
		"  - the PLACE it takes in that order: the first one that leaves is the first one;\n" +
		"  - the DAY it expires ('2026-03-17');\n" +
		"  - the DAYS that are left of it.\n\n" +
		"The rotation follows the EARLIEST EXPIRY: the unit that expires first is the first one that leaves " +
		"the book. The units that expire on the same day keep a stable order: the production first, and the " +
		"position of the unit in it.\n\n" +
		"The answer carries the units the bakery still holds: the units that already left the book are not " +
		"offered any more, and the ones that were sold, wasted, consumed or given away are gone.\n\n" +
		"The answer carries NO money: the valuation of the stock follows the FIFO, and it is another rule. " +
		"The rotation says what leaves first and when it expires, and never what it is worth.\n\n" +
		"The business is REQUIRED: it is the business whose book is read, and PACA never invents one.\n\n" +
		"The product is OPTIONAL: send the identity of the product, or the name the owner reads of it " +
		"('Pan de molde'), to read the rotation of that product alone, and leave it out to read the rotation " +
		"of the whole book of the business.\n\n" +
		"This tool changes nothing: it reads the units of the book and it writes nothing."
}

// readRotationInput is the input of the tool that reads the rotation of the
// book.
type readRotationInput struct {
	// BusinessID is the business whose book is read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose book is read. It is required."`

	// RecipeID is the identity of the product the caller asks about.
	RecipeID int64 `json:"recipe_id,omitempty" jsonschema:"The identity of the product whose rotation is read. It is optional: send the recipe_name instead when the name carries only one product."`

	// RecipeName is the name the owner reads of the product she asks about.
	RecipeName string `json:"recipe_name,omitempty" jsonschema:"The name the owner reads of the product whose rotation is read ('Pan de molde'). It is optional: leave it out to read the rotation of the whole book of the business."`
}

// rotationUnitOutput is what the answer carries about one unit of the book.
type rotationUnitOutput struct {
	// UnitID is the identity of the unit.
	UnitID int64 `json:"unit_id"`

	// Order is the place the unit takes in the rotation: the first one
	// that leaves is the first one.
	Order int `json:"order"`

	// ProductionID is the production the unit came out of.
	ProductionID int64 `json:"production_id"`

	// Position is the order the unit was made in.
	Position int `json:"position"`

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
}

// readRotationOutput is the answer of the tool that reads the rotation of
// the book.
type readRotationOutput struct {
	// Day is the day the answer was read on.
	Day string `json:"day"`

	// RecipeID is the product the rotation is of. It is zero when the
	// rotation is of the whole book of the business.
	RecipeID int64 `json:"recipe_id,omitempty"`

	// RecipeName is the name the owner reads of the product. It is empty
	// when the rotation is of the whole book of the business.
	RecipeName string `json:"recipe_name,omitempty"`

	// Units are the units of the book, in the order they leave it.
	Units []rotationUnitOutput `json:"units"`
}

// registerRotationTools registers the tools of the rotation.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerRotationTools(reader ports.RotationReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadRotation,
		Description: descriptionReadRotation,
	}, s.readRotation(reader))
}

// readRotation is the handler of the tool that reads the rotation of the
// book.
//
// The tool reads the units that are still in the book and answers with the
// order they leave it: it writes nothing.
//
// The business is required: the tool refuses the business that is not
// there, and it never asks the use case for the rotation of nobody.
func (s *Server) readRotation(reader ports.RotationReader) sdk.ToolHandlerFor[readRotationInput, readRotationOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readRotationInput) (*sdk.CallToolResult, readRotationOutput, error) {
		if reader == nil {
			return nil, readRotationOutput{}, ports.ErrNoDatabase
		}

		if input.BusinessID <= 0 {
			return nil, readRotationOutput{}, s.toolError(recipe.ErrMissingBusiness)
		}

		read, err := reader.Read(ctx, ports.ReadRotation{
			BusinessID: business.ID(input.BusinessID),
			RecipeID:   input.RecipeID,
			RecipeName: input.RecipeName,
		})
		if err != nil {
			return nil, readRotationOutput{}, s.toolError(err)
		}

		answer := readRotationOutput{
			Day:        read.Day.String(),
			RecipeID:   int64(read.RecipeID),
			RecipeName: read.RecipeName,
			Units:      make([]rotationUnitOutput, 0, len(read.Units)),
		}

		for _, one := range read.Units {
			answer.Units = append(answer.Units, rotationUnitOutput{
				UnitID:       int64(one.UnitID),
				Order:        one.Order,
				ProductionID: int64(one.ProductionID),
				Position:     one.Position,
				RecipeID:     int64(one.RecipeID),
				RecipeName:   one.RecipeName,
				ProducedOn:   one.ProducedOn.String(),
				Grams:        one.Grams,
				ExpiresOn:    one.ExpiresOn.String(),
				DaysLeft:     one.DaysLeft,
			})
		}

		return nil, answer, nil
	}
}
