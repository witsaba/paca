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

// theDayOfTheSale is the day the sale of the test happens on.
func theDayOfTheSale() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC))
}

// theProductionOfTheSale is the production of the test: ten units of
// bread, worth ten thousand pesos.
func theProductionOfTheSale() recipe.Production {
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
		Day:        theDayOfTheSale(),
		GramsIn:    900,
		Money:      10000,
		Units:      units,
	}
}

// theUnitsOfTheSale are the units of the production of the test that are
// still in the book.
func theUnitsOfTheSale() []recipe.ExpiringUnit {
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
			ProducedOn: theDayOfTheSale(),
		})
	}

	return units
}

// theUnitsOfTheSaleInTheOrderOfTheExpiry answers the units of the
// production of the test in the order the rotation takes them.
//
// The unit that expires first carries the HIGHEST identity of the book, so
// the order of the identities and the order of the expiry disagree: only
// the expiry decides the order in which the units leave the book.
func theUnitsOfTheSaleInTheOrderOfTheExpiry() []recipe.ExpiringUnit {
	units := theUnitsOfTheSale()

	for index := range units {
		one := units[index]
		one.ExpiresOn = ledger.NewDate(
			time.Date(2026, time.March, 15+(10-index), 0, 0, 0, 0, time.UTC))
		units[index] = one
	}

	return units
}

// theDefinitionOfTheSale is the sale the owner confirmed: three units of
// bread for five thousand pesos.
func theDefinitionOfTheSale() recipe.SaleDefinition {
	return recipe.SaleDefinition{
		RecipeID:      recipe.ID(9),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Unit:          "unidad",
		Money:         5000,
		Day:           theDayOfTheSale(),
		Utterance:     "vendí tres panes",
		ConfirmedWith: "sí",
	}
}

// TestTheSaleTakesTheUnitsThatExpireFirst proves the rule of the rotation:
// the sale takes the units of the product that are still in it, IN THE
// ORDER THEY EXPIRE, and the valuation by the FIFO is another rule.
//
// The unit that expires first carries the highest identity of the book, so
// the order of the identities and the order of the expiry disagree, and
// the test says the whole ordered set of the identities that left.
func TestTheSaleTakesTheUnitsThatExpireFirst(t *testing.T) {
	t.Parallel()

	units := theUnitsOfTheSaleInTheOrderOfTheExpiry()

	sale, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		units,
		theDefinitionOfTheSale(),
	)
	if err != nil {
		t.Fatalf("the sale was refused: %v", err)
	}

	// The three units that expire first are the three last identities of
	// the book: the tenth, the ninth and the eighth.
	want := []recipe.UnitID{10, 9, 8}

	if len(sale.Units) != len(want) {
		t.Fatalf("the sale took %d units, want %d", len(sale.Units), len(want))
	}

	for index, one := range sale.Units {
		if one.UnitID != want[index] {
			t.Errorf("the unit %d that left the book is the unit %d, want the unit %d",
				index+1, one.UnitID, want[index])
		}
	}
}

// TestTheSaleOfAnExpiredUnitIsRefused proves the rule of the domain: a
// unit that expired is never sold, and the sale does not skip it to take a
// fresher one.
func TestTheSaleOfAnExpiredUnitIsRefused(t *testing.T) {
	t.Parallel()

	units := theUnitsOfTheSale()

	// The first unit expired yesterday: the sale of it is refused, and the
	// sale does not take the next one instead.
	expired := units[0]
	expired.ExpiresOn = ledger.NewDate(time.Date(2026, time.March, 13, 0, 0, 0, 0, time.UTC))
	units[0] = expired

	_, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		units,
		theDefinitionOfTheSale(),
	)
	if !errors.Is(err, recipe.ErrExpiredUnit) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrExpiredUnit)
	}
}

