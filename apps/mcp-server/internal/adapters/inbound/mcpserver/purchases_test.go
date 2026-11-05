package mcpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// stubPurchaseDrafter is the use case that keeps the draft of a purchase,
// in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubPurchaseDrafter struct {
	draft purchase.Draft
	err   error
	got   ports.DraftPurchase
	calls int
}

// Draft answers the draft of the test, and remembers what the tool asked.
func (s *stubPurchaseDrafter) Draft(_ context.Context, request ports.DraftPurchase) (purchase.Draft, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return purchase.Draft{}, s.err
	}

	return s.draft, nil
}

// connectWithPurchases starts the transport with the stub of the drafts.
func connectWithPurchases(t *testing.T, drafter ports.PurchaseDrafter) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		DraftPurchase:       drafter,
	})

	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)

	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: server.URL}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the transport: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Logf("the session did not close: %v", err)
		}
	})

	return session
}

// theAnswerOfTheDraft is the answer the stub of the use case gives.
func theAnswerOfTheDraft() purchase.Draft {
	day := ledger.NewDate(draftMoment())

	return purchase.Draft{
		ID:         42,
		BusinessID: business.ID(77),
		State:      purchase.StateProposed,
		Revision:   1,
		Utterance:  "Compré 25 kilos de harina",
		Lines: []purchase.Line{
			{
				Name:         "Harina",
				Measure:      purchase.Measure{Quantity: 25, Unit: "kg"},
				Price:        purchase.Price{Total: 12000},
				IngredientID: ingredient.ID(10),
			},
		},
		Missing:    []string{"the price of Pan"},
		Ambiguous:  []purchase.Ambiguity{{Name: "harina", Options: []string{"Harina panadera", "Harina de trigo"}}},
		Inferred:   []string{"the day of the purchase is the day of today"},
		Day:        day,
		Market:     "Plaza de Mercado",
		Invoice:    "FV-001",
		ProposedOn: day,
		ExpiresOn:  ledger.NewDate(draftMoment().AddDate(0, 0, purchase.DraftLifetimeDays)),
	}
}

// draftMoment is the moment the draft of the test was proposed.
func draftMoment() time.Time {
	return time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC)
}

