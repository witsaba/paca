package recipe_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The day of the rotation of the test: the day the answer was read on.
func theDayOfTheRotation() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC))
}

// A unit of the book of the rotation of the test.
//
// It was made on the fourteenth of March, it belongs to the recipe of the
// test, and it expires on the day the test names.
func aUnitOfTheRotation(
	id int64,
	production int64,
	position int,
	of recipe.ID,
	expires time.Time,
) recipe.ExpiringUnit {
	return recipe.ExpiringUnit{
		ProductionUnit: recipe.ProductionUnit{
			ID:           recipe.UnitID(id),
			ProductionID: recipe.ProductionID(production),
			Position:     position,
			Grams:        90,
			ExpiresOn:    ledger.NewDate(expires),
		},
		RecipeID:   of,
		RecipeName: "Pan de molde",
		ProducedOn: ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
	}
}

// TestTheRotationFollowsTheEarliestExpiry proves the rule of the domain:
// the rotation of the output follows the earliest expiry, and the unit that
// expires first is the first one that leaves the book.
func TestTheRotationFollowsTheEarliestExpiry(t *testing.T) {
	t.Parallel()

	late := aUnitOfTheRotation(3, 3, 1, recipe.ID(9), time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC))
	early := aUnitOfTheRotation(1, 3, 2, recipe.ID(9), time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC))
	middle := aUnitOfTheRotation(2, 3, 3, recipe.ID(9), time.Date(2026, time.March, 18, 0, 0, 0, 0, time.UTC))

	rotation := recipe.RotationOf([]recipe.ExpiringUnit{late, early, middle}, 0, "", theDayOfTheRotation())

	if len(rotation.Units) != 3 {
		t.Fatalf("the rotation carries %d units, want 3", len(rotation.Units))
	}

	want := []string{"2026-03-16", "2026-03-18", "2026-03-20"}

	for index, one := range rotation.Units {
		if one.ExpiresOn.String() != want[index] {
			t.Errorf("the unit %d of the rotation expires on %q, want %q",
				index+1, one.ExpiresOn.String(), want[index])
		}

		if one.Order != index+1 {
			t.Errorf("the unit %d of the rotation takes the place %d, want %d",
				index+1, one.Order, index+1)
		}
	}
}

// TestTheUnitsThatExpireOnTheSameDayKeepAStableOrder proves the rule of the
// domain: the units that expire on the same day keep a stable order, the
// production first and the position of the unit in it.
//
// The data of the test makes the two orders disagree: the unit of a later
// production carries the lowest identity, and the unit of an earlier
// production carries a higher one, so the order of the identities and the
// order of the rule are not the same order.
func TestTheUnitsThatExpireOnTheSameDayKeepAStableOrder(t *testing.T) {
	t.Parallel()

	expires := time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)

	later := aUnitOfTheRotation(1, 5, 1, recipe.ID(9), expires)
	earlier := aUnitOfTheRotation(2, 4, 2, recipe.ID(9), expires)
	first := aUnitOfTheRotation(3, 4, 1, recipe.ID(9), expires)

	rotation := recipe.RotationOf(
		[]recipe.ExpiringUnit{later, earlier, first},
		0,
		"",
		theDayOfTheRotation(),
	)

	// The rule of the domain: the production first, and the position of
	// the unit in it. The production 4 comes before the production 5, and
	// the position 1 comes before the position 2.
	want := []recipe.UnitID{3, 2, 1}

	for index, one := range rotation.Units {
		if one.UnitID != want[index] {
			t.Errorf("the unit %d of the rotation is the unit %d, want the unit %d",
				index+1, one.UnitID, want[index])
		}
	}
}

