package mcpserver

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/party"
)

// The names of the tools of the parties.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that opens a party
// is not a tool that creates a row.
const (
	// ToolOpenParty opens a party of the business: the person, the store
	// or the farm that buys from it or that it buys from.
	ToolOpenParty = "open_party"

	// ToolReadParties reads the parties of the business.
	ToolReadParties = "read_parties"
)

// The descriptions of the tools of the parties.
//
// The kinds of a party and the roles it holds belong to the domain: the
// description reads them and never writes them a second time.
var (
	descriptionOpenParty   = buildDescriptionOpenParty()
	descriptionReadParties = buildDescriptionReadParties()
)

// buildDescriptionOpenParty renders the description of the tool that opens
// a party.
//
// The kinds of a party and the roles it holds belong to the domain: the
// description reads them and never writes them a second time.
func buildDescriptionOpenParty() string {
	return "Open a party of the business: the person, the store or the farm that buys from the bakery, or that the " +
		"bakery buys from.\n\n" +
		"The business is REQUIRED: it is the business of its own — the one that knows the clients and the providers " +
		"of the bakery. The Personal Business of a person holds the money of that person and not the clients of " +
		"the bakery, and PACA refuses a party of it.\n\n" +
		"The kind is REQUIRED: it is one of " + theKindsOfTheParties() + ".\n\n" +
		"  - a store REQUIRES the legal human that represents it ('legal_human_name'), and PACA refuses a store " +
		"without one;\n" +
		"  - a farm CARRIES the legal human and its legal figure only when it has one: some farms have a legal " +
		"figure and some do not, and PACA never invents one;\n" +
		"  - a person REFUSES the legal human: it is a person, and the legal human that represents a store or a " +
		"farm is NEVER a second person of PACA.\n\n" +
		"The name is REQUIRED: it is the name the owner reads ('La Panadería de Don Jaime'). The name of a party " +
		"is not unique inside the business: two parties may share it, and the reading of the parties lists them " +
		"all.\n\n" +
		"The roles are REQUIRED: they are the roles the party holds (" + theRolesOfTheParties() + "). A party may " +
		"hold both of them, and a party that holds none of them is refused: it is neither a client nor a " +
		"provider.\n\n" +
		"The phone, the email and the address are OPTIONAL: they are the facts of the party, and PACA never " +
		"invents them. The phone carries its country code ('+57 321 253 9984'), and the email is a valid address.\n\n" +
		"The legal figure ('legal_figure') is OPTIONAL: it is the legal figure of the legal human ('S.A.S.', " +
		"'Ltda.'), and it is the figure of a farm that has one.\n\n" +
		"The tax identity ('tax_id') is OPTIONAL: it is the NIT of a person, of a store or of a farm. PACA never " +
		"asks for it, and it never assumes one: the book does not know the format of a tax identity, and it keeps " +
		"it the way the owner said it.\n\n" +
		"The answer carries the party as it is now: its identity, its kind, its name, its roles, the way of " +
		"reaching it, its address, the legal human that represents it and its tax identity."
}

// buildDescriptionReadParties renders the description of the tool that
// reads the parties of a business.
//
// The kinds of a party and the roles it holds belong to the domain: the
// description reads them and never writes them a second time.
func buildDescriptionReadParties() string {
	return "Read the parties of the business: the person, the store and the farm that it knows.\n\n" +
		"The business is REQUIRED: it is the business whose parties are read, and PACA never invents one.\n\n" +
		"The answer carries every party of the business, in the order of their identity, with:\n\n" +
		"  - its kind (" + theKindsOfTheParties() + ");\n" +
		"  - the name the owner reads, and the roles it holds (" + theRolesOfTheParties() + ");\n" +
		"  - the way of reaching it: the phone and the email;\n" +
		"  - its address;\n" +
		"  - the legal human that represents it, and the legal figure of that human when it has one.\n\n" +
		"The name of a party is not unique inside the business: the answer carries every party, and the ones that " +
		"share a name among them.\n\n" +
		"It writes nothing."
}

// openPartyInput is the input of the tool that opens a party.
type openPartyInput struct {
	// BusinessID is the business that knows the party and that owns it.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business that knows the party: the business of its own, the one with the clients and the providers of the bakery. It is required, and PACA never invents one."`

	// Kind is the kind of the party.
	Kind string `json:"kind" jsonschema:"The kind of the party: 'person', 'store' or 'farm'. It is required."`

	// Name is the name the owner reads.
	Name string `json:"name" jsonschema:"The name the owner reads ('La Panadería de Don Jaime'). It is required, and it is not unique inside the business."`

	// Roles are the roles the party holds.
	Roles []string `json:"roles" jsonschema:"The roles the party holds: 'client', 'provider', or both of them. It is required, and a party that holds none of them is refused."`

	// Phone is the phone number of the party.
	Phone string `json:"phone,omitempty" jsonschema:"The phone number of the party, with its country code ('+57 321 253 9984'). It is optional, and PACA never invents it."`

	// Email is the email address of the party.
	Email string `json:"email,omitempty" jsonschema:"The email address of the party. It is optional, and PACA never invents it."`

	// Address is the address of the party.
	Address string `json:"address,omitempty" jsonschema:"The address of the party. It is optional, and PACA never invents it."`

	// LegalHumanName is the name of the human that represents the party.
	LegalHumanName string `json:"legal_human_name,omitempty" jsonschema:"The name of the legal human that represents the party. It is REQUIRED for a store, it is optional for a farm, and a person refuses it."`

	// LegalFigure is the legal figure the legal human has.
	LegalFigure string `json:"legal_figure,omitempty" jsonschema:"The legal figure of the legal human ('S.A.S.', 'Ltda.'). It is optional: some farms have one and some do not."`

	// TaxID is the tax identity of the party.
	TaxID string `json:"tax_id,omitempty" jsonschema:"The tax identity of the party: the NIT of a person, of a store or of a farm ('900123456-7'). It is optional, and PACA never assumes one."`
}

