package recipe_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The day of the production of the test.
func theDayOfTheExit() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC))
}

// The production of the test: ten units of bread, worth ten thousand pesos.
func theProductionOfTheExit() recipe.Production {
	units := make([]recipe.ProductionUnit, 0, 10)

	for position := 1; position <= 10; position++ {
		units = append(units, recipe.ProductionUnit{
			Position:  position,
			Grams:     90,
			ExpiresOn: ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)),
		})
	}

	return recipe.Production{
		ID:         recipe.ProductionID(3),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(9),
		RecipeName: "Pan de molde",
		Day:        theDayOfTheExit(),
		GramsIn:    900,
		Money:      10000,
		Units:      units,
	}
}

// The units of the production of the test that are still in the book.
func theUnitsOfTheExit() []recipe.ExpiringUnit {
	units := make([]recipe.ExpiringUnit, 0, 10)

	for position := 1; position <= 10; position++ {
		units = append(units, recipe.ExpiringUnit{
			ProductionUnit: recipe.ProductionUnit{
				ID:           recipe.UnitID(int64(position)),
				ProductionID: recipe.ProductionID(3),
				Position:     position,
				Grams:        90,
				ExpiresOn:    ledger.NewDate(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)),
			},
			RecipeID:   recipe.ID(9),
			RecipeName: "Pan de molde",
			ProducedOn: theDayOfTheExit(),
		})
	}

	return units
}

// The definition of the consumption of the test: three units the owner ate.
func theDefinitionOfTheExit() recipe.ExitDefinition {
	return recipe.ExitDefinition{
		RecipeID:      recipe.ID(9),
		RecipeName:    "Pan de molde",
		Kind:          recipe.ExitConsumption,
		Quantity:      3,
		Unit:          "unidad",
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
		Utterance:     "me comí tres panes",
		ConfirmedWith: "sí",
	}
}

// TestTheExitCarriesTheProductTheQuantityTheUnitAndTheDay proves the rule
// of the domain: the exit carries what the owner said: the product, the
// quantity, the unit, the kind of the exit and the day it happened.
func TestTheExitCarriesTheProductTheQuantityTheUnitAndTheDay(t *testing.T) {
	t.Parallel()

	exit, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		theDefinitionOfTheExit(),
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if exit.Kind != recipe.ExitConsumption {
		t.Errorf("the kind of the exit is %q, want %q", exit.Kind, recipe.ExitConsumption)
	}

	if exit.RecipeID != recipe.ID(9) {
		t.Errorf("the product of the exit is %d, want %d", exit.RecipeID, recipe.ID(9))
	}

	if exit.RecipeName != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", exit.RecipeName, "Pan de molde")
	}

	if exit.Quantity != 3 {
		t.Errorf("the quantity of the exit is %v, want 3", exit.Quantity)
	}

	if exit.Unit != "unidad" {
		t.Errorf("the unit of the exit is %q, want %q", exit.Unit, "unidad")
	}

	if exit.Day.String() != "2026-03-14" {
		t.Errorf("the day of the exit is %q, want %q", exit.Day.String(), "2026-03-14")
	}

	if exit.ConfirmedWith != "sí" {
		t.Errorf("the word of the owner is %q, want %q", exit.ConfirmedWith, "sí")
	}

	if exit.Utterance != "me comí tres panes" {
		t.Errorf("the sentence of the owner is %q, want the one she said", exit.Utterance)
	}
}

// TestTheExitReleasesTheCostOfTheUnitsThatLeft proves the rule of the
// domain: the cost of the exit is the cost of the units that left the
// book, and it is derived, never typed.
func TestTheExitReleasesTheCostOfTheUnitsThatLeft(t *testing.T) {
	t.Parallel()

	exit, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		theDefinitionOfTheExit(),
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	// The production cost ten thousand pesos for ten units: a unit is
	// worth a thousand pesos, and three units are worth three thousand.
	if exit.Cost != 3000 {
		t.Errorf("the cost of the exit is %d, want %d", exit.Cost, 3000)
	}

	if len(exit.Units) != 3 {
		t.Fatalf("the exit took %d units, want 3", len(exit.Units))
	}
}

// TestTheExitTakesTheUnitsThatExpireFirst proves the rule of the
// rotation: the units that leave the book are the ones that expire first.
func TestTheExitTakesTheUnitsThatExpireFirst(t *testing.T) {
	t.Parallel()

	units := theUnitsOfTheExit()

	// The first unit expires before the others: it is the one that leaves
	// the book first.
	early := units[0]
	early.ExpiresOn = ledger.NewDate(time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC))
	units[0] = early

	exit, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		units,
		theDefinitionOfTheExit(),
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if exit.Units[0].UnitID != units[0].ID {
		t.Errorf("the exit took the unit %d, want the one that expires first (%d)",
			exit.Units[0].UnitID, units[0].ID)
	}
}