// TestTheSaleOfMoreUnitsThanTheBookHoldsIsRefused proves the rule of the
// domain: the book never writes a sale of units it does not hold.
//
// The sale carries its own error: the error of the waste is the error of a
// loss, and the owner is not told about a loss when she sold nothing.
func TestTheSaleOfMoreUnitsThanTheBookHoldsIsRefused(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheSale()
	definition.Quantity = 12

	_, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		definition,
	)
	if !errors.Is(err, recipe.ErrNotEnoughUnitsToSell) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotEnoughUnitsToSell)
	}
	if errors.Is(err, recipe.ErrNotEnoughUnits) {
		t.Fatalf("the error is %v, want the error of the sale and not the error of the waste", err)
	}
}

// TestTheCostOfTheSaleIsTheSumOfTheCostsOfTheUnitsSold proves the rule of
// the valuation: the cost of a unit is derived from the money of the
// production it came out of, and the cost of the sale is the sum of them.
func TestTheCostOfTheSaleIsTheSumOfTheCostsOfTheUnitsSold(t *testing.T) {
	t.Parallel()

	sale, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		theDefinitionOfTheSale(),
	)
	if err != nil {
		t.Fatalf("the sale was refused: %v", err)
	}

	// The production cost ten thousand pesos for ten units: a unit is
	// worth a thousand pesos, and three units are worth three thousand.
	if sale.Cost != 3000 {
		t.Errorf("the cost of the sale is %d, want %d", sale.Cost, 3000)
	}

	if sale.Money != 5000 {
		t.Errorf("the money of the sale is %d, want %d", sale.Money, 5000)
	}
}

// TestTheSaleIsRefusedWithoutAWholeNumberOfUnits proves the rule of the
// domain: the book never splits a unit.
func TestTheSaleIsRefusedWithoutAWholeNumberOfUnits(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheSale()
	definition.Quantity = 2.5

	_, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		definition,
	)
	if !errors.Is(err, recipe.ErrInvalidSaleQuantity) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidSaleQuantity)
	}
}

// TestTheSaleIsRefusedWithAWeight proves the rule of the domain: the sale
// counts the units it sold, and the book never converts a weight into a
// number of units.
func TestTheSaleIsRefusedWithAWeight(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheSale()
	definition.Unit = "kg"

	_, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		definition,
	)
	if !errors.Is(err, recipe.ErrInvalidSaleUnit) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidSaleUnit)
	}
}

// TestTheSaleIsRefusedWithoutMoney proves the rule of the domain: a sale
// moves money in, and the money is never inferred.
func TestTheSaleIsRefusedWithoutMoney(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheSale()
	definition.Money = 0

	_, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		definition,
	)
	if !errors.Is(err, recipe.ErrInvalidSaleMoney) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidSaleMoney)
	}
}

// TestTheSaleIsRefusedWithoutTheDayItWasMade proves the rule of the
// domain: the day is what says whether the unit expired.
func TestTheSaleIsRefusedWithoutTheDayItWasMade(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheSale()
	definition.Day = ledger.Date{}

	_, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		definition,
	)
	if !errors.Is(err, recipe.ErrMissingSaleDay) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingSaleDay)
	}
}

// TestTheSaleIsRefusedWithoutTheWordOfTheOwner proves the rule of the
// domain: the sale is written with the confirmation of the owner.
func TestTheSaleIsRefusedWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheSale()
	definition.ConfirmedWith = ""

	_, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		definition,
	)
	if !errors.Is(err, recipe.ErrMissingSaleConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingSaleConfirmation)
	}
}

// TestTheSaleIsRefusedWithoutABusiness proves the rule of the domain: the
// sale belongs to a business.
func TestTheSaleIsRefusedWithoutABusiness(t *testing.T) {
	t.Parallel()

	_, err := recipe.SaleOf(
		business.ID(0),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		theDefinitionOfTheSale(),
	)
	if !errors.Is(err, recipe.ErrMissingBusiness) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingBusiness)
	}
}

// TestTheSaleIsRefusedWhenTheBookHoldsNoProductWithThatName proves the
// rule of the domain: the book never guesses which product she sold.
func TestTheSaleIsRefusedWhenTheBookHoldsNoProductWithThatName(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheSale()
	definition.RecipeID = 0
	definition.RecipeName = "Pan de queso"

	_, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		definition,
	)
	if !errors.Is(err, recipe.ErrUnknownRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownRecipe)
	}
}

