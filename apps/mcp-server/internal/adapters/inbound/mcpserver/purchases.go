package mcpserver

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The names of the tools of the utterance.
//
// The names are the ones the agent reads in the list of the tools, so they
// say what the tool does and not what it writes: a tool that proposes a
// purchase is not a tool that buys it.
const (
	// ToolDraftPurchase proposes the purchase the owner said, and keeps it
	// as a draft of the business.
	ToolDraftPurchase = "draft_purchase"

	// ToolConfirmPurchase writes the purchase of the draft the owner
	// confirmed, and moves the money of the payment.
	ToolConfirmPurchase = "confirm_purchase"

	// ToolCancelDraft drops the draft the owner does not want.
	ToolCancelDraft = "cancel_draft"

	// ToolReadDrafts reads the pending drafts of a business, so that a
	// lost conversation can be recovered.
	ToolReadDrafts = "read_drafts"
)

// The description of the tool of the utterance.
//
// The lifetime of the draft and the way the money is read come from the
// domain: the description reads them and never writes them a second time.
var (
	descriptionDraftPurchase   = buildDescriptionDraftPurchase()
	descriptionConfirmPurchase = buildDescriptionConfirmPurchase()
	descriptionCancelDraft     = buildDescriptionCancelDraft()
	descriptionReadDrafts      = buildDescriptionReadDrafts()
)

// buildDescriptionDraftPurchase renders the description of the tool that
// drafts a purchase.
//
// The lifetime of a draft belongs to the domain, and the description reads
// it: the number is a rule, not a promise of the transport.
func buildDescriptionDraftPurchase() string {
	return "Propose the purchase the owner just said, and keep it as a DRAFT of the business.\n\n" +
		"The business is REQUIRED: it is the business the purchase belongs to, and PACA never invents one.\n\n" +
		"The sentence of the owner is REQUIRED: it is her own words, verbatim, and it is kept with the draft so " +
		"the entry can be audited and derived again. Send the sentence as she said it, not a summary of it.\n\n" +
		"Every line carries the name as the owner said it, the quantity, the unit and the money. At least one line " +
		"is REQUIRED:\n\n" +
		"  - the name is the name she said ('Harina', 'harina panadera'). A name PACA does not know is not an " +
		"error: it is recorded as she said it.\n" +
		"  - the quantity is a number, and it is positive.\n" +
		"  - the unit is REQUIRED: the quantity and its unit travel together ('kg', 'g', 'paquete', 'unidad'). " +
		"PACA refuses a line without a unit.\n" +
		"  - the money is the money that was paid ('total_paid') or the price of one unit ('unit_price'), in " +
		"the currency of the book ('" + business.DefaultCurrency + "'). Never send both unless they agree: " +
		"PACA refuses the pair that does not.\n" +
		"  - the identity of the ingredient of the catalog ('ingredient_id') is OPTIONAL: send the identity the " +
		"tool " + ToolMatchItems + " answered for the name, and leave it out when the catalog does not know " +
		"the name.\n\n" +
		"The money is never inferred: leave it out when the owner did not say it, and the answer will say that " +
		"the draft still needs it. PACA refuses a price it filled by itself, and it never writes a price the " +
		"owner did not say.\n\n" +
		"The day is OPTIONAL: it is the day of the purchase, written the way a person writes it ('2026-03-01'), " +
		"and it means the day of today when it is empty. The market is OPTIONAL and the invoice is OPTIONAL: " +
		"they stay absent when the owner named none.\n\n" +
		"The ambiguities are OPTIONAL: every name PACA was not sure about carries the options she can choose " +
		"between, so the question has an answer that is one word.\n\n" +
		"The draft is a proposal, not a fact: it waits for the yes of the owner, and it expires in " +
		fmt.Sprint(purchase.DraftLifetimeDays) +
		" days if she says nothing. Name the draft to correct it: the amendment grows its revision.\n\n" +
		"The answer carries what PACA understood, what the draft still needs, what PACA was not sure about and " +
		"what PACA filled by itself.\n\n" +
		"This tool changes nothing in the book: it writes a draft and nothing else."
}

