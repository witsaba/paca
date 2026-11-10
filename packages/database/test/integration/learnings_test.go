//go:build integration

package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/learning"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/user"
)

// learningsRepository returns the repository of the learning log, bound to
// the database of the suite.
func learningsRepository(t *testing.T) *postgres.LearningRepository {
	t.Helper()

	repo, err := postgres.NewLearningRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the learning log could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheLearnings opens the business that keeps the log of the test.
func theBakeryOfTheLearnings(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Learnings")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner),
		businessOf(business.KindBusiness, "La Panadería De Las Lecciones"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theBookOfTheLearnings writes the stock, the recipe and the production the
// entries of the test are about.
func theBookOfTheLearnings(
	t *testing.T,
	ctx context.Context,
	bakery business.Business,
) (recipe.Recipe, recipe.Production) {
	t.Helper()

	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)

	production, err := productionsRepository(t).Record(ctx, theProductionOfTheTest(t, bakery, baked, layers, flour))
	if err != nil {
		t.Fatalf("the production could not be written: %v", err)
	}

	return baked, production
}

// theDayOfTheLearnings reads a day of the book of the test.
func theDayOfTheLearnings(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day of the entry could not be read: %v", err)
	}

	return parsed
}

// theEntryOfTheTest builds one entry of the log the tests write.
func theEntryOfTheTest(
	t *testing.T,
	bakery business.Business,
	baked recipe.Recipe,
	production *recipe.Production,
	happened, learned string,
	answers *learning.ID,
) learning.Entry {
	t.Helper()

	built, err := learning.New(bakery.ID, theDayOfTheLearnings(t, "2026-03-14"), baked, production,
		happened, learned, answers)
	if err != nil {
		t.Fatalf("the entry of the test was refused: %v", err)
	}

	return built
}

// theEntryOf reads one entry of the log by its identity.
func theEntryOf(t *testing.T, held []learning.Entry, id learning.ID) learning.Entry {
	t.Helper()

	for _, one := range held {
		if one.ID == id {
			return one
		}
	}

	t.Fatalf("the log holds no entry with the identity %d", id)

	return learning.Entry{}
}

// theSecondBatchOfTheLearnings writes a second recipe of the bakery, so that
// the tests can name a batch and a production of another one.
func theSecondBatchOfTheLearnings(
	t *testing.T,
	ctx context.Context,
	bakery business.Business,
	flour ingredient.ID,
) recipe.Recipe {
	t.Helper()

	built, err := recipe.New(bakery.ID, recipe.Definition{
		Name: "Pan de centeno",
		Ingredients: []recipe.Ingredient{
			{IngredientID: flour, Grams: 500},
		},
		Steps: []recipe.Step{
			{Instruction: "Hornear."},
		},
		Yield:         recipe.Yield{Units: 10, Grams: 90},
		ShelfLifeDays: 3,
		Utterance:     "El pan de centeno lleva 500 gramos de harina",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the second batch was refused: %v", err)
	}

	written, err := recipesRepository(t).Record(ctx, built)
	if err != nil {
		t.Fatalf("the second batch could not be written: %v", err)
	}

	return written
}

// theFingerprintOfTheLog answers the bytes of the whole log of a business:
// every column of every row, in the order the rows were written. A write that
// was refused leaves the fingerprint as it was, and that is how the tests
// prove that the log is never rewritten and that a refusal writes nothing.
func theFingerprintOfTheLog(t *testing.T, ctx context.Context, own business.ID) string {
	t.Helper()

	rows, err := theSuite.db.QueryContext(ctx, `
		SELECT id, business_id, day, recipe_id, coalesce(production_id, 0), happened, learned,
		       coalesce(answers_id, 0), created_at
		FROM learnings
		WHERE business_id = $1
		ORDER BY id`, int64(own))
	if err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}
	defer func() { _ = rows.Close() }()

	sum := sha256.New()

	for rows.Next() {
		var id, businessID, recipeID, productionID, answersID int64
		var day, createdAt time.Time
		var happened, learned string

		if err := rows.Scan(&id, &businessID, &day, &recipeID, &productionID, &happened, &learned,
			&answersID, &createdAt); err != nil {
			t.Fatalf("the row of the log could not be read: %v", err)
		}

		fmt.Fprintf(sum, "%d|%d|%s|%d|%d|%s|%s|%d|%s\n", id, businessID, day.Format("2006-01-02"),
			recipeID, productionID, happened, learned, answersID, createdAt.UTC().Format(time.RFC3339Nano))
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}

	return hex.EncodeToString(sum.Sum(nil))
}

// TestTheLearningIsKeptWithItsBatchAndItsWords proves the rules of the table
// against a real database: the entry is written with the day, the batch, the
// production it is about, what happened and what was learned, and it is read
// back as it was written.
func TestTheLearningIsKeptWithItsBatchAndItsWords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)
	bakery := theBakeryOfTheLearnings(t, ctx)
	baked, production := theBookOfTheLearnings(t, ctx, bakery)

	written, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, &production,
		"  La masa subió más lento que la receta dice  ", "  Leudar dos horas más en invierno  ", nil))
	if err != nil {
		t.Fatalf("the entry could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Fatal("the entry carries no identity: the storage generates it")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.Day.String() != "2026-03-14" {
		t.Errorf("the day is %s, want 2026-03-14", written.Day)
	}
	if written.RecipeID != baked.ID {
		t.Errorf("the batch is %d, want %d", written.RecipeID, baked.ID)
	}
	if written.RecipeName != baked.Name {
		t.Errorf("the name of the batch is %q, want %q", written.RecipeName, baked.Name)
	}
	if written.ProductionID != production.ID {
		t.Errorf("the production is %d, want %d", written.ProductionID, production.ID)
	}
	if written.Happened != "La masa subió más lento que la receta dice" {
		t.Errorf("what happened is %q, want the words of the owner", written.Happened)
	}
	if written.Learned != "Leudar dos horas más en invierno" {
		t.Errorf("what was learned is %q, want the words of the owner", written.Learned)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}

	read := theEntryOf(t, held, written.ID)
	if read.RecipeName != baked.Name {
		t.Errorf("the name of the batch read back is %q, want %q", read.RecipeName, baked.Name)
	}
	if read.ProductionID != production.ID {
		t.Errorf("the production read back is %d, want %d", read.ProductionID, production.ID)
	}
	if read.Learned != "Leudar dos horas más en invierno" {
		t.Errorf("what was learned, read back, is %q, want the words of the owner", read.Learned)
	}
}

