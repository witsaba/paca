package learning

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theDayOfTheTest is a day of the book of the bakery.
func theDayOfTheTest(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day %q could not be read: %v", day, err)
	}

	return parsed
}

// theBatchOfTheTest is the batch the entries of the tests are about: the
// recipe of the bakery, with the name the owner reads.
func theBatchOfTheTest() recipe.Recipe {
	return recipe.Recipe{
		ID:         recipe.ID(9),
		BusinessID: business.ID(77),
		Name:       "Pan de molde",
	}
}

// theProductionOfTheTest is one run of the batch.
func theProductionOfTheTest() recipe.Production {
	return recipe.Production{
		ID:         recipe.ProductionID(4),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(9),
		RecipeName: "Pan de molde",
	}
}

// TestTheEntryCarriesWhatHappenedAndWhatWasLearned proves that the domain
// builds the entry of the log: the day, the batch it is about, what happened
// and what was learned.
//
// The entry names the batch and never copies it: the identity and the name
// of the recipe travel with the entry, and the recipe stays the owner of
// them.
func TestTheEntryCarriesWhatHappenedAndWhatWasLearned(t *testing.T) {
	t.Parallel()

	built, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-14"), theBatchOfTheTest(), nil,
		"  La masa subió más lento que la receta dice  ", "  Leudar dos horas más en invierno  ", nil)
	if err != nil {
		t.Fatalf("the entry was refused: %v", err)
	}

	if built.ID != 0 {
		t.Errorf("the identity is %d, want 0: the storage generates it", built.ID)
	}
	if built.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", built.BusinessID)
	}
	if built.Day.String() != "2026-03-14" {
		t.Errorf("the day is %s, want 2026-03-14", built.Day)
	}
	if built.RecipeID != recipe.ID(9) {
		t.Errorf("the batch is %d, want 9", built.RecipeID)
	}
	if built.RecipeName != "Pan de molde" {
		t.Errorf("the name of the batch is %q, want %q", built.RecipeName, "Pan de molde")
	}
	if built.ProductionID != 0 {
		t.Errorf("the production is %d, want 0: the entry is about the batch alone", built.ProductionID)
	}
	if built.Happened != "La masa subió más lento que la receta dice" {
		t.Errorf("what happened is %q, want the words of the owner", built.Happened)
	}
	if built.Learned != "Leudar dos horas más en invierno" {
		t.Errorf("what was learned is %q, want the words of the owner", built.Learned)
	}
	if built.Answers != 0 {
		t.Errorf("the entry answers %d, want none", built.Answers)
	}
}

// TestTheEntryThatCarriesAProductionNamesOneRunOfTheBatch proves that the
// entry about one run of the batch carries the production, and that the
// production is a run of the NAMED batch.
func TestTheEntryThatCarriesAProductionNamesOneRunOfTheBatch(t *testing.T) {
	t.Parallel()

	run := theProductionOfTheTest()

	built, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-14"), theBatchOfTheTest(), &run,
		"El horno corrió caliente", "Bajar el horno a 200 grados", nil)
	if err != nil {
		t.Fatalf("the entry was refused: %v", err)
	}

	if built.ProductionID != recipe.ProductionID(4) {
		t.Errorf("the production is %d, want 4", built.ProductionID)
	}
	if built.RecipeID != recipe.ID(9) {
		t.Errorf("the batch is %d, want 9", built.RecipeID)
	}
}

// TestTheEntryThatAnswersAnEntryCarriesTheEntryItAnswers proves that a
// correction is a NEW entry that answers the old one: the log is never
// rewritten.
func TestTheEntryThatAnswersAnEntryCarriesTheEntryItAnswers(t *testing.T) {
	t.Parallel()

	answered := ID(3)

	built, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-15"), theBatchOfTheTest(), nil,
		"La masa quedó dura", "Amasar diez minutos más", &answered)
	if err != nil {
		t.Fatalf("the entry was refused: %v", err)
	}

	if built.Answers != ID(3) {
		t.Errorf("the entry answers %d, want 3", built.Answers)
	}
}

// TestTheEntryOfNoBusinessIsRefused proves the rule of the domain: the log
// belongs to a business.
func TestTheEntryOfNoBusinessIsRefused(t *testing.T) {
	t.Parallel()

	_, err := New(0, theDayOfTheTest(t, "2026-03-14"), theBatchOfTheTest(), nil,
		"La masa subió lento", "Leudar más", nil)
	if !errors.Is(err, ErrMissingBusiness) {
		t.Fatalf("the entry of no business answered %v, want %v", err, ErrMissingBusiness)
	}
}

