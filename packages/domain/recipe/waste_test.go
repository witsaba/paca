package recipe_test

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The day of the production of the test.
func theDayOfTheWaste() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC))
}

// The production of the test: ten units of bread, worth ten thousand pesos.
func theProductionOfTheWaste() recipe.Production {
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
		Day:        theDayOfTheWaste(),
		GramsIn:    900,
		Money:      10000,
		Units:      units,
	}
}

// The units of the production of the test that are still in the book.
func theUnitsOfTheWaste() []recipe.ExpiringUnit {
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
			ProducedOn: theDayOfTheWaste(),
		})
	}

	return units
}

// The definition of the waste of the test: three units that were burnt.
func theDefinitionOfTheWaste() recipe.WasteDefinition {
	return recipe.WasteDefinition{
		RecipeID:      recipe.ID(9),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Reason:        recipe.WasteBurnt,
		Day:           ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
		Utterance:     "se me quemaron tres panes",
		ConfirmedWith: "sí",
	}
}

// TestTheWasteCarriesTheProductTheQuantityTheUnitAndTheReason proves the
// rule of the domain: the waste is a movement that carries the product, the
// quantity, the unit and the reason.
func TestTheWasteCarriesTheProductTheQuantityTheUnitAndTheReason(t *testing.T) {
	t.Parallel()

	waste, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		theDefinitionOfTheWaste(),
	)
	if err != nil {
		t.Fatalf("the waste was refused: %v", err)
	}

	if waste.RecipeID != recipe.ID(9) {
		t.Errorf("the product of the waste is %d, want %d", waste.RecipeID, recipe.ID(9))
	}

	if waste.RecipeName != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", waste.RecipeName, "Pan de molde")
	}

	if waste.Quantity != 3 {
		t.Errorf("the quantity of the waste is %v, want 3", waste.Quantity)
	}

	if waste.Unit != "unidad" {
		t.Errorf("the unit of the waste is %q, want %q", waste.Unit, "unidad")
	}

	if waste.Reason != recipe.WasteBurnt {
		t.Errorf("the reason of the waste is %q, want %q", waste.Reason, recipe.WasteBurnt)
	}

	if waste.Day.String() != "2026-03-14" {
		t.Errorf("the day of the waste is %q, want %q", waste.Day.String(), "2026-03-14")
	}

	if waste.ConfirmedWith != "sí" {
		t.Errorf("the word of the owner is %q, want %q", waste.ConfirmedWith, "sí")
	}
}

// TestTheWasteReleasesTheCostOfWhatItWasWorth proves the rule of the
// domain: the money of the waste is the money of the units that left, and
// it is derived, never typed.
func TestTheWasteReleasesTheCostOfWhatItWasWorth(t *testing.T) {
	t.Parallel()

	waste, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		theDefinitionOfTheWaste(),
	)
	if err != nil {
		t.Fatalf("the waste was refused: %v", err)
	}

	// The production cost ten thousand pesos for ten units: a unit is
	// worth a thousand pesos, and three units are worth three thousand.
	if waste.Money != 3000 {
		t.Errorf("the money of the waste is %d, want %d", waste.Money, 3000)
	}

	if len(waste.Units) != 3 {
		t.Fatalf("the waste took %d units, want 3", len(waste.Units))
	}
}

// TestTheWasteTakesTheUnitsThatExpireFirst proves the rule of the
// rotation: the units that leave the book are the ones that expire first,
// and the valuation by FIFO is another rule.
func TestTheWasteTakesTheUnitsThatExpireFirst(t *testing.T) {
	t.Parallel()

	units := theUnitsOfTheWaste()

	// The first unit expires before the others: it is the one that leaves
	// the book first.
	early := units[0]
	early.ExpiresOn = ledger.NewDate(time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC))
	units[0] = early

	// The last unit expires after the others: it stays in the book.
	late := units[9]
	late.ID = recipe.UnitID(20)
	late.ExpiresOn = ledger.NewDate(time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC))
	units[9] = late

	waste, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		units,
		theDefinitionOfTheWaste(),
	)
	if err != nil {
		t.Fatalf("the waste was refused: %v", err)
	}

	if len(waste.Units) != 3 {
		t.Fatalf("the waste took %d units, want 3", len(waste.Units))
	}

	if waste.Units[0].UnitID != recipe.UnitID(1) {
		t.Errorf("the first unit that left is %d, want the one that expires first", waste.Units[0].UnitID)
	}

	for _, one := range waste.Units {
		if one.UnitID == recipe.UnitID(20) {
			t.Error("the unit that expires last left the book, want the ones that expire first")
		}
	}
}

