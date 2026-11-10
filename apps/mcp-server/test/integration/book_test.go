//go:build integration

// The proof of the whole slice, end to end.
//
// The test starts a PostgreSQL, applies the schema of the migrations, starts
// the server against that database, and walks the story of the money of a
// person through the protocol the agent speaks: the onboarding, the salary,
// the loan of a friend, the payment of the loan, and the closing of the
// month.
//
// The proof is not a heartbeat: a server that answers does not prove that
// the book of the money works. Every number the test asserts is a number the
// agent reads in the answer of a tool.
package integration_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"

	core "github.com/witsaba/paca/packages/database"
)

// The settings of the database the story runs against.
const (
	// defaultStoryPostgresImage is the PostgreSQL the story runs against. It is
	// the version the stack of the monorepository runs.
	defaultStoryPostgresImage = "postgres:18.6-bookworm"

	// storyPostgresPort is the port PostgreSQL listens on inside the
	// container.
	storyPostgresPort = "5432/tcp"

	// storyPostgresUser is the owner of the database of the container.
	storyPostgresUser = "paca"

	// storyPostgresPassword is the password of the owner of the database of
	// the container. The database lives inside the container of the test,
	// and it is thrown away with it.
	storyPostgresPassword = "paca"

	// storyPostgresDatabase is the name of the database of the container.
	storyPostgresDatabase = "paca"

	// storyPostgresAlias is the name the database carries on the network of
	// the test. It is the name the stack of the monorepository uses too.
	storyPostgresAlias = "postgres"
)

// The onboarding of the story: the person and the accounts of their book.
type storyOnboarding struct {
	DocumentID string `json:"document_id"`
	FullName   string `json:"full_name"`
	Status     string `json:"status"`
	BusinessID string `json:"business_id"`
	Accounts   []struct {
		ID   string `json:"id"`
		Kind string `json:"kind"`
		Name string `json:"name"`
	} `json:"accounts"`
}

// The debts of the story.
type storyDebts struct {
	BusinessID int64 `json:"business_id"`
	Debts      []struct {
		AccountID    int64  `json:"account_id"`
		Kind         string `json:"kind"`
		Name         string `json:"name"`
		Counterparty string `json:"counterparty"`
		DueDate      string `json:"due_date"`
		Amount       int64  `json:"amount"`
	} `json:"debts"`
}

// The closing of the month of the story.
type storySummary struct {
	BusinessID int64  `json:"business_id"`
	FromDate   string `json:"from_date"`
	ToDate     string `json:"to_date"`
	Income     int64  `json:"income"`
	Expenses   int64  `json:"expenses"`
	Net        int64  `json:"net"`
	Debts      []struct {
		AccountID    int64  `json:"account_id"`
		Counterparty string `json:"counterparty"`
		Amount       int64  `json:"amount"`
	} `json:"debts"`
}