// TestTheEntryAboutTheBatchAloneCarriesNoProduction proves the rule of the
// table: the entry that is about the batch and not about one run of it
// carries no production, and it is read back with none.
func TestTheEntryAboutTheBatchAloneCarriesNoProduction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)
	bakery := theBakeryOfTheLearnings(t, ctx)
	baked, _ := theBookOfTheLearnings(t, ctx, bakery)

	written, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"El horno corrió caliente", "Bajar el horno a 200 grados", nil))
	if err != nil {
		t.Fatalf("the entry could not be written: %v", err)
	}

	if written.ProductionID != 0 {
		t.Errorf("the production is %d, want none: the entry is about the batch alone", written.ProductionID)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}

	read := theEntryOf(t, held, written.ID)
	if read.ProductionID != 0 {
		t.Errorf("the production read back is %d, want none", read.ProductionID)
	}
}

// TestTheLogIsNeverRewritten proves the rule of the table: a correction is a
// NEW entry that answers the old one. The entry that was answered keeps its
// own words and its own identity, and the correction is a row of its own.
func TestTheLogIsNeverRewritten(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)
	bakery := theBakeryOfTheLearnings(t, ctx)
	baked, _ := theBookOfTheLearnings(t, ctx, bakery)

	first, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento", "Leudar una hora más", nil))
	if err != nil {
		t.Fatalf("the entry could not be written: %v", err)
	}

	answered := first.ID

	second, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento otra vez", "Leudar dos horas más", &answered))
	if err != nil {
		t.Fatalf("the correction could not be written: %v", err)
	}

	if second.ID == first.ID {
		t.Fatalf("the correction carries the identity %d of the entry it answers", second.ID)
	}
	if second.Answers != first.ID {
		t.Errorf("the correction answers %d, want %d", second.Answers, first.ID)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}
	if len(held) != 2 {
		t.Fatalf("the log holds %d entries, want 2: the correction is a row of its own", len(held))
	}

	read := theEntryOf(t, held, first.ID)
	if read.Learned != "Leudar una hora más" {
		t.Errorf("the entry that was answered says %q, want its own words", read.Learned)
	}
	if read.Answers != 0 {
		t.Errorf("the entry that was answered answers %d, want none", read.Answers)
	}

	current := learning.TheLog(held).Current()
	if len(current) != 1 || current[0].ID != second.ID {
		t.Errorf("the current entries are %+v, want the correction %d", current, second.ID)
	}
}

