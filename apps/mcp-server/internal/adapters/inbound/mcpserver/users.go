package mcpserver

import (
	"context"
	"errors"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/core/domain/user"
)

// The names of the three tools of the users.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that starts an
// onboarding is not a tool that creates a row.
const (
	// ToolOnboardUser registers a person and starts their onboarding.
	ToolOnboardUser = "onboard_user"

	// ToolChangeUserStatus moves a user to another status.
	ToolChangeUserStatus = "change_user_status"

	// ToolUpdateUserContactData writes the data of a user.
	ToolUpdateUserContactData = "update_user_contact_data"
)

// The descriptions of the three tools.
//
// A description is not a comment: it is the instruction the agent reads
// before it calls the tool. It says what the tool needs, what it does with
// what it receives, and what it must not do.
const (
	// descriptionOnboardUser tells the agent how to start an onboarding.
	descriptionOnboardUser = "Begin the onboarding of a person who uses PACA, and register them.\n\n" +
		"The national document ID is REQUIRED: it is the document of the person and PACA never invents one. " +
		"Ask the person for it, and if they do not know it, do not call this tool.\n\n" +
		"The full name is OPTIONAL. Ask kindly for it, because a person registered with their name is a person " +
		"the team recognises later. If the person prefers not to give it, register them anyway: an onboarding " +
		"without a name is still a valid onboarding.\n\n" +
		"The phone is OPTIONAL, but it must carry the country code (for example '+57 321 253 9984'). PACA cleans " +
		"the number and stores the digits only ('573212539984'). A number without a country code is refused.\n\n" +
		"The email is OPTIONAL, but it must be a valid address. Ask the person to confirm it before you register it.\n\n" +
		"Try to gather as much information as the person is willing to give - the name, the phone and the email - " +
		"and then register the user with everything you have.\n\n" +
		"The new user starts in the 'onboarding' status, with no reason attached to it."

	// descriptionChangeUserStatus tells the agent how to move a user.
	descriptionChangeUserStatus = "Move a user to another status, and say why when the status needs it.\n\n" +
		"The status of a user may move like this:\n" +
		"  onboarding -> active, inactive or blocked\n" +
		"  active     -> inactive or blocked\n" +
		"  inactive   -> active or blocked\n" +
		"  blocked    -> active or inactive\n\n" +
		"A user never returns to 'onboarding': that status belongs to a person who has just been registered, and " +
		"it is never a destination.\n\n" +
		"The observation is REQUIRED when the new status is 'inactive' or 'blocked': those are the states that " +
		"say why the person is not using the product, so ask the person for the reason. The observation must be " +
		"left out when the new status is 'active', because that status carries no reason. The database refuses " +
		"the wrong combination, and so does this tool."

	// descriptionUpdateUserContactData tells the agent how to write data.
	descriptionUpdateUserContactData = "Update the data of a user who is already registered: the full name, the " +
		"phone and the email.\n\n" +
		"Only the fields you send are updated: an empty field is a field that stays as it is. If the person wants " +
		"to correct the name and the email, send those two and leave the phone empty; if they want to correct all " +
		"three, send all three.\n\n" +
		"The phone must carry the country code (for example '+57 321 253 9984'); PACA cleans the number and stores " +
		"the digits only ('573212539984'). The email must be a valid address.\n\n" +
		"This tool never changes the status of the user: use " + ToolChangeUserStatus + " for that."
)

// onboardUserInput is the input of the tool that starts an onboarding.
type onboardUserInput struct {
	// DocumentID is the national document ID of the person.
	DocumentID string `json:"document_id" jsonschema:"The national document ID of the person, with digits only: '1.234.567' and '1234567' are the same person. It is required, and PACA never invents one."`

	// FullName is the complete name of the person.
	FullName string `json:"full_name,omitempty" jsonschema:"The complete name of the person. It is optional: ask kindly for it, and register the person without it when they prefer not to give it."`

	// Phone is the phone number of the person.
	Phone string `json:"phone,omitempty" jsonschema:"The phone of the person, with the country code: '+57 321 253 9984'. It is optional. PACA cleans the number and stores the digits only."`

	// Email is the email address of the person.
	Email string `json:"email,omitempty" jsonschema:"The email address of the person. It is optional, and it must be a valid address."`
}