// TestTheExitOfAnExpiredUnitIsAllowed proves the rule of the domain: only
// the sale refuses the unit that expired. The bread that expired is still
// bread for the owner and for the one she gives it to, and the book never
// refuses a movement the owner declared.
func TestTheExitOfAnExpiredUnitIsAllowed(t *testing.T) {
	t.Parallel()

	units := theUnitsOfTheExit()

	expired := units[0]
	expired.ExpiresOn = ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC))
	units[0] = expired

	exit, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		units,
		theDefinitionOfTheExit(),
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if len(exit.Units) != 3 {
		t.Fatalf("the exit took %d units, want 3", len(exit.Units))
	}
}

// TestTheConsumptionWaitsForTheProfessionalOfTheTaxes proves the rule of
// the domain: the tax treatment of the consumption is pending the
// professional of the taxes, and the exit names the question.
func TestTheConsumptionWaitsForTheProfessionalOfTheTaxes(t *testing.T) {
	t.Parallel()

	exit, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		theDefinitionOfTheExit(),
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if exit.TaxTreatment != recipe.TaxPendingTheProfessional {
		t.Errorf("the tax treatment of the exit is %q, want %q",
			exit.TaxTreatment, recipe.TaxPendingTheProfessional)
	}

	if exit.TaxQuestion == "" {
		t.Fatal("the exit names no question about the taxes")
	}

	if !containsTheQuestionOfTheTaxes(exit.TaxQuestion) {
		t.Errorf("the question of the exit does not name the professional of the taxes: %q",
			exit.TaxQuestion)
	}
}

// TestTheDonationWaitsForTheProfessionalOfTheTaxes proves the rule of the
// domain: the book decides no tax treatment of its own, and the exit
// carries the question instead of a silent default.
func TestTheDonationWaitsForTheProfessionalOfTheTaxes(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.Kind = recipe.ExitDonation

	exit, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if exit.TaxTreatment != recipe.TaxPendingTheProfessional {
		t.Errorf("the tax treatment of the exit is %q, want %q",
			exit.TaxTreatment, recipe.TaxPendingTheProfessional)
	}

	if exit.TaxQuestion == "" {
		t.Fatal("the exit names no question about the taxes")
	}
}

// TestTheDonationIsASocialExpenseOfTheBusiness proves the rule of the
// domain: the donation lands in the account of the social expense of the
// business, and the consumption releases the cost like a sale does.
func TestTheDonationIsASocialExpenseOfTheBusiness(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.Kind = recipe.ExitDonation

	donation, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if donation.TheExpenseRole() != account.RoleDonation {
		t.Errorf("the cost of the donation lands in the account of the role %q, want %q",
			donation.TheExpenseRole(), account.RoleDonation)
	}

	consumption, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		theDefinitionOfTheExit(),
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if consumption.TheExpenseRole() != account.RoleCostOfSales {
		t.Errorf("the cost of the consumption lands in the account of the role %q, want %q",
			consumption.TheExpenseRole(), account.RoleCostOfSales)
	}
}

// TestTheCostOfTheExitLeavesTheInventory proves the rule of the domain:
// the cost of the units that left the book leaves the inventory and lands
// in the account of the expense of the exit.
func TestTheCostOfTheExitLeavesTheInventory(t *testing.T) {
	t.Parallel()

	exit, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		theDefinitionOfTheExit(),
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	movement, err := exit.CostMovement(account.ID(5), account.ID(3))
	if err != nil {
		t.Fatalf("the movement of the cost was refused: %v", err)
	}

	if movement.DebitAccountID != account.ID(5) {
		t.Errorf("the cost lands in the account %d, want 5", movement.DebitAccountID)
	}

	if movement.CreditAccountID != account.ID(3) {
		t.Errorf("the cost comes from the account %d, want the inventory (3)",
			movement.CreditAccountID)
	}

	if movement.Amount != 3000 {
		t.Errorf("the movement carries %d pesos, want 3000", movement.Amount)
	}

	// The sentence of the owner is what she reads in the book.
	if movement.Description != "me comí tres panes" {
		t.Errorf("the description of the movement is %q, want the sentence of the owner",
			movement.Description)
	}
}