// buildDescriptionConfirmPurchase renders the description of the tool that
// confirms a draft.
//
// The lifetime of a draft belongs to the domain, and the description reads
// it: the number is a rule, not a promise of the transport.
func buildDescriptionConfirmPurchase() string {
	return "Confirm the draft the owner just saw, and write the purchase in the book.\n\n" +
		"The business is REQUIRED: it is the business the purchase belongs to.\n\n" +
		"The draft is REQUIRED: it is the draft PACA showed her.\n\n" +
		"The revision is REQUIRED: it is the revision she saw. A draft that changed under her eyes is refused: " +
		"show it again and ask for the yes once more.\n\n" +
		"The word is REQUIRED: it is the word she used to confirm it ('sí', 'confirmado'), and it is kept with " +
		"the record.\n\n" +
		"The account that pays is OPTIONAL: leave it out and the money comes from the cash of the business, and " +
		"name it when she paid with another account.\n\n" +
		"The confirmation writes the purchase and its lines, and it moves the money of the payment from the account " +
		"that pays to the inventory of the business: the three of them are written together or not at all.\n\n" +
		"The confirmation grows the catalog of the business too: a name the catalog does not know becomes an " +
		"ingredient of the business, recorded as the owner said it. A name the catalog already knows keeps the " +
		"ingredient it has. The answer names the names the catalog did not know, so the agent can tell the owner " +
		"that a name is new.\n\n" +
		"A draft that expired is refused: the price of a market does not wait, and a draft expires in " +
		fmt.Sprint(purchase.DraftLifetimeDays) +
		" days if she says nothing. A draft that still needs the money of one of its lines is refused: the money " +
		"is never inferred.\n\n" +
		"The answer carries the purchase that was written and the three parts of the answer of PACA: what it " +
		"recorded, what it did not understand and what it still needs."
}

// buildDescriptionCancelDraft renders the description of the tool that
// cancels a draft.
//
// The states of a draft belong to the domain, and the description reads
// them: the vocabulary is a rule, not a promise of the transport.
func buildDescriptionCancelDraft() string {
	return "Drop the draft the owner does not want, and answer with the draft you cancelled.\n\n" +
		"The business is REQUIRED: it is the business the draft belongs to.\n\n" +
		"The draft is REQUIRED: it is the draft PACA showed her.\n\n" +
		"The draft stops waiting: it moves to the state 'cancelled' and it never waits again. A draft that is " +
		"confirmed already and a draft that is cancelled already are refused: they do not wait any more.\n\n" +
		"A draft that expired is refused too: the price of a market does not wait, and the owner drafts the " +
		"purchase again.\n\n" +
		"This tool changes nothing in the book: it drops the draft and nothing else. The money of the book does " +
		"not move, and no purchase is written."
}

// buildDescriptionReadDrafts renders the description of the tool that
// reads the pending drafts.
//
// The states of a draft and its lifetime belong to the domain, and the
// description reads them: the vocabulary is a rule, not a promise of the
// transport.
func buildDescriptionReadDrafts() string {
	return "Read the pending drafts of a business, so that a lost conversation can be recovered.\n\n" +
		"The business is REQUIRED: it is the business whose drafts are read.\n\n" +
		"The day is OPTIONAL: it is the day of the reading, written the way a person writes it ('2026-03-01'), " +
		"and it means the day of today when it is empty. The day decides which drafts are still pending.\n\n" +
		"The answer carries every pending draft with what it still needs, what it was not sure about and what " +
		"PACA filled by itself. Show the drafts to the owner and ask for the yes of the one she means.\n\n" +
		"The states of a draft are " + namesOfTheStates() + ": it waits in '" + purchase.StateProposed.String() +
		"' and it ends in one of the other three. A draft that nobody answered expires in " +
		fmt.Sprint(purchase.DraftLifetimeDays) +
		" days, and an expired draft is not pending any more.\n\n" +
		"This tool changes nothing in the book: it reads the drafts and nothing else."
}

