package mcpserver

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
)

// The names of the tools of the businesses.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that opens a
// business is not a tool that creates a row.
const (
	// ToolOpenBusiness opens a business of a person, with the book of
	// accounts it starts with.
	ToolOpenBusiness = "open_business"

	// ToolReadBusinesses reads the businesses of a person.
	ToolReadBusinesses = "read_businesses"
)

// The descriptions of the tools of the businesses.
//
// The list of the kinds of a business is read from the domain: it belongs to
// the domain, and the description never writes it a second time. The seed of
// the book of a business is read from the domain too.
var (
	descriptionOpenBusiness   = buildDescriptionOpenBusiness()
	descriptionReadBusinesses = "Read the businesses of a person: the Personal Business and the other " +
		"businesses they own.\n\n" +
		"The document ID of the owner is REQUIRED: it is the document of the person whose businesses you " +
		"want to read, and PACA never invents one. Ask the person for it, and do not call this tool without it.\n\n" +
		"The answer carries every business of the person, with its identity, its name, its kind and the " +
		"currency of its book.\n\n" +
		"A business is of these kinds: " + theKindsOfTheBusinesses() + ". The kind 'personal' is the book of " +
		"the money of the person, and the kind 'business' is the business of their own.\n\n" +
		"This tool changes nothing: use " + ToolOpenBusiness + " to open a business."
)

// buildDescriptionOpenBusiness renders the description of the tool that
// opens a business.
//
// The list of the kinds of a business is read from the domain: it belongs to
// the domain, and the description never writes it a second time. The seed of
// the book of a business is read from the domain too.
func buildDescriptionOpenBusiness() string {
	return "Open a business for a person: the business itself, and the book of accounts it starts with.\n\n" +
		"The document ID of the owner is REQUIRED: it is the document of the person who owns the business, " +
		"and PACA never invents one. Ask the person for it, and do not call this tool without it.\n\n" +
		"The name is REQUIRED: it is the name the person reads in their book ('Panadería del barrio'). A " +
		"business without a name has nothing to show in the book of its owner, and PACA refuses it. Ask the " +
		"person for the name, and do not call this tool without it.\n\n" +
		"The currency is OPTIONAL: it is the currency of the book of the business, and it means the peso " +
		"('" + business.DefaultCurrency + "') when it is empty.\n\n" +
		"The business is always of the kind '" + business.KindBusiness.String() + "': the Personal Business of " +
		"a person is opened by the onboarding, and never here.\n\n" +
		"The name of a business is taken inside the businesses of its owner: PACA refuses a name that is " +
		"already there, and it refuses to open a business of the owner with the name of another business of " +
		"the same owner. The same name in the businesses of another person is free.\n\n" +
		"The business and its accounts are written in one transaction: the book of a business starts whole " +
		"or not at all, and the business is not there when its accounts are not.\n\n" +
		"The book of the business starts with these accounts: " + theNamesOfTheBusinessAccounts() + ".\n\n" +
		"A debt is not one of them: it is created when it happens."
}

// openBusinessInput is the input of the tool that opens a business.
type openBusinessInput struct {
	// OwnerID is the national document ID of the person who owns the
	// business.
	OwnerID string `json:"owner_id" jsonschema:"The national document ID of the person who owns the business, with digits only. It is required, and PACA never invents one."`

	// Name is the name the person reads in their book.
	Name string `json:"name" jsonschema:"The name the person reads in their book ('Panadería del barrio'). It is required: a business without a name has nothing to show in the book of its owner."`

	// Currency is the currency of the book of the business.
	Currency string `json:"currency,omitempty" jsonschema:"The currency of the book of the business. It is optional: it means the peso ('COP') when it is empty."`
}

// readBusinessesInput is the input of the tool that reads the businesses of
// a person.
type readBusinessesInput struct {
	// OwnerID is the national document ID of the person who owns the
	// businesses.
	OwnerID string `json:"owner_id" jsonschema:"The national document ID of the person whose businesses you want to read, with digits only. It is required, and PACA never invents one."`
}