// TestTheWasteIsRefusedWithoutAWholeNumberOfUnits proves the rule of the
// domain: the waste of the bakery counts the units it lost, and the book
// never splits a unit.
func TestTheWasteIsRefusedWithoutAWholeNumberOfUnits(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheWaste()
	definition.Quantity = 2.5

	_, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		definition,
	)
	if !errors.Is(err, recipe.ErrInvalidWasteQuantity) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidWasteQuantity)
	}
}

// TestTheWasteIsRefusedWithAWeight proves the rule of the domain: the waste
// counts the units it lost, and the book never converts a weight into a
// number of units.
func TestTheWasteIsRefusedWithAWeight(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheWaste()
	definition.Unit = "kg"

	_, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		definition,
	)
	if !errors.Is(err, recipe.ErrInvalidWasteUnit) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidWasteUnit)
	}
}

// TestTheWasteIsRefusedWithoutAReason proves the rule of the domain: the
// reason is what a later analysis reads, and it is never guessed.
func TestTheWasteIsRefusedWithoutAReason(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheWaste()
	definition.Reason = ""

	_, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		definition,
	)
	if !errors.Is(err, recipe.ErrUnknownWasteReason) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownWasteReason)
	}
}

// TestTheWasteIsRefusedWithoutTheWordOfTheOwner proves the rule of the
// domain: the waste is written with the confirmation of the owner.
func TestTheWasteIsRefusedWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheWaste()
	definition.ConfirmedWith = ""

	_, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		definition,
	)
	if !errors.Is(err, recipe.ErrMissingWasteConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingWasteConfirmation)
	}
}

// TestTheWasteIsRefusedWithoutTheDayItHappened proves the rule of the
// domain: the waste of no day is a waste of no time.
func TestTheWasteIsRefusedWithoutTheDayItHappened(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheWaste()
	definition.Day = ledger.Date{}

	_, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		definition,
	)
	if !errors.Is(err, recipe.ErrMissingWasteDay) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingWasteDay)
	}
}

// TestTheWasteIsRefusedWithoutAProduct proves the rule of the domain: the
// waste names the product it lost.
func TestTheWasteIsRefusedWithoutAProduct(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheWaste()
	definition.RecipeID = 0
	definition.RecipeName = ""

	_, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		definition,
	)
	if !errors.Is(err, recipe.ErrMissingRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingRecipe)
	}
}

// TestTheWasteIsRefusedWhenTheNameCarriesMoreThanOneProduct proves the rule
// of the domain: the book never guesses which product the owner lost.
func TestTheWasteIsRefusedWhenTheNameCarriesMoreThanOneProduct(t *testing.T) {
	t.Parallel()

	other := theProductionOfTheWaste()
	other.ID = recipe.ProductionID(4)
	other.RecipeID = recipe.ID(10)

	definition := theDefinitionOfTheWaste()
	definition.RecipeID = 0

	_, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste(), other},
		theUnitsOfTheWaste(),
		definition,
	)
	if !errors.Is(err, recipe.ErrAmbiguousRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrAmbiguousRecipe)
	}
}

// TestTheWasteIsRefusedWhenTheBookHoldsNoUnitOfTheProduct proves the rule
// of the domain: the waste of what the book does not hold is refused.
func TestTheWasteIsRefusedWhenTheBookHoldsNoUnitOfTheProduct(t *testing.T) {
	t.Parallel()

	_, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		[]recipe.ExpiringUnit{},
		theDefinitionOfTheWaste(),
	)
	if !errors.Is(err, recipe.ErrNotEnoughUnits) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotEnoughUnits)
	}
}