// TestTheTableRefusesTheAnswerOfTheEntryOfAnotherBusiness proves the rule of
// the table: the answer carries the business with it, and the composite
// foreign key refuses the answer of an entry of another business.
func TestTheTableRefusesTheAnswerOfTheEntryOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)

	other := theBakeryOfTheLearnings(t, ctx)
	otherBatch, _ := theBookOfTheLearnings(t, ctx, other)

	foreign, err := repo.Create(ctx, theEntryOfTheTest(t, other, otherBatch, nil,
		"La masa subió lento", "Leudar una hora más", nil))
	if err != nil {
		t.Fatalf("the entry of the other business could not be written: %v", err)
	}

	mine := theBakeryOfTheLearnings(t, ctx)
	myBatch, _ := theBookOfTheLearnings(t, ctx, mine)

	answered := foreign.ID

	written, err := repo.Create(ctx, theEntryOfTheTest(t, mine, myBatch, nil,
		"La masa quedó dura", "Amasar más", &answered))
	if !errors.Is(err, learning.ErrEntryOfAnotherBusiness) {
		t.Fatalf("the answer of the entry of another business answered %v (%+v), want %v",
			err, written, learning.ErrEntryOfAnotherBusiness)
	}

	held, err := repo.List(ctx, mine.ID)
	if err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Fatalf("the refused answer left %d entries behind, want none", len(held))
	}
}

// TestTheTableRefusesTheSecondAnswerOfTheSameEntry proves the rule of the
// table: the log is a chain and not a tree. The partial unique index refuses
// the second answer of the same entry, and the first one stays in the book.
func TestTheTableRefusesTheSecondAnswerOfTheSameEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)
	bakery := theBakeryOfTheLearnings(t, ctx)
	baked, _ := theBookOfTheLearnings(t, ctx, bakery)

	first, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento", "Leudar una hora más", nil))
	if err != nil {
		t.Fatalf("the entry could not be written: %v", err)
	}

	answered := first.ID

	if _, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento otra vez", "Leudar dos horas más", &answered)); err != nil {
		t.Fatalf("the correction could not be written: %v", err)
	}

	written, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento una tercera vez", "Leudar tres horas más", &answered))
	if !errors.Is(err, learning.ErrAlreadyAnswered) {
		t.Fatalf("the second answer of the same entry answered %v (%+v), want %v",
			err, written, learning.ErrAlreadyAnswered)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}
	if len(held) != 2 {
		t.Fatalf("the log holds %d entries, want 2: the refused answer is not a row", len(held))
	}
}

// TestTheTableRefusesTheProductionOfAnotherBatch proves the rule of the
// table: the production of the entry is a run of the NAMED batch. The
// production and the batch are both of the business, so the composite foreign
// key of the batch is the one that refuses the entry, and the test says which
// rule of the table refused it.
//
// The domain refuses the same entry first (learning.ErrAnotherBatch): the
// table holds the rule as its defense in depth, and a writer that never asked
// the domain is the one this test proves.
func TestTheTableRefusesTheProductionOfAnotherBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)
	bakery := theBakeryOfTheLearnings(t, ctx)

	flour, layers := theStockOfTheProductions(t, ctx, bakery)
	baked := theRecipeOfTheProductions(t, ctx, bakery, flour.ID)
	second := theSecondBatchOfTheLearnings(t, ctx, bakery, flour.ID)

	run, err := productionsRepository(t).Record(ctx, theProductionOfTheTest(t, bakery, second, layers, flour))
	if err != nil {
		t.Fatalf("the production of the second batch could not be written: %v", err)
	}

	written, err := repo.Create(ctx, theEntryOfACallerThatBypassesTheDomain(t, bakery.ID, baked, &run))
	if !errors.Is(err, learning.ErrAnotherBatch) {
		t.Fatalf("the entry naming the batch %d about the production %d of the batch %d answered %v (%+v), want %v",
			baked.ID, run.ID, run.RecipeID, err, written, learning.ErrAnotherBatch)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}
	if len(held) != 0 {
		t.Fatalf("the refused entry left %d entries behind, want none", len(held))
	}
}

// TestTheTwoIdenticalEntriesAreTwoEntries proves the documented rule of the
// identity: the identity of an entry is its own and never its text. Two
// entries that say the same thing on the same day are two entries, and the log
// never refuses them and never merges them.
func TestTheTwoIdenticalEntriesAreTwoEntries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)
	bakery := theBakeryOfTheLearnings(t, ctx)
	baked, _ := theBookOfTheLearnings(t, ctx, bakery)

	first, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento", "Leudar más", nil))
	if err != nil {
		t.Fatalf("the first entry could not be written: %v", err)
	}

	second, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento", "Leudar más", nil))
	if err != nil {
		t.Fatalf("the identical entry was refused: %v", err)
	}

	if first.ID == second.ID {
		t.Fatalf("the two entries carry the identity %d", first.ID)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}

	current := learning.TheLog(held).Current()
	if len(held) != 2 || len(current) != 2 {
		t.Fatalf("the log holds %d entries and %d current, want 2 and 2", len(held), len(current))
	}
}

