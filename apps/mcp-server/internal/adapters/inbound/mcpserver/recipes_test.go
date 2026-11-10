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
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// stubRecipeRecorder is the use case that writes the recipe of the bakery,
// in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubRecipeRecorder struct {
	recorded recipe.Recipe
	err      error
	got      ports.RecordRecipe
	calls    int
}

// Record answers the recipe of the test, and remembers what the tool asked.
func (s *stubRecipeRecorder) Record(_ context.Context, request ports.RecordRecipe) (recipe.Recipe, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return recipe.Recipe{}, s.err
	}

	return s.recorded, nil
}

// connectWithTheRecipes starts the transport with the stub of the recipes.
func connectWithTheRecipes(t *testing.T, recorder ports.RecipeRecorder) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		RecordRecipe:        recorder,
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

// theAnswerOfTheRecipe is the answer the stub of the use case gives.
func theAnswerOfTheRecipe() recipe.Recipe {
	return recipe.Recipe{
		ID:            recipe.ID(9),
		BusinessID:    business.ID(77),
		Name:          "Pan de molde",
		Ingredients:   []recipe.Ingredient{{IngredientID: ingredient.ID(10), Grams: 500}},
		Steps:         []recipe.Step{{Instruction: "Hornear.", Minutes: 45, Temperature: 220}},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		Utterance:     "El pan de molde lleva 500 gramos de harina",
		ConfirmedWith: "sí",
	}
}

// theArgumentsOfTheRecipe are the arguments the agent sends.
func theArgumentsOfTheRecipe() map[string]any {
	return map[string]any{
		"business_id": 77,
		"name":        "Pan de molde",
		"ingredients": []any{
			map[string]any{"ingredient_id": 10, "grams": 500},
		},
		"steps": []any{
			map[string]any{
				"instruction": "Mezclar la harina con el agua y amasar.",
				"cue":         "la masa que holds la forma",
				"minutes":     20,
				"temperature": 24,
			},
			map[string]any{"instruction": "Hornear.", "minutes": 45, "temperature": 220},
		},
		"yield_units":     10,
		"yield_grams":     90,
		"shelf_life_days": 3,
		"utterance":       "El pan de molde lleva 500 gramos de harina",
		"confirmed_with":  "sí",
	}
}

// TestTheRecipeToolWritesTheRecipe proves that the tool carries what the
// agent sends to the use case, and that the answer carries the recipe the
// owner confirmed.
func TestTheRecipeToolWritesTheRecipe(t *testing.T) {
	t.Parallel()

	recorder := &stubRecipeRecorder{recorded: theAnswerOfTheRecipe()}
	session := connectWithTheRecipes(t, recorder)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolRecordRecipe,
		Arguments: theArgumentsOfTheRecipe(),
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if recorder.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", recorder.calls)
	}
	if recorder.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", recorder.got.BusinessID)
	}
	if recorder.got.Name != "Pan de molde" {
		t.Errorf("the name is %q, want %q", recorder.got.Name, "Pan de molde")
	}
	if len(recorder.got.Ingredients) != 1 {
		t.Fatalf("the recipe carries %d ingredients, want 1", len(recorder.got.Ingredients))
	}
	if recorder.got.Ingredients[0].IngredientID != 10 {
		t.Errorf("the ingredient is %d, want 10", recorder.got.Ingredients[0].IngredientID)
	}
	if recorder.got.Ingredients[0].Grams != 500 {
		t.Errorf("the grams of the ingredient are %v, want 500", recorder.got.Ingredients[0].Grams)
	}
	if len(recorder.got.Steps) != 2 {
		t.Fatalf("the recipe carries %d steps, want 2", len(recorder.got.Steps))
	}
	if recorder.got.Steps[0].Instruction != "Mezclar la harina con el agua y amasar." {
		t.Errorf("the first step is %q, want the one of the owner", recorder.got.Steps[0].Instruction)
	}
	if recorder.got.Steps[0].Cue != "la masa que holds la forma" {
		t.Errorf("the cue of the first step is %q, want the one of the owner", recorder.got.Steps[0].Cue)
	}
	if recorder.got.Steps[0].Minutes != 20 || recorder.got.Steps[0].Temperature != 24 {
		t.Errorf("the first step is %+v, want 20 minutes at 24 degrees", recorder.got.Steps[0])
	}
	if recorder.got.Steps[1].Instruction != "Hornear." {
		t.Errorf("the second step is %q, want %q", recorder.got.Steps[1].Instruction, "Hornear.")
	}
	if recorder.got.YieldUnits != 10 {
		t.Errorf("the yield is %d units, want 10", recorder.got.YieldUnits)
	}
	if recorder.got.YieldGrams != 90 {
		t.Errorf("the weight of the unit is %v grams, want 90", recorder.got.YieldGrams)
	}
	if recorder.got.ShelfLifeDays != 3 {
		t.Errorf("the shelf life is %d days, want 3", recorder.got.ShelfLifeDays)
	}
	if recorder.got.Utterance != "El pan de molde lleva 500 gramos de harina" {
		t.Errorf("the sentence is %q, want the one of the owner", recorder.got.Utterance)
	}
	if recorder.got.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", recorder.got.ConfirmedWith, "sí")
	}

	answer := textOf(t, result)
	for _, want := range []string{"Pan de molde", "Hornear.", "500", "90", "3"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q:\n%s", want, answer)
		}
	}
}

// TestTheDescriptionOfTheRecipeToolSaysWhatItNeeds proves that the
// description of the tool is a contract with the agent: it says what is
// required, and it says what the recipe is.
func TestTheDescriptionOfTheRecipeToolSaysWhatItNeeds(t *testing.T) {
	t.Parallel()

	session := connectWithTheRecipes(t, &stubRecipeRecorder{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolRecordRecipe)

	for _, want := range []string{
		"business is REQUIRED",
		"name is REQUIRED",
		"ingredient_id is REQUIRED",
		"grams is REQUIRED",
		"instruction is REQUIRED",
		"cue is OPTIONAL",
		"minutes is OPTIONAL",
		"temperature is OPTIONAL",
		"yield_units is REQUIRED",
		"yield_grams is REQUIRED",
		"shelf_life_days is REQUIRED",
		"confirmed_with is REQUIRED",
		"never measured",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheErrorsOfTheRecipesReachTheAgentWithAMessage proves that every
// error the recipe can return reaches the agent with a message of its own.
func TestTheErrorsOfTheRecipesReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	for _, failure := range recipe.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			recorder := &stubRecipeRecorder{err: failure}
			session := connectWithTheRecipes(t, recorder)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      mcpserver.ToolRecordRecipe,
				Arguments: theArgumentsOfTheRecipe(),
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

// TestTheRecipeToolWithoutADatabaseIsRefused proves that the tool is there
// even when the service has no database, and that it says so.
func TestTheRecipeToolWithoutADatabaseIsRefused(t *testing.T) {
	t.Parallel()

	session := connectWithTheRecipes(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      mcpserver.ToolRecordRecipe,
		Arguments: theArgumentsOfTheRecipe(),
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the refusal of the service")
	}

	answer := textOf(t, result)
	if !strings.Contains(answer, "database") {
		t.Errorf("the answer does not say that the database is not there: %s", answer)
	}
}
