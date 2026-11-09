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

// stubRotationReader is the use case that reads the rotation, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubRotationReader struct {
	answer recipe.Rotation
	err    error
	got    ports.ReadRotation
	calls  int
}

// Read answers the rotation of the test, and remembers what the tool asked.
func (s *stubRotationReader) Read(_ context.Context, request ports.ReadRotation) (recipe.Rotation, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Rotation{}, s.err
	}

	return s.answer, nil
}

// connectWithTheRotation starts the transport with the stub of the
// rotation.
func connectWithTheRotation(t *testing.T, reader ports.RotationReader) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		ReadRotation:        reader,
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

// theRotationOfTheTool is the answer the tool of the test gives: two units
// of the book, and the first one is the one that leaves first.
func theRotationOfTheTool(t *testing.T) recipe.Rotation {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-15")
	if err != nil {
		t.Fatalf("the day of the test could not be built: %v", err)
	}

	return recipe.Rotation{
		Day: day,
		Units: []recipe.RotationUnit{
			{
				UnitID:       recipe.UnitID(1),
				Order:        1,
				ProductionID: recipe.ProductionID(3),
				Position:     1,
				RecipeID:     recipe.ID(9),
				RecipeName:   "Pan de molde",
				ProducedOn:   ledger.NewDate(day.Time().AddDate(0, 0, -1)),
				Grams:        90,
				ExpiresOn:    ledger.NewDate(day.Time().AddDate(0, 0, 2)),
				DaysLeft:     2,
			},
			{
				UnitID:       recipe.UnitID(2),
				Order:        2,
				ProductionID: recipe.ProductionID(3),
				Position:     2,
				RecipeID:     recipe.ID(9),
				RecipeName:   "Pan de molde",
				ProducedOn:   ledger.NewDate(day.Time().AddDate(0, 0, -1)),
				Grams:        90,
				ExpiresOn:    ledger.NewDate(day.Time().AddDate(0, 0, 5)),
				DaysLeft:     5,
			},
		},
	}
}

// theRotationAnswer is the answer of the tool of the rotation, decoded.
type theRotationAnswer struct {
	Day   string `json:"day"`
	Units []struct {
		UnitID     int64  `json:"unit_id"`
		Order      int    `json:"order"`
		RecipeName string `json:"recipe_name"`
		ExpiresOn  string `json:"expires_on"`
		DaysLeft   int    `json:"days_left"`
	} `json:"units"`
}

// TestTheReadRotationToolAnswersWhatLeavesFirstAndWhenItExpires proves
// that the tool answers the order in which the units leave the book and the
// day each one expires.
func TestTheReadRotationToolAnswersWhatLeavesFirstAndWhenItExpires(t *testing.T) {
	t.Parallel()

	reader := &stubRotationReader{answer: theRotationOfTheTool(t)}
	session := connectWithTheRotation(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadRotation,
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

	var answer theRotationAnswer

	jsonOf(t, result, &answer)

	if len(answer.Units) != 2 {
		t.Fatalf("the answer carries %d units, want 2", len(answer.Units))
	}

	if answer.Units[0].ExpiresOn != "2026-03-17" {
		t.Errorf("the first unit expires on %q, want %q", answer.Units[0].ExpiresOn, "2026-03-17")
	}

	if answer.Units[0].Order != 1 {
		t.Errorf("the first unit takes the place %d, want 1", answer.Units[0].Order)
	}

	if answer.Units[1].Order != 2 {
		t.Errorf("the second unit takes the place %d, want 2", answer.Units[1].Order)
	}

	if answer.Units[0].DaysLeft != 2 {
		t.Errorf("the first unit has %d days left, want 2", answer.Units[0].DaysLeft)
	}

	if answer.Units[1].ExpiresOn != "2026-03-20" {
		t.Errorf("the second unit expires on %q, want %q", answer.Units[1].ExpiresOn, "2026-03-20")
	}

	answer2 := textOf(t, result)
	for _, want := range []string{"Pan de molde", "2026-03-17", "2026-03-15"} {
		if !strings.Contains(answer2, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer2)
		}
	}
}

// TestTheReadRotationToolOfOneProductCarriesTheName proves that the tool
// asks for the rotation of the product the owner named, and that it names
// it in the answer.
func TestTheReadRotationToolOfOneProductCarriesTheName(t *testing.T) {
	t.Parallel()

	rotation := theRotationOfTheTool(t)
	rotation.RecipeID = recipe.ID(9)
	rotation.RecipeName = "Pan de molde"

	reader := &stubRotationReader{answer: rotation}
	session := connectWithTheRotation(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadRotation,
		Arguments: map[string]any{
			"business_id": 77,
			"recipe_id":   9,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if reader.got.RecipeID != 9 {
		t.Errorf("the product is %d, want 9", reader.got.RecipeID)
	}

	answer := textOf(t, result)
	if !strings.Contains(answer, "Pan de molde") {
		t.Errorf("the answer does not carry the name of the product:\n%s", answer)
	}
}

// TestTheReadRotationToolRefusesTheBusinessThatIsNotThere proves that the
// business is required: the tool refuses the business that is not there, and
// it never asks the use case for the rotation of nobody.
func TestTheReadRotationToolRefusesTheBusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	reader := &stubRotationReader{answer: theRotationOfTheTool(t)}
	session := connectWithTheRotation(t, reader)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadRotation,
		Arguments: map[string]any{
			"business_id": 0,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a rotation, want the refusal of the business that is not there")
	}
	if reader.calls != 0 {
		t.Errorf("the use case was called %d times, want none", reader.calls)
	}
}

// TestTheReadRotationToolWithoutADatabaseSaysSo proves that the tool is
// there even when the service has no database, and that it says so.
func TestTheReadRotationToolWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	session := connectWithTheRotation(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadRotation,
		Arguments: map[string]any{
			"business_id": 77,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered a rotation, want the news that the database is not there")
	}
}

// TestTheDescriptionOfTheRotationIsAContractWithTheAgent proves that the
// description of the tool says what the tool answers: the order in which the
// units leave the book, the day they expire, and the fact that the answer
// carries no money.
func TestTheDescriptionOfTheRotationIsAContractWithTheAgent(t *testing.T) {
	t.Parallel()

	session := connectWithTheRotation(t, &stubRotationReader{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadRotation)

	for _, want := range []string{
		"business is REQUIRED",
		"product is OPTIONAL",
		"EARLIEST EXPIRY",
		"PLACE it takes in that order",
		"DAY it expires",
		"DAYS that are left",
		"NO money",
		"changes nothing",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}
