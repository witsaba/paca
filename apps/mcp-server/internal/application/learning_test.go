package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/learning"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// fakeLearnings is the fake of the storage of the learning log.
//
// It remembers the entries it holds and the ones it wrote, and it answers
// what the test tells it to answer.
type fakeLearnings struct {
	held    []learning.Entry
	written []learning.Entry
	err     error
	listed  business.ID
}

// Create writes the entry and remembers it.
func (f *fakeLearnings) Create(_ context.Context, entry learning.Entry) (learning.Entry, error) {
	if f.err != nil {
		return learning.Entry{}, f.err
	}

	entry.ID = learning.ID(len(f.written) + 100)
	f.written = append(f.written, entry)

	return entry, nil
}

// List answers the entries of the business, and remembers that it was asked.
func (f *fakeLearnings) List(_ context.Context, own business.ID) ([]learning.Entry, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]learning.Entry, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// fakeLearningProductions is the fake of the reading of the productions.
//
// The entry of the log names the production it is about, and the use case
// reads the productions of the business to know which one it is.
type fakeLearningProductions struct {
	held []recipe.Production
	err  error
	list business.ID
}

// List answers the productions of the business.
func (f *fakeLearningProductions) List(_ context.Context, own business.ID) ([]recipe.Production, error) {
	f.list = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]recipe.Production, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// theBatchOfTheLearnings is the batch of the bakery the entries are about.
func theBatchOfTheLearnings() recipe.Recipe {
	return recipe.Recipe{
		ID:         recipe.ID(9),
		BusinessID: business.ID(77),
		Name:       "Pan de molde",
	}
}

// theRunOfTheLearnings is one run of that batch.
func theRunOfTheLearnings() recipe.Production {
	return recipe.Production{
		ID:         recipe.ProductionID(4),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(9),
		RecipeName: "Pan de molde",
	}
}

// theEntryOfTheLearnings is one entry the storage of the test holds.
func theEntryOfTheLearnings(t *testing.T, id learning.ID, learned string, answers learning.ID) learning.Entry {
	t.Helper()

	day, err := ledger.ParseDate("2026-03-14")
	if err != nil {
		t.Fatalf("the day of the entry could not be read: %v", err)
	}

	built, err := learning.New(business.ID(77), day, theBatchOfTheLearnings(), nil,
		"La masa subió lento", learned, nil)
	if err != nil {
		t.Fatalf("the entry of the test was refused: %v", err)
	}

	built.ID = id
	built.Answers = answers

	return built
}

// theLearningService wires the use case with the fakes of the test.
func theLearningService(
	learnings *fakeLearnings,
	recipes *fakeRecipes,
	productions *fakeLearningProductions,
	existing *business.Business,
) *application.LearningService {
	return application.NewLearningService(learnings, &fakeBusinesses{existing: existing}, recipes, productions)
}

// theBakeryOfTheLearnings is the business that keeps the log of the tests.
func theBakeryOfTheLearnings() business.Business {
	return business.Business{ID: business.ID(77), Kind: business.KindBusiness, Name: "La Panadería"}
}

