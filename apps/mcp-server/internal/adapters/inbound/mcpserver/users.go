package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/order"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/user"
)

// The names of the tools of the people.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that starts an
// onboarding is not a tool that creates a row.
//
// The registration of a person is not one of them: the tool of the
// onboarding registers the person and opens their book, and one tool that
// does the whole way is one tool the agent cannot call halfway.
const (
	// ToolGetUser reads the data of a person who is already registered.
	ToolGetUser = "get_user"

	// ToolListUsers reads the people PACA knows.
	ToolListUsers = "list_users"

	// ToolChangeUserStatus moves a user to another status.
	ToolChangeUserStatus = "change_user_status"

	// ToolUpdateUserContactData writes the data of a user.
	ToolUpdateUserContactData = "update_user_contact_data"
)

// The descriptions of the tools of the users.
//
// A description is not a comment: it is the instruction the agent reads
// before it calls the tool. It says what the tool needs, what it does with
// what it receives, and what it must not do.
const (
	// descriptionChangeUserStatusHead tells the agent what the tool does and
	// what it may do with what it receives.
	//
	// The map of the states is not written here: it is read from the domain
	// and rendered for the agent, so the instruction and the rule never
	// drift apart.
	descriptionChangeUserStatusHead = "Move a user to another status, and say why when the status needs it.\n\n" +
		"The status of a user may move like this:\n"

	// descriptionChangeUserStatusTail tells the agent what it must not do.
	descriptionChangeUserStatusTail = "\nA user never returns to 'onboarding': that status belongs to a person who has just been registered, and " +
		"it is never a destination.\n\n" +
		"The observation is REQUIRED when the new status is 'inactive' or 'blocked': those are the states that " +
		"say why the person is not using the product, so ask the person for the reason. The observation must be " +
		"left out when the new status is 'active', because that status carries no reason. PACA refuses the wrong " +
		"combination, and so does the database."

	// descriptionUpdateUserContactData tells the agent how to write data.
	descriptionUpdateUserContactData = "Update the data of a user who is already registered: the full name, the " +
		"phone and the email.\n\n" +
		"Only the fields you send are updated: an empty field is a field that stays as it is. If the person wants " +
		"to correct the name and the email, send those two and leave the phone empty; if they want to correct all " +
		"three, send all three.\n\n" +
		"The phone must carry the country code (for example '+57 321 253 9984'); PACA cleans the number and stores " +
		"the digits only ('573212539984'). The email must be a valid address.\n\n" +
		"This tool never changes the status of the user: use " + ToolChangeUserStatus + " for that."

	// descriptionListUsers tells the agent how to read the people.
	descriptionListUsers = "Read the people PACA knows: their document ID, the full name, the phone, the email " +
		"and the status.\n\n" +
		"It answers with the people in the order of their document ID, and with as many of them as you ask for. " +
		"Ask for the people you need, and ask again when the answer carries more people than you asked for.\n\n" +
		"This tool changes nothing: use " + ToolGetUser + " to read one person, " + ToolUpdateUserContactData +
		" to correct the data of a person, and " + ToolChangeUserStatus + " to move them to another status."

	// descriptionGetUser tells the agent how to read the data of a person.
	descriptionGetUser = "Read the data of a person who is already registered: the document ID, the full name, " +
		"the phone, the email and the status.\n\n" +
		"It answers with what PACA knows about the person, and it fails when no user carries that document ID.\n\n" +
		"This tool changes nothing: use " + ToolUpdateUserContactData + " to correct the data of the person, and " +
		ToolChangeUserStatus + " to move them to another status."
)

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

// getUserInput is the input of the tool that reads the data of a user.
type getUserInput struct {
	// DocumentID is the national document ID of the person.
	DocumentID string `json:"document_id" jsonschema:"The national document ID of the person, with digits only: '1.234.567' and '1234567' are the same person. It is required."`
}

// listUsersInput is the input of the tool that reads the people.
type listUsersInput struct {
	// Limit is the number of people the agent wants to read.
	Limit int `json:"limit,omitempty" jsonschema:"The number of people you want to read. It is optional: leave it out to read the default number of people."`
}

// listUsersOutput is what the tool that reads the people answers.
type listUsersOutput struct {
	// Users are the people PACA knows, in the order of their document ID.
	Users []userOutput `json:"users"`
}

// userOutput is what the tools answer.
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
		Name:        ToolGetUser,
		Description: descriptionGetUser,
	}, s.getUser(users))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolListUsers,
		Description: descriptionListUsers,
	}, s.listUsers(users))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolChangeUserStatus,
		Description: descriptionOfTheStatusTool(),
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
//
// The list comes from the domain, so a state the domain does not know cannot
// reach the schema, and a state the domain adds reaches it.
func statusInputSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[changeUserStatusInput](nil)
	if err != nil {
		panic("the input schema of the tool that changes the status could not be built: " + err.Error())
	}

	if status, ok := schema.Properties["status"]; ok {
		targets := user.AllTargets()
		enum := make([]any, 0, len(targets))
		for _, target := range targets {
			enum = append(enum, target.String())
		}
		status.Enum = enum
	}

	return schema
}

