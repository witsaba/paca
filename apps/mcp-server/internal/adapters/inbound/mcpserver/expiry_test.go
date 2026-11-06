package mcpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// stubShelfLifeCorrector is the use case that corrects the shelf life, in
// memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubShelfLifeCorrector struct {
	answer ports.CorrectedShelfLife
	err    error
	got    ports.CorrectShelfLife
	calls  int
}

// Correct answers the correction of the test, and remembers what the tool
// asked.
func (s *stubShelfLifeCorrector) Correct(_ context.Context, request ports.CorrectShelfLife) (ports.CorrectedShelfLife, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return ports.CorrectedShelfLife{}, s.err
	}

	return s.answer, nil
}

// stubExpiringReader is the use case that reads what expires, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubExpiringReader struct {
	answer recipe.Expiry
	err    error
	got    ports.ReadExpiring
	calls  int
}

// Read answers the expiry of the test, and remembers what the tool asked.
func (s *stubExpiringReader) Read(_ context.Context, request ports.ReadExpiring) (recipe.Expiry, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Expiry{}, s.err
	}

	return s.answer, nil
}

// connectWithTheExpiry starts the transport with the stubs of the expiry.
func connectWithTheExpiry(
	t *testing.T,
	corrector ports.ShelfLifeCorrector,
	reader ports.ExpiringReader,
) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		CorrectShelfLife:    corrector,
		ReadExpiring:        reader,
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

// theCorrectionOfTheTool is the correction the tool of the test answers.
func theCorrectionOfTheTool(t *testing.T) ports.CorrectedShelfLife {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-16")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	return ports.CorrectedShelfLife{
		Correction: recipe.ShelfLifeCorrection{
			ID:            recipe.CorrectionID(1),
			BusinessID:    business.ID(77),
			RecipeID:      recipe.ID(9),
			RecipeName:    "Pan de molde",
			ShelfLifeDays: 5,
			Day:           day,
			Utterance:     "el pan de molde dura cinco dias",
			ConfirmedWith: "sí",
		},
		Units: []recipe.CorrectedUnit{
			{
				UnitID: recipe.UnitID(1),
				Was:    ledger.NewDate(day.Time().AddDate(0, 0, 1)),
				Now:    ledger.NewDate(day.Time().AddDate(0, 0, 3)),
			},
		},
	}
}

// theExpiryOfTheTool is the answer the tool of the test gives: two units,
// and the first one is already expired.
func theExpiryOfTheTool(t *testing.T) recipe.Expiry {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-15")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	return recipe.Expiry{
		Day: day,
		Units: []recipe.ExpiringUnit{
			{
				ProductionUnit: recipe.ProductionUnit{
					ID:           recipe.UnitID(1),
					ProductionID: recipe.ProductionID(3),
					Position:     1,
					Grams:        90,
					ExpiresOn:    ledger.NewDate(day.Time().AddDate(0, 0, 2)),
				},
				RecipeID:   recipe.ID(9),
				RecipeName: "Pan de molde",
				ProducedOn: ledger.NewDate(day.Time().AddDate(0, 0, -1)),
			},
			{
				ProductionUnit: recipe.ProductionUnit{
					ID:           recipe.UnitID(2),
					ProductionID: recipe.ProductionID(3),
					Position:     2,
					Grams:        90,
					ExpiresOn:    ledger.NewDate(day.Time().AddDate(0, 0, -1)),
				},
				RecipeID:   recipe.ID(9),
				RecipeName: "Pan de molde",
				ProducedOn: ledger.NewDate(day.Time().AddDate(0, 0, -4)),
			},
		},
	}
}