// namesOfTheStates renders the states of a draft in a sentence the agent
// reads.
//
// The vocabulary belongs to the domain: the description reads it and never
// writes it a second time.
func namesOfTheStates() string {
	names := make([]string, 0, len(purchase.States()))

	for _, one := range purchase.States() {
		names = append(names, "'"+one.String()+"'")
	}

	return strings.Join(names, ", ")
}

// draftPurchaseLineInput is one line of the candidate the agent read.
type draftPurchaseLineInput struct {
	// Name is the name as the owner said it.
	Name string `json:"name" jsonschema:"The name of the thing as the owner said it ('Harina'). It is required."`

	// Quantity is how much of it the owner bought.
	Quantity float64 `json:"quantity" jsonschema:"How much of it the owner bought. It is required and it is positive."`

	// Unit is the unit of that quantity.
	Unit string `json:"unit" jsonschema:"The unit of the quantity ('kg', 'g', 'paquete', 'unidad'). It is required: the quantity and its unit travel together."`

	// TotalPaid is the money that was paid for the whole line.
	TotalPaid int64 `json:"total_paid,omitempty" jsonschema:"The money that was paid for the whole line, in pesos. It is optional: leave it out when the owner did not say it."`

	// UnitPrice is the price of one unit of the line.
	UnitPrice int64 `json:"unit_price,omitempty" jsonschema:"The price of one unit of the line, in pesos. It is optional: leave it out when the owner did not say it."`

	// Inferred says that the price was filled by the agent.
	Inferred bool `json:"price_inferred,omitempty" jsonschema:"True when the price was filled by PACA and not said by the owner. The money is never inferred, and PACA refuses a price it filled."`

	// IngredientID is the identity of the ingredient of the catalog the
	// name belongs to.
	IngredientID int64 `json:"ingredient_id,omitempty" jsonschema:"The identity of the ingredient of the catalog the name belongs to. It is optional: send the identity the tool match_items answered, and leave it out when the catalog does not know the name."`
}

// draftPurchaseAmbiguityInput is one name the agent was not sure about.
type draftPurchaseAmbiguityInput struct {
	// Name is the name as the owner said it.
	Name string `json:"name" jsonschema:"The name as the owner said it ('harina'). It is required."`

	// Options are the candidates the agent found.
	Options []string `json:"options" jsonschema:"The candidates PACA found, so the owner chooses with one word. It is required, and it is never empty."`
}

// draftPurchaseInput is the input of the tool that drafts a purchase.
type draftPurchaseInput struct {
	// BusinessID is the business the purchase belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the purchase belongs to. It is required."`

	// DraftID is the draft the candidate corrects.
	DraftID int64 `json:"draft_id,omitempty" jsonschema:"The identity of the draft this candidate corrects. It is optional: leave it out when the candidate is a new one."`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance" jsonschema:"The sentence of the owner, verbatim. It is required, and it is kept with the draft."`

	// Lines are the lines the agent read.
	Lines []draftPurchaseLineInput `json:"lines" jsonschema:"The lines of the purchase, as PACA read them. At least one line is required."`

	// Day is the day of the purchase.
	Day string `json:"day,omitempty" jsonschema:"The day of the purchase, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`

	// Market is the market the purchase happened in.
	Market string `json:"market,omitempty" jsonschema:"The market the purchase happened in ('Plaza de Mercado'). It is optional."`

	// Invoice is the invoice of the purchase.
	Invoice string `json:"invoice,omitempty" jsonschema:"The invoice of the purchase ('FV-001'). It is optional."`

	// Ambiguities are the names the agent was not sure about.
	Ambiguities []draftPurchaseAmbiguityInput `json:"ambiguities,omitempty" jsonschema:"The names PACA was not sure about, each one with the options the owner chooses between. It is optional."`
}

