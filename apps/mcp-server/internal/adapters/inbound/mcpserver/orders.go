package mcpserver

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/order"
)

// The names of the tools of the orders.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records an
// order is not a tool that creates a row.
const (
	// ToolRecordOrder writes an order of the business: the party that
	// ordered, the products and the day it is due.
	ToolRecordOrder = "record_order"

	// ToolReadOrders reads the orders of the business and the round of a
	// day: what is still owed.
	ToolReadOrders = "read_orders"
)

// The descriptions of the tools of the orders.
//
// The states of an order belong to the domain: the description reads them
// and never writes them a second time.
var (
	descriptionRecordOrder = buildDescriptionRecordOrder()
	descriptionReadOrders  = buildDescriptionReadOrders()
)

// buildDescriptionRecordOrder renders the description of the tool that
// records an order.
//
// The states of an order belong to the domain: the description reads them
// and never writes them a second time.
func buildDescriptionRecordOrder() string {
	return "Record an order of the bakery: what the bakery owes to a client, and the day it is due.\n\n" +
		"An order carries the party that ordered, the products it ordered with the quantities of them, and the day " +
		"it is due. The order is written OPEN: an order that was not delivered stays open, and the round of the " +
		"day answers what is still owed. The delivery of an order is written with the route of the round, and " +
		"PACA does not write it here.\n\n" +
		"The business is REQUIRED: it is the business of its own — the one that sells the bread. The Personal " +
		"Business of a person holds the money of that person, and not the orders of the bakery, and PACA refuses " +
		"the orders of it.\n\n" +
		"The party is REQUIRED: it is the client that ordered, and it is named EITHER by 'party_id' OR by the name " +
		"the owner reads ('party_name'). The agent sends one of the two, never neither of them. A party that does " +
		"not hold the role of the client is refused: the bakery does not owe bread to a provider.\n\n" +
		"The day it is due ('due_on') is REQUIRED: it is the day the bread is owed, written as '2026-03-14'. PACA " +
		"never invents one.\n\n" +
		"Every line of the order carries the product, the quantity of it and the unit the owner said. At least one " +
		"line is REQUIRED ('lines'):\n\n" +
		"  - the product is named EITHER by 'product_id' OR by the name the owner reads ('product_name'). The " +
		"products are the ones of the price list of the business, and PACA refuses a product of another business;\n" +
		"  - the quantity ('quantity') is REQUIRED: how many units of the product the line carries (12). It is a " +
		"positive whole number of units: the book never splits a unit, and PACA never guesses the quantity;\n" +
		"  - the unit ('unit') is REQUIRED: the unit the owner said ('pan', 'unidad'). The quantity and its unit " +
		"travel together, and PACA never converts a weight into a number of units.\n\n" +
		"The answer carries the order that was written: its identity ('id'), the business that owns it " +
		"('business_id'), the party that ordered ('party_id'), the name the owner reads of that party " +
		"('party_name'), the day it is due ('due_on'), the state of the order ('state') and its lines ('lines') " +
		"with the product of each one ('product_id'), the name the owner reads of it ('product_name'), the quantity " +
		"('quantity') and the unit ('unit')."
}

// buildDescriptionReadOrders renders the description of the tool that reads
// the orders of a business.
//
// The states of an order belong to the domain: the description reads them
// and never writes them a second time.
func buildDescriptionReadOrders() string {
	return "Read the orders of the business and the round of a day: what is still owed.\n\n" +
		"The round of a day carries the orders that are DUE on it: the bread the bakery owes that day. The round " +
		"answers what is still owed: the orders it carries that were not delivered. An order that was not " +
		"delivered stays open.\n\n" +
		"The business is REQUIRED: it is the business whose orders are read, and PACA never invents one. A " +
		"business that is not there is refused: the answer is never an empty list for a business that does not " +
		"exist.\n\n" +
		"The day ('day') is OPTIONAL: it is the day of the round that is read, written as '2026-03-14'. The round " +
		"of that day answers the orders that are due on it, and the whole book of the orders is read when the " +
		"agent names none.\n\n" +
		"The answer carries the day of the round ('day'), the orders the round carries ('orders') and the ones " +
		"that are still owed ('owed'). Every order carries its identity ('id'), the business that owns it " +
		"('business_id'), the party that ordered ('party_id'), the name the owner reads of that party " +
		"('party_name'), the day it is due ('due_on'), the state of the order ('state') and its lines ('lines') " +
		"with the product of each one ('product_id'), the name the owner reads of it ('product_name'), the quantity " +
		"('quantity') and the unit ('unit').\n\n" +
		"The state of an order is one of " + theStatesOfTheOrders() + ".\n\n" +
		"It writes nothing."
}