// descriptionOfTheStatusTool returns the instruction of the tool that moves a
// user.
//
// The words are the ones of the protocol, and the map of the states is the
// one of the domain: the instruction is rendered from it, never written a
// second time.
func descriptionOfTheStatusTool() string {
	return descriptionChangeUserStatusHead + descriptionOfTheTransitions() + descriptionChangeUserStatusTail
}

// descriptionOfTheTransitions renders the map of the states of the domain for
// the agent.
//
// Every state is a line, and the destinations of the line are the ones the
// domain allows.
func descriptionOfTheTransitions() string {
	all := user.AllStatuses()

	width := 0
	for _, status := range all {
		if length := len(status.String()); length > width {
			width = length
		}
	}

	var text strings.Builder
	for _, status := range all {
		targets := status.AllowedTargets()
		if len(targets) == 0 {
			continue
		}

		fmt.Fprintf(&text, "  %-*s-> %s\n", width+1, status.String(), namesOf(targets))
	}

	return text.String()
}

// namesOf returns the names of the states, in a sentence an agent reads.
func namesOf(targets []user.Status) string {
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.String())
	}

	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return names[0]
	}

	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}

// getUser is the handler of the tool that reads the data of a user.
//
// The tool reads the person the document ID carries: it changes nothing, and
// it answers with what PACA knows about the person.
func (s *Server) getUser(users ports.UsersQuerier) sdk.ToolHandlerFor[getUserInput, userOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input getUserInput) (*sdk.CallToolResult, userOutput, error) {
		if users == nil {
			return nil, userOutput{}, ports.ErrNoDatabase
		}

		found, err := users.Find(ctx, input.DocumentID)
		if err != nil {
			return nil, userOutput{}, s.toolError(err)
		}

		return nil, outputOf(found), nil
	}
}

// listUsers is the handler of the tool that reads the people.
//
// The tool reads the people PACA knows: it changes nothing, and it answers
// with as many of them as the agent asked for.
func (s *Server) listUsers(users ports.UsersQuerier) sdk.ToolHandlerFor[listUsersInput, listUsersOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input listUsersInput) (*sdk.CallToolResult, listUsersOutput, error) {
		if users == nil {
			return nil, listUsersOutput{}, ports.ErrNoDatabase
		}

		found, err := users.List(ctx, input.Limit)
		if err != nil {
			return nil, listUsersOutput{}, s.toolError(err)
		}

		answer := listUsersOutput{Users: make([]userOutput, 0, len(found))}
		for _, person := range found {
			answer.Users = append(answer.Users, outputOf(person))
		}

		return nil, answer, nil
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
			return nil, userOutput{}, s.toolError(err)
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
			return nil, userOutput{}, s.toolError(err)
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
//
// The text of a Go error never reaches the agent. An error the service does
// not know is logged, and the agent reads a message that says what happened
// without saying how the service failed.
func (s *Server) toolError(err error) error {
	for known, message := range theMessages {
		if errors.Is(err, known) {
			return errors.New(message)
		}
	}

	// The error is one the domain carries, and nobody mapped it: that is a
	// defect of the service, and the log says so.
	if isErrorOfTheDomain(err) {
		s.logger.Error("the error of the domain has no message for the agent",
			slog.Any("error", err))
		return errors.New("the service does not know this failure yet")
	}

	s.logger.Error("the tool failed for a reason the service does not know",
		slog.Any("error", err))

	return errors.New("the tool failed for a reason the service does not know")
}

// isErrorOfTheDomain tells whether the error is one the domain carries.
//
// The catalogue of the domain names them all, and the book of a person has
// its own: the people, the businesses, the accounts and the movements. The
// catalog of the names, the recipes of the bakery, the parties, the price
// list and the orders have theirs. The service uses them to tell a missing
// mapping from a failure it did not expect: the first one is a defect of
// the service, the second one belongs to the library that failed.
func isErrorOfTheDomain(err error) bool {
	known := user.Errors()
	known = append(known, business.Errors()...)
	known = append(known, account.Errors()...)
	known = append(known, ledger.Errors()...)
	known = append(known, purchase.Errors()...)
	known = append(known, ingredient.Errors()...)
	known = append(known, recipe.Errors()...)
	known = append(known, party.Errors()...)
	known = append(known, product.Errors()...)
	known = append(known, order.Errors()...)

	for _, one := range known {
		if errors.Is(err, one) {
			return true
		}
	}

	return false
}
