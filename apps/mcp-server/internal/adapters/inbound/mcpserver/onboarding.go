package mcpserver

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
)

// ToolCompleteOnboarding is the name of the tool that completes the
// onboarding of a person.
const ToolCompleteOnboarding = "complete_onboarding"

// descriptionCompleteOnboarding tells the agent how to complete the
// onboarding of a person.
//
// The list of the accounts is read from the domain: the seed of the personal
// business belongs to the domain, and the description never restates it.
var descriptionCompleteOnboarding = buildDescriptionCompleteOnboarding()

// buildDescriptionCompleteOnboarding renders the description of the tool.
//
// The list of the accounts is read from the domain: the seed of the personal
// business belongs to the domain, and the description never restates it.
func buildDescriptionCompleteOnboarding() string {
	return "Complete the onboarding of a person who uses PACA: register them, open their " +
		"Personal Business, and move them to the 'active' status.\n\n" +
		"This is the way PACA onboards a person, and it is ONE call: the person, their business and the accounts of " +
		"their business are written in one transaction, and they are either all there or none of them is.\n\n" +
		"The national document ID is REQUIRED: it is the document of the person and PACA never invents one. " +
		"Ask the person for it, and if they do not know it, do not call this tool.\n\n" +
		"The full name is REQUIRED: a person is registered with their name, and PACA refuses the registration without it. " +
		"Ask the person for their name kindly, and do not call this tool until they give it.\n\n" +
		"The phone is OPTIONAL, but it must carry the country code (for example '+57 321 253 9984'). PACA cleans " +
		"the number and stores the digits only ('573212539984'). A number without a country code is refused.\n\n" +
		"The email is OPTIONAL, but it must be a valid address. Ask the person to confirm it before you register it, " +
		"and register them without it when they prefer not to give it.\n\n" +
		"Try to gather as much information as the person is willing to give - the name, the phone and the email - " +
		"and then register the person with everything you have.\n\n" +
		"The book of the person starts empty: PACA does not invent the money they have. Ask them what they have in " +
		"cash and in the bank, and record it with the tools of the book when this tool answers.\n\n" +
		"This tool is safe to call again for a person that is already registered: PACA completes the person that is " +
		"there and does not register them twice.\n\n" +
		"The book of the person starts with these accounts: " + theNamesOfTheDefaultAccounts() + ".\n\n" +
		"An account is of these kinds: " + theKindsOfTheAccounts() + "."
}

// completeOnboardingInput is the input of the tool that completes the
// onboarding of a person.
type completeOnboardingInput struct {
	// DocumentID is the national document ID of the person.
	DocumentID string `json:"document_id" jsonschema:"The national document ID of the person, with digits only: '1.234.567' and '1234567' are the same person. It is required, and PACA never invents one."`

	// FullName is the complete name of the person.
	FullName string `json:"full_name,omitempty" jsonschema:"The complete name of the person. It is required: ask the person for their name kindly, and do not call this tool without it."`

	// Phone is the phone number of the person.
	Phone string `json:"phone,omitempty" jsonschema:"The phone number of the person, with the country code (for example '+57 321 253 9984'). It is optional: ask kindly for it, and register the person without it when they prefer not to give it."`

	// Email is the email address of the person.
	Email string `json:"email,omitempty" jsonschema:"The email address of the person. It is optional: ask kindly for it, and register the person without it when they prefer not to give it."`
}

// completeOnboardingOutput is the answer of the tool that completes the
// onboarding of a person.
type completeOnboardingOutput struct {
	// DocumentID is the document of the person.
	DocumentID string `json:"document_id"`

	// FullName is the name of the person.
	FullName string `json:"full_name"`

	// Status is the state of the person after the onboarding.
	Status string `json:"status"`

	// BusinessID is the identity of the personal business of the person.
	BusinessID string `json:"business_id"`

	// BusinessName is the name of the personal business of the person.
	BusinessName string `json:"business_name"`

	// BusinessKind is the kind of the business of the person.
	BusinessKind string `json:"business_kind"`

	// Currency is the currency of the book of the person.
	Currency string `json:"currency"`

	// Accounts are the accounts the book of the person starts with.
	Accounts []accountOutput `json:"accounts"`
}

// accountOutput is one account of the book of a person.
type accountOutput struct {
	// ID is the identity of the account.
	ID string `json:"id"`

	// Kind is the kind of the account.
	Kind string `json:"kind"`

	// Name is the name the person reads in their book.
	Name string `json:"name"`
}

// registerOnboardingTools registers the tool that completes the onboarding.
//
// The tool is a property of the transport: it does not depend on the
// database. A server without a database still exposes it, and every call
// answers that the database is not configured.
func (s *Server) registerOnboardingTools(onboarding ports.OnboardingCompleter) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolCompleteOnboarding,
		Description: descriptionCompleteOnboarding,
	}, s.completeOnboarding(onboarding))
}

// completeOnboarding is the handler of the tool that completes the
// onboarding of a person.
func (s *Server) completeOnboarding(onboarding ports.OnboardingCompleter) sdk.ToolHandlerFor[completeOnboardingInput, completeOnboardingOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input completeOnboardingInput) (*sdk.CallToolResult, completeOnboardingOutput, error) {
		if onboarding == nil {
			return nil, completeOnboardingOutput{}, ports.ErrNoDatabase
		}

		completed, err := onboarding.Complete(ctx, input.DocumentID, input.FullName, input.Phone, input.Email)
		if err != nil {
			return nil, completeOnboardingOutput{}, s.toolError(err)
		}

		accounts := make([]accountOutput, 0, len(completed.Accounts))
		for _, one := range completed.Accounts {
			accounts = append(accounts, accountOutput{
				ID:   fmt.Sprintf("%d", one.ID),
				Kind: one.Kind.String(),
				Name: one.Name,
			})
		}

		return nil, completeOnboardingOutput{
			DocumentID:   fmt.Sprintf("%d", completed.User.ID),
			FullName:     completed.User.FullName,
			Status:       completed.User.Status.String(),
			BusinessID:   fmt.Sprintf("%d", completed.Business.ID),
			BusinessName: completed.Business.Name,
			BusinessKind: completed.Business.Kind.String(),
			Currency:     completed.Business.Currency,
			Accounts:     accounts,
		}, nil
	}
}

// theNamesOfTheDefaultAccounts renders the seed of the personal business for
// the description of the tool.
//
// The seed belongs to the domain: the description reads it and never writes
// it a second time.
func theNamesOfTheDefaultAccounts() string {
	seed := account.DefaultPersonalAccounts()

	names := make([]string, 0, len(seed))
	for _, one := range seed {
		names = append(names, one.Name)
	}

	return strings.Join(names, ", ")
}

// theKindsOfTheAccounts renders the kinds of an account for the description
// of the tool.
//
// The kinds belong to the domain: the description reads them and never
// writes them a second time.
func theKindsOfTheAccounts() string {
	kinds := account.AllKinds()

	names := make([]string, 0, len(kinds))
	for _, one := range kinds {
		names = append(names, one.String())
	}

	return strings.Join(names, ", ")
}