// businessOutput is what the tools of the businesses answer about one
// business.
type businessOutput struct {
	// ID is the identity of the business.
	ID string `json:"id"`

	// Name is the name the person reads.
	Name string `json:"name"`

	// Kind is the kind of the business.
	Kind string `json:"kind"`

	// Currency is the currency of the book of the business.
	Currency string `json:"currency"`
}

// openBusinessOutput is the answer of the tool that opens a business.
type openBusinessOutput struct {
	// Business is the business that was opened.
	Business businessOutput `json:"business"`

	// Accounts are the accounts the book of the business starts with.
	Accounts []accountOutput `json:"accounts"`
}

// readBusinessesOutput is the answer of the tool that reads the businesses
// of a person.
type readBusinessesOutput struct {
	// Businesses are the businesses of the person.
	Businesses []businessOutput `json:"businesses"`
}

// registerBusinessTools registers the tools of the businesses.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerBusinessTools(opener ports.BusinessOpener, querier ports.BusinessQuerier) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolOpenBusiness,
		Description: descriptionOpenBusiness,
	}, s.openBusiness(opener))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadBusinesses,
		Description: descriptionReadBusinesses,
	}, s.readBusinesses(querier))
}

// openBusiness is the handler of the tool that opens a business.
//
// The tool writes the business and the book it starts with, and it answers
// with both of them.
func (s *Server) openBusiness(opener ports.BusinessOpener) sdk.ToolHandlerFor[openBusinessInput, openBusinessOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input openBusinessInput) (*sdk.CallToolResult, openBusinessOutput, error) {
		if opener == nil {
			return nil, openBusinessOutput{}, ports.ErrNoDatabase
		}

		opened, err := opener.Open(ctx, input.OwnerID, input.Name, input.Currency)
		if err != nil {
			return nil, openBusinessOutput{}, s.toolError(err)
		}

		accounts := make([]accountOutput, 0, len(opened.Accounts))
		for _, one := range opened.Accounts {
			accounts = append(accounts, accountOutput{
				ID:   fmt.Sprintf("%d", one.ID),
				Kind: one.Kind.String(),
				Name: one.Name,
			})
		}

		return nil, openBusinessOutput{
			Business: outputOfBusiness(opened.Business),
			Accounts: accounts,
		}, nil
	}
}

// readBusinesses is the handler of the tool that reads the businesses of a
// person.
//
// The tool reads the businesses the person owns: it changes nothing, and it
// answers with every one of them.
func (s *Server) readBusinesses(querier ports.BusinessQuerier) sdk.ToolHandlerFor[readBusinessesInput, readBusinessesOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readBusinessesInput) (*sdk.CallToolResult, readBusinessesOutput, error) {
		if querier == nil {
			return nil, readBusinessesOutput{}, ports.ErrNoDatabase
		}

		owned, err := querier.List(ctx, input.OwnerID)
		if err != nil {
			return nil, readBusinessesOutput{}, s.toolError(err)
		}

		businesses := make([]businessOutput, 0, len(owned))
		for _, one := range owned {
			businesses = append(businesses, outputOfBusiness(one))
		}

		return nil, readBusinessesOutput{Businesses: businesses}, nil
	}
}

// outputOfBusiness maps the model of the domain onto the answer of the tools
// of the businesses.
func outputOfBusiness(model business.Business) businessOutput {
	return businessOutput{
		ID:       fmt.Sprintf("%d", model.ID),
		Name:     model.Name,
		Kind:     model.Kind.String(),
		Currency: model.Currency,
	}
}

// theKindsOfTheBusinesses renders the kinds of a business for the
// description of the tool.
//
// The kinds belong to the domain: the description reads them and never
// writes them a second time.
func theKindsOfTheBusinesses() string {
	kinds := business.AllKinds()

	names := make([]string, 0, len(kinds))
	for _, one := range kinds {
		names = append(names, one.String())
	}

	return strings.Join(names, ", ")
}

// theNamesOfTheBusinessAccounts renders the seed of the book of a business
// for the description of the tool.
//
// The seed belongs to the domain: the description reads it and never writes
// it a second time.
func theNamesOfTheBusinessAccounts() string {
	seed := account.DefaultBusinessAccounts()

	names := make([]string, 0, len(seed))
	for _, one := range seed {
		names = append(names, one.Name)
	}

	return strings.Join(names, ", ")
}
