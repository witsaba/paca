package mcpserver

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/order"
)

// The names of the tools of the route.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that records a
// route is not a tool that creates a row.
const (
	// ToolRecordRoute writes the route of the round of a day: the order
	// the agent will visit the clients in, and what each visit costs.
	ToolRecordRoute = "record_route"
)

// The descriptions of the tools of the route.
var descriptionRecordRoute = buildDescriptionRecordRoute()

// buildDescriptionRecordRoute renders the description of the tool that
// records the route of a round.
//
// The description says what the tool does, what it needs, and what it
// answers: it is the contract the agent reads before it calls the tool.
func buildDescriptionRecordRoute() string {
	return "Record the route of the round of a day: the order you will visit the clients in, and what each visit costs.\n\n" +
		"You are the one that knows the streets, and PACA never computes the route: you write back the order you " +
		"will visit the clients in, and the money of each visit. The route is written back INTO the round of that day, " +
		"and writing it DELIVERS the orders of its stops: the orders you visit are delivered with the route.\n\n" +
		"The business is REQUIRED: it is the business of its own — the one that sells the bread. The Personal " +
		"Business of a person holds the money of that person, and not the rounds of the bakery, and PACA refuses " +
		"the route of it.\n\n" +
		"The day ('day') is REQUIRED: it is the day of the round the route is of, written as '2026-03-14'. PACA " +
		"never invents one.\n\n" +
		"The stops ('stops') are REQUIRED: at least one stop, and the stops are in the ORDER you will visit the " +
		"clients in. Every stop carries:\n\n" +
		"  - the order it visits ('order_id'): the identity of an order the business still OWES, which is what you " +
		"read the whole: EVERY open order of the business, the ones that are due on the day and the ones that fell " +
		"due on an earlier day and were not delivered. PACA refuses a stop that names an order of another " +
		"business, and a stop that names an order that was delivered already;\n" +
		"  - what the visit costs ('cost'): the money of the transport to that client, in WHOLE PESOS. It is " +
		"REQUIRED — send it in every stop: it is never negative, and it may be zero, because a stop the owner " +
		"walks to costs nothing.\n\n" +
		"Every order is visited ONCE: a route that names the same order in two stops is refused, because the total " +
		"of the route counts the money of a stop once.\n\n" +
		"The route of a business and a day is written ONCE: a second route for the same day is refused, because " +
		"the round was already planned.\n\n" +
		"The answer carries the route that was written: its identity ('id'), the business that owns it " +
		"('business_id'), the day it is of ('day'), its stops ('stops') with the order of every one of them " +
		"('order_id') and what it costs ('cost'), and the TOTAL of the route ('total'): the money of every stop, " +
		"which PACA derives from the stops and which you never write."
}

// recordRouteStopInput is one stop of the route the agent read.
type recordRouteStopInput struct {
	// OrderID is the order the stop visits.
	OrderID int64 `json:"order_id,omitempty" jsonschema:"The identity of the order the stop visits: an order the business still owes — EVERY open order of the business, the one that is due on the day of the round or the one that fell due on an earlier day. Send it always: PACA never guesses the order of a stop, and the domain refuses the stop that names an order of another business or an order that was delivered already."`

	// Cost is what the stop costs.
	//
	// The money of the stop is required, as every other money of the
	// transport is: the schema refuses the stop that carries no cost, and
	// the domain is the one that decides whether the cost may enter the
	// book — it is never negative, and zero is the stop the owner walks
	// to.
	Cost int64 `json:"cost" jsonschema:"What the stop costs: the money of the transport to that client, in whole pesos. It is required and you send it in every stop: it is never negative, and it may be zero when the owner walks to that client."`
}

// recordRouteInput is the input of the tool that records the route.
type recordRouteInput struct {
	// BusinessID is the business the route belongs to.
	BusinessID int64 `json:"business_id,omitempty" jsonschema:"The identity of the business the route belongs to: the business of its own, the one that sells the bread. Send it always: PACA never invents one, and the domain refuses the route when it is left out."`

	// Day is the day of the round the route is of.
	Day string `json:"day,omitempty" jsonschema:"The day of the round the route is of, written as '2026-03-14'. Send it always: the route is of a day, and PACA never invents one."`

	// Stops are the stops of the route.
	Stops []recordRouteStopInput `json:"stops,omitempty" jsonschema:"The stops of the route, in the order you will visit the clients in. At least one stop is required, and the domain refuses the route that carries none."`
}

// routeStopOutput is what the tools of the route answer about one stop.
type routeStopOutput struct {
	// OrderID is the order the stop visits.
	OrderID int64 `json:"order_id"`

	// Cost is what the stop costs.
	Cost int64 `json:"cost"`
}

// routeOutput is what the tools of the route answer about a route.
type routeOutput struct {
	// ID is the identity of the route.
	ID int64 `json:"id"`

	// BusinessID is the business that owns the route.
	BusinessID int64 `json:"business_id"`

	// Day is the day the route is of.
	Day string `json:"day"`

	// Total is what the route costs: the money of every stop.
	//
	// The total is derived by PACA from the stops, and the agent never
	// writes it.
	Total int64 `json:"total"`

	// Stops are the stops of the route, in the order the agent will visit
	// the clients in.
	Stops []routeStopOutput `json:"stops"`
}

// recordRouteOutput is the answer of the tool that records the route.
type recordRouteOutput struct {
	// Route is the route that was written.
	Route routeOutput `json:"route"`
}

// registerRouteTools registers the tools of the route.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerRouteTools(recorder ports.RouteRecorder) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolRecordRoute,
		Description: descriptionRecordRoute,
	}, s.recordRoute(recorder))
}

// recordRoute is the handler of the tool that records the route of a round.
//
// The tool writes the route of the round of the day, and the route delivers
// the orders of its stops.
func (s *Server) recordRoute(recorder ports.RouteRecorder) sdk.ToolHandlerFor[recordRouteInput, recordRouteOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input recordRouteInput) (*sdk.CallToolResult, recordRouteOutput, error) {
		if recorder == nil {
			return nil, recordRouteOutput{}, ports.ErrNoDatabase
		}

		stops := make([]ports.RouteStop, 0, len(input.Stops))
		for _, one := range input.Stops {
			stops = append(stops, ports.RouteStop{
				OrderID: one.OrderID,
				Cost:    one.Cost,
			})
		}

		written, err := recorder.Record(ctx, ports.RecordRoute{
			BusinessID: businessIDOf(input.BusinessID),
			Day:        input.Day,
			Stops:      stops,
		})
		if err != nil {
			return nil, recordRouteOutput{}, s.toolError(err)
		}

		return nil, recordRouteOutput{Route: outputOfRoute(written)}, nil
	}
}

// outputOfRoute maps the model of the domain onto the answer of the tools
// of the route.
func outputOfRoute(model order.Route) routeOutput {
	stops := make([]routeStopOutput, 0, len(model.Stops))

	for _, one := range model.Stops {
		stops = append(stops, routeStopOutput{
			OrderID: int64(one.OrderID),
			Cost:    int64(one.Cost),
		})
	}

	return routeOutput{
		ID:         int64(model.ID),
		BusinessID: int64(model.BusinessID),
		Day:        model.Day.String(),
		Total:      int64(model.Total()),
		Stops:      stops,
	}
}