// recordOrderLineInput is one line of the order the agent reads.
type recordOrderLineInput struct {
	// ProductID is the identity of the product the line is of.
	ProductID int64 `json:"product_id,omitempty" jsonschema:"The identity of the product the line is of. Send it OR the name the owner reads, never neither of them."`

	// ProductName is the name the owner reads of the product.
	ProductName string `json:"product_name,omitempty" jsonschema:"The name the owner reads of the product ('Pan de molde'). Send it OR the identity of the product, never neither of them."`

	// Quantity is how many units of the product the line carries.
	Quantity int `json:"quantity,omitempty" jsonschema:"How many units of the product the line carries (12). Send it always: it is a positive whole number of units, PACA never fills it, and the domain refuses the order when it is left out."`

	// Unit is the unit of that quantity, the way the owner said it.
	Unit string `json:"unit,omitempty" jsonschema:"The unit of the quantity, the way the owner said it ('pan', 'unidad'). Send it always: the quantity and its unit travel together, and the domain refuses the order when it is left out."`
}

// recordOrderInput is the input of the tool that records an order.
type recordOrderInput struct {
	// BusinessID is the business the order belongs to.
	BusinessID int64 `json:"business_id,omitempty" jsonschema:"The identity of the business the order belongs to: the business of its own, the one that sells the bread. Send it always: PACA never invents one, and the domain refuses the order when it is left out."`

	// PartyID is the identity of the party that ordered.
	PartyID int64 `json:"party_id,omitempty" jsonschema:"The identity of the client that ordered. Send it OR the name the owner reads, never neither of them."`

	// PartyName is the name the owner reads of the party that ordered.
	PartyName string `json:"party_name,omitempty" jsonschema:"The name the owner reads of the client that ordered ('Doña Rosalba'). Send it OR the identity of the party, never neither of them."`

	// DueOn is the day the order is due on.
	DueOn string `json:"due_on,omitempty" jsonschema:"The day the order is due on, written as '2026-03-14'. Send it always: PACA never invents one, and the domain refuses the order when it is left out."`

	// Lines are the lines of the order.
	Lines []recordOrderLineInput `json:"lines,omitempty" jsonschema:"The lines of the order: the product, the quantity of it and the unit the owner said. At least one line is required, and the domain refuses the order that carries none."`
}

// readOrdersInput is the input of the tool that reads the orders of a
// business.
type readOrdersInput struct {
	// BusinessID is the business whose orders are read.
	BusinessID int64 `json:"business_id,omitempty" jsonschema:"The identity of the business whose orders are read. Send it always: PACA never invents one, and the domain refuses the reading when it is left out."`

	// Day is the day of the round that is read.
	Day string `json:"day,omitempty" jsonschema:"The day of the round that is read, written as '2026-03-14'. It is optional: the round of that day answers the orders that are due on it, and the whole book of the orders is read when it is left out."`
}

// orderLineOutput is what the tools of the orders answer about one line of
// an order.
type orderLineOutput struct {
	// ProductID is the product the line is of.
	ProductID int64 `json:"product_id"`

	// ProductName is the name the owner reads of the product.
	ProductName string `json:"product_name"`

	// Quantity is how many units of the product the line carries.
	Quantity int `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`
}

// orderOutput is what the tools of the orders answer about one order.
type orderOutput struct {
	// ID is the identity of the order.
	ID int64 `json:"id"`

	// BusinessID is the business that owns the order.
	BusinessID int64 `json:"business_id"`

	// PartyID is the party that ordered.
	PartyID int64 `json:"party_id"`

	// PartyName is the name the owner reads of the party that ordered.
	PartyName string `json:"party_name"`

	// DueOn is the day the order is due on.
	DueOn string `json:"due_on"`

	// State is the state of the order: the open one, or the delivered
	// one.
	State string `json:"state"`

	// Lines are the lines of the order.
	Lines []orderLineOutput `json:"lines"`
}

// recordOrderOutput is the answer of the tool that records an order.
type recordOrderOutput struct {
	// Order is the order that was written.
	Order orderOutput `json:"order"`
}