// TestTheWholeStoryOfTheMoneyOfAPerson proves the slice end to end: the
// person arrives, Paca opens their book, the money moves, a friend lends
// money, the loan is paid, and the month is closed.
func TestTheWholeStoryOfTheMoneyOfAPerson(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	shared, err := network.New(ctx)
	if err != nil {
		t.Fatalf("the network of the test did not start: %v", err)
	}
	t.Cleanup(func() {
		if err := shared.Remove(context.Background()); err != nil {
			t.Logf("the network of the test did not go away: %v", err)
		}
	})

	database := startTheDatabaseOfTheStory(t, ctx, shared)
	defer func() {
		if err := database.Terminate(context.Background()); err != nil {
			t.Logf("the database container did not terminate: %v", err)
		}
	}()

	applyTheSchemaOfTheStory(t, ctx, database)
	address := startTheServerOfTheStory(t, ctx, shared)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: address + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the server: %v", err)
	}
	defer session.Close()

	// The person arrives: the onboarding opens the business and its
	// accounts.
	var onboarding storyOnboarding
	callTheTool(t, ctx, session, "complete_onboarding", map[string]any{
		"document_id": "1010101010",
		"full_name":   "Ana Rivera",
	}, &onboarding)

	if onboarding.Status != "active" {
		t.Fatalf("the person is %q, want %q", onboarding.Status, "active")
	}
	if onboarding.BusinessID == "" {
		t.Fatal("the onboarding opened no business")
	}

	cash := theAccountOfTheStory(t, onboarding, "Efectivo")
	salary := theAccountOfTheStory(t, onboarding, "Salario")

	// The onboarding answers with the identities of the book as text, and
	// the tools of the book take them as numbers: the agent reads one and
	// writes the other, and the proof walks the same way.
	businessID := theNumberOfTheStory(t, onboarding.BusinessID)

	// The salary lands in the cash and comes from the salary.
	callTheTool(t, ctx, session, "record_movement", map[string]any{
		"business_id":     businessID,
		"from_account_id": salary,
		"to_account_id":   cash,
		"amount":          1000000,
		"date":            "2026-03-01",
		"description":     "Salary of March",
	}, nil)

	// A friend lends the money: the debt is opened and the money lands in
	// the cash.
	var debt struct {
		Debt struct {
			AccountID    int64  `json:"account_id"`
			Counterparty string `json:"counterparty"`
			Amount       int64  `json:"amount"`
		} `json:"debt"`
	}
	callTheTool(t, ctx, session, "record_debt", map[string]any{
		"business_id":   businessID,
		"counterparty":  "Bruno",
		"amount":        500000,
		"to_account_id": cash,
		"date":          "2026-03-05",
		"due_date":      "2026-04-30",
		"description":   "Loan of Bruno",
	}, &debt)

	if debt.Debt.Counterparty != "Bruno" {
		t.Fatalf("the debt names %q, want %q", debt.Debt.Counterparty, "Bruno")
	}
	if debt.Debt.Amount != 500000 {
		t.Fatalf("the debt holds %d, want 500000", debt.Debt.Amount)
	}

	// The debt is paid in part: what remains is what the person owes.
	var payment struct {
		Debt struct {
			Amount int64 `json:"amount"`
		} `json:"debt"`
	}
	callTheTool(t, ctx, session, "record_debt_payment", map[string]any{
		"business_id":     businessID,
		"debt_account_id": debt.Debt.AccountID,
		"amount":          100000,
		"from_account_id": cash,
		"date":            "2026-03-10",
		"description":     "Payment to Bruno",
	}, &payment)

	if payment.Debt.Amount != 400000 {
		t.Fatalf("the debt holds %d, want 400000", payment.Debt.Amount)
	}

	// The debts of the book: what the person owes, to whom.
	var debts storyDebts
	callTheTool(t, ctx, session, "list_debts", map[string]any{
		"business_id": businessID,
	}, &debts)

	if len(debts.Debts) != 1 {
		t.Fatalf("the book holds %d debts, want 1", len(debts.Debts))
	}
	if debts.Debts[0].Amount != 400000 {
		t.Errorf("the debt holds %d, want 400000", debts.Debts[0].Amount)
	}
	if debts.Debts[0].DueDate != "2026-04-30" {
		t.Errorf("the debt falls due on %q, want %q", debts.Debts[0].DueDate, "2026-04-30")
	}

	// The closing of March: what the person earned, what they spent, and
	// what they still owe.
	var summary storySummary
	callTheTool(t, ctx, session, "read_month_summary", map[string]any{
		"business_id": businessID,
		"year":        2026,
		"month":       3,
	}, &summary)

	if summary.Income != 1000000 {
		t.Errorf("the income of March is %d, want 1000000", summary.Income)
	}
	if summary.Expenses != 0 {
		t.Errorf("the expenses of March are %d, want 0", summary.Expenses)
	}
	if summary.Net != 1000000 {
		t.Errorf("the net of March is %d, want 1000000", summary.Net)
	}
	if len(summary.Debts) != 1 || summary.Debts[0].Amount != 400000 {
		t.Errorf("the debts of March are %+v, want the 400000 that is owed to Bruno", summary.Debts)
	}
}