// TestThePurchaseToolKeepsTheDraft proves that the tool carries what the
// agent sends to the use case, and that the answer carries the draft and
// the three parts of the answer of Paca.
func TestThePurchaseToolKeepsTheDraft(t *testing.T) {
	t.Parallel()

	drafter := &stubPurchaseDrafter{draft: theAnswerOfTheDraft()}
	session := connectWithPurchases(t, drafter)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolDraftPurchase,
		Arguments: map[string]any{
			"business_id": 77,
			"utterance":   "Compré 25 kilos de harina",
			"day":         "2026-03-14",
			"market":      "Plaza de Mercado",
			"invoice":     "FV-001",
			"lines": []any{
				map[string]any{
					"name":          "Harina",
					"quantity":      25,
					"unit":          "kg",
					"total_paid":    12000,
					"ingredient_id": 10,
				},
			},
			"ambiguities": []any{
				map[string]any{
					"name":    "harina",
					"options": []any{"Harina panadera", "Harina de trigo"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if drafter.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", drafter.calls)
	}
	if drafter.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", drafter.got.BusinessID)
	}
	if drafter.got.Utterance != "Compré 25 kilos de harina" {
		t.Errorf("the sentence is %q, want %q", drafter.got.Utterance, "Compré 25 kilos de harina")
	}
	if drafter.got.Day != "2026-03-14" {
		t.Errorf("the day is %q, want %q", drafter.got.Day, "2026-03-14")
	}
	if drafter.got.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", drafter.got.Market, "Plaza de Mercado")
	}
	if drafter.got.Invoice != "FV-001" {
		t.Errorf("the invoice is %q, want %q", drafter.got.Invoice, "FV-001")
	}
	if len(drafter.got.Lines) != 1 {
		t.Fatalf("the use case received %d lines, want 1", len(drafter.got.Lines))
	}
	if drafter.got.Lines[0].Name != "Harina" {
		t.Errorf("the name of the line is %q, want %q", drafter.got.Lines[0].Name, "Harina")
	}
	if drafter.got.Lines[0].Quantity != 25 {
		t.Errorf("the quantity is %v, want 25", drafter.got.Lines[0].Quantity)
	}
	if drafter.got.Lines[0].Unit != "kg" {
		t.Errorf("the unit is %q, want %q", drafter.got.Lines[0].Unit, "kg")
	}
	if drafter.got.Lines[0].Total != 12000 {
		t.Errorf("the money is %d, want 12000", drafter.got.Lines[0].Total)
	}
	if drafter.got.Lines[0].IngredientID != 10 {
		t.Errorf("the identity of the ingredient is %d, want 10", drafter.got.Lines[0].IngredientID)
	}
	if len(drafter.got.Ambiguities) != 1 {
		t.Fatalf("the use case received %d ambiguities, want 1", len(drafter.got.Ambiguities))
	}
	if len(drafter.got.Ambiguities[0].Options) != 2 {
		t.Errorf("the options are %v, want the two of the test", drafter.got.Ambiguities[0].Options)
	}

	answer := textOf(t, result)
	for _, want := range []string{
		"42", "proposed", "Harina", "25", "kg", "12000", "10",
		"the price of Pan",
		"Harina panadera",
		"the day of the purchase is the day of today",
		"2026-03-15", "2026-03-22",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestThePurchaseToolAnswersWithoutADatabase proves that a server without
// a database still exposes the tool, and that the call says so.
func TestThePurchaseToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithPurchases(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolDraftPurchase,
		Arguments: map[string]any{
			"business_id": 77,
			"utterance":   "Compré harina",
			"lines": []any{
				map[string]any{"name": "Harina", "quantity": 25, "unit": "kg", "total_paid": 12000},
			},
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the service")
	}

	if !strings.Contains(textOf(t, result), "database") {
		t.Errorf("the answer does not say that the database is not there: %s", textOf(t, result))
	}
}

// TestThePurchaseToolExplainsTheFailureToTheAgent proves that the failure
// of the use case reaches the agent with a message of its own, and never as
// the raw error of the domain.
func TestThePurchaseToolExplainsTheFailureToTheAgent(t *testing.T) {
	t.Parallel()

	drafter := &stubPurchaseDrafter{err: purchase.ErrInferredPrice}
	session := connectWithPurchases(t, drafter)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolDraftPurchase,
		Arguments: map[string]any{
			"business_id": 77,
			"utterance":   "Compré harina",
			"lines": []any{
				map[string]any{"name": "Harina", "quantity": 25, "unit": "kg", "total_paid": 12000},
			},
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the use case")
	}

	answer := textOf(t, result)
	if answer == purchase.ErrInferredPrice.Error() {
		t.Errorf("the answer is the raw error of the domain: %s", answer)
	}
	if !strings.Contains(answer, "never inferred") {
		t.Errorf("the answer does not explain the failure: %s", answer)
	}
}

// TestTheDescriptionOfThePurchaseToolSaysWhatItNeeds proves that the
// description of the tool is a contract with the agent: it says what is
// required, what the empty field means, and that it changes nothing in the
// book.
func TestTheDescriptionOfThePurchaseToolSaysWhatItNeeds(t *testing.T) {
	t.Parallel()

	session := connectWithPurchases(t, &stubPurchaseDrafter{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolDraftPurchase)

	for _, want := range []string{
		"business is REQUIRED",
		"sentence of the owner is REQUIRED",
		"At least one line is REQUIRED",
		"unit is REQUIRED",
		"never inferred",
		"day is OPTIONAL",
		"day of today",
		"market is OPTIONAL",
		"invoice is OPTIONAL",
		"changes nothing in the book",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheErrorsOfThePurchasesReachTheAgentWithAMessage proves that every
// error the drafts can return reaches the agent with a message of its own.
func TestTheErrorsOfThePurchasesReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	for _, failure := range purchase.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			drafter := &stubPurchaseDrafter{err: failure}
			session := connectWithPurchases(t, drafter)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolDraftPurchase,
				Arguments: map[string]any{
					"business_id": 77,
					"utterance":   "Compré harina",
					"lines": []any{
						map[string]any{"name": "Harina", "quantity": 25, "unit": "kg", "total_paid": 12000},
					},
				},
			})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool answered no error, want the failure of the use case")
			}

			answer := textOf(t, result)
			if answer == failure.Error() {
				t.Fatalf("the answer is the raw error of the domain: %s", answer)
			}
			if !strings.Contains(answer, ":") {
				t.Errorf("the answer says nothing about what to do: %s", answer)
			}
		})
	}
}

// stubPurchaseConfirmer is the use case that writes the purchase of a
// confirmed draft, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubPurchaseConfirmer struct {
	confirmation purchase.Confirmation
	err          error
	got          ports.ConfirmPurchase
	calls        int
}

// Confirm answers the confirmation of the test, and remembers what the tool
// asked.
func (s *stubPurchaseConfirmer) Confirm(_ context.Context, request ports.ConfirmPurchase) (purchase.Confirmation, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return purchase.Confirmation{}, s.err
	}

	return s.confirmation, nil
}