// draftLineOutput is what the answer carries about one line of the draft.
type draftLineOutput struct {
	// Name is the name as the owner said it.
	Name string `json:"name"`

	// Quantity is how much of it the owner bought.
	Quantity float64 `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`

	// TotalPaid is the money that was paid for the whole line.
	TotalPaid int64 `json:"total_paid,omitempty"`

	// UnitPrice is the price of one unit of the line.
	UnitPrice int64 `json:"unit_price,omitempty"`

	// IngredientID is the identity of the ingredient of the catalog the
	// name belongs to.
	IngredientID int64 `json:"ingredient_id,omitempty"`
}

// draftAmbiguityOutput is what the answer carries about one name the agent
// was not sure about.
type draftAmbiguityOutput struct {
	// Name is the name as the owner said it.
	Name string `json:"name"`

	// Options are the candidates the agent found.
	Options []string `json:"options"`
}

// draftOutput is the draft the answer carries.
type draftOutput struct {
	// ID is the identity of the draft.
	ID int64 `json:"id"`

	// BusinessID is the business the draft belongs to.
	BusinessID int64 `json:"business_id"`

	// State is the state of the draft.
	State string `json:"state"`

	// Revision is the number of the revision of the draft.
	Revision int `json:"revision"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance"`

	// Day is the day of the purchase.
	Day string `json:"day"`

	// Market is the market the purchase happened in.
	Market string `json:"market,omitempty"`

	// Invoice is the invoice of the purchase.
	Invoice string `json:"invoice,omitempty"`

	// ProposedOn is the day the draft was proposed.
	ProposedOn string `json:"proposed_on"`

	// ExpiresOn is the day the draft expires.
	ExpiresOn string `json:"expires_on"`
}

// draftPurchaseOutput is the answer of the tool that drafts a purchase.
//
// The answer carries the three parts of the answer of Paca: what it
// understood, what it did not understand and what it still needs.
type draftPurchaseOutput struct {
	// Draft is the draft the business keeps.
	Draft draftOutput `json:"draft"`

	// Understood is what PACA understood: the lines of the purchase.
	Understood []draftLineOutput `json:"understood"`

	// Missing is what the draft still needs.
	Missing []string `json:"missing"`

	// Ambiguous is what PACA was not sure about, with the options it
	// offers.
	Ambiguous []draftAmbiguityOutput `json:"ambiguous"`

	// Inferred is what PACA filled by itself.
	Inferred []string `json:"inferred"`
}

// confirmPurchaseInput is the input of the tool that confirms a draft.
type confirmPurchaseInput struct {
	// BusinessID is the business the purchase belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the purchase belongs to. It is required."`

	// DraftID is the draft the owner confirmed.
	DraftID int64 `json:"draft_id" jsonschema:"The identity of the draft the owner confirmed. It is required."`

	// Revision is the revision of the draft the owner saw.
	Revision int `json:"revision" jsonschema:"The revision of the draft the owner saw. It is required: a draft that changed under her eyes is refused."`

	// ConfirmedWith is the word the owner used to confirm the draft.
	ConfirmedWith string `json:"confirmed_with" jsonschema:"The word the owner used to confirm it ('sí', 'confirmado'). It is required, and it is kept with the record."`

	// FromAccountID is the account the money of the payment comes from.
	FromAccountID int64 `json:"from_account_id,omitempty" jsonschema:"The identity of the account the money of the payment comes from. It is optional: leave it out and the money comes from the cash of the business."`
}

// purchaseLineOutput is what the answer carries about one line of the
// purchase.
type purchaseLineOutput struct {
	// Name is the name as the owner said it.
	Name string `json:"name"`

	// Quantity is how much of it the owner bought.
	Quantity float64 `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`

	// TotalPaid is the money that was paid for the whole line.
	TotalPaid int64 `json:"total_paid"`

	// UnitPrice is the price of one unit of the line.
	UnitPrice int64 `json:"unit_price,omitempty"`

	// IngredientID is the identity of the ingredient of the catalog the
	// name belongs to.
	IngredientID int64 `json:"ingredient_id,omitempty"`
}