// TestTheEntryOfNoDayIsRefused proves the rule of the domain: the entry
// carries the day it is about.
func TestTheEntryOfNoDayIsRefused(t *testing.T) {
	t.Parallel()

	_, err := New(business.ID(77), ledger.Date{}, theBatchOfTheTest(), nil,
		"La masa subió lento", "Leudar más", nil)
	if !errors.Is(err, ErrMissingDay) {
		t.Fatalf("the entry of no day answered %v, want %v", err, ErrMissingDay)
	}
}

// TestTheEntryAboutNoBatchIsRefused proves the rule of the domain: the entry
// is about a batch, and a batch that is not there is not a batch.
func TestTheEntryAboutNoBatchIsRefused(t *testing.T) {
	t.Parallel()

	_, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-14"), recipe.Recipe{}, nil,
		"La masa subió lento", "Leudar más", nil)
	if !errors.Is(err, ErrMissingRecipe) {
		t.Fatalf("the entry about no batch answered %v, want %v", err, ErrMissingRecipe)
	}
}

// TestTheEntryAboutTheBatchOfAnotherBusinessIsRefused proves the rule of the
// domain: the batch of the entry is a batch of the business that owns the
// entry.
func TestTheEntryAboutTheBatchOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	foreign := theBatchOfTheTest()
	foreign.BusinessID = business.ID(88)

	_, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-14"), foreign, nil,
		"La masa subió lento", "Leudar más", nil)
	if !errors.Is(err, ErrRecipeOfAnotherBusiness) {
		t.Fatalf("the entry about a foreign batch answered %v, want %v", err, ErrRecipeOfAnotherBusiness)
	}
}

// TestTheEntryAboutTheProductionOfAnotherBusinessIsRefused proves the rule of
// the domain: the production of the entry is a production of the business
// that owns the entry.
func TestTheEntryAboutTheProductionOfAnotherBusinessIsRefused(t *testing.T) {
	t.Parallel()

	foreign := theProductionOfTheTest()
	foreign.BusinessID = business.ID(88)

	_, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-14"), theBatchOfTheTest(), &foreign,
		"La masa subió lento", "Leudar más", nil)
	if !errors.Is(err, ErrProductionOfAnotherBusiness) {
		t.Fatalf("the entry about a foreign production answered %v, want %v", err, ErrProductionOfAnotherBusiness)
	}
}

// TestTheEntryAboutTheProductionOfAnotherBatchIsRefused proves the rule of
// the domain: the production of the entry is a run of the NAMED batch, and
// the book never carries two truths for the same fact.
func TestTheEntryAboutTheProductionOfAnotherBatchIsRefused(t *testing.T) {
	t.Parallel()

	anotherRun := theProductionOfTheTest()
	anotherRun.RecipeID = recipe.ID(10)

	_, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-14"), theBatchOfTheTest(), &anotherRun,
		"La masa subió lento", "Leudar más", nil)
	if !errors.Is(err, ErrAnotherBatch) {
		t.Fatalf("the entry about the production of another batch answered %v, want %v", err, ErrAnotherBatch)
	}
}

// TestTheEntryThatSaysNothingHappenedIsRefused proves the rule of the domain:
// the entry that says nothing is not an entry of the log.
func TestTheEntryThatSaysNothingHappenedIsRefused(t *testing.T) {
	t.Parallel()

	for _, happened := range []string{"", "   ", "\t\n"} {
		_, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-14"), theBatchOfTheTest(), nil,
			happened, "Leudar más", nil)
		if !errors.Is(err, ErrMissingHappened) {
			t.Fatalf("the entry that says %q happened answered %v, want %v", happened, err, ErrMissingHappened)
		}
	}
}

// TestTheEntryThatTaughtNothingIsRefused proves the rule of the domain: the
// log is what PACA learns from, and an entry that taught nothing taught
// nothing.
func TestTheEntryThatTaughtNothingIsRefused(t *testing.T) {
	t.Parallel()

	for _, learned := range []string{"", "   ", "\t\n"} {
		_, err := New(business.ID(77), theDayOfTheTest(t, "2026-03-14"), theBatchOfTheTest(), nil,
			"La masa subió lento", learned, nil)
		if !errors.Is(err, ErrMissingLearned) {
			t.Fatalf("the entry that learned %q answered %v, want %v", learned, err, ErrMissingLearned)
		}
	}
}