// TestTheRefusedEntriesLeaveTheLogAsItWas proves the acceptance of the
// increment in its strongest form: the log is never rewritten, and a write
// that was REFUSED leaves the stored log byte for byte as it was. The
// fingerprint of the whole log of the business is taken after a written entry
// and a written correction, and the two refusals that follow it — the second
// answer of the same entry and the answer of an entry of another business —
// change nothing.
func TestTheRefusedEntriesLeaveTheLogAsItWas(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)

	other := theBakeryOfTheLearnings(t, ctx)
	otherBatch, _ := theBookOfTheLearnings(t, ctx, other)

	foreign, err := repo.Create(ctx, theEntryOfTheTest(t, other, otherBatch, nil,
		"El horno corrió caliente", "Bajar el horno", nil))
	if err != nil {
		t.Fatalf("the entry of the other business could not be written: %v", err)
	}

	bakery := theBakeryOfTheLearnings(t, ctx)
	baked, _ := theBookOfTheLearnings(t, ctx, bakery)

	first, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento", "Leudar una hora más", nil))
	if err != nil {
		t.Fatalf("the entry could not be written: %v", err)
	}

	answered := first.ID

	if _, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento otra vez", "Leudar dos horas más", &answered)); err != nil {
		t.Fatalf("the correction could not be written: %v", err)
	}

	before := theFingerprintOfTheLog(t, ctx, bakery.ID)
	beforeOther := theFingerprintOfTheLog(t, ctx, other.ID)

	written, err := repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa subió lento una tercera vez", "Leudar tres horas más", &answered))
	if !errors.Is(err, learning.ErrAlreadyAnswered) {
		t.Fatalf("the second answer of the same entry answered %v (%+v), want %v",
			err, written, learning.ErrAlreadyAnswered)
	}

	foreignAnswer := foreign.ID

	written, err = repo.Create(ctx, theEntryOfTheTest(t, bakery, baked, nil,
		"La masa quedó dura", "Amasar más", &foreignAnswer))
	if !errors.Is(err, learning.ErrEntryOfAnotherBusiness) {
		t.Fatalf("the answer of the entry of another business answered %v (%+v), want %v",
			err, written, learning.ErrEntryOfAnotherBusiness)
	}

	if after := theFingerprintOfTheLog(t, ctx, bakery.ID); after != before {
		t.Fatalf("the refused writes changed the log: %s -> %s", before, after)
	}

	if after := theFingerprintOfTheLog(t, ctx, other.ID); after != beforeOther {
		t.Fatalf("the refused writes changed the log of the other business: %s -> %s", beforeOther, after)
	}
}

// TestTheEntriesOfAnotherBusinessAreNotListed proves the rule of the table:
// the log of a business is read for that business, and the entries of
// another one are not in it.
func TestTheEntriesOfAnotherBusinessAreNotListed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)

	mine := theBakeryOfTheLearnings(t, ctx)
	myBatch, _ := theBookOfTheLearnings(t, ctx, mine)

	other := theBakeryOfTheLearnings(t, ctx)
	otherBatch, _ := theBookOfTheLearnings(t, ctx, other)

	if _, err := repo.Create(ctx, theEntryOfTheTest(t, mine, myBatch, nil,
		"La masa subió lento", "Leudar una hora más", nil)); err != nil {
		t.Fatalf("the entry of the business could not be written: %v", err)
	}

	if _, err := repo.Create(ctx, theEntryOfTheTest(t, other, otherBatch, nil,
		"El horno corrió caliente", "Bajar el horno", nil)); err != nil {
		t.Fatalf("the entry of the other business could not be written: %v", err)
	}

	held, err := repo.List(ctx, mine.ID)
	if err != nil {
		t.Fatalf("the log could not be read: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the log of the business holds %d entries, want 1: %+v", len(held), held)
	}
	if held[0].BusinessID != mine.ID {
		t.Errorf("the entry is of the business %d, want %d", held[0].BusinessID, mine.ID)
	}
}