// readOrdersOutput is the answer of the tool that reads the orders of a
// business.
type readOrdersOutput struct {
	// Day is the day of the round that was read.
	Day string `json:"day,omitempty"`

	// Orders are the orders the round carries.
	Orders []orderOutput `json:"orders"`

	// Owed are the orders of the round that are still owed: the ones
	// that were not delivered.
	Owed []orderOutput `json:"owed"`
}

// registerOrderTools registers the tools of the orders.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerOrderTools(recorder ports.OrderRecorder, reader ports.OrdersReader) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordOrder,
		Description: descriptionRecordOrder,
	}, s.recordOrder(recorder))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadOrders,
		Description: descriptionReadOrders,
	}, s.readOrders(reader))
}

// recordOrder is the handler of the tool that records an order.
//
// The tool writes the order of the business with its lines and answers with
// it. The order is written open: it was not delivered.
func (s *Server) recordOrder(recorder ports.OrderRecorder) sdk.ToolHandlerFor[recordOrderInput, recordOrderOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordOrderInput) (*sdk.CallToolResult, recordOrderOutput, error) {
		if recorder == nil {
			return nil, recordOrderOutput{}, ports.ErrNoDatabase
		}

		lines := make([]ports.OrderLine, 0, len(input.Lines))
		for _, one := range input.Lines {
			lines = append(lines, ports.OrderLine{
				ProductID:   one.ProductID,
				ProductName: one.ProductName,
				Quantity:    one.Quantity,
				Unit:        one.Unit,
			})
		}

		written, err := recorder.Record(ctx, ports.RecordOrder{
			BusinessID: businessIDOf(input.BusinessID),
			PartyID:    input.PartyID,
			PartyName:  input.PartyName,
			DueOn:      input.DueOn,
			Lines:      lines,
		})
		if err != nil {
			return nil, recordOrderOutput{}, s.toolError(err)
		}

		return nil, recordOrderOutput{Order: outputOfOrder(written)}, nil
	}
}

// readOrders is the handler of the tool that reads the orders of a
// business.
//
// The tool reads the round of the day: it changes nothing, and it answers
// with the orders it carries and with the ones that are still owed.
func (s *Server) readOrders(reader ports.OrdersReader) sdk.ToolHandlerFor[readOrdersInput, readOrdersOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readOrdersInput) (*sdk.CallToolResult, readOrdersOutput, error) {
		if reader == nil {
			return nil, readOrdersOutput{}, ports.ErrNoDatabase
		}

		round, err := reader.Read(ctx, ports.ReadOrders{
			BusinessID: businessIDOf(input.BusinessID),
			Day:        input.Day,
		})
		if err != nil {
			return nil, readOrdersOutput{}, s.toolError(err)
		}

		orders := make([]orderOutput, 0, len(round.Orders))
		for _, one := range round.Orders {
			orders = append(orders, outputOfOrder(one))
		}

		owed := make([]orderOutput, 0, len(round.Orders))
		for _, one := range round.Owed() {
			owed = append(owed, outputOfOrder(one))
		}

		answer := readOrdersOutput{Orders: orders, Owed: owed}
		if !round.Day.Time().IsZero() {
			answer.Day = round.Day.String()
		}

		return nil, answer, nil
	}
}

// outputOfOrder maps the model of the domain onto the answer of the tools
// of the orders.
func outputOfOrder(model order.Order) orderOutput {
	lines := make([]orderLineOutput, 0, len(model.Lines))
	for _, one := range model.Lines {
		lines = append(lines, orderLineOutput{
			ProductID:   int64(one.ProductID),
			ProductName: one.ProductName,
			Quantity:    one.Quantity,
			Unit:        one.Unit,
		})
	}

	return orderOutput{
		ID:         int64(model.ID),
		BusinessID: int64(model.BusinessID),
		PartyID:    int64(model.PartyID),
		PartyName:  model.PartyName,
		DueOn:      model.DueOn.String(),
		State:      model.State.String(),
		Lines:      lines,
	}
}

// theStatesOfTheOrders renders the states of an order for the description
// of the tool.
//
// The states belong to the domain: the description reads them and never
// writes them a second time.
func theStatesOfTheOrders() string {
	states := order.AllStates()

	names := make([]string, 0, len(states))
	for _, one := range states {
		names = append(names, one.String())
	}

	return strings.Join(names, ", ")
}