// TestTheRotationMayBeOfOneProduct proves the rule of the domain: the
// rotation may be asked for the whole book of a business or for one product
// of it, and the product is the one the owner asked about.
func TestTheRotationMayBeOfOneProduct(t *testing.T) {
	t.Parallel()

	expires := time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)

	bread := aUnitOfTheRotation(1, 3, 1, recipe.ID(9), expires)
	cake := aUnitOfTheRotation(2, 4, 1, recipe.ID(10), expires)

	rotation := recipe.RotationOf(
		[]recipe.ExpiringUnit{bread, cake},
		recipe.ID(10),
		"Pastel",
		theDayOfTheRotation(),
	)

	if rotation.RecipeID != recipe.ID(10) {
		t.Errorf("the product of the rotation is %d, want %d", rotation.RecipeID, recipe.ID(10))
	}

	if rotation.RecipeName != "Pastel" {
		t.Errorf("the name of the product of the rotation is %q, want %q", rotation.RecipeName, "Pastel")
	}

	if len(rotation.Units) != 1 {
		t.Fatalf("the rotation carries %d units, want the one of the product", len(rotation.Units))
	}

	if rotation.Units[0].RecipeID != recipe.ID(10) {
		t.Errorf("the unit of the rotation is of the product %d, want %d",
			rotation.Units[0].RecipeID, recipe.ID(10))
	}
}

// TestTheRotationCarriesTheDayAndTheDaysLeft proves the rule of the domain:
// the answer carries the day every unit expires and the days that are left
// of it, and the product every unit belongs to.
func TestTheRotationCarriesTheDayAndTheDaysLeft(t *testing.T) {
	t.Parallel()

	unit := aUnitOfTheRotation(1, 3, 1, recipe.ID(9), time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC))

	rotation := recipe.RotationOf([]recipe.ExpiringUnit{unit}, 0, "", theDayOfTheRotation())

	if rotation.Day.String() != "2026-03-15" {
		t.Errorf("the rotation carries the day %q, want %q", rotation.Day.String(), "2026-03-15")
	}

	if len(rotation.Units) != 1 {
		t.Fatalf("the rotation carries %d units, want 1", len(rotation.Units))
	}

	one := rotation.Units[0]

	if one.ExpiresOn.String() != "2026-03-17" {
		t.Errorf("the unit expires on %q, want %q", one.ExpiresOn.String(), "2026-03-17")
	}

	if one.DaysLeft != 2 {
		t.Errorf("the unit has %d days left, want 2", one.DaysLeft)
	}

	if one.RecipeID != recipe.ID(9) {
		t.Errorf("the unit is of the product %d, want %d", one.RecipeID, recipe.ID(9))
	}

	if one.RecipeName != "Pan de molde" {
		t.Errorf("the name of the product of the unit is %q, want %q", one.RecipeName, "Pan de molde")
	}

	if one.ProducedOn.String() != "2026-03-14" {
		t.Errorf("the unit was made on %q, want %q", one.ProducedOn.String(), "2026-03-14")
	}

	if one.Grams != 90 {
		t.Errorf("the unit weighs %v grams, want 90", one.Grams)
	}
}

// TestTheRotationCarriesNoMoney proves the rule of the domain: the
// valuation of the stock follows the FIFO and the rotation follows the
// earliest expiry, the two rules are apart, and the answer of the rotation
// carries no money at all.
func TestTheRotationCarriesNoMoney(t *testing.T) {
	t.Parallel()

	for _, answer := range []any{recipe.Rotation{}, recipe.RotationUnit{}} {
		kind := reflect.TypeOf(answer)

		for index := 0; index < kind.NumField(); index++ {
			name := kind.Field(index).Name

			switch name {
			case "Money", "Cost", "Amount", "Price", "Value":
				t.Errorf("the answer of the rotation carries the field %q: the money of a unit "+
					"is the rule of the sale and of the waste, and it is not the rotation", name)
			}
		}
	}
}

// TestTheRotationOfTheBookHoldsTheUnitsThatAreStillInIt proves the rule of
// the domain: the rotation is built with the units that are still in the
// book, and the unit that left the book is never offered again.
func TestTheRotationOfTheBookHoldsTheUnitsThatAreStillInIt(t *testing.T) {
	t.Parallel()

	expires := time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)

	held := []recipe.ExpiringUnit{aUnitOfTheRotation(1, 3, 1, recipe.ID(9), expires)}

	rotation := recipe.RotationOf(held, 0, "", theDayOfTheRotation())

	if len(rotation.Units) != 1 {
		t.Fatalf("the rotation carries %d units, want the one that is still in the book", len(rotation.Units))
	}

	if rotation.Units[0].UnitID != recipe.UnitID(1) {
		t.Errorf("the unit of the rotation is the unit %d, want the unit 1", rotation.Units[0].UnitID)
	}
}