// theEntryOfACallerThatBypassesTheDomain builds an entry straight from its
// fields, the way a writer that never asked the domain would: the tests of
// the defense in depth of the table write what the domain refuses, and the
// table is the only thing that can refuse it.
func theEntryOfACallerThatBypassesTheDomain(
	t *testing.T,
	own business.ID,
	batch recipe.Recipe,
	production *recipe.Production,
) learning.Entry {
	t.Helper()

	entry := learning.Entry{
		BusinessID: own,
		Day:        theDayOfTheLearnings(t, "2026-03-14"),
		RecipeID:   batch.ID,
		RecipeName: batch.Name,
		Happened:   "La masa subió lento",
		Learned:    "Leudar una hora más",
	}

	if production != nil {
		entry.ProductionID = production.ID
	}

	return entry
}

// TestTheTableRefusesTheBatchOfAnotherBusiness proves the rule of the table:
// the batch of the entry is a batch of the business that owns the log.
func TestTheTableRefusesTheBatchOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)

	other := theBakeryOfTheLearnings(t, ctx)
	otherBatch, _ := theBookOfTheLearnings(t, ctx, other)

	mine := theBakeryOfTheLearnings(t, ctx)

	written, err := repo.Create(ctx, theEntryOfACallerThatBypassesTheDomain(t, mine.ID, otherBatch, nil))
	if !errors.Is(err, learning.ErrRecipeOfAnotherBusiness) {
		t.Fatalf("the entry about the batch of another business answered %v (%+v), want %v",
			err, written, learning.ErrRecipeOfAnotherBusiness)
	}
}

// TestTheTableRefusesTheProductionOfAnotherBusiness proves the rule of the
// table: the production of the entry is a production of the business that
// owns the log.
//
// The production of the other business is a run of the batch of this one: a
// row only a writer that bypassed the domain could write. The composite
// foreign key of the business is then the only key that refuses the entry,
// so the test says which rule of the table refused it.
func TestTheTableRefusesTheProductionOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)

	other := theBakeryOfTheLearnings(t, ctx)
	_, otherRun := theBookOfTheLearnings(t, ctx, other)

	mine := theBakeryOfTheLearnings(t, ctx)
	myBatch, _ := theBookOfTheLearnings(t, ctx, mine)

	foreignRun := otherRun
	foreignRun.BusinessID = other.ID
	foreignRun.RecipeID = myBatch.ID

	recorded, err := productionsRepository(t).Record(ctx, foreignRun)
	if err != nil {
		t.Fatalf("the production of the other business could not be written: %v", err)
	}

	written, err := repo.Create(ctx, theEntryOfACallerThatBypassesTheDomain(t, mine.ID, myBatch, &recorded))
	if !errors.Is(err, learning.ErrProductionOfAnotherBusiness) {
		t.Fatalf("the entry about the production of another business answered %v (%+v), want %v",
			err, written, learning.ErrProductionOfAnotherBusiness)
	}
}

// TestTheLearningOfABusinessThatIsNotThereIsRefused proves the rule of the
// table: the entry belongs to a business that is there.
func TestTheLearningOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)
	bakery := theBakeryOfTheLearnings(t, ctx)
	baked, _ := theBookOfTheLearnings(t, ctx, bakery)

	entry := theEntryOfTheTest(t, bakery, baked, nil, "La masa subió lento", "Leudar una hora más", nil)
	entry.BusinessID = business.ID(999999999)

	written, err := repo.Create(ctx, entry)
	if !errors.Is(err, learning.ErrBusinessNotFound) {
		t.Fatalf("the entry of a business that is not there answered %v (%+v), want %v",
			err, written, learning.ErrBusinessNotFound)
	}
}

// TestTheTableRefusesTheEntryThatSaysNothing proves the rule of the table: an
// entry that says nothing happened or taught nothing is not an entry of the
// log. The domain refuses it first, and the table holds it as its defense in
// depth.
func TestTheTableRefusesTheEntryThatSaysNothing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := learningsRepository(t)
	bakery := theBakeryOfTheLearnings(t, ctx)
	baked, _ := theBookOfTheLearnings(t, ctx, bakery)

	said := theEntryOfTheTest(t, bakery, baked, nil, "La masa subió lento", "Leudar una hora más", nil)

	cases := []struct {
		name   string
		broken func(learning.Entry) learning.Entry
	}{
		{
			name:   "the entry that says nothing happened",
			broken: func(entry learning.Entry) learning.Entry { entry.Happened = "   "; return entry },
		},
		{
			name:   "the entry that taught nothing",
			broken: func(entry learning.Entry) learning.Entry { entry.Learned = "   "; return entry },
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			written, err := repo.Create(ctx, test.broken(said))
			if !errors.Is(err, learning.ErrInvalidValue) {
				t.Fatalf("the row answered %v (%+v), want %v", err, written, learning.ErrInvalidValue)
			}
		})
	}
}