// purchaseOutput is the purchase the answer carries.
type purchaseOutput struct {
	// ID is the identity of the purchase.
	ID int64 `json:"id"`

	// DraftID is the draft the purchase came from.
	DraftID int64 `json:"draft_id"`

	// BusinessID is the business that bought.
	BusinessID int64 `json:"business_id"`

	// Day is the day of the purchase.
	Day string `json:"day"`

	// Market is the market the purchase happened in.
	Market string `json:"market,omitempty"`

	// Invoice is the invoice of the purchase.
	Invoice string `json:"invoice,omitempty"`

	// Utterance is the sentence of the owner, verbatim.
	Utterance string `json:"utterance"`

	// ConfirmedWith is the word the owner used to confirm it.
	ConfirmedWith string `json:"confirmed_with"`

	// Total is the money of the whole purchase.
	Total int64 `json:"total"`

	// Lines are the lines of the purchase.
	Lines []purchaseLineOutput `json:"lines"`
}

// confirmPurchaseOutput is the answer of the tool that confirms a draft.
//
// The answer carries the three parts of the answer of Paca: what it
// recorded, what it did not understand and what it still needs.
type confirmPurchaseOutput struct {
	// Purchase is the purchase that was written.
	Purchase purchaseOutput `json:"purchase"`

	// Recorded is what the confirmation recorded.
	Recorded []purchaseLineOutput `json:"recorded"`

	// NotUnderstood is what PACA did not understand, with the options it
	// offered.
	NotUnderstood []draftAmbiguityOutput `json:"not_understood"`

	// StillNeeds is what the purchase still needs.
	StillNeeds []string `json:"still_needs"`

	// NewNames are the names the catalog of the business did not know: the
	// names that became ingredients of the business with this purchase.
	NewNames []string `json:"new_names"`
}

// cancelDraftInput is the input of the tool that cancels a draft.
type cancelDraftInput struct {
	// BusinessID is the business the draft belongs to.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business the draft belongs to. It is required."`

	// DraftID is the draft the owner does not want.
	DraftID int64 `json:"draft_id" jsonschema:"The identity of the draft the owner does not want. It is required."`
}

// cancelDraftOutput is the answer of the tool that cancels a draft.
//
// The answer carries the draft that was cancelled and the three parts of
// the answer of Paca: what it understood, what it was not sure about and
// what it filled by itself.
type cancelDraftOutput struct {
	// Draft is the draft that was cancelled.
	Draft draftOutput `json:"draft"`

	// Understood is what PACA understood: the lines of the draft.
	Understood []draftLineOutput `json:"understood"`

	// Missing is what the draft still needed.
	Missing []string `json:"missing"`

	// Ambiguous is what PACA was not sure about, with the options it
	// offered.
	Ambiguous []draftAmbiguityOutput `json:"ambiguous"`

	// Inferred is what PACA filled by itself.
	Inferred []string `json:"inferred"`
}

// readDraftsInput is the input of the tool that reads the pending drafts.
type readDraftsInput struct {
	// BusinessID is the business whose drafts are read.
	BusinessID int64 `json:"business_id" jsonschema:"The identity of the business whose pending drafts are read. It is required."`

	// Day is the day of the reading.
	Day string `json:"day,omitempty" jsonschema:"The day of the reading, written as '2026-03-01'. It is optional: it means the day of today when it is empty."`
}

// pendingDraftOutput is what the answer carries about one pending draft.
type pendingDraftOutput struct {
	// Draft is the draft that waits.
	Draft draftOutput `json:"draft"`

	// Understood is what PACA understood: the lines of the draft.
	Understood []draftLineOutput `json:"understood"`

	// Missing is what the draft still needs.
	Missing []string `json:"missing"`

	// Ambiguous is what PACA was not sure about, with the options it
	// offers.
	Ambiguous []draftAmbiguityOutput `json:"ambiguous"`

	// Inferred is what PACA filled by itself.
	Inferred []string `json:"inferred"`
}

// readDraftsOutput is the answer of the tool that reads the pending
// drafts.
type readDraftsOutput struct {
	// Drafts are the pending drafts of the business.
	Drafts []pendingDraftOutput `json:"drafts"`
}