// TestTheEntryIsWrittenWithTheBatchAndTheWordsOfTheOwner proves that the use
// case writes the entry of the log: the day, the batch of the business, what
// happened and what was learned, in the words of the owner.
//
// The entry NAMES the batch: the identity and the name of the recipe travel
// with the entry, and the use case never copies the recipe.
func TestTheEntryIsWrittenWithTheBatchAndTheWordsOfTheOwner(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theBatchOfTheLearnings()}}
	service := theLearningService(learnings, recipes, &fakeLearningProductions{}, &bakery)

	written, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID: 77,
		Day:        "2026-03-14",
		RecipeID:   9,
		Happened:   "  La masa subió más lento que la receta dice  ",
		Learned:    "  Leudar dos horas más en invierno  ",
	})
	if err != nil {
		t.Fatalf("the entry was refused: %v", err)
	}

	if recipes.found != recipe.ID(9) {
		t.Errorf("the batch read is %d, want 9", recipes.found)
	}
	if written.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", written.BusinessID)
	}
	if written.Day.String() != "2026-03-14" {
		t.Errorf("the day is %s, want 2026-03-14", written.Day)
	}
	if written.RecipeID != recipe.ID(9) {
		t.Errorf("the batch is %d, want 9", written.RecipeID)
	}
	if written.RecipeName != "Pan de molde" {
		t.Errorf("the name of the batch is %q, want %q", written.RecipeName, "Pan de molde")
	}
	if written.ProductionID != 0 {
		t.Errorf("the production is %d, want none: the entry is about the batch alone", written.ProductionID)
	}
	if written.Happened != "La masa subió más lento que la receta dice" {
		t.Errorf("what happened is %q, want the words of the owner", written.Happened)
	}
	if written.Learned != "Leudar dos horas más en invierno" {
		t.Errorf("what was learned is %q, want the words of the owner", written.Learned)
	}
	if len(learnings.written) != 1 {
		t.Fatalf("the storage wrote %d entries, want 1", len(learnings.written))
	}
}

// TestTheEntryAboutOneRunOfTheBatchCarriesTheProduction proves that the use
// case reads the production the agent names and that the entry carries it:
// the entry is about one run of the batch, and the run is of the business.
func TestTheEntryAboutOneRunOfTheBatchCarriesTheProduction(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theBatchOfTheLearnings()}}
	productions := &fakeLearningProductions{held: []recipe.Production{theRunOfTheLearnings()}}
	service := theLearningService(learnings, recipes, productions, &bakery)

	written, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID:   77,
		Day:          "2026-03-14",
		RecipeID:     9,
		ProductionID: 4,
		Happened:     "El horno corrió caliente",
		Learned:      "Bajar el horno a 200 grados",
	})
	if err != nil {
		t.Fatalf("the entry was refused: %v", err)
	}

	if productions.list != business.ID(77) {
		t.Errorf("the productions were read for the business %d, want 77", productions.list)
	}
	if written.ProductionID != recipe.ProductionID(4) {
		t.Errorf("the production is %d, want 4", written.ProductionID)
	}
	if written.RecipeID != recipe.ID(9) {
		t.Errorf("the batch is %d, want 9", written.RecipeID)
	}
}

// TestTheCorrectionAnswersTheEntryItNames proves that a correction is a NEW
// entry that answers the old one: the log is never rewritten, and the entry
// that was answered keeps its own words.
func TestTheCorrectionAnswersTheEntryItNames(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{held: []learning.Entry{
		theEntryOfTheLearnings(t, learning.ID(3), "Leudar una hora más", 0),
	}}
	recipes := &fakeRecipes{held: []recipe.Recipe{theBatchOfTheLearnings()}}
	service := theLearningService(learnings, recipes, &fakeLearningProductions{}, &bakery)

	written, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID: 77,
		Day:        "2026-03-15",
		RecipeID:   9,
		Happened:   "La masa quedó dura",
		Learned:    "Amasar diez minutos más",
		AnswersID:  3,
	})
	if err != nil {
		t.Fatalf("the correction was refused: %v", err)
	}

	if learnings.listed != business.ID(77) {
		t.Errorf("the log was read for the business %d, want 77", learnings.listed)
	}
	if written.Answers != learning.ID(3) {
		t.Errorf("the correction answers %d, want 3", written.Answers)
	}
	if written.ID == learning.ID(3) {
		t.Errorf("the correction carries the identity %d of the entry it answers", written.ID)
	}
	if learnings.held[0].Learned != "Leudar una hora más" {
		t.Errorf("the entry that was answered says %q, want its own words: the log is never rewritten",
			learnings.held[0].Learned)
	}
}