// TestTheDescriptionOfTheExitIsTheWordsOfTheBook proves the rule of the
// domain: the book reads its own words when the owner said no sentence.
func TestTheDescriptionOfTheExitIsTheWordsOfTheBook(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.Utterance = ""

	exit, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if exit.Description() != "Consumo: 3 unidades de Pan de molde" {
		t.Errorf("the description of the exit is %q, want the words of the book",
			exit.Description())
	}
}

// TestTheExitIsRefusedWithoutABusiness proves the rule of the domain: the
// exit belongs to the business that owns it.
func TestTheExitIsRefusedWithoutABusiness(t *testing.T) {
	t.Parallel()

	_, err := recipe.ExitOf(
		business.ID(0),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		theDefinitionOfTheExit(),
	)
	if !errors.Is(err, recipe.ErrMissingBusiness) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingBusiness)
	}
}

// TestTheExitOfAnotherKindIsRefused proves the rule of the domain: the
// exits of the book are the consumption of the owner and the donation, and
// no other kind is guessed.
func TestTheExitOfAnotherKindIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.Kind = recipe.ExitKind("sale")

	_, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if !errors.Is(err, recipe.ErrUnknownExitKind) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownExitKind)
	}
}

// TestTheExitOfNoWholeNumberOfUnitsIsRefused proves the rule of the
// domain: the exits of the book count the units they took, and the book
// never splits a unit.
func TestTheExitOfNoWholeNumberOfUnitsIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.Quantity = 2.5

	_, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if !errors.Is(err, recipe.ErrInvalidExitQuantity) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidExitQuantity)
	}
}

// TestTheExitOfAnotherUnitIsRefused proves the rule of the domain: the
// book never converts a weight into a number of units.
func TestTheExitOfAnotherUnitIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.Unit = "gramos"

	_, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if !errors.Is(err, recipe.ErrInvalidExitUnit) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidExitUnit)
	}
}

// TestTheExitOfNoDayIsRefused proves the rule of the domain: the exit
// carries the day it happened.
func TestTheExitOfNoDayIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.Day = ledger.Date{}

	_, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if !errors.Is(err, recipe.ErrMissingExitDay) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingExitDay)
	}
}

// TestTheExitWithoutTheWordOfTheOwnerIsRefused proves the rule of the
// domain: the exit is written with the confirmation of the owner.
func TestTheExitWithoutTheWordOfTheOwnerIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.ConfirmedWith = ""

	_, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if !errors.Is(err, recipe.ErrMissingExitConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingExitConfirmation)
	}
}

// TestTheExitOfUnitsTheBookDoesNotHoldIsRefused proves the rule of the
// domain: the exit is of what the bakery still has.
func TestTheExitOfUnitsTheBookDoesNotHoldIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.Quantity = 30

	_, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if !errors.Is(err, recipe.ErrNotEnoughUnitsToExit) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotEnoughUnitsToExit)
	}
}

// TestTheExitOfAProductThatIsNotThereIsRefused proves the rule of the
// domain: the exit names a product of the business.
func TestTheExitOfAProductThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.RecipeID = 0
	definition.RecipeName = "El pan que no existe"

	_, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if !errors.Is(err, recipe.ErrUnknownRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownRecipe)
	}
}

// TestTheExitWithoutTheTaxTreatmentIsRefused proves the rule of the
// domain: an exit carries the tax treatment it waits for, and a row that
// carries none is never read as an exit.
func TestTheExitWithoutTheTaxTreatmentIsRefused(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewExit(business.ID(77), recipe.ExitFact{
		RecipeID:      recipe.ID(9),
		RecipeName:    "Pan de molde",
		Kind:          recipe.ExitConsumption,
		Day:           theDayOfTheExit(),
		Quantity:      3,
		Unit:          "unidad",
		Cost:          3000,
		TaxTreatment:  "",
		TaxQuestion:   "la pregunta",
		ConfirmedWith: "sí",
		Units: []recipe.ExitedUnit{
			{UnitID: recipe.UnitID(1), ProductionID: recipe.ProductionID(3), Position: 1, Grams: 90},
		},
	})
	if !errors.Is(err, recipe.ErrMissingTaxTreatment) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingTaxTreatment)
	}
}

// TestTheDescriptionOfTheDonationIsTheWordsOfTheBook proves the rule of
// the domain: the book reads its own words for the donation too.
func TestTheDescriptionOfTheDonationIsTheWordsOfTheBook(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheExit()
	definition.Kind = recipe.ExitDonation
	definition.Utterance = ""

	exit, err := recipe.ExitOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheExit()},
		theUnitsOfTheExit(),
		definition,
	)
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if exit.Description() != "Donación: 3 unidades de Pan de molde" {
		t.Errorf("the description of the exit is %q, want the words of the book",
			exit.Description())
	}
}