// connectWithTheUtterance starts the transport with the stubs of the
// utterance.
func connectWithTheUtterance(t *testing.T, drafter ports.PurchaseDrafter, confirmer ports.PurchaseConfirmer) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		DraftPurchase:       drafter,
		ConfirmPurchase:     confirmer,
	})

	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)

	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: server.URL}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the transport: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Logf("the session did not close: %v", err)
		}
	})

	return session
}

// theAnswerOfTheConfirmation is the answer the stub of the use case gives.
func theAnswerOfTheConfirmation() purchase.Confirmation {
	day := ledger.NewDate(draftMoment())

	return purchase.Confirmation{
		Purchase: purchase.Purchase{
			ID:            purchase.ID(11),
			DraftID:       purchase.DraftID(42),
			BusinessID:    business.ID(77),
			Day:           day,
			Market:        "Plaza de Mercado",
			Invoice:       "FV-001",
			Utterance:     "Compré 25 kilos de harina",
			ConfirmedWith: "sí",
			Lines: []purchase.Line{
				{
					Name:    "Harina",
					Measure: purchase.Measure{Quantity: 25, Unit: "kg"},
					Price:   purchase.Price{Total: 12000, Unit: 480},
				},
			},
			Total: 12000,
		},
		Recorded: []purchase.Line{
			{
				Name:    "Harina",
				Measure: purchase.Measure{Quantity: 25, Unit: "kg"},
				Price:   purchase.Price{Total: 12000, Unit: 480},
			},
		},
		NotUnderstood: []purchase.Ambiguity{
			{Name: "harina", Options: []string{"Harina panadera", "Harina de trigo"}},
		},
		StillNeeds: []string{},
	}
}

// TestTheConfirmationToolWritesThePurchase proves that the tool carries
// what the agent sends to the use case, and that the answer carries the
// purchase and the three parts of the answer of Paca.
func TestTheConfirmationToolWritesThePurchase(t *testing.T) {
	t.Parallel()

	confirmer := &stubPurchaseConfirmer{confirmation: theAnswerOfTheConfirmation()}
	session := connectWithTheUtterance(t, &stubPurchaseDrafter{}, confirmer)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolConfirmPurchase,
		Arguments: map[string]any{
			"business_id":     77,
			"draft_id":        42,
			"revision":        2,
			"confirmed_with":  "sí",
			"from_account_id": 2,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if confirmer.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", confirmer.calls)
	}
	if confirmer.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", confirmer.got.BusinessID)
	}
	if confirmer.got.DraftID != purchase.DraftID(42) {
		t.Errorf("the draft is %d, want 42", confirmer.got.DraftID)
	}
	if confirmer.got.Revision != 2 {
		t.Errorf("the revision is %d, want 2", confirmer.got.Revision)
	}
	if confirmer.got.ConfirmedWith != "sí" {
		t.Errorf("the word is %q, want %q", confirmer.got.ConfirmedWith, "sí")
	}
	if confirmer.got.FromAccountID != account.ID(2) {
		t.Errorf("the account that pays is %d, want 2", confirmer.got.FromAccountID)
	}

	answer := textOf(t, result)
	for _, want := range []string{
		"11", "42", "77", "2026-03-15", "Plaza de Mercado", "FV-001",
		"Compré 25 kilos de harina", "sí", "12000", "480", "Harina", "25", "kg",
		"Harina panadera",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheConfirmationToolAnswersWithoutADatabase proves that a server
// without a database still exposes the tool, and that the call says so.
func TestTheConfirmationToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithTheUtterance(t, &stubPurchaseDrafter{}, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolConfirmPurchase,
		Arguments: map[string]any{
			"business_id":    77,
			"draft_id":       42,
			"revision":       1,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the service")
	}

	if !strings.Contains(textOf(t, result), "database") {
		t.Errorf("the answer does not say that the database is not there: %s", textOf(t, result))
	}
}

// TestTheConfirmationToolExplainsTheFailureToTheAgent proves that the
// failure of the use case reaches the agent with a message of its own, and
// never as the raw error of the domain.
func TestTheConfirmationToolExplainsTheFailureToTheAgent(t *testing.T) {
	t.Parallel()

	confirmer := &stubPurchaseConfirmer{err: purchase.ErrExpiredDraft}
	session := connectWithTheUtterance(t, &stubPurchaseDrafter{}, confirmer)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolConfirmPurchase,
		Arguments: map[string]any{
			"business_id":    77,
			"draft_id":       42,
			"revision":       1,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the use case")
	}

	answer := textOf(t, result)
	if answer == purchase.ErrExpiredDraft.Error() {
		t.Errorf("the answer is the raw error of the domain: %s", answer)
	}
	if !strings.Contains(answer, "expired") {
		t.Errorf("the answer does not explain the failure: %s", answer)
	}
}

// TestTheDescriptionOfTheConfirmationToolSaysWhatItNeeds proves that the
// description of the tool is a contract with the agent: it says what is
// required, what the empty field means, and what the tool writes.
func TestTheDescriptionOfTheConfirmationToolSaysWhatItNeeds(t *testing.T) {
	t.Parallel()

	session := connectWithTheUtterance(t, &stubPurchaseDrafter{}, &stubPurchaseConfirmer{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolConfirmPurchase)

	for _, want := range []string{
		"business is REQUIRED",
		"draft is REQUIRED",
		"revision is REQUIRED",
		"word is REQUIRED",
		"account that pays is OPTIONAL",
		"cash of the business",
		"writes the purchase",
		"moves the money",
		"names the names the catalog did not know",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheErrorsOfTheConfirmationReachTheAgentWithAMessage proves that every
// error the confirmation can return reaches the agent with a message of its
// own.
func TestTheErrorsOfTheConfirmationReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	for _, failure := range purchase.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			confirmer := &stubPurchaseConfirmer{err: failure}
			session := connectWithTheUtterance(t, &stubPurchaseDrafter{}, confirmer)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolConfirmPurchase,
				Arguments: map[string]any{
					"business_id":    77,
					"draft_id":       42,
					"revision":       1,
					"confirmed_with": "sí",
				},
			})
			if err != nil {
				t.Fatalf("the tool failed: %v", err)
			}
			if !result.IsError {
				t.Fatal("the tool answered no error, want the failure of the use case")
			}

			answer := textOf(t, result)
			if answer == failure.Error() {
				t.Fatalf("the answer is the raw error of the domain: %s", answer)
			}
			if !strings.Contains(answer, ":") {
				t.Errorf("the answer says nothing about what to do: %s", answer)
			}
		})
	}
}