// registerPurchaseTools registers the tools of the utterance.
//
// The tools are registered even when the service has no database: the list
// of the tools is a property of the server, and a call without a database
// answers that the database is not there.
func (s *Server) registerPurchaseTools(
	drafter ports.PurchaseDrafter,
	confirmer ports.PurchaseConfirmer,
	canceller ports.PurchaseCanceller,
	reader ports.DraftsReader,
) {
	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolDraftPurchase,
		Description: descriptionDraftPurchase,
	}, s.draftPurchase(drafter))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolConfirmPurchase,
		Description: descriptionConfirmPurchase,
	}, s.confirmPurchase(confirmer))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolCancelDraft,
		Description: descriptionCancelDraft,
	}, s.cancelDraft(canceller))

	sdk.AddTool(s.server, &sdk.Tool{
		Name:        ToolReadDrafts,
		Description: descriptionReadDrafts,
	}, s.readDrafts(reader))
}

// draftPurchase is the handler of the tool that drafts a purchase.
//
// The tool keeps a draft and answers with it: it never writes the book.
func (s *Server) draftPurchase(drafter ports.PurchaseDrafter) sdk.ToolHandlerFor[draftPurchaseInput, draftPurchaseOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input draftPurchaseInput) (*sdk.CallToolResult, draftPurchaseOutput, error) {
		if drafter == nil {
			return nil, draftPurchaseOutput{}, ports.ErrNoDatabase
		}

		drafted, err := drafter.Draft(ctx, requestOf(input))
		if err != nil {
			return nil, draftPurchaseOutput{}, s.toolError(err)
		}

		return nil, outputOfDraft(drafted), nil
	}
}

// confirmPurchase is the handler of the tool that confirms a draft.
//
// The tool writes the purchase and the money of the payment, and answers
// with the purchase and the three parts of the answer of Paca.
func (s *Server) confirmPurchase(confirmer ports.PurchaseConfirmer) sdk.ToolHandlerFor[confirmPurchaseInput, confirmPurchaseOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input confirmPurchaseInput) (*sdk.CallToolResult, confirmPurchaseOutput, error) {
		if confirmer == nil {
			return nil, confirmPurchaseOutput{}, ports.ErrNoDatabase
		}

		confirmed, err := confirmer.Confirm(ctx, ports.ConfirmPurchase{
			BusinessID:    business.ID(input.BusinessID),
			DraftID:       purchase.DraftID(input.DraftID),
			Revision:      input.Revision,
			ConfirmedWith: input.ConfirmedWith,
			FromAccountID: account.ID(input.FromAccountID),
		})
		if err != nil {
			return nil, confirmPurchaseOutput{}, s.toolError(err)
		}

		return nil, outputOfConfirmation(confirmed), nil
	}
}

// cancelDraft is the handler of the tool that cancels a draft.
//
// The tool drops the draft the owner does not want and answers with it: it
// writes nothing in the book.
func (s *Server) cancelDraft(canceller ports.PurchaseCanceller) sdk.ToolHandlerFor[cancelDraftInput, cancelDraftOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input cancelDraftInput) (*sdk.CallToolResult, cancelDraftOutput, error) {
		if canceller == nil {
			return nil, cancelDraftOutput{}, ports.ErrNoDatabase
		}

		cancelled, err := canceller.Cancel(ctx, ports.CancelDraft{
			BusinessID: business.ID(input.BusinessID),
			DraftID:    purchase.DraftID(input.DraftID),
		})
		if err != nil {
			return nil, cancelDraftOutput{}, s.toolError(err)
		}

		return nil, outputOfCancellation(cancelled), nil
	}
}

