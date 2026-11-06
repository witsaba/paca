package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The fake of the storage of the labor.
//
// It remembers the labor it wrote: the use case writes one, and the test
// proves what it wrote.
type fakeLabors struct {
	written []recipe.Labor
	held    []recipe.Labor
	err     error
	listed  business.ID
}

// Record writes the labor and remembers it.
func (f *fakeLabors) Record(_ context.Context, written recipe.Labor) (recipe.Labor, error) {
	if f.err != nil {
		return recipe.Labor{}, f.err
	}

	written.ID = recipe.LaborID(4)
	f.written = append(f.written, written)

	return written, nil
}

// List answers the labor of the business, and remembers it.
func (f *fakeLabors) List(_ context.Context, own business.ID) ([]recipe.Labor, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]recipe.Labor, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// theLaborOfTheTest is the labor the owner declared: two hours with her
// hands, one in the oven and half an hour of waiting, at twenty thousand
// pesos the hour.
func theLaborOfTheTest(t *testing.T) recipe.Labor {
	t.Helper()

	hours, err := recipe.NewHours(2, 1, 0.5)
	if err != nil {
		t.Fatalf("the hours of the labor were refused: %v", err)
	}

	labour, err := recipe.NewLabor(business.ID(77), recipe.LaborDefinition{
		Hours:         hours,
		HourValue:     ledger.Amount(20000),
		Day:           ledger.NewDate(theClock().Now()),
		Utterance:     "Hoy trabajé tres horas y media en la panadería",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the labor was refused: %v", err)
	}

	return labour
}

// TestTheLaborIsWrittenWithTheHoursAndTheValueOfTheHour proves that the
// use case writes the hours the owner worked and the value of the hour she
// declared, and that it never infers either of them.
func TestTheLaborIsWrittenWithTheHoursAndTheValueOfTheHour(t *testing.T) {
	t.Parallel()

	labors := &fakeLabors{}
	service := application.NewLaborService(labors, theClock())

	written, err := service.Record(t.Context(), ports.RecordLabor{
		BusinessID:    business.ID(77),
		HandsOnHours:  2,
		OvenHours:     1,
		WaitingHours:  0.5,
		HourValue:     20000,
		Day:           "2026-03-15",
		Utterance:     "Hoy trabajé tres horas y media en la panadería",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the labor was refused: %v", err)
	}

	if len(labors.written) != 1 {
		t.Fatalf("the use case wrote %d labors, want 1", len(labors.written))
	}

	if written.Hours.HandsOn != 2 || written.Hours.Oven != 1 || written.Hours.Waiting != 0.5 {
		t.Errorf("the hours of the labor are %+v, want two hours, one hour and half an hour", written.Hours)
	}
	if written.HourValue != ledger.Amount(20000) {
		t.Errorf("the value of the hour is %d, want 20000", written.HourValue)
	}
	if written.Money != ledger.Amount(70000) {
		t.Errorf("the money of the labor is %d, want 70000", written.Money)
	}
	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day of the labor is %q, want %q", written.Day.String(), "2026-03-15")
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
}

// TestTheLaborWithoutHoursIsRefused proves that the use case refuses a
// labor without hours: a labor without hours is nothing.
func TestTheLaborWithoutHoursIsRefused(t *testing.T) {
	t.Parallel()

	labors := &fakeLabors{}
	service := application.NewLaborService(labors, theClock())

	_, err := service.Record(t.Context(), ports.RecordLabor{
		BusinessID:    business.ID(77),
		HourValue:     20000,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingHours) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingHours)
	}
	if len(labors.written) != 0 {
		t.Errorf("the use case wrote %d labors, want none", len(labors.written))
	}
}

// TestTheLaborWithoutTheValueOfTheHourIsRefused proves that the use case
// refuses a labor whose value of the hour was not declared: the money of
// the work of a person is never inferred.
func TestTheLaborWithoutTheValueOfTheHourIsRefused(t *testing.T) {
	t.Parallel()

	labors := &fakeLabors{}
	service := application.NewLaborService(labors, theClock())

	_, err := service.Record(t.Context(), ports.RecordLabor{
		BusinessID:    business.ID(77),
		HandsOnHours:  2,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingHourValue) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingHourValue)
	}
	if len(labors.written) != 0 {
		t.Errorf("the use case wrote %d labors, want none", len(labors.written))
	}
}

// TestTheLaborOfTheDayOfToday proves that the day of a labor the agent
// names none is the day of today: the day is one of the three things Paca
// may fill by herself.
func TestTheLaborOfTheDayOfToday(t *testing.T) {
	t.Parallel()

	labors := &fakeLabors{}
	service := application.NewLaborService(labors, theClock())

	written, err := service.Record(t.Context(), ports.RecordLabor{
		BusinessID:    business.ID(77),
		HandsOnHours:  2,
		HourValue:     20000,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the labor was refused: %v", err)
	}

	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day of the labor is %q, want the day of today", written.Day.String())
	}
}

// TestTheLaborWithoutADatabaseSaysSo proves that a service without a
// database says so, and it says it in the only capability it has.
func TestTheLaborWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewLaborService(nil, theClock())

	_, err := service.Record(t.Context(), ports.RecordLabor{
		BusinessID:    business.ID(77),
		HandsOnHours:  2,
		HourValue:     20000,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// theLaborOfTheBusiness is the labor the business declared: the value of
// the hour the owner declared, the one the labor of a batch is valued
// with.
func theLaborOfTheBusiness(t *testing.T) *fakeLabors {
	t.Helper()

	return &fakeLabors{held: []recipe.Labor{theLaborOfTheTest(t)}}
}