// stubPurchaseCanceller is the use case that cancels a draft, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubPurchaseCanceller struct {
	cancelled purchase.Draft
	err       error
	got       ports.CancelDraft
	calls     int
}

// Cancel answers the cancelled draft of the test, and remembers what the
// tool asked.
func (s *stubPurchaseCanceller) Cancel(_ context.Context, request ports.CancelDraft) (purchase.Draft, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return purchase.Draft{}, s.err
	}

	return s.cancelled, nil
}

// stubDraftsReader is the use case that reads the pending drafts, in
// memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubDraftsReader struct {
	drafts []purchase.Draft
	err    error
	got    ports.ReadDrafts
	calls  int
}

// Read answers the drafts of the test, and remembers what the tool asked.
func (s *stubDraftsReader) Read(_ context.Context, request ports.ReadDrafts) ([]purchase.Draft, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return nil, s.err
	}

	return s.drafts, nil
}

// connectWithThePendingDrafts starts the transport with the stubs of the
// tools of the pending drafts.
func connectWithThePendingDrafts(
	t *testing.T,
	canceller ports.PurchaseCanceller,
	reader ports.DraftsReader,
) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		CancelDraft:         canceller,
		ReadDrafts:          reader,
	})

	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)

	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: server.URL}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the transport: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Logf("the session did not close: %v", err)
		}
	})

	return session
}

// theCancelledDraft is the answer the stub of the use case gives.
func theCancelledDraft() purchase.Draft {
	cancelled := theAnswerOfTheDraft()
	cancelled.State = purchase.StateCancelled

	return cancelled
}

