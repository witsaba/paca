// Package mcpserver adapts the service to the Model Context Protocol.
//
// The transport is the official Go SDK of the protocol. It registers no tool
// yet: this is the initialization of the server, not its first feature.
package mcpserver

import (
	"log/slog"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
)

// Options configures the MCP transport.
type Options struct {
	// ServiceName is the name the MCP clients see.
	ServiceName string
	// ServiceVersion is the version the MCP clients see.
	ServiceVersion string
	// Instructions tells the MCP clients what the server is for.
	Instructions string
	// Logger is the logger of the transport.
	Logger *slog.Logger
	// MaxRequestBodyBytes is the largest request body the transport accepts.
	MaxRequestBodyBytes int64
	// TrustedOrigins are the origins the transport trusts. The rule of the
	// Origin is the same origin as the Host header when there are none.
	TrustedOrigins []string
	// Users is the use case of the users, the one behind the three tools.
	//
	// It may be nil: a server without a database still exposes the tools,
	// and every call answers that the database is not there.
	Users ports.UsersQuerier

	// Onboarding is the use case that completes the onboarding of a person.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	Onboarding ports.OnboardingCompleter

	// OpenBusiness is the use case that opens a business of a person, with
	// the book of accounts it starts with.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	OpenBusiness ports.BusinessOpener

	// ReadBusinesses is the use case that reads the businesses of a person.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadBusinesses ports.BusinessQuerier

	// Book is the use case of the book of a business: the movements and the
	// balances.
	//
	// It may be nil: a server without a database still exposes the tools,
	// and every call answers that the database is not there.
	Book ports.BookQuerier

	// Debts is the use case of the debts of a business: the debts the book
	// still holds, and the payments that close them.
	//
	// It may be nil: a server without a database still exposes the tools,
	// and every call answers that the database is not there.
	Debts ports.DebtQuerier

	// Summary is the use case of the closing of a month.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	Summary ports.SummaryQuerier

	// DraftPurchase is the use case that keeps the purchase the owner said
	// as a draft of the business.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	DraftPurchase ports.PurchaseDrafter

	// ConfirmPurchase is the use case that writes the purchase of the draft
	// the owner confirmed, and moves the money of the payment.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ConfirmPurchase ports.PurchaseConfirmer

	// MatchItems is the use case that matches a name the owner said
	// against the catalog of the names the business buys.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	MatchItems ports.ItemMatcher

	// ReadIngredients is the use case that reads the catalog of the names
	// the business buys, with what is known of every ingredient and what
	// is still missing of it.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadIngredients ports.IngredientsReader

	// CompleteIngredient is the use case that writes what the owner
	// declared about an ingredient of the catalog.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	CompleteIngredient ports.IngredientCompleter

	// CancelDraft is the use case that drops the draft the owner does not
	// want.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	CancelDraft ports.PurchaseCanceller

	// ReadDrafts is the use case that reads the pending drafts of a
	// business, so that a lost conversation can be recovered.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadDrafts ports.DraftsReader

	// RecordRecipe is the use case that writes the recipe of the bakery
	// that the owner confirmed.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordRecipe ports.RecipeRecorder

	// RecordReference is the use case that writes the reference prices the
	// research of PACA found, with the confirmation of the owner.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordReference ports.ReferenceRecorder

	// ReadReferences is the use case that reads the reference prices of a
	// business, with their source and their day.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadReferences ports.ReferencesReader

	// ReadStock is the use case that reads the stock of a business: what
	// it holds and what it cost.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadStock ports.StockReader

	// ReadBatchCost is the use case that reads the cost of a batch of a
	// recipe: what the recipe costs right now, derived from the layers of
	// the stock.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadBatchCost ports.BatchCostReader

	// ReadCapacity is the use case that reads the capacity of a weekend:
	// what the hours the owner has carry of the recipes of the business.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadCapacity ports.CapacityReader

	// RecordLabor is the use case that writes the labor the owner worked
	// and the value of the hour she declared.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordLabor ports.LaborRecorder

	// RecordOverhead is the use case that writes the overhead of the
	// house: the electricity the owner paid, the hours the oven ran and
	// the allocation she declared.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordOverhead ports.OverheadRecorder

	// ReadRealPrice is the use case that reads the report of the real
	// price of a product: the three layers of the cost of a batch apart
	// and the price of the owner beside them.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadRealPrice ports.RealPriceReader

	// ReadMargin is the use case that reads the margin of a product: what
	// the owner charges above the direct cost of the batch, and what the
	// market said about the same ingredients.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadMargin ports.MarginReader

	// RecordProduction is the use case that writes the production of the
	// bakery: what came out of the oven on one day, with the units that
	// came out of it and the day each one expires.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordProduction ports.ProductionRecorder

	// CorrectShelfLife is the use case that writes the correction of the
	// shelf life the owner declared: the days what a recipe makes keeps.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	CorrectShelfLife ports.ShelfLifeCorrector

	// ReadExpiring is the use case that reads what expires: the units of
	// the bakery and the day each one expires.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadExpiring ports.ExpiringReader

	// RecordWaste is the use case that writes the waste of the bakery:
	// what she lost, the reason, and the money of what it was worth.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordWaste ports.WasteRecorder

	// RecordSale is the use case that writes the sale of the bakery: what
	// she sold, the money the client paid, and the cost it released.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordSale ports.SaleRecorder

	// RecordExit is the use case that writes the exit of a unit of the
	// book: the consumption of the owner or the donation.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordExit ports.ExitRecorder

	// ReadRotation is the use case that reads the rotation of the book:
	// the order in which the units that are still in it leave it.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadRotation ports.RotationReader

	// OpenParty is the use case that opens a party of the business: the
	// person, the store or the farm that buys from it or that it buys
	// from.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	OpenParty ports.PartyOpener

	// ReadParties is the use case that reads the parties of the business.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadParties ports.PartiesReader

	// RecordInvoice is the use case that writes the invoice that came
	// with the purchase: its number, its day, its total and the tax that
	// travels with it.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordInvoice ports.InvoiceRecorder

	// ReadInvoices is the use case that reads the invoices of the
	// business.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadInvoices ports.InvoicesReader

	// SetPrice is the use case that sets the price of a product for a
	// channel: the retail or the wholesale.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	SetPrice ports.PriceRecorder

	// ReadPrices is the use case that reads the price list of the
	// business.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadPrices ports.PricesReader

	// RecordOrder is the use case that writes the orders of the bakery:
	// what the bakery owes to a client, and the day it is due.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordOrder ports.OrderRecorder

	// ReadOrders is the use case that reads the orders of the business
	// and the round of a day: what is still owed.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadOrders ports.OrdersReader

	// RecordRoute is the use case that writes the route of the round of a
	// day: the order the agent will visit the clients in, and what each
	// visit costs.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordRoute ports.RouteRecorder

	// RecordCommitment is the use case that writes the promises of the
	// bakery: what is due and the day it is due, and the day a promise was
	// met.
	//
	// It may be nil: a server without a database still exposes the tools,
	// and every call answers that the database is not there.
	RecordCommitment ports.CommitmentRecorder

	// ReadCommitments is the use case that reads the calendar of the
	// promises of a business: what is due and what is still owed.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadCommitments ports.CommitmentReader

	// RecordLearning is the use case that writes the learning log of the
	// bakery: what happened, in the words of the owner, and what was
	// learned from it.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	RecordLearning ports.LearningRecorder

	// ReadLearning is the use case that reads the learning log of the
	// bakery: the entries in the order they were written, and the ones that
	// are the last word of their chain.
	//
	// It may be nil: a server without a database still exposes the tool,
	// and every call answers that the database is not there.
	ReadLearning ports.LearningReader
}