// TestTheKindsOfTheExitAreTheTwoOfTheBook proves the rule of the domain:
// the exits of the bakery are the consumption of the owner and the
// donation, and the caller receives a copy of the list.
func TestTheKindsOfTheExitAreTheTwoOfTheBook(t *testing.T) {
	t.Parallel()

	kinds := recipe.AllExitKinds()
	if len(kinds) != 2 {
		t.Fatalf("the book holds %d kinds of the exit, want 2", len(kinds))
	}

	if kinds[0] != recipe.ExitConsumption || kinds[1] != recipe.ExitDonation {
		t.Errorf("the kinds of the exit are %v, want the consumption and the donation", kinds)
	}

	kinds[0] = recipe.ExitKind("changed")

	if recipe.AllExitKinds()[0] != recipe.ExitConsumption {
		t.Error("the list of the kinds changed under the caller")
	}
}

// TestTheBrokenRowOfTheExitIsRefused proves the rule of the domain: a row
// that another writer broke is refused, and it is never read as an exit.
func TestTheBrokenRowOfTheExitIsRefused(t *testing.T) {
	t.Parallel()

	whole := recipe.ExitFact{
		RecipeID:      recipe.ID(9),
		RecipeName:    "Pan de molde",
		Kind:          recipe.ExitConsumption,
		Day:           theDayOfTheExit(),
		Quantity:      3,
		Unit:          "unidad",
		Cost:          3000,
		TaxTreatment:  recipe.TaxPendingTheProfessional,
		TaxQuestion:   "la pregunta de los impuestos",
		ConfirmedWith: "sí",
		Units: []recipe.ExitedUnit{
			{UnitID: recipe.UnitID(1), ProductionID: recipe.ProductionID(3), Position: 1, Grams: 90},
		},
	}

	for name, failure := range map[string]struct {
		fact recipe.ExitFact
		own  business.ID
		want error
	}{
		"the business is not there": {
			fact: whole, own: business.ID(0), want: recipe.ErrMissingBusiness,
		},
		"the kind is not one of the two": {
			fact: recipe.ExitFact{
				RecipeID: whole.RecipeID, RecipeName: whole.RecipeName,
				Kind: recipe.ExitKind("sale"), Day: whole.Day, Quantity: whole.Quantity,
				Unit: whole.Unit, Cost: whole.Cost, TaxTreatment: whole.TaxTreatment,
				TaxQuestion: whole.TaxQuestion, ConfirmedWith: whole.ConfirmedWith,
				Units: whole.Units,
			},
			own: business.ID(77), want: recipe.ErrUnknownExitKind,
		},
		"the product is not there": {
			fact: recipe.ExitFact{
				RecipeID: 0, RecipeName: whole.RecipeName, Kind: whole.Kind, Day: whole.Day,
				Quantity: whole.Quantity, Unit: whole.Unit, Cost: whole.Cost,
				TaxTreatment: whole.TaxTreatment, TaxQuestion: whole.TaxQuestion,
				ConfirmedWith: whole.ConfirmedWith, Units: whole.Units,
			},
			own: business.ID(77), want: recipe.ErrMissingRecipe,
		},
		"the day is not there": {
			fact: recipe.ExitFact{
				RecipeID: whole.RecipeID, RecipeName: whole.RecipeName, Kind: whole.Kind,
				Quantity: whole.Quantity, Unit: whole.Unit, Cost: whole.Cost,
				TaxTreatment: whole.TaxTreatment, TaxQuestion: whole.TaxQuestion,
				ConfirmedWith: whole.ConfirmedWith, Units: whole.Units,
			},
			own: business.ID(77), want: recipe.ErrMissingExitDay,
		},
		"the quantity is not a whole number of units": {
			fact: recipe.ExitFact{
				RecipeID: whole.RecipeID, RecipeName: whole.RecipeName, Kind: whole.Kind,
				Day: whole.Day, Quantity: 2.5, Unit: whole.Unit, Cost: whole.Cost,
				TaxTreatment: whole.TaxTreatment, TaxQuestion: whole.TaxQuestion,
				ConfirmedWith: whole.ConfirmedWith, Units: whole.Units,
			},
			own: business.ID(77), want: recipe.ErrInvalidExitQuantity,
		},
		"the unit is another one": {
			fact: recipe.ExitFact{
				RecipeID: whole.RecipeID, RecipeName: whole.RecipeName, Kind: whole.Kind,
				Day: whole.Day, Quantity: whole.Quantity, Unit: "gramos", Cost: whole.Cost,
				TaxTreatment: whole.TaxTreatment, TaxQuestion: whole.TaxQuestion,
				ConfirmedWith: whole.ConfirmedWith, Units: whole.Units,
			},
			own: business.ID(77), want: recipe.ErrInvalidExitUnit,
		},
		"the cost is negative": {
			fact: recipe.ExitFact{
				RecipeID: whole.RecipeID, RecipeName: whole.RecipeName, Kind: whole.Kind,
				Day: whole.Day, Quantity: whole.Quantity, Unit: whole.Unit, Cost: -1,
				TaxTreatment: whole.TaxTreatment, TaxQuestion: whole.TaxQuestion,
				ConfirmedWith: whole.ConfirmedWith, Units: whole.Units,
			},
			own: business.ID(77), want: recipe.ErrInvalidExitCost,
		},
		"the tax treatment is not there": {
			fact: recipe.ExitFact{
				RecipeID: whole.RecipeID, RecipeName: whole.RecipeName, Kind: whole.Kind,
				Day: whole.Day, Quantity: whole.Quantity, Unit: whole.Unit, Cost: whole.Cost,
				TaxTreatment: "", TaxQuestion: whole.TaxQuestion,
				ConfirmedWith: whole.ConfirmedWith, Units: whole.Units,
			},
			own: business.ID(77), want: recipe.ErrMissingTaxTreatment,
		},
		"the question of the taxes is not there": {
			fact: recipe.ExitFact{
				RecipeID: whole.RecipeID, RecipeName: whole.RecipeName, Kind: whole.Kind,
				Day: whole.Day, Quantity: whole.Quantity, Unit: whole.Unit, Cost: whole.Cost,
				TaxTreatment: whole.TaxTreatment, TaxQuestion: "",
				ConfirmedWith: whole.ConfirmedWith, Units: whole.Units,
			},
			own: business.ID(77), want: recipe.ErrMissingTaxTreatment,
		},
		"the exit carries no unit": {
			fact: recipe.ExitFact{
				RecipeID: whole.RecipeID, RecipeName: whole.RecipeName, Kind: whole.Kind,
				Day: whole.Day, Quantity: whole.Quantity, Unit: whole.Unit, Cost: whole.Cost,
				TaxTreatment: whole.TaxTreatment, TaxQuestion: whole.TaxQuestion,
				ConfirmedWith: whole.ConfirmedWith,
			},
			own: business.ID(77), want: recipe.ErrNotEnoughUnitsToExit,
		},
		"the word of the owner is not there": {
			fact: recipe.ExitFact{
				RecipeID: whole.RecipeID, RecipeName: whole.RecipeName, Kind: whole.Kind,
				Day: whole.Day, Quantity: whole.Quantity, Unit: whole.Unit, Cost: whole.Cost,
				TaxTreatment: whole.TaxTreatment, TaxQuestion: whole.TaxQuestion,
				Units: whole.Units,
			},
			own: business.ID(77), want: recipe.ErrMissingExitConfirmation,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := recipe.NewExit(failure.own, failure.fact)
			if !errors.Is(err, failure.want) {
				t.Fatalf("the error is %v, want %v", err, failure.want)
			}
		})
	}
}