// TestTheEntryIsReadBackThroughItsOwnConstructor proves that the row is read
// back through the own constructor of the domain: the entry the storage
// holds is the entry the domain builds, and a row another writer broke is
// refused.
func TestTheEntryIsReadBackThroughItsOwnConstructor(t *testing.T) {
	t.Parallel()

	built, err := NewFromFacts(Facts{
		ID:           ID(3),
		BusinessID:   business.ID(77),
		Day:          theDayOfTheTest(t, "2026-03-14"),
		RecipeID:     recipe.ID(9),
		RecipeName:   "Pan de molde",
		ProductionID: recipe.ProductionID(4),
		Happened:     "La masa subió lento",
		Learned:      "Leudar más",
		Answers:      ID(2),
	})
	if err != nil {
		t.Fatalf("the entry read back was refused: %v", err)
	}

	if built.ID != ID(3) {
		t.Errorf("the identity is %d, want 3", built.ID)
	}
	if built.ProductionID != recipe.ProductionID(4) {
		t.Errorf("the production is %d, want 4", built.ProductionID)
	}
	if built.Answers != ID(2) {
		t.Errorf("the entry answers %d, want 2", built.Answers)
	}

	broken := []struct {
		name    string
		facts   Facts
		failure error
	}{
		{
			name: "the row of no business",
			facts: Facts{
				ID: ID(3), Day: theDayOfTheTest(t, "2026-03-14"), RecipeID: recipe.ID(9),
				Happened: "La masa subió lento", Learned: "Leudar más",
			},
			failure: ErrMissingBusiness,
		},
		{
			name: "the row of no day",
			facts: Facts{
				ID: ID(3), BusinessID: business.ID(77), RecipeID: recipe.ID(9),
				Happened: "La masa subió lento", Learned: "Leudar más",
			},
			failure: ErrMissingDay,
		},
		{
			name: "the row about no batch",
			facts: Facts{
				ID: ID(3), BusinessID: business.ID(77), Day: theDayOfTheTest(t, "2026-03-14"),
				Happened: "La masa subió lento", Learned: "Leudar más",
			},
			failure: ErrMissingRecipe,
		},
		{
			name: "the row that says nothing happened",
			facts: Facts{
				ID: ID(3), BusinessID: business.ID(77), Day: theDayOfTheTest(t, "2026-03-14"), RecipeID: recipe.ID(9),
				Learned: "Leudar más",
			},
			failure: ErrMissingHappened,
		},
		{
			name: "the row that taught nothing",
			facts: Facts{
				ID: ID(3), BusinessID: business.ID(77), Day: theDayOfTheTest(t, "2026-03-14"), RecipeID: recipe.ID(9),
				Happened: "La masa subió lento",
			},
			failure: ErrMissingLearned,
		},
	}

	for _, test := range broken {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewFromFacts(test.facts); !errors.Is(err, test.failure) {
				t.Fatalf("the row answered %v, want %v", err, test.failure)
			}
		})
	}
}

// TestTheLogBelongsToABusinessOfItsOwn proves the rule of the domain: the
// log belongs to the business that sells the bread, and the Personal
// Business of a person is not one.
func TestTheLogBelongsToABusinessOfItsOwn(t *testing.T) {
	t.Parallel()

	if err := OwnedBy(business.KindBusiness); err != nil {
		t.Fatalf("the log of a business of its own was refused: %v", err)
	}

	for _, kind := range []business.Kind{business.KindPersonal, business.Kind("unknown")} {
		if err := OwnedBy(kind); !errors.Is(err, ErrWrongBusiness) {
			t.Errorf("the log of a business of the kind %q answered %v, want %v", kind, err, ErrWrongBusiness)
		}
	}
}

// TestTheErrorsOfTheLearningAreCatalogued proves that the catalogue of the
// domain names every error of the log, once each: a caller proves that it
// maps every one of them through this list.
func TestTheErrorsOfTheLearningAreCatalogued(t *testing.T) {
	t.Parallel()

	catalogue := Errors()
	if len(catalogue) == 0 {
		t.Fatal("the catalogue of the learning is empty")
	}

	seen := make(map[error]bool, len(catalogue))
	for _, failure := range catalogue {
		if failure == nil {
			t.Fatalf("the catalogue holds a nil error: %v", catalogue)
		}
		if seen[failure] {
			t.Errorf("the catalogue names %v twice", failure)
		}
		seen[failure] = true
	}
}