// readPartiesInput is the input of the tool that reads the parties of a
// business.
type readPartiesInput struct {
	// BusinessID is the business whose parties are read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose parties are read. It is required, and PACA never invents one."`
}

// partyOutput is what the tools of the parties answer about one party.
type partyOutput struct {
	// ID is the identity of the party.
	ID int64 `json:"id"`

	// BusinessID is the business that knows the party.
	BusinessID int64 `json:"business_id"`

	// Kind is the kind of the party.
	Kind string `json:"kind"`

	// Name is the name the owner reads.
	Name string `json:"name"`

	// Roles are the roles the party holds.
	Roles []string `json:"roles"`

	// Phone is the phone number of the party.
	Phone string `json:"phone,omitempty"`

	// Email is the email address of the party.
	Email string `json:"email,omitempty"`

	// Address is the address of the party.
	Address string `json:"address,omitempty"`

	// LegalHumanName is the name of the human that represents the party.
	LegalHumanName string `json:"legal_human_name,omitempty"`

	// LegalFigure is the legal figure the legal human has.
	LegalFigure string `json:"legal_figure,omitempty"`

	// TaxID is the tax identity of the party.
	TaxID string `json:"tax_id,omitempty"`
}

// openPartyOutput is the answer of the tool that opens a party.
type openPartyOutput struct {
	// Party is the party that was opened.
	Party partyOutput `json:"party"`
}

// readPartiesOutput is the answer of the tool that reads the parties of a
// business.
type readPartiesOutput struct {
	// Parties are the parties of the business.
	Parties []partyOutput `json:"parties"`
}

// registerPartyTools registers the tools of the parties.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerPartyTools(opener ports.PartyOpener, reader ports.PartiesReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolOpenParty,
		Description: descriptionOpenParty,
	}, s.openParty(opener))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadParties,
		Description: descriptionReadParties,
	}, s.readParties(reader))
}

// openParty is the handler of the tool that opens a party.
//
// The tool writes the party of the business and answers with it.
func (s *Server) openParty(opener ports.PartyOpener) sdk.ToolHandlerFor[openPartyInput, openPartyOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input openPartyInput) (*sdk.CallToolResult, openPartyOutput, error) {
		if opener == nil {
			return nil, openPartyOutput{}, ports.ErrNoDatabase
		}

		opened, err := opener.Open(ctx, ports.OpenParty{
			BusinessID:     input.BusinessID,
			Kind:           input.Kind,
			Name:           input.Name,
			Roles:          input.Roles,
			Phone:          input.Phone,
			Email:          input.Email,
			Address:        input.Address,
			LegalHumanName: input.LegalHumanName,
			LegalFigure:    input.LegalFigure,
			TaxID:          input.TaxID,
		})
		if err != nil {
			return nil, openPartyOutput{}, s.toolError(err)
		}

		return nil, openPartyOutput{Party: outputOfParty(opened)}, nil
	}
}

// readParties is the handler of the tool that reads the parties of a
// business.
//
// The tool reads the parties the business knows: it changes nothing, and it
// answers with every one of them.
func (s *Server) readParties(reader ports.PartiesReader) sdk.ToolHandlerFor[readPartiesInput, readPartiesOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readPartiesInput) (*sdk.CallToolResult, readPartiesOutput, error) {
		if reader == nil {
			return nil, readPartiesOutput{}, ports.ErrNoDatabase
		}

		held, err := reader.Read(ctx, ports.ReadParties{BusinessID: input.BusinessID})
		if err != nil {
			return nil, readPartiesOutput{}, s.toolError(err)
		}

		parties := make([]partyOutput, 0, len(held))
		for _, one := range held {
			parties = append(parties, outputOfParty(one))
		}

		return nil, readPartiesOutput{Parties: parties}, nil
	}
}

// outputOfParty maps the model of the domain onto the answer of the tools
// of the parties.
func outputOfParty(model party.Party) partyOutput {
	roles := make([]string, 0, len(model.Roles))
	for _, role := range model.Roles {
		roles = append(roles, role.String())
	}

	return partyOutput{
		ID:             int64(model.ID),
		BusinessID:     int64(model.BusinessID),
		Kind:           model.Kind.String(),
		Name:           model.Name,
		Roles:          roles,
		Phone:          model.Phone,
		Email:          model.Email,
		Address:        model.Address,
		LegalHumanName: model.LegalHuman.Name,
		LegalFigure:    model.LegalHuman.Figure,
		TaxID:          model.TaxID,
	}
}

// theKindsOfTheParties renders the kinds of a party for the description of
// the tool.
//
// The kinds belong to the domain: the description reads them and never
// writes them a second time.
func theKindsOfTheParties() string {
	kinds := party.AllKinds()

	names := make([]string, 0, len(kinds))
	for _, one := range kinds {
		names = append(names, one.String())
	}

	return strings.Join(names, ", ")
}

// theRolesOfTheParties renders the roles of a party for the description of
// the tool.
//
// The roles belong to the domain: the description reads them and never
// writes them a second time.
func theRolesOfTheParties() string {
	roles := party.AllRoles()

	names := make([]string, 0, len(roles))
	for _, one := range roles {
		names = append(names, one.String())
	}

	return strings.Join(names, ", ")
}