// TestTheAnswerOfAnEntryThatIsNotThereIsRefused proves that the domain
// decides which entries may be answered: the log refuses the answer of an
// entry it does not hold, and nothing is written.
func TestTheAnswerOfAnEntryThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theBatchOfTheLearnings()}}
	service := theLearningService(learnings, recipes, &fakeLearningProductions{}, &bakery)

	_, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID: 77,
		Day:        "2026-03-15",
		RecipeID:   9,
		Happened:   "La masa quedó dura",
		Learned:    "Amasar más",
		AnswersID:  3,
	})
	if !errors.Is(err, learning.ErrEntryNotFound) {
		t.Fatalf("the answer of an entry that is not there answered %v, want %v", err, learning.ErrEntryNotFound)
	}
	if len(learnings.written) != 0 {
		t.Fatalf("the refused answer wrote %d entries, want none", len(learnings.written))
	}
}

// TestTheSecondAnswerOfAnEntryIsRefused proves that the log is a chain and
// not a tree: the entry that was answered once is never answered again.
func TestTheSecondAnswerOfAnEntryIsRefused(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{held: []learning.Entry{
		theEntryOfTheLearnings(t, learning.ID(3), "Leudar una hora más", 0),
		theEntryOfTheLearnings(t, learning.ID(4), "Leudar dos horas más", learning.ID(3)),
	}}
	recipes := &fakeRecipes{held: []recipe.Recipe{theBatchOfTheLearnings()}}
	service := theLearningService(learnings, recipes, &fakeLearningProductions{}, &bakery)

	_, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID: 77,
		Day:        "2026-03-16",
		RecipeID:   9,
		Happened:   "La masa subió lento otra vez",
		Learned:    "Leudar tres horas más",
		AnswersID:  3,
	})
	if !errors.Is(err, learning.ErrAlreadyAnswered) {
		t.Fatalf("the second answer answered %v, want %v", err, learning.ErrAlreadyAnswered)
	}
}

// TestTheEntryOfNoDayIsRefused proves that the domain decides whether the
// entry carries a day: the use case passes what the agent sent, and the
// domain refuses the entry of no day.
func TestTheEntryOfNoDayIsRefused(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	recipes := &fakeRecipes{held: []recipe.Recipe{theBatchOfTheLearnings()}}
	learnings := &fakeLearnings{}
	service := theLearningService(learnings, recipes, &fakeLearningProductions{}, &bakery)

	_, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID: 77,
		RecipeID:   9,
		Happened:   "La masa subió lento",
		Learned:    "Leudar una hora más",
	})
	if !errors.Is(err, learning.ErrMissingDay) {
		t.Fatalf("the entry of no day answered %v, want %v", err, learning.ErrMissingDay)
	}
	if len(learnings.written) != 0 {
		t.Fatalf("the refused entry wrote %d entries, want none", len(learnings.written))
	}
}

// TestTheEntryOfAMalformedDayIsRefusedAsADayOfTheBook proves that the use
// case tells a day that is NOT a day of the book apart from no day at all: a
// malformed day is refused with ledger.ErrInvalidDate, and not with the
// refusal of the entry that carries no day.
//
// The rule of the day is the rule of the book and the use case holds no copy
// of it: it reads the day the agent sent, and the parse of a day that is not
// a day is a refusal the agent can see.
func TestTheEntryOfAMalformedDayIsRefusedAsADayOfTheBook(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theBatchOfTheLearnings()}}
	service := theLearningService(learnings, recipes, &fakeLearningProductions{}, &bakery)

	for _, day := range []string{"2026-02-30", "14/03/2026", "the day of the market"} {
		_, err := service.Record(t.Context(), ports.RecordLearning{BusinessID: 77, Day: day, RecipeID: 9,
			Happened: "La masa subió lento", Learned: "Leudar una hora más"})
		if !errors.Is(err, ledger.ErrInvalidDate) {
			t.Errorf("the day %q answered %v, want %v", day, err, ledger.ErrInvalidDate)
		}
	}

	if len(learnings.written) != 0 {
		t.Fatalf("the refused entries wrote %d rows, want none", len(learnings.written))
	}
}