// TestTheWasteIsRefusedWhenTheBookHoldsFewerUnitsThanSheLost proves the
// rule of the domain: the book never writes a waste of units it does not
// hold.
func TestTheWasteIsRefusedWhenTheBookHoldsFewerUnitsThanSheLost(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheWaste()
	definition.Quantity = 12

	_, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		definition,
	)
	if !errors.Is(err, recipe.ErrNotEnoughUnits) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotEnoughUnits)
	}
}

// TestTheWasteMovesTheMoneyToTheAccountOfTheWaste proves the rule of the
// domain: the money of the waste is never lost from the book, it lands in
// the account of the waste and it leaves the inventory.
func TestTheWasteMovesTheMoneyToTheAccountOfTheWaste(t *testing.T) {
	t.Parallel()

	waste, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		theDefinitionOfTheWaste(),
	)
	if err != nil {
		t.Fatalf("the waste was refused: %v", err)
	}

	movement, err := waste.Movement(account.ID(5), account.ID(3))
	if err != nil {
		t.Fatalf("the movement of the waste was refused: %v", err)
	}

	if movement.DebitAccountID != account.ID(5) {
		t.Errorf("the money lands in the account %d, want the waste", movement.DebitAccountID)
	}

	if movement.CreditAccountID != account.ID(3) {
		t.Errorf("the money comes from the account %d, want the inventory", movement.CreditAccountID)
	}

	if movement.Amount != 3000 {
		t.Errorf("the movement carries %d pesos, want %d", movement.Amount, 3000)
	}

	if movement.OccurredOn.String() != "2026-03-14" {
		t.Errorf("the movement happens on %q, want %q", movement.OccurredOn.String(), "2026-03-14")
	}

	if movement.Description != "se me quemaron tres panes" {
		t.Errorf("the movement says %q, want the sentence of the owner", movement.Description)
	}
}

// TestTheWasteOfTheBookCarriesTheWordsOfTheBook proves the rule of the
// domain: the movement of the waste says what it was when the owner said
// no sentence.
func TestTheWasteOfTheBookCarriesTheWordsOfTheBook(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheWaste()
	definition.Utterance = ""

	waste, err := recipe.WasteOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheWaste()},
		theUnitsOfTheWaste(),
		definition,
	)
	if err != nil {
		t.Fatalf("the waste was refused: %v", err)
	}

	movement, err := waste.Movement(account.ID(5), account.ID(3))
	if err != nil {
		t.Fatalf("the movement of the waste was refused: %v", err)
	}

	if movement.Description != "Merma: 3 unidades de Pan de molde (burnt)" {
		t.Errorf("the movement says %q, want the words of the book", movement.Description)
	}
}

// TestTheReasonsOfTheWasteAreTheFourOfTheBook proves the rule of the
// domain: the reason of the waste is one of the four reasons of the book,
// and it is read the way a person reads it.
func TestTheReasonsOfTheWasteAreTheFourOfTheBook(t *testing.T) {
	t.Parallel()

	reasons := recipe.AllWasteReasons()
	if len(reasons) != 4 {
		t.Fatalf("the book knows %d reasons, want 4", len(reasons))
	}

	for _, one := range reasons {
		if !one.Valid() {
			t.Errorf("the reason %q is not one of the four of the book", one)
		}
	}

	parsed, err := recipe.ParseWasteReason("quemado")
	if err != nil {
		t.Fatalf("the reason of the owner was refused: %v", err)
	}

	if parsed != recipe.WasteBurnt {
		t.Errorf("the reason of the owner is %q, want %q", parsed, recipe.WasteBurnt)
	}

	if _, err := recipe.ParseWasteReason("roto"); err == nil {
		t.Error("a reason the book does not know was accepted")
	}
}