// TestTheCorrectShelfLifeToolWritesTheDaysAndTheWord proves that the tool
// writes the correction the owner confirmed: the recipe she corrected, the
// number of days, the day and the word she used.
func TestTheCorrectShelfLifeToolWritesTheDaysAndTheWord(t *testing.T) {
	t.Parallel()

	corrector := &stubShelfLifeCorrector{answer: theCorrectionOfTheTool(t)}
	session := connectWithTheExpiry(t, corrector, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCorrectShelfLife,
		Arguments: map[string]any{
			"business_id":     77,
			"recipe_name":     "Pan de molde",
			"shelf_life_days": 5,
			"day":             "2026-03-16",
			"utterance":       "el pan de molde dura cinco dias",
			"confirmed_with":  "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if corrector.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", corrector.calls)
	}
	if corrector.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", corrector.got.BusinessID)
	}
	if corrector.got.RecipeName != "Pan de molde" {
		t.Errorf("the name of the recipe is %q, want %q", corrector.got.RecipeName, "Pan de molde")
	}
	if corrector.got.ShelfLifeDays != 5 {
		t.Errorf("the days are %d, want 5", corrector.got.ShelfLifeDays)
	}
	if corrector.got.Day != "2026-03-16" {
		t.Errorf("the day is %q, want %q", corrector.got.Day, "2026-03-16")
	}
	if corrector.got.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", corrector.got.ConfirmedWith, "sí")
	}

	answer := textOf(t, result)
	for _, want := range []string{"Pan de molde", "2026-03-16", "2026-03-19"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheCorrectShelfLifeToolRefusesTheDaysThatAreNotThere proves that the
// number of days is required: the book never fills it by itself.
func TestTheCorrectShelfLifeToolRefusesTheDaysThatAreNotThere(t *testing.T) {
	t.Parallel()

	corrector := &stubShelfLifeCorrector{answer: theCorrectionOfTheTool(t)}
	session := connectWithTheExpiry(t, corrector, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCorrectShelfLife,
		Arguments: map[string]any{
			"business_id":     77,
			"recipe_name":     "Pan de molde",
			"shelf_life_days": 0,
			"day":             "2026-03-16",
			"confirmed_with":  "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a correction, want the refusal of the days that are not there")
	}
	if corrector.calls != 0 {
		t.Errorf("the use case was called %d times, want none", corrector.calls)
	}
}

// TestTheCorrectShelfLifeToolWithoutADatabaseSaysSo proves that the tool is
// there even when the service has no database, and that it says so.
func TestTheCorrectShelfLifeToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheExpiry(t, nil, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCorrectShelfLife,
		Arguments: map[string]any{
			"business_id":     77,
			"recipe_name":     "Pan de molde",
			"shelf_life_days": 5,
			"confirmed_with":  "sí",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a correction, want the news that the database is not there")
	}
}

// TestTheReadExpiringToolAnswersWhatExpiresAndWhen proves that the tool
// answers the units of the business with the day each one expires, the days
// that are left and whether it is already expired.
func TestTheReadExpiringToolAnswersWhatExpiresAndWhen(t *testing.T) {
	t.Parallel()

	reader := &stubExpiringReader{answer: theExpiryOfTheTool(t)}
	session := connectWithTheExpiry(t, nil, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadExpiring,
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
	for _, want := range []string{"Pan de molde", "2026-03-17", "2026-03-14", "2026-03-15"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheReadExpiringToolRefusesTheBusinessThatIsNotThere proves that the
// business is required: the tool refuses the business that is not there, and
// it never asks the use case for the expiry of nobody.
func TestTheReadExpiringToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	reader := &stubExpiringReader{answer: theExpiryOfTheTool(t)}
	session := connectWithTheExpiry(t, nil, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadExpiring,
		Arguments: map[string]any{
			"business_id": 0,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered an expiry, want the refusal of the business that is not there")
	}
	if reader.calls != 0 {
		t.Errorf("the use case was called %d times, want none", reader.calls)
	}
}

// TestTheDescriptionOfTheExpiryToolsIsAContractWithTheAgent proves that the
// descriptions of the tools say what the tools do: the correction of the
// shelf life, the days that are never invented, and the read of what
// expires.
func TestTheDescriptionOfTheExpiryToolsIsAContractWithTheAgent(t *testing.T) {
	t.Parallel()

	session := connectWithTheExpiry(t, &stubShelfLifeCorrector{}, &stubExpiringReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	correction := descriptionOf(t, tools.Tools, mcpserver.ToolCorrectShelfLife)
	for _, want := range []string{
		"business is REQUIRED",
		"recipe is REQUIRED",
		"day is OPTIONAL",
		"utterance is OPTIONAL",
		"confirmed_with is REQUIRED",
		"NEVER invented",
		"KEPT",
	} {
		if !strings.Contains(correction, want) {
			t.Errorf("the description does not say %q:\n%s", want, correction)
		}
	}

	reading := descriptionOf(t, tools.Tools, mcpserver.ToolReadExpiring)
	for _, want := range []string{
		"business is REQUIRED",
		"DAYS that are left",
		"NEVER sold",
		"changes nothing",
	} {
		if !strings.Contains(reading, want) {
			t.Errorf("the description does not say %q:\n%s", want, reading)
		}
	}
}