// TestTheEntryAboutNoBatchIsRefused proves that the domain decides whether
// the entry names a batch: the use case asks for no recipe when the agent
// names none, and the domain refuses it.
func TestTheEntryAboutNoBatchIsRefused(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	recipes := &fakeRecipes{}
	service := theLearningService(&fakeLearnings{}, recipes, &fakeLearningProductions{}, &bakery)

	_, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID: 77,
		Day:        "2026-03-14",
		Happened:   "La masa subió lento",
		Learned:    "Leudar una hora más",
	})
	if !errors.Is(err, learning.ErrMissingRecipe) {
		t.Fatalf("the entry about no batch answered %v, want %v", err, learning.ErrMissingRecipe)
	}
	if recipes.found != 0 {
		t.Errorf("the recipes were read for the identity %d, want none: the agent named no batch", recipes.found)
	}
}

// TestTheEntryAboutTheBatchOfAnotherBusinessIsRefused proves that the domain
// decides whose batch the entry is about: the recipe of another business is
// refused, and nothing is written.
func TestTheEntryAboutTheBatchOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	foreign := theBatchOfTheLearnings()
	foreign.BusinessID = business.ID(88)

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{}
	recipes := &fakeRecipes{held: []recipe.Recipe{foreign}}
	service := theLearningService(learnings, recipes, &fakeLearningProductions{}, &bakery)

	_, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID: 77,
		Day:        "2026-03-14",
		RecipeID:   9,
		Happened:   "La masa subió lento",
		Learned:    "Leudar una hora más",
	})
	if !errors.Is(err, learning.ErrRecipeOfAnotherBusiness) {
		t.Fatalf("the entry about the batch of another business answered %v, want %v",
			err, learning.ErrRecipeOfAnotherBusiness)
	}
	if len(learnings.written) != 0 {
		t.Fatalf("the refused entry wrote %d entries, want none", len(learnings.written))
	}
}

// TestTheEntryAboutAProductionThatIsNotOfTheBusinessIsRefused proves that the
// entry about a production the business does not hold is refused: the
// production the log names is a production of that business.
func TestTheEntryAboutAProductionThatIsNotOfTheBusinessIsRefused(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theBatchOfTheLearnings()}}
	productions := &fakeLearningProductions{held: []recipe.Production{theRunOfTheLearnings()}}
	service := theLearningService(learnings, recipes, productions, &bakery)

	_, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID:   77,
		Day:          "2026-03-14",
		RecipeID:     9,
		ProductionID: 5,
		Happened:     "La masa subió lento",
		Learned:      "Leudar una hora más",
	})
	if !errors.Is(err, learning.ErrProductionOfAnotherBusiness) {
		t.Fatalf("the entry about a production the business does not hold answered %v, want %v",
			err, learning.ErrProductionOfAnotherBusiness)
	}
	if len(learnings.written) != 0 {
		t.Fatalf("the refused entry wrote %d entries, want none", len(learnings.written))
	}
}

// TestTheEntryAboutAProductionOfAnotherBatchIsRefused proves that the
// production the entry names is a run of the NAMED batch, and the domain is
// the one that says it: the book never carries two truths for the same fact.
func TestTheEntryAboutAProductionOfAnotherBatchIsRefused(t *testing.T) {
	t.Parallel()

	anotherRun := theRunOfTheLearnings()
	anotherRun.RecipeID = recipe.ID(10)

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theBatchOfTheLearnings()}}
	productions := &fakeLearningProductions{held: []recipe.Production{anotherRun}}
	service := theLearningService(learnings, recipes, productions, &bakery)

	_, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID:   77,
		Day:          "2026-03-14",
		RecipeID:     9,
		ProductionID: 4,
		Happened:     "La masa subió lento",
		Learned:      "Leudar una hora más",
	})
	if !errors.Is(err, learning.ErrAnotherBatch) {
		t.Fatalf("the entry about a production of another batch answered %v, want %v", err, learning.ErrAnotherBatch)
	}
}