// readDrafts is the handler of the tool that reads the pending drafts.
//
// The tool reads the drafts and answers with them: it writes nothing.
func (s *Server) readDrafts(reader ports.DraftsReader) sdk.ToolHandlerFor[readDraftsInput, readDraftsOutput] {
	return func(ctx context.Context, _ *sdk.CallToolRequest, input readDraftsInput) (*sdk.CallToolResult, readDraftsOutput, error) {
		if reader == nil {
			return nil, readDraftsOutput{}, ports.ErrNoDatabase
		}

		read, err := reader.Read(ctx, ports.ReadDrafts{
			BusinessID: business.ID(input.BusinessID),
			Day:        input.Day,
		})
		if err != nil {
			return nil, readDraftsOutput{}, s.toolError(err)
		}

		return nil, outputOfPendingDrafts(read), nil
	}
}

// requestOf turns what the agent sent into the request of the use case.
//
// The port speaks in the words of the agent and in numbers: the use case
// turns them into the model of the domain, and the domain decides what may
// enter the book.
func requestOf(input draftPurchaseInput) ports.DraftPurchase {
	lines := make([]ports.PurchaseLine, 0, len(input.Lines))
	for _, one := range input.Lines {
		lines = append(lines, ports.PurchaseLine{
			Name:         one.Name,
			Quantity:     one.Quantity,
			Unit:         one.Unit,
			Total:        one.TotalPaid,
			UnitPrice:    one.UnitPrice,
			Inferred:     one.Inferred,
			IngredientID: one.IngredientID,
		})
	}

	ambiguities := make([]ports.PurchaseAmbiguity, 0, len(input.Ambiguities))
	for _, one := range input.Ambiguities {
		ambiguities = append(ambiguities, ports.PurchaseAmbiguity{
			Name:    one.Name,
			Options: one.Options,
		})
	}

	return ports.DraftPurchase{
		BusinessID:  business.ID(input.BusinessID),
		DraftID:     purchase.DraftID(input.DraftID),
		Utterance:   input.Utterance,
		Lines:       lines,
		Day:         input.Day,
		Market:      input.Market,
		Invoice:     input.Invoice,
		Ambiguities: ambiguities,
	}
}

// outputOfDraft maps the model of the domain onto the answer of the tool.
//
// The answer carries the three parts of the answer of Paca: what it
// understood, what it did not understand and what it still needs.
func outputOfDraft(draft purchase.Draft) draftPurchaseOutput {
	return draftPurchaseOutput{
		Draft:      draftOutputOf(draft),
		Understood: understoodOf(draft),
		Missing:    emptyIfNil(draft.Missing),
		Ambiguous:  ambiguousOf(draft),
		Inferred:   emptyIfNil(draft.Inferred),
	}
}

// outputOfCancellation maps the draft that was cancelled onto the answer of
// the tool.
//
// The answer carries the draft and the three parts of the answer of Paca:
// what it understood, what it was not sure about and what it filled.
func outputOfCancellation(cancelled purchase.Draft) cancelDraftOutput {
	return cancelDraftOutput{
		Draft:      draftOutputOf(cancelled),
		Understood: understoodOf(cancelled),
		Missing:    emptyIfNil(cancelled.Missing),
		Ambiguous:  ambiguousOf(cancelled),
		Inferred:   emptyIfNil(cancelled.Inferred),
	}
}

// outputOfPendingDrafts maps the pending drafts onto the answer of the
// tool.
//
// Every draft carries the three parts of the answer of Paca: what it
// understood, what it still needs and what it was not sure about.
func outputOfPendingDrafts(drafts []purchase.Draft) readDraftsOutput {
	pending := make([]pendingDraftOutput, 0, len(drafts))

	for _, one := range drafts {
		pending = append(pending, pendingDraftOutput{
			Draft:      draftOutputOf(one),
			Understood: understoodOf(one),
			Missing:    emptyIfNil(one.Missing),
			Ambiguous:  ambiguousOf(one),
			Inferred:   emptyIfNil(one.Inferred),
		})
	}

	return readDraftsOutput{Drafts: pending}
}