// TestTheSaleIsRefusedWhenTheNameCarriesMoreThanOneProduct proves the rule
// of the domain: the book never guesses which product she sold.
func TestTheSaleIsRefusedWhenTheNameCarriesMoreThanOneProduct(t *testing.T) {
	t.Parallel()

	other := theProductionOfTheSale()
	other.ID = recipe.ProductionID(4)
	other.RecipeID = recipe.ID(10)

	definition := theDefinitionOfTheSale()
	definition.RecipeID = 0

	_, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale(), other},
		theUnitsOfTheSale(),
		definition,
	)
	if !errors.Is(err, recipe.ErrAmbiguousRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrAmbiguousRecipe)
	}
}

// TestTheSaleMovesTheMoneyInAndReleasesTheCost proves the rule of the
// domain: the sale moves the money in and it releases the cost of the
// units it sells.
func TestTheSaleMovesTheMoneyInAndReleasesTheCost(t *testing.T) {
	t.Parallel()

	sale, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		theDefinitionOfTheSale(),
	)
	if err != nil {
		t.Fatalf("the sale was refused: %v", err)
	}

	money, err := sale.MoneyMovement(account.ID(1), account.ID(4))
	if err != nil {
		t.Fatalf("the movement of the money was refused: %v", err)
	}

	if money.DebitAccountID != account.ID(1) {
		t.Errorf("the money lands in the account %d, want the one that receives it", money.DebitAccountID)
	}

	if money.CreditAccountID != account.ID(4) {
		t.Errorf("the money comes from the account %d, want the sales", money.CreditAccountID)
	}

	if money.Amount != 5000 {
		t.Errorf("the movement carries %d pesos, want %d", money.Amount, 5000)
	}

	if money.OccurredOn.String() != "2026-03-14" {
		t.Errorf("the movement happens on %q, want %q", money.OccurredOn.String(), "2026-03-14")
	}

	cost, err := sale.CostMovement(account.ID(5), account.ID(3))
	if err != nil {
		t.Fatalf("the movement of the cost was refused: %v", err)
	}

	if cost.DebitAccountID != account.ID(5) {
		t.Errorf("the cost lands in the account %d, want the cost of the sales", cost.DebitAccountID)
	}

	if cost.CreditAccountID != account.ID(3) {
		t.Errorf("the cost comes from the account %d, want the inventory", cost.CreditAccountID)
	}

	if cost.Amount != 3000 {
		t.Errorf("the movement carries %d pesos, want %d", cost.Amount, 3000)
	}
}

// TestTheSaleOfTheBookCarriesTheWordsOfTheBook proves the rule of the
// domain: the movement of the sale says what it was when the owner said no
// sentence.
func TestTheSaleOfTheBookCarriesTheWordsOfTheBook(t *testing.T) {
	t.Parallel()

	definition := theDefinitionOfTheSale()
	definition.Utterance = ""

	sale, err := recipe.SaleOf(
		business.ID(77),
		[]recipe.Production{theProductionOfTheSale()},
		theUnitsOfTheSale(),
		definition,
	)
	if err != nil {
		t.Fatalf("the sale was refused: %v", err)
	}

	money, err := sale.MoneyMovement(account.ID(1), account.ID(4))
	if err != nil {
		t.Fatalf("the movement of the money was refused: %v", err)
	}

	if money.Description != "Venta: 3 unidades de Pan de molde" {
		t.Errorf("the movement says %q, want the words of the book", money.Description)
	}

	cost, err := sale.CostMovement(account.ID(5), account.ID(3))
	if err != nil {
		t.Fatalf("the movement of the cost was refused: %v", err)
	}

	if cost.Description != "Costo de la venta: 3 unidades de Pan de molde" {
		t.Errorf("the movement of the cost says %q, want the words of the book", cost.Description)
	}
}