// changeUserStatusInput is the input of the tool that moves a user.
type changeUserStatusInput struct {
	// DocumentID is the national document ID of the person.
	DocumentID string `json:"document_id" jsonschema:"The national document ID of the person, with digits only. It is required."`

	// Status is the status the user moves to.
	Status string `json:"status" jsonschema:"The status the user moves to: 'active', 'inactive' or 'blocked'. A user never returns to the onboarding."`

	// StatusObservations is the reason of the move.
	StatusObservations string `json:"status_observations,omitempty" jsonschema:"The reason of the move. It is required when the new status is 'inactive' or 'blocked', and it must be left out when the new status is 'active'."`
}

// updateUserContactDataInput is the input of the tool that writes the data
// of a user.
type updateUserContactDataInput struct {
	// DocumentID is the national document ID of the person.
	DocumentID string `json:"document_id" jsonschema:"The national document ID of the person, with digits only. It is required."`

	// FullName is the complete name of the person.
	FullName string `json:"full_name,omitempty" jsonschema:"The complete name of the person. Leave it empty when it does not change."`

	// Phone is the phone number of the person.
	Phone string `json:"phone,omitempty" jsonschema:"The phone of the person, with the country code. Leave it empty when it does not change."`

	// Email is the email address of the person.
	Email string `json:"email,omitempty" jsonschema:"The email address of the person. Leave it empty when it does not change."`
}

// userOutput is what the three tools answer.
//
// The answer carries the row as it is now, so the agent can tell the person
// what PACA knows about them.
type userOutput struct {
	// ID is the national document ID of the person.
	ID int64 `json:"id"`

	// FullName is the complete name of the person.
	FullName string `json:"full_name,omitempty"`

	// Status is the status of the user.
	Status string `json:"status"`

	// StatusObservations is the reason of the current status.
	StatusObservations string `json:"status_observations,omitempty"`

	// Phone is the phone number of the person.
	Phone string `json:"phone,omitempty"`

	// Email is the email address of the person.
	Email string `json:"email,omitempty"`

	// CreatedAt is the moment the row was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt is the moment the row was last written.
	UpdatedAt string `json:"updated_at"`
}

// registerUserTools registers the three tools of the users.
//
// The tools are registered even when the service has no database: the list of
// the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerUserTools(users ports.UsersQuerier) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolOnboardUser,
		Description: descriptionOnboardUser,
	}, s.onboardUser(users))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolChangeUserStatus,
		Description: descriptionChangeUserStatus,
		InputSchema: statusInputSchema(),
	}, s.changeUserStatus(users))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolUpdateUserContactData,
		Description: descriptionUpdateUserContactData,
	}, s.updateUserContactData(users))
}

// statusInputSchema returns the input schema of the tool that moves a user.
//
// The schema is the one the generator infers from the input of the tool, with
// the list of the statuses the user may move to: the agent reads the list
// instead of guessing a state.
func statusInputSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[changeUserStatusInput](nil)
	if err != nil {
		panic("the input schema of the tool that changes the status could not be built: " + err.Error())
	}

	if status, ok := schema.Properties["status"]; ok {
		status.Enum = []any{"active", "inactive", "blocked"}
	}

	return schema
}

// onboardUser is the handler of the tool that starts an onboarding.
func (s *Server) onboardUser(users ports.UsersQuerier) sdk.ToolHandlerFor[onboardUserInput, userOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input onboardUserInput) (*sdk.CallToolResult, userOutput, error) {
		if users == nil {
			return nil, userOutput{}, ports.ErrNoDatabase
		}

		created, err := users.Onboard(ctx, input.DocumentID, input.FullName, input.Phone, input.Email)
		if err != nil {
			return nil, userOutput{}, toolError(err)
		}

		return nil, outputOf(created), nil
	}
}

