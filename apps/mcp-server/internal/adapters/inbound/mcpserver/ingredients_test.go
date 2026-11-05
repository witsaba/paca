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
)

// stubItemMatcher is the use case that matches a name against the catalog,
// in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubItemMatcher struct {
	match ingredient.Match
	err   error
	got   ports.MatchItems
	calls int
}

// Match answers the match of the test, and remembers what the tool asked.
func (s *stubItemMatcher) Match(_ context.Context, request ports.MatchItems) (ingredient.Match, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return ingredient.Match{}, s.err
	}

	return s.match, nil
}

// connectWithTheCatalog starts the transport with the stub of the catalog.
func connectWithTheCatalog(t *testing.T, matcher ports.ItemMatcher) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		MatchItems:          matcher,
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

// theAnswerOfTheMatch is the answer the stub of the use case gives.
func theAnswerOfTheMatch() ingredient.Match {
	return ingredient.Match{
		Name:   "harina",
		Status: ingredient.StatusAmbiguous,
		Options: []ingredient.Option{
			{ID: 10, Name: "Harina panadera"},
			{ID: 20, Name: "Harina de trigo"},
		},
	}
}

// TestTheMatchToolAnswersTheCandidates proves that the tool carries what
// the agent sends to the use case, and that the answer carries the status
// of the name and the options it offers.
func TestTheMatchToolAnswersTheCandidates(t *testing.T) {
	t.Parallel()

	matcher := &stubItemMatcher{match: theAnswerOfTheMatch()}
	session := connectWithTheCatalog(t, matcher)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMatchItems,
		Arguments: map[string]any{
			"business_id": 77,
			"name":        "harina",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if matcher.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", matcher.calls)
	}
	if matcher.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", matcher.got.BusinessID)
	}
	if matcher.got.Name != "harina" {
		t.Errorf("the name is %q, want %q", matcher.got.Name, "harina")
	}

	answer := textOf(t, result)
	for _, want := range []string{
		"harina", "ambiguous", "Harina panadera", "Harina de trigo", "10", "20",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheMatchToolAnswersTheNameItDoesNotKnow proves that a name the
// catalog does not know is not an error: the answer says that it is new.
func TestTheMatchToolAnswersTheNameItDoesNotKnow(t *testing.T) {
	t.Parallel()

	matcher := &stubItemMatcher{match: ingredient.Match{
		Name:   "polvasa",
		Status: ingredient.StatusUnknown,
		New:    true,
	}}
	session := connectWithTheCatalog(t, matcher)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMatchItems,
		Arguments: map[string]any{
			"business_id": 77,
			"name":        "polvasa",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	answer := textOf(t, result)
	for _, want := range []string{"polvasa", "unknown", "true"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheMatchToolAnswersWithoutADatabase proves that a server without a
// database still exposes the tool, and that the call says so.
func TestTheMatchToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithTheCatalog(t, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMatchItems,
		Arguments: map[string]any{
			"business_id": 77,
			"name":        "harina",
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

// TestTheMatchToolExplainsTheFailureToTheAgent proves that the failure of
// the use case reaches the agent with a message of its own, and never as
// the raw error of the domain.
func TestTheMatchToolExplainsTheFailureToTheAgent(t *testing.T) {
	t.Parallel()

	matcher := &stubItemMatcher{err: ingredient.ErrDuplicateName}
	session := connectWithTheCatalog(t, matcher)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolMatchItems,
		Arguments: map[string]any{
			"business_id": 77,
			"name":        "harina",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the use case")
	}

	answer := textOf(t, result)
	if answer == ingredient.ErrDuplicateName.Error() {
		t.Errorf("the answer is the raw error of the domain: %s", answer)
	}
	if !strings.Contains(answer, ":") {
		t.Errorf("the answer says nothing about what to do: %s", answer)
	}
}

// TestTheDescriptionOfTheMatchToolSaysWhatItNeeds proves that the
// description of the tool is a contract with the agent: it says what is
// required, what the answer carries, and that it changes nothing.
func TestTheDescriptionOfTheMatchToolSaysWhatItNeeds(t *testing.T) {
	t.Parallel()

	session := connectWithTheCatalog(t, &stubItemMatcher{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolMatchItems)

	for _, want := range []string{
		"business is REQUIRED",
		"name is REQUIRED",
		"exact",
		"ambiguous",
		"unknown",
		"new",
		"changes nothing",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// stubIngredientsReader is the use case that reads the catalog of a
// business, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubIngredientsReader struct {
	items []ingredient.Ingredient
	err   error
	got   ports.ReadIngredients
	calls int
}

// Read answers the ingredients of the business, and remembers what the tool
// asked.
func (s *stubIngredientsReader) Read(_ context.Context, request ports.ReadIngredients) ([]ingredient.Ingredient, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return nil, s.err
	}

	return s.items, nil
}

// stubIngredientCompleter is the use case that completes an ingredient of
// the catalog, in memory.
//
// It answers what the test tells it to answer, and it remembers what the
// tool asked it.
type stubIngredientCompleter struct {
	completed ingredient.Ingredient
	err       error
	got       ports.CompleteIngredient
	calls     int
}

// Complete answers the completion of the test, and remembers what the tool
// asked.
func (s *stubIngredientCompleter) Complete(_ context.Context, request ports.CompleteIngredient) (ingredient.Ingredient, error) {
	s.calls++
	s.got = request

	if s.err != nil {
		return ingredient.Ingredient{}, s.err
	}

	return s.completed, nil
}

// connectWithTheCatalogOfTheBusiness starts the transport with the stubs
// of the catalog: the one that matches a name, the one that reads it and
// the one that completes an ingredient.
func connectWithTheCatalogOfTheBusiness(
	t *testing.T,
	matcher ports.ItemMatcher,
	reader ports.IngredientsReader,
	completer ports.IngredientCompleter,
) *sdk.ClientSession {
	t.Helper()

	transport := mcpserver.New(mcpserver.Options{
		ServiceName:         "paca-mcp-server",
		ServiceVersion:      "0.1.0",
		Instructions:        "The PACA MCP server.",
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBodyBytes: 1 << 20,
		MatchItems:          matcher,
		ReadIngredients:     reader,
		CompleteIngredient:  completer,
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

// theIngredientOfTheCatalog is an ingredient the tests of the tools use.
func theIngredientOfTheCatalog(t *testing.T) ingredient.Ingredient {
	t.Helper()

	built, err := ingredient.New(business.ID(77), "Grasa de cerdo", []string{"grasa"})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	built.ID = ingredient.ID(10)

	completed, err := ingredient.Complete(built, ingredient.Declaration{
		Unit:         ingredient.UnitKilogram,
		PackageGrams: 1000,
		Allergens:    []string{"gluten"},
		Certification: ingredient.Certification{
			Name:     "Orgánico",
			Evidence: "Certificado de la finca, 2026-02-01",
		},
	})
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}

	return completed
}

// TestTheReadIngredientsToolAnswersTheCatalog proves that the tool carries
// the business the agent names, and that the answer carries what is known
// of every ingredient and what is still missing of it.
func TestTheReadIngredientsToolAnswersTheCatalog(t *testing.T) {
	t.Parallel()

	reader := &stubIngredientsReader{items: []ingredient.Ingredient{theIngredientOfTheCatalog(t)}}
	session := connectWithTheCatalogOfTheBusiness(t, nil, reader, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadIngredients,
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
		"Grasa de cerdo", "kg", "gluten", "Orgánico",
		"Certificado de la finca, 2026-02-01", "grasa", "grams_per_unit",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheReadIngredientsToolAnswersWhatIsMissing proves that the answer
// carries what the ingredient still needs from its owner: the agent asks
// for exactly what she has not said yet.
func TestTheReadIngredientsToolAnswersWhatIsMissing(t *testing.T) {
	t.Parallel()

	built, err := ingredient.New(business.ID(77), "Harina panadera", nil)
	if err != nil {
		t.Fatalf("the ingredient was refused: %v", err)
	}
	built.ID = ingredient.ID(10)

	reader := &stubIngredientsReader{items: []ingredient.Ingredient{built}}
	session := connectWithTheCatalogOfTheBusiness(t, nil, reader, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadIngredients,
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

	answer := textOf(t, result)
	if !strings.Contains(answer, "the unit the ingredient is bought in") {
		t.Errorf("the answer does not name what is missing: %s", answer)
	}
	if strings.Contains(answer, "grams_per_unit") {
		t.Errorf("the answer carries the weight of a unit the book does not know: %s", answer)
	}
}

// TestTheReadIngredientsToolAnswersWithoutADatabase proves that a server
// without a database still exposes the tool, and that the call says so.
func TestTheReadIngredientsToolAnswersWithoutADatabase(t *testing.T) {
	t.Parallel()

	session := connectWithTheCatalogOfTheBusiness(t, nil, nil, nil)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolReadIngredients,
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

// TestTheCompleteIngredientToolWritesWhatTheOwnerDeclared proves that the
// tool carries what the owner declared, and that the answer carries the
// ingredient as it is now.
func TestTheCompleteIngredientToolWritesWhatTheOwnerDeclared(t *testing.T) {
	t.Parallel()

	completer := &stubIngredientCompleter{completed: theIngredientOfTheCatalog(t)}
	session := connectWithTheCatalogOfTheBusiness(t, nil, nil, completer)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCompleteIngredient,
		Arguments: map[string]any{
			"business_id":            77,
			"ingredient_id":          10,
			"unit":                   "kg",
			"package_grams":          1000,
			"allergens":              []any{"gluten"},
			"certification":          "Orgánico",
			"certification_evidence": "Certificado de la finca, 2026-02-01",
			"aliases":                []any{"grasa de cerdo"},
			"conversion":             0.95,
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("the tool answered an error: %s", textOf(t, result))
	}

	if completer.calls != 1 {
		t.Errorf("the use case was called %d times, want 1", completer.calls)
	}
	if completer.got.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", completer.got.BusinessID)
	}
	if completer.got.IngredientID != 10 {
		t.Errorf("the ingredient is %d, want 10", completer.got.IngredientID)
	}
	if completer.got.Unit != "kg" {
		t.Errorf("the unit is %q, want %q", completer.got.Unit, "kg")
	}
	if completer.got.PackageGrams != 1000 {
		t.Errorf("the weight of the package is %v, want 1000", completer.got.PackageGrams)
	}
	if len(completer.got.Allergens) != 1 || completer.got.Allergens[0] != "gluten" {
		t.Errorf("the allergens are %v, want the one of the owner", completer.got.Allergens)
	}
	if completer.got.CertificationName != "Orgánico" {
		t.Errorf("the certification is %q, want %q", completer.got.CertificationName, "Orgánico")
	}
	if completer.got.CertificationEvidence != "Certificado de la finca, 2026-02-01" {
		t.Errorf("the evidence is %q, want the document of the owner", completer.got.CertificationEvidence)
	}
	if len(completer.got.Aliases) != 1 || completer.got.Aliases[0] != "grasa de cerdo" {
		t.Errorf("the aliases are %v, want the one of the owner", completer.got.Aliases)
	}
	if completer.got.Conversion != 0.95 {
		t.Errorf("the conversion is %v, want 0.95", completer.got.Conversion)
	}

	answer := textOf(t, result)
	for _, want := range []string{"Grasa de cerdo", "kg", "Orgánico"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer does not carry %q: %s", want, answer)
		}
	}
}

// TestTheCompleteIngredientToolExplainsTheFailureToTheAgent proves that
// the failure of the use case reaches the agent with a message of its own.
func TestTheCompleteIngredientToolExplainsTheFailureToTheAgent(t *testing.T) {
	t.Parallel()

	completer := &stubIngredientCompleter{err: ingredient.ErrMissingEvidence}
	session := connectWithTheCatalogOfTheBusiness(t, nil, nil, completer)

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: mcpserver.ToolCompleteIngredient,
		Arguments: map[string]any{
			"business_id":   77,
			"ingredient_id": 10,
			"certification": "Orgánico",
		},
	})
	if err != nil {
		t.Fatalf("the tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("the tool answered no error, want the failure of the use case")
	}

	answer := textOf(t, result)
	if answer == ingredient.ErrMissingEvidence.Error() {
		t.Errorf("the answer is the raw error of the domain: %s", answer)
	}
	if !strings.Contains(answer, ":") {
		t.Errorf("the answer says nothing about what to do: %s", answer)
	}
}

// TestTheDescriptionOfTheReadIngredientsToolSaysWhatItAnswers proves that
// the description of the tool is a contract with the agent: it names the
// units of the book and what the answer carries.
func TestTheDescriptionOfTheReadIngredientsToolSaysWhatItAnswers(t *testing.T) {
	t.Parallel()

	session := connectWithTheCatalogOfTheBusiness(t, nil, &stubIngredientsReader{}, nil)

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolReadIngredients)

	for _, want := range []string{
		"business is REQUIRED",
		"'g', 'kg', 'paquete', 'unidad'",
		"allergens",
		"certification",
		"missing",
		"changes nothing",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheDescriptionOfTheCompleteIngredientToolSaysThatTheEvidenceIsRequired
// proves that the description of the tool tells the agent that a claim
// carries its evidence, and that the units of the book are the four.
func TestTheDescriptionOfTheCompleteIngredientToolSaysThatTheEvidenceIsRequired(t *testing.T) {
	t.Parallel()

	session := connectWithTheCatalogOfTheBusiness(t, nil, nil, &stubIngredientCompleter{})

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("the client could not list the tools: %v", err)
	}

	description := descriptionOf(t, tools.Tools, mcpserver.ToolCompleteIngredient)

	for _, want := range []string{
		"ingredient_id",
		"'g', 'kg', 'paquete', 'unidad'",
		"certification_evidence",
		"evidence is REQUIRED",
		"aliases",
		"conversion",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("the description does not say %q:\n%s", want, description)
		}
	}
}

// TestTheErrorsOfTheCatalogReachTheAgentWithAMessage proves that every
// error the catalog can return reaches the agent with a message of its own.
func TestTheErrorsOfTheCatalogReachTheAgentWithAMessage(t *testing.T) {
	t.Parallel()

	for _, failure := range ingredient.Errors() {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()

			matcher := &stubItemMatcher{err: failure}
			session := connectWithTheCatalog(t, matcher)

			result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name: mcpserver.ToolMatchItems,
				Arguments: map[string]any{
					"business_id": 77,
					"name":        "harina",
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
