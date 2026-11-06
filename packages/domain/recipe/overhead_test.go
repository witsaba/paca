package recipe_test

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theDayOfTheOverhead is the day the overhead of the test was declared.
func theDayOfTheOverhead() ledger.Date {
	return ledger.NewDate(time.Date(2026, time.March, 7, 9, 0, 0, 0, time.UTC))
}

// TestTheOverheadIsTheElectricityAndTheHoursOfTheOven proves the rule of
// the domain: the overhead is the electricity of the house and the hours of
// the oven, and the allocation is declared by the owner.
//
// The money of the overhead is the electricity of the house times the
// allocation: the money of the book counts whole pesos.
func TestTheOverheadIsTheElectricityAndTheHoursOfTheOven(t *testing.T) {
	t.Parallel()

	overhead, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
		Electricity:   ledger.Amount(120000),
		OvenHours:     40,
		Allocation:    0.5,
		Day:           theDayOfTheOverhead(),
		Utterance:     "Pagué la luz del mes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the overhead was refused: %v", err)
	}

	if overhead.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", overhead.BusinessID)
	}
	if overhead.Electricity != ledger.Amount(120000) {
		t.Errorf("the electricity is %d, want 120000", overhead.Electricity)
	}
	if overhead.OvenHours != 40 {
		t.Errorf("the hours of the oven are %v, want 40", overhead.OvenHours)
	}
	if overhead.Allocation != 0.5 {
		t.Errorf("the allocation is %v, want 0.5", overhead.Allocation)
	}
	if overhead.Money != ledger.Amount(60000) {
		t.Errorf("the money of the overhead is %d, want 60000", overhead.Money)
	}
	if overhead.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", overhead.ConfirmedWith, "sí")
	}
}

// TestTheOverheadWithoutTheElectricityIsRefused proves the rule of the
// domain: the money of the electricity of the house is declared by the
// owner, and it is never inferred.
func TestTheOverheadWithoutTheElectricityIsRefused(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
		OvenHours:     40,
		Allocation:    0.5,
		Day:           theDayOfTheOverhead(),
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingElectricity) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingElectricity)
	}
}

// TestTheOverheadWithoutTheHoursOfTheOvenIsRefused proves the rule of the
// domain: without the hours of the oven the money of the oven has no rate,
// and a rate nobody declared is a rate PACA invented.
func TestTheOverheadWithoutTheHoursOfTheOvenIsRefused(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
		Electricity:   ledger.Amount(120000),
		Allocation:    0.5,
		Day:           theDayOfTheOverhead(),
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingOvenHours) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingOvenHours)
	}
}

// TestTheAllocationThatIsNotAShareIsRefused proves the rule of the domain:
// the allocation is the share of the electricity of the house that belongs
// to the oven, and it is greater than zero and not greater than the
// electricity itself.
func TestTheAllocationThatIsNotAShareIsRefused(t *testing.T) {
	t.Parallel()

	for _, allocation := range []float64{0, -0.5, 1.5} {
		_, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
			Electricity:   ledger.Amount(120000),
			OvenHours:     40,
			Allocation:    allocation,
			Day:           theDayOfTheOverhead(),
			ConfirmedWith: "sí",
		})
		if !errors.Is(err, recipe.ErrInvalidAllocation) {
			t.Fatalf("the error of the allocation %v is %v, want %v",
				allocation, err, recipe.ErrInvalidAllocation)
		}
	}
}

// TestTheOverheadOfNoBusinessIsRefused proves the rule of the domain: the
// overhead of the bakery belongs to the business that works it.
func TestTheOverheadOfNoBusinessIsRefused(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewOverhead(business.ID(0), recipe.OverheadDefinition{
		Electricity:   ledger.Amount(120000),
		OvenHours:     40,
		Allocation:    0.5,
		Day:           theDayOfTheOverhead(),
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingBusiness) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingBusiness)
	}
}

// TestTheOverheadWithoutTheDayIsRefused proves the rule of the domain: the
// overhead carries the day it was declared.
func TestTheOverheadWithoutTheDayIsRefused(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
		Electricity:   ledger.Amount(120000),
		OvenHours:     40,
		Allocation:    0.5,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingDay) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingDay)
	}
}