// changeUserStatus is the handler of the tool that moves a user.
func (s *Server) changeUserStatus(users ports.UsersQuerier) sdk.ToolHandlerFor[changeUserStatusInput, userOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input changeUserStatusInput) (*sdk.CallToolResult, userOutput, error) {
		if users == nil {
			return nil, userOutput{}, ports.ErrNoDatabase
		}

		moved, err := users.ChangeStatus(ctx, input.DocumentID, input.Status, input.StatusObservations)
		if err != nil {
			return nil, userOutput{}, toolError(err)
		}

		return nil, outputOf(moved), nil
	}
}

// updateUserContactData is the handler of the tool that writes the data of a
// user.
func (s *Server) updateUserContactData(users ports.UsersQuerier) sdk.ToolHandlerFor[updateUserContactDataInput, userOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input updateUserContactDataInput) (*sdk.CallToolResult, userOutput, error) {
		if users == nil {
			return nil, userOutput{}, ports.ErrNoDatabase
		}

		updated, err := users.UpdateContactData(ctx, input.DocumentID, input.FullName, input.Phone, input.Email)
		if err != nil {
			return nil, userOutput{}, toolError(err)
		}

		return nil, outputOf(updated), nil
	}
}

// outputOf maps the model of the domain to the answer of the tools.
//
// The timestamps leave as text, in the format the protocol carries: an agent
// reads them, and it does not compute with them.
func outputOf(model user.User) userOutput {
	return userOutput{
		ID:                 int64(model.ID),
		FullName:           model.FullName,
		Status:             model.Status.String(),
		StatusObservations: model.StatusObservations,
		Phone:              model.Phone,
		Email:              model.Email,
		CreatedAt:          timestamp(model.CreatedAt),
		UpdatedAt:          timestamp(model.UpdatedAt),
	}
}

// timestamp returns the moment in the format the protocol carries.
func timestamp(moment time.Time) string {
	if moment.IsZero() {
		return ""
	}
	return moment.UTC().Format(time.RFC3339)
}

// toolError turns the error of a use case into the message the agent reads.
//
// An agent cannot fix a protocol error, and it cannot fix a message that
// says nothing: the message says what happened and what the agent has to do
// about it.
func toolError(err error) error {
	switch {
	case errors.Is(err, user.ErrInvalidID):
		return errors.New("the document ID is not a number: ask the person for the national document ID, with digits only")
	case errors.Is(err, user.ErrInvalidPhone):
		return errors.New("the phone number is not an international number: ask for it again with the country code, for example '+57 321 253 9984'")
	case errors.Is(err, user.ErrInvalidEmail):
		return errors.New("the email address is not valid: ask the person to confirm it")
	case errors.Is(err, user.ErrAlreadyExists):
		return errors.New("there is already a user with that document ID: do not register the person again")
	case errors.Is(err, user.ErrEmailTaken):
		return errors.New("that email address is already registered: ask the person for another one")
	case errors.Is(err, user.ErrPhoneTaken):
		return errors.New("that phone number is already registered: ask the person for another one")
	case errors.Is(err, user.ErrNotFound):
		return errors.New("there is no user with that document ID: check the document with the person before trying again")
	case errors.Is(err, user.ErrInvalidTransition):
		return errors.New("that status change is not allowed: a user never returns to the onboarding, and a status never changes to itself")
	case errors.Is(err, user.ErrObservationRequired):
		return errors.New("that status needs a reason: ask the person why, and send the observation")
	case errors.Is(err, user.ErrUnexpectedObservation):
		return errors.New("that status takes no reason: send the observation only when the new status is inactive or blocked")
	case errors.Is(err, user.ErrUnknownStatus):
		return errors.New("the status must be active, inactive or blocked")
	case errors.Is(err, ports.ErrNoDatabase):
		return errors.New("the service has no database configured: the users cannot be reached")
	case errors.Is(err, user.ErrInvalidValue):
		return errors.New("the database refused the row: check the name, the phone and the email of the person")
	}

	return err
}