// draftOutputOf maps the draft of the domain onto the answer of the tool.
func draftOutputOf(draft purchase.Draft) draftOutput {
	return draftOutput{
		ID:         int64(draft.ID),
		BusinessID: int64(draft.BusinessID),
		State:      draft.State.String(),
		Revision:   draft.Revision,
		Utterance:  draft.Utterance,
		Day:        draft.Day.String(),
		Market:     draft.Market,
		Invoice:    draft.Invoice,
		ProposedOn: draft.ProposedOn.String(),
		ExpiresOn:  draft.ExpiresOn.String(),
	}
}

// understoodOf maps the lines the agent read onto the answer of the tool.
func understoodOf(draft purchase.Draft) []draftLineOutput {
	understood := make([]draftLineOutput, 0, len(draft.Lines))

	for _, one := range draft.Lines {
		understood = append(understood, draftLineOutput{
			Name:         one.Name,
			Quantity:     one.Measure.Quantity,
			Unit:         one.Measure.Unit,
			TotalPaid:    int64(one.Price.Total),
			UnitPrice:    int64(one.Price.Unit),
			IngredientID: int64(one.IngredientID),
		})
	}

	return understood
}

// ambiguousOf maps the names the agent was not sure about onto the answer
// of the tool.
func ambiguousOf(draft purchase.Draft) []draftAmbiguityOutput {
	ambiguous := make([]draftAmbiguityOutput, 0, len(draft.Ambiguous))

	for _, one := range draft.Ambiguous {
		ambiguous = append(ambiguous, draftAmbiguityOutput{
			Name:    one.Name,
			Options: one.Options,
		})
	}

	return ambiguous
}

// outputOfConfirmation maps the outcome of the confirmation onto the answer
// of the tool.
//
// The answer carries the three parts of the answer of Paca: what it
// recorded, what it did not understand and what it still needs.
func outputOfConfirmation(confirmation purchase.Confirmation) confirmPurchaseOutput {
	lines := make([]purchaseLineOutput, 0, len(confirmation.Purchase.Lines))
	for _, one := range confirmation.Purchase.Lines {
		lines = append(lines, outputOfPurchaseLine(one))
	}

	recorded := make([]purchaseLineOutput, 0, len(confirmation.Recorded))
	for _, one := range confirmation.Recorded {
		recorded = append(recorded, outputOfPurchaseLine(one))
	}

	notUnderstood := make([]draftAmbiguityOutput, 0, len(confirmation.NotUnderstood))
	for _, one := range confirmation.NotUnderstood {
		notUnderstood = append(notUnderstood, draftAmbiguityOutput{
			Name:    one.Name,
			Options: one.Options,
		})
	}

	return confirmPurchaseOutput{
		Purchase: purchaseOutput{
			ID:            int64(confirmation.Purchase.ID),
			DraftID:       int64(confirmation.Purchase.DraftID),
			BusinessID:    int64(confirmation.Purchase.BusinessID),
			Day:           confirmation.Purchase.Day.String(),
			Market:        confirmation.Purchase.Market,
			Invoice:       confirmation.Purchase.Invoice,
			Utterance:     confirmation.Purchase.Utterance,
			ConfirmedWith: confirmation.Purchase.ConfirmedWith,
			Total:         int64(confirmation.Purchase.Total),
			Lines:         lines,
		},
		Recorded:      recorded,
		NotUnderstood: notUnderstood,
		StillNeeds:    emptyIfNil(confirmation.StillNeeds),
		NewNames:      emptyIfNil(confirmation.NewNames),
	}
}

// outputOfPurchaseLine maps one line of the purchase onto the answer of the
// tool.
func outputOfPurchaseLine(one purchase.Line) purchaseLineOutput {
	return purchaseLineOutput{
		Name:         one.Name,
		Quantity:     one.Measure.Quantity,
		Unit:         one.Measure.Unit,
		TotalPaid:    int64(one.Price.Total),
		UnitPrice:    int64(one.Price.Unit),
		IngredientID: int64(one.IngredientID),
	}
}

// emptyIfNil answers an empty list when the list is not there.
//
// The answer of a tool is read by an agent, and a list that is not there is
// a list the agent has to guess.
func emptyIfNil(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}