// TestTheNameOfTheExitIsReadTheWayAPersonReadsIt proves the rule of the
// domain: the kind of the exit is read the way the owner says it, and the
// answer is the kind the book knows.
func TestTheNameOfTheExitIsReadTheWayAPersonReadsIt(t *testing.T) {
	t.Parallel()

	for spoken, want := range map[string]recipe.ExitKind{
		"consumption": recipe.ExitConsumption,
		"Consumo":     recipe.ExitConsumption,
		"consumido":   recipe.ExitConsumption,
		"donation":    recipe.ExitDonation,
		"Donación":    recipe.ExitDonation,
		"regalo":      recipe.ExitDonation,
	} {
		kind, err := recipe.ParseExitKind(spoken)
		if err != nil {
			t.Fatalf("the kind %q was refused: %v", spoken, err)
		}

		if kind != want {
			t.Errorf("the kind %q is %q, want %q", spoken, kind, want)
		}
	}

	if _, err := recipe.ParseExitKind("waste"); !errors.Is(err, recipe.ErrUnknownExitKind) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownExitKind)
	}
}

// containsTheQuestionOfTheTaxes reports whether the question names the
// professional of the taxes.
func containsTheQuestionOfTheTaxes(question string) bool {
	return len(question) > 0 && strings.Contains(question, "profesional de los impuestos")
}