// TestTheCancellationToolDropsTheDraft proves that the tool carries what
// the agent sends to the use case, and that the answer carries the draft it
// cancelled.
func TestTheCancellationToolDropsTheDraft(t *testing.T) {
	t.Parallel()

	canceller := &stubPurchaseCanceller{cancelled: theCancelledDraft()}
	session := connectWithThePendingDrafts(t, canceller, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCancelDraft,
		Arguments: map[string]any{
			"business_id": 77,
			"draft_id":    42,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if canceller.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", canceller.calls)
	}
	if canceller.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", canceller.got.BusinessID)
	}
	if canceller.got.DraftID != purchase.DraftID(42) {
		t.Errorf("the draft is %d, want 42", canceller.got.DraftID)
	}

	answer := textOf(t, result)
	for _, want := range []string{"42", "cancelled", "Compré 25 kilos de harina"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheCancellationToolAnswersWithoutADatabase proves that a server
// without a database still exposes the tool, and that the call says so.
func TestTheCancellationToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithThePendingDrafts(t, nil, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCancelDraft,
		Arguments: map[string]any{
			"business_id": 77,
			"draft_id":    42,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the service")
	}

	if !strings.Contains(textOf(t, result), "database") {
		t.Errorf("the answer does not say that the database is not there: %s", textOf(t, result))
	}
}

// TestTheCancellationToolExplainsTheFailureToTheAgent proves that the
// failure of the use case reaches the agent with a message of its own, and
// never as the raw error of the domain.
func TestTheCancellationToolExplainsTheFailureToTheAgent(t *testing.T) {
	t.Parallel()

	canceller := &stubPurchaseCanceller{err: purchase.ErrNotCancellable}
	session := connectWithThePendingDrafts(t, canceller, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCancelDraft,
		Arguments: map[string]any{
			"business_id": 77,
			"draft_id":    42,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the use case")
	}

	answer := textOf(t, result)
	if answer == purchase.ErrNotCancellable.Error() {
		t.Errorf("the answer is the raw error of the domain: %s", answer)
	}
	if !strings.Contains(answer, "wait") {
		t.Errorf("the answer does not explain the failure: %s", answer)
	}
}

// TestTheDescriptionOfTheCancellationToolSaysWhatItNeeds proves that the
// description of the tool is a contract with the agent: it says what is
// required and that the tool changes nothing in the book.
func TestTheDescriptionOfTheCancellationToolSaysWhatItNeeds(t *testing.T) {
	t.Parallel()

	session := connectWithThePendingDrafts(t, &stubPurchaseCanceller{}, nil)

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolCancelDraft)

	for _, want := range []string{
		"business is REQUIRED",
		"draft is REQUIRED",
		"changes nothing in the book",
		"cancelled",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheReadToolAnswersThePendingDrafts proves that the tool carries what
// the agent sends to the use case, and that the answer carries every
// pending draft with what it still needs, what it was not sure about and
// what Paca filled.
func TestTheReadToolAnswersThePendingDrafts(t *testing.T) {
	t.Parallel()

	reader := &stubDraftsReader{drafts: []purchase.Draft{theAnswerOfTheDraft()}}
	session := connectWithThePendingDrafts(t, nil, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadDrafts,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if reader.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", reader.calls)
	}
	if reader.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", reader.got.BusinessID)
	}

	answer := textOf(t, result)
	for _, want := range []string{
		"42", "proposed", "Harina", "25", "kg", "12000",
		"the price of Pan",
		"Harina panadera",
		"the day of the purchase is the day of today",
		"2026-03-15", "2026-03-22",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheReadToolAnswersWithoutADatabase proves that a server without a
// database still exposes the tool, and that the call says so.
func TestTheReadToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithThePendingDrafts(t, nil, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadDrafts,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the service")
	}

	if !strings.Contains(textOf(t, result), "database") {
		t.Errorf("the answer does not say that the database is not there: %s", textOf(t, result))
	}
}

// TestTheDescriptionOfTheReadToolSaysWhatItNeeds proves that the
// description of the tool is a contract with the agent: it says what is
// required and that the tool changes nothing in the book.
func TestTheDescriptionOfTheReadToolSaysWhatItNeeds(t *testing.T) {
	t.Parallel()

	session := connectWithThePendingDrafts(t, nil, &stubDraftsReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadDrafts)

	for _, want := range []string{
		"business is REQUIRED",
		"day is OPTIONAL",
		"day of today",
		"changes nothing in the book",
		"what it still needs",
		"what it was not sure about",
		"what PACA filled",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheConfirmationToolNamesTheNamesTheCatalogDidNotKnow proves that the
// answer of the confirmation names the names the catalog did not know, so
// the agent can tell the owner that a name is new.
func TestTheConfirmationToolNamesTheNamesTheCatalogDidNotKnow(t *testing.T) {
	t.Parallel()

	answer := theAnswerOfTheConfirmation()
	answer.NewNames = []string{"Harina"}

	confirmer := &stubPurchaseConfirmer{confirmation: answer}
	session := connectWithTheUtterance(t, &stubPurchaseDrafter{}, confirmer)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolConfirmPurchase,
		Arguments: map[string]any{
			"business_id":    77,
			"draft_id":       42,
			"revision":       1,
			"confirmed_with": "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if !strings.Contains(textOf(t, result), "new_names") {
		t.Errorf("the answer does not name the new names: %s", textOf(t, result))
	}
}