// Server is the MCP transport of the service.
type Server struct {
	server  *sdk.Server
	handler http.Handler
	logger  *slog.Logger
}

// New builds the MCP transport.
//
// The transport is stateless: the protocol revision the SDK negotiates has no
// protocol-level session, and a stateless server cannot leak one.
//
// The protections of the SDK stay on. The DNS-rebinding protection rejects a
// request that arrives through the loopback interface with a foreign Host
// header, and the body size limit rejects an oversized request. The cross
// origin protection of the standard library rejects a cross-origin request
// from a browser, and lets the requests of a non-browser client through.
//
// The three tools of the users are registered here: the list of the tools is
// a property of the transport, and it does not depend on the database.
func New(opts Options) *Server {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	server := sdk.NewServer(
		&sdk.Implementation{
			Name:    opts.ServiceName,
			Title:   opts.ServiceName,
			Version: opts.ServiceVersion,
		},
		&sdk.ServerOptions{
			Logger:       opts.Logger,
			Instructions: opts.Instructions,
		},
	)

	transport := &Server{server: server, logger: opts.Logger}
	if transport.logger == nil {
		transport.logger = slog.Default()
	}

	transport.registerUserTools(opts.Users)
	transport.registerOnboardingTools(opts.Onboarding)
	transport.registerBusinessTools(opts.OpenBusiness, opts.ReadBusinesses)
	transport.registerBookTools(opts.Book)
	transport.registerDebtTools(opts.Debts, opts.Summary)
	transport.registerPurchaseTools(opts.DraftPurchase, opts.ConfirmPurchase, opts.CancelDraft, opts.ReadDrafts)
	transport.registerIngredientTools(opts.MatchItems, opts.ReadIngredients, opts.CompleteIngredient)
	transport.registerRecipeTools(opts.RecordRecipe)
	transport.registerReferenceTools(opts.RecordReference, opts.ReadReferences)
	transport.registerStockTools(opts.ReadStock)
	transport.registerBatchCostTools(opts.ReadBatchCost)
	transport.registerCapacityTools(opts.ReadCapacity)
	transport.registerLaborTools(opts.RecordLabor)
	transport.registerOverheadTools(opts.RecordOverhead)
	transport.registerRealPriceTools(opts.ReadRealPrice)
	transport.registerMarginTools(opts.ReadMargin)
	transport.registerProductionTools(opts.RecordProduction)
	transport.registerExpiryTools(opts.CorrectShelfLife, opts.ReadExpiring)
	transport.registerWasteTools(opts.RecordWaste)
	transport.registerSaleTools(opts.RecordSale)
	transport.registerExitTools(opts.RecordExit)
	transport.registerRotationTools(opts.ReadRotation)
	transport.registerPartyTools(opts.OpenParty, opts.ReadParties)
	transport.registerInvoiceTools(opts.RecordInvoice, opts.ReadInvoices)
	transport.registerPriceTools(opts.SetPrice, opts.ReadPrices)
	transport.registerOrderTools(opts.RecordOrder, opts.ReadOrders)
	transport.registerRouteTools(opts.RecordRoute)
	transport.registerCommitmentTools(opts.RecordCommitment, opts.ReadCommitments)
	transport.registerLearningTools(opts.RecordLearning, opts.ReadLearning)

	handler := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server { return server },
		&sdk.StreamableHTTPOptions{
			Logger:                       opts.Logger,
			Stateless:                    true,
			MaxRequestBodyBytes:          opts.MaxRequestBodyBytes,
			PropagateRequestCancellation: true,
		},
	)

	// The rule of the Origin is stated here and not left to the default:
	// the protection of the standard library trusts the same origin as the
	// Host header of the request, and the origins the service trusts are
	// the ones the operator named.
	//
	// The origins are validated by the configuration of the service: an
	// origin that the protection of the standard library cannot read is
	// refused there, and the transport says so in its log instead of
	// trusting it.
	protection := http.NewCrossOriginProtection()
	for _, origin := range opts.TrustedOrigins {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			logger.Error("the origin is not one the transport can trust",
				slog.String("origin", origin),
				slog.Any("error", err),
			)
		}
	}

	transport.handler = protection.Handler(handler)

	return transport
}

// Handler returns the HTTP handler of the transport.
func (s *Server) Handler() http.Handler {
	return s.handler
}