// TestTheOverheadWithoutTheWordOfTheOwnerIsRefused proves the rule of the
// domain: the overhead is written with the word the owner used to confirm
// it, the way the purchase is written.
func TestTheOverheadWithoutTheWordOfTheOwnerIsRefused(t *testing.T) {
	t.Parallel()

	_, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
		Electricity: ledger.Amount(120000),
		OvenHours:   40,
		Allocation:  0.5,
		Day:         theDayOfTheOverhead(),
	})
	if !errors.Is(err, recipe.ErrMissingConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingConfirmation)
	}
}

// TestTheRateOfTheOverheadIsTheLastTheOwnerDeclared proves the rule of the
// domain: the rate of the house does not wait, and the last declaration is
// the one that counts.
func TestTheRateOfTheOverheadIsTheLastTheOwnerDeclared(t *testing.T) {
	t.Parallel()

	older, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
		Electricity:   ledger.Amount(100000),
		OvenHours:     40,
		Allocation:    0.5,
		Day:           ledger.NewDate(time.Date(2026, time.February, 7, 9, 0, 0, 0, time.UTC)),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the overhead was refused: %v", err)
	}

	older.ID = recipe.OverheadID(1)

	newer, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
		Electricity:   ledger.Amount(120000),
		OvenHours:     40,
		Allocation:    0.5,
		Day:           theDayOfTheOverhead(),
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the overhead was refused: %v", err)
	}

	newer.ID = recipe.OverheadID(2)

	rate := recipe.TheOverheadRateOf([]recipe.Overhead{older, newer})
	if rate.Unknown {
		t.Fatal("the rate of the overhead is not derivable, want the rate of the last declaration")
	}
	if rate.MoneyPerOvenHour != 1500 {
		t.Errorf("the money of the hour of the oven is %v, want 1500", rate.MoneyPerOvenHour)
	}
}

// TestTheRateOfABusinessThatDeclaredNoOverheadIsNotKnown proves the rule of
// the domain: the money of the house is never inferred, and the batch that
// needs it says that it does not know it.
func TestTheRateOfABusinessThatDeclaredNoOverheadIsNotKnown(t *testing.T) {
	t.Parallel()

	rate := recipe.TheOverheadRateOf(nil)
	if !rate.Unknown {
		t.Fatal("the rate of the overhead is derivable, want the news that it is not known")
	}
	if rate.Reason == "" {
		t.Error("the rate of the overhead carries no reason, want the words of the book")
	}
}

// TestTheOverheadOfABatchIsDerivedFromTheHoursOfItsOven proves the rule of
// the domain: the overhead of a batch is derived from the hours of the oven
// of its steps, and never typed.
//
// The steps of the hands and of the waiting are the labor of the batch, not
// the overhead of it.
func TestTheOverheadOfABatchIsDerivedFromTheHoursOfItsOven(t *testing.T) {
	t.Parallel()

	overhead, err := recipe.OverheadOf(theRecipeOfTheLabor(t), recipe.OverheadRate{
		MoneyPerOvenHour: 1500,
	})
	if err != nil {
		t.Fatalf("the overhead was refused: %v", err)
	}

	if overhead.OvenHours != 0.75 {
		t.Errorf("the hours of the oven are %v, want 0.75", overhead.OvenHours)
	}
	if overhead.Money != ledger.Amount(1125) {
		t.Errorf("the money of the overhead is %d, want 1125", overhead.Money)
	}
	if overhead.Unknown {
		t.Error("the money of the overhead is not derivable, want the money of the hours")
	}
}

// TestTheOverheadOfABatchIsNotKnownWhenTheHouseIsNotKnown proves the rule
// of the domain: the money of the overhead is never inferred, and the batch
// says that it does not know it.
func TestTheOverheadOfABatchIsNotKnownWhenTheHouseIsNotKnown(t *testing.T) {
	t.Parallel()

	overhead, err := recipe.OverheadOf(theRecipeOfTheLabor(t), recipe.OverheadRate{
		Unknown: true,
		Reason:  "the owner has not declared the overhead of the house",
	})
	if err != nil {
		t.Fatalf("the overhead was refused: %v", err)
	}

	if !overhead.Unknown {
		t.Fatal("the money of the overhead is derivable, want the news that it is not known")
	}
	if overhead.Money != ledger.Amount(0) {
		t.Errorf("the money of the overhead is %d, want none", overhead.Money)
	}
	if overhead.Reason == "" {
		t.Error("the overhead carries no reason, want the words of the book")
	}
}