// TestTheLogOfABusinessThatIsNotThereIsRefused proves that the business of the
// log is read first, and that the agent is told about the log and not about
// the onboarding of a person.
func TestTheLogOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	service := theLearningService(&fakeLearnings{}, &fakeRecipes{}, &fakeLearningProductions{}, nil)

	_, err := service.Read(t.Context(), ports.ReadLearnings{BusinessID: 77})
	if !errors.Is(err, learning.ErrBusinessNotFound) {
		t.Fatalf("the log of a business that is not there answered %v, want %v",
			err, learning.ErrBusinessNotFound)
	}
}

// TestTheLogOfThePersonalBusinessIsRefused proves the rule of the domain: the
// log belongs to a business of its own, and the Personal Business of a person
// keeps the money of that person and not the lessons of the bakery.
func TestTheLogOfThePersonalBusinessIsRefused(t *testing.T) {
	t.Parallel()

	personal := business.Business{ID: business.ID(77), Kind: business.KindPersonal, Name: "Mi Negocio"}
	service := theLearningService(&fakeLearnings{}, &fakeRecipes{}, &fakeLearningProductions{}, &personal)

	_, err := service.Record(t.Context(), ports.RecordLearning{
		BusinessID: 77,
		Day:        "2026-03-14",
		RecipeID:   9,
		Happened:   "La masa subió lento",
		Learned:    "Leudar una hora más",
	})
	if !errors.Is(err, learning.ErrWrongBusiness) {
		t.Fatalf("the entry of the personal business answered %v, want %v", err, learning.ErrWrongBusiness)
	}
}

// TestTheReadingAnswersTheLogAndItsCurrentEntries proves that the reading of
// the log answers the entries of the business in the order they were written,
// and the ones that are the last word of their chain: the corrections of the
// past are in the log, and the last word is what Paca tells the owner.
func TestTheReadingAnswersTheLogAndItsCurrentEntries(t *testing.T) {
	t.Parallel()

	bakery := theBakeryOfTheLearnings()
	learnings := &fakeLearnings{held: []learning.Entry{
		theEntryOfTheLearnings(t, learning.ID(3), "Leudar una hora más", 0),
		theEntryOfTheLearnings(t, learning.ID(4), "Leudar dos horas más", learning.ID(3)),
		theEntryOfTheLearnings(t, learning.ID(5), "Bajar el horno a 200 grados", 0),
	}}
	service := theLearningService(learnings, &fakeRecipes{}, &fakeLearningProductions{}, &bakery)

	log, err := service.Read(t.Context(), ports.ReadLearnings{BusinessID: 77})
	if err != nil {
		t.Fatalf("the log was refused: %v", err)
	}

	if learnings.listed != business.ID(77) {
		t.Errorf("the log was read for the business %d, want 77", learnings.listed)
	}
	if len(log.Entries) != 3 {
		t.Fatalf("the log carries %d entries, want 3", len(log.Entries))
	}
	if log.Entries[1].Answers != learning.ID(3) {
		t.Errorf("the entry %d answers %d, want 3", log.Entries[1].ID, log.Entries[1].Answers)
	}

	current := log.Current()
	if len(current) != 2 {
		t.Fatalf("the log holds %d current entries, want 2: %+v", len(current), current)
	}
	if current[0].ID != learning.ID(4) {
		t.Errorf("the first current entry is %d, want 4: the last word of the chain", current[0].ID)
	}
	if current[1].ID != learning.ID(5) {
		t.Errorf("the second current entry is %d, want 5", current[1].ID)
	}
}

// TestTheLogWithoutStorageSaysSo proves that the use case with no database
// says it, and that it never panics on a port with no storage behind it.
func TestTheLogWithoutStorageSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewLearningService(nil, nil, nil, nil)

	if _, err := service.Record(t.Context(), ports.RecordLearning{BusinessID: 77}); !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the entry without a database answered %v, want %v", err, ports.ErrNoDatabase)
	}
	if _, err := service.Read(t.Context(), ports.ReadLearnings{BusinessID: 77}); !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the log without a database answered %v, want %v", err, ports.ErrNoDatabase)
	}
}