// TestTheWholeStoryOfTheCapacityOfTheWeekend proves the slice of the capacity
// end to end: the owner buys the flour, she writes the recipe of the bread,
// and the tool answers what the hours of her weekend carry of it.
//
// The proof is not the list of the tools: a tool that lists is a tool that the
// server can answer with the service of the capacity wired to nothing, and the
// owner would read that her book has no database for the rest of its days.
// The test calls the tool, and every number it asserts is a number the agent
// reads in the answer.
func TestTheWholeStoryOfTheCapacityOfTheWeekend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	shared, err := network.New(ctx)
	if err != nil {
		t.Fatalf("the network of the test did not start: %v", err)
	}
	t.Cleanup(func() {
		if err := shared.Remove(context.Background()); err != nil {
			t.Logf("the network of the test did not go away: %v", err)
		}
	})

	database := startTheDatabaseOfTheStory(t, ctx, shared)
	defer func() {
		if err := database.Terminate(context.Background()); err != nil {
			t.Logf("the database container did not terminate: %v", err)
		}
	}()

	applyTheSchemaOfTheStory(t, ctx, database)
	address := startTheServerOfTheStory(t, ctx, shared)

	client := sdk.NewClient(&sdk.Implementation{Name: "paca-mcp-server-test", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: address + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("the client could not connect to the server: %v", err)
	}
	defer session.Close()

	// The person arrives, and Paca opens the bakery: a business of its own,
	// with the book of accounts of a business.
	callTheTool(t, ctx, session, "complete_onboarding", map[string]any{
		"document_id": "1010101012",
		"full_name":   "Ana Rivera",
	}, nil)

	var bakery struct {
		Business struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"business"`
		Accounts []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
			Name string `json:"name"`
		} `json:"accounts"`
	}
	callTheTool(t, ctx, session, "open_business", map[string]any{
		"owner_id": "1010101012",
		"name":     "Panadería del barrio",
	}, &bakery)

	if bakery.Business.Kind != "business" {
		t.Fatalf("the bakery is a business of the kind %q, want %q", bakery.Business.Kind, "business")
	}

	businessID := theNumberOfTheStory(t, bakery.Business.ID)

	cash := int64(0)
	for _, one := range bakery.Accounts {
		if one.Name == "Efectivo" {
			cash = theNumberOfTheStory(t, one.ID)
		}
	}
	if cash == 0 {
		t.Fatal("the book of the bakery holds no cash to pay the flour with")
	}

	// The flour comes into the house: the draft the agent proposes, and
	// the confirmation of the owner. The name the catalog did not know
	// becomes the ingredient the recipe names.
	var draft struct {
		Draft struct {
			ID       int64 `json:"id"`
			Revision int   `json:"revision"`
		} `json:"draft"`
	}
	callTheTool(t, ctx, session, "draft_purchase", map[string]any{
		"business_id": businessID,
		"utterance":   "Compré un kilo de harina por 5000 pesos.",
		"lines": []map[string]any{
			{"name": "Harina", "quantity": 1, "unit": "kg", "total_paid": 5000},
		},
	}, &draft)

	callTheTool(t, ctx, session, "confirm_purchase", map[string]any{
		"business_id":     businessID,
		"draft_id":        draft.Draft.ID,
		"revision":        draft.Draft.Revision,
		"confirmed_with":  "sí",
		"from_account_id": cash,
	}, nil)

	var catalog struct {
		Ingredients []struct {
			ID int64 `json:"id"`
		} `json:"ingredients"`
	}
	callTheTool(t, ctx, session, "read_ingredients", map[string]any{
		"business_id": businessID,
	}, &catalog)

	if len(catalog.Ingredients) != 1 {
		t.Fatalf("the catalog holds %d ingredients, want the one the purchase wrote", len(catalog.Ingredients))
	}

	flour := catalog.Ingredients[0].ID

	// The recipe the owner confirmed: twenty minutes of her hands to
	// amass and fifteen to shape, forty minutes of the oven, two hours of
	// the waiting, and twelve units of what it makes.
	var baked struct {
		ID int64 `json:"id"`
	}
	callTheTool(t, ctx, session, "record_recipe", map[string]any{
		"business_id": businessID,
		"name":        "Pan de molde",
		"ingredients": []map[string]any{
			{"ingredient_id": flour, "grams": 500},
		},
		"steps": []map[string]any{
			{"instruction": "Amasar.", "minutes": 20, "labor_kind": "hands_on"},
			{"instruction": "Formar.", "minutes": 15, "labor_kind": "hands_on"},
			{"instruction": "Hornear.", "minutes": 40, "labor_kind": "oven"},
			{"instruction": "Reposar.", "minutes": 120, "labor_kind": "waiting"},
		},
		"yield_units":     12,
		"yield_grams":     90,
		"shelf_life_days": 3,
		"confirmed_with":  "sí",
	}, &baked)

	// The capacity of the weekend: the two hours of her hands carry three
	// batches of the bread, because the batch needs the thirty-five
	// minutes of her hands and not the forty of the oven nor the two hours
	// of the waiting.
	var capacity struct {
		Hours   float64 `json:"hours"`
		Recipes []struct {
			RecipeID int64 `json:"recipe_id"`
			Minutes  int   `json:"minutes"`
			Batches  int   `json:"batches"`
			Units    int   `json:"units"`
			Limited  bool  `json:"limited"`
		} `json:"recipes"`
	}
	callTheTool(t, ctx, session, "read_capacity", map[string]any{
		"business_id": businessID,
		"hours":       2,
	}, &capacity)

	if capacity.Hours != 2 {
		t.Errorf("the capacity carries %v hours, want the 2 the owner declared", capacity.Hours)
	}
	if len(capacity.Recipes) != 1 {
		t.Fatalf("the capacity carries %d recipes, want the one the book holds", len(capacity.Recipes))
	}

	one := capacity.Recipes[0]
	if one.RecipeID != baked.ID {
		t.Errorf("the capacity is of the recipe %d, want %d", one.RecipeID, baked.ID)
	}
	if one.Minutes != 35 {
		t.Errorf("the minutes of the batch are %d, want the 35 of the two steps of the hands", one.Minutes)
	}
	if one.Batches != 3 || one.Units != 36 {
		t.Errorf("the capacity carries %d batches and %d units, want 3 and 36", one.Batches, one.Units)
	}
	if !one.Limited {
		t.Error("the recipe of the hands of the owner is not limited by the hours")
	}
}

// startTheDatabaseOfTheStory starts the PostgreSQL of the story and answers
// the container that holds it.
func startTheDatabaseOfTheStory(t *testing.T, ctx context.Context, shared *testcontainers.DockerNetwork) testcontainers.Container {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: storyPostgresImage(),
			Env: map[string]string{
				"POSTGRES_USER":     storyPostgresUser,
				"POSTGRES_PASSWORD": storyPostgresPassword,
				"POSTGRES_DB":       storyPostgresDatabase,
				"TZ":                "UTC",
				"PGTZ":              "UTC",
			},
			ExposedPorts: []string{storyPostgresPort},
			WaitingFor: wait.ForListeningPort(storyPostgresPort).
				WithStartupTimeout(3 * time.Minute),
			Networks: []string{shared.Name},
			NetworkAliases: map[string][]string{
				shared.Name: {storyPostgresAlias},
			},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("the database of the story did not start: %v", err)
	}

	return container
}

// applyTheSchemaOfTheStory applies the migrations of the schema to the
// database of the story.
//
// The migrations are applied by the library that owns them, the same way the
// writer of the schema applies them: the service does not migrate on boot.
func applyTheSchemaOfTheStory(t *testing.T, ctx context.Context, container testcontainers.Container) {
	t.Helper()

	port, err := container.MappedPort(ctx, storyPostgresPort)
	if err != nil {
		t.Fatalf("the port of the database could not be read: %v", err)
	}

	url := fmt.Sprintf("postgres://%s:%s@127.0.0.1:%s/%s?sslmode=disable",
		storyPostgresUser, storyPostgresPassword, port.Port(), storyPostgresDatabase)

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("the database of the story could not be opened: %v", err)
	}
	defer db.Close()

	if err := waitForTheDatabaseOfTheStory(ctx, db); err != nil {
		t.Fatalf("the database of the story did not answer: %v", err)
	}

	migrator, err := core.NewMigrator(db, slog.Default())
	if err != nil {
		t.Fatalf("the migrator of the schema could not be built: %v", err)
	}
	if _, err := migrator.Up(ctx); err != nil {
		t.Fatalf("the schema could not be applied: %v", err)
	}
}

// storyPostgresImage returns the image of the database the story runs
// against.
//
// The image can be changed with PACA_TEST_POSTGRES_IMAGE, for a machine that
// keeps another PostgreSQL around.
func storyPostgresImage() string {
	if image := os.Getenv("PACA_TEST_POSTGRES_IMAGE"); image != "" {
		return image
	}

	return defaultStoryPostgresImage
}

// waitForTheDatabaseOfTheStory asks the database whether it answers, and it
// asks again while the server is still starting up.
//
// The container opens the port of PostgreSQL before the server accepts the
// connections, and the first question of the story can arrive in that
// window: the database answers with the code 57P03, the server is starting
// up. The suite waits for the answer instead of failing on it.
func waitForTheDatabaseOfTheStory(ctx context.Context, db *sql.DB) error {
	deadline := time.Now().Add(2 * time.Minute)

	for {
		err := db.PingContext(ctx)
		if err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// startTheServerOfTheStory starts the server against the database of the
// story and answers the address the client reaches it at.
func startTheServerOfTheStory(t *testing.T, ctx context.Context, shared *testcontainers.DockerNetwork) string {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        imageName + ":" + imageTag,
			ExposedPorts: []string{containerPort + "/tcp"},
			Env: map[string]string{
				"PACA_MCP_HOST":                 "0.0.0.0",
				"PACA_MCP_ALLOW_NON_LOOPBACK":   "true",
				"PACA_MCP_PORT":                 containerPort,
				"PACA_DATABASE_URL":             "postgres://" + storyPostgresUser + ":" + storyPostgresPassword + "@" + storyPostgresAlias + ":5432/" + storyPostgresDatabase + "?sslmode=disable",
				"PACA_DATABASE_CONNECT_TIMEOUT": "30s",
			},
			WaitingFor: wait.ForHTTP("/healthz").
				WithPort(containerPort + "/tcp").
				WithStatusCodeMatcher(func(status int) bool {
					return status == 200
				}).
				WithStartupTimeout(3 * time.Minute),
			Networks: []string{shared.Name},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("the server container did not start: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("the server container did not terminate: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("the host of the server container could not be read: %v", err)
	}
	port, err := container.MappedPort(ctx, containerPort)
	if err != nil {
		t.Fatalf("the port of the server container could not be read: %v", err)
	}

	return fmt.Sprintf("http://%s:%s", host, port.Port())
}

// callTheTool calls a tool of the server and reads the answer of the tool.
//
// The answer of a tool is JSON, and the test reads it the way the agent
// reads it: the fields the agent acts on.
func callTheTool(t *testing.T, ctx context.Context, session *sdk.ClientSession, name string, arguments map[string]any, answer any) {
	t.Helper()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("the tool %q failed: %v", name, err)
	}
	if result.IsError {
		t.Fatalf("the tool %q answered an error: %s", name, theTextOfTheStory(result))
	}

	if answer == nil {
		return
	}

	text := theTextOfTheStory(result)
	if err := json.Unmarshal([]byte(text), answer); err != nil {
		t.Fatalf("the answer of the tool %q is not JSON: %v: %s", name, err, text)
	}
}

// theTextOfTheStory returns the text of the answer of a tool.
func theTextOfTheStory(result *sdk.CallToolResult) string {
	text := ""
	for _, content := range result.Content {
		if textContent, ok := content.(*sdk.TextContent); ok {
			text += textContent.Text
		}
	}

	return text
}

// theAccountOfTheStory returns the account of the book of the person that
// carries that name.
//
// The answer of the onboarding carries the identities as text, and the
// tools of the book take them as numbers: the account is translated here,
// the way the agent translates it.
func theAccountOfTheStory(t *testing.T, onboarding storyOnboarding, name string) int64 {
	t.Helper()

	for _, one := range onboarding.Accounts {
		if one.Name == name {
			return theNumberOfTheStory(t, one.ID)
		}
	}

	t.Fatalf("the book of the person holds no account named %q", name)

	return 0
}

// theNumberOfTheStory reads the identity the onboarding answered with.
func theNumberOfTheStory(t *testing.T, identity string) int64 {
	t.Helper()

	number, err := strconv.ParseInt(identity, 10, 64)
	if err != nil {
		t.Fatalf("the identity %q is not a number: %v", identity, err)
	}

	return number
}
