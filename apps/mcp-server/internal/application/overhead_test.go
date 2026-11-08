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

// The fake of the storage of the overhead.
//
// It remembers the overhead it wrote: the use case writes one, and the test
// proves what it wrote.
type fakeOverheads struct {
	written []recipe.Overhead
	held    []recipe.Overhead
	err     error
	listed  business.ID
}

// Record writes the overhead and remembers it.
func (f *fakeOverheads) Record(_ context.Context, written recipe.Overhead) (recipe.Overhead, error) {
	if f.err != nil {
		return recipe.Overhead{}, f.err
	}

	written.ID = recipe.OverheadID(4)
	f.written = append(f.written, written)

	return written, nil
}

// List answers the overhead of the business, and remembers it.
func (f *fakeOverheads) List(_ context.Context, own business.ID) ([]recipe.Overhead, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]recipe.Overhead, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// theOverheadOfTheTest is the overhead the owner declared: the electricity
// of the house, the hours the oven ran and the share of the electricity
// that belongs to the oven.
func theOverheadOfTheTest(t *testing.T) recipe.Overhead {
	t.Helper()

	overhead, err := recipe.NewOverhead(business.ID(77), recipe.OverheadDefinition{
		Electricity:   ledger.Amount(120000),
		OvenHours:     40,
		Allocation:    0.5,
		Day:           ledger.NewDate(theClock().Now()),
		Utterance:     "Pagué la luz del mes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the overhead was refused: %v", err)
	}

	return overhead
}

// TestTheOverheadIsWrittenWithTheElectricityAndTheAllocation proves that
// the use case writes the electricity of the house, the hours of the oven
// and the allocation the owner declared, and that it never infers any of
// them.
func TestTheOverheadIsWrittenWithTheElectricityAndTheAllocation(t *testing.T) {
	t.Parallel()

	overheads := &fakeOverheads{}
	service := application.NewOverheadService(overheads, theClock())

	written, err := service.Record(t.Context(), ports.RecordOverhead{
		BusinessID:    business.ID(77),
		Electricity:   120000,
		OvenHours:     40,
		Allocation:    0.5,
		Day:           "2026-03-15",
		Utterance:     "Pagué la luz del mes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the overhead was refused: %v", err)
	}

	if len(overheads.written) != 1 {
		t.Fatalf("the use case wrote %d overheads, want 1", len(overheads.written))
	}

	if written.Electricity != ledger.Amount(120000) {
		t.Errorf("the electricity of the house is %d, want 120000", written.Electricity)
	}
	if written.OvenHours != 40 {
		t.Errorf("the hours of the oven are %v, want 40", written.OvenHours)
	}
	if written.Allocation != 0.5 {
		t.Errorf("the allocation is %v, want 0.5", written.Allocation)
	}
	if written.Money != ledger.Amount(60000) {
		t.Errorf("the money of the overhead is %d, want 60000", written.Money)
	}
	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day of the overhead is %q, want %q", written.Day.String(), "2026-03-15")
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
}

// TestTheOverheadWithoutTheElectricityIsRefused proves that the use case
// refuses an overhead without the electricity of the house: the money of
// the house is never inferred.
func TestTheOverheadWithoutTheElectricityIsRefused(t *testing.T) {
	t.Parallel()

	overheads := &fakeOverheads{}
	service := application.NewOverheadService(overheads, theClock())

	_, err := service.Record(t.Context(), ports.RecordOverhead{
		BusinessID:    business.ID(77),
		OvenHours:     40,
		Allocation:    0.5,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingElectricity) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingElectricity)
	}
	if len(overheads.written) != 0 {
		t.Errorf("the use case wrote %d overheads, want none", len(overheads.written))
	}
}

// TestTheOverheadWithoutTheHoursOfTheOvenIsRefused proves that the use
// case refuses an overhead without the hours of the oven: without them the
// money of the oven has no rate.
func TestTheOverheadWithoutTheHoursOfTheOvenIsRefused(t *testing.T) {
	t.Parallel()

	overheads := &fakeOverheads{}
	service := application.NewOverheadService(overheads, theClock())

	_, err := service.Record(t.Context(), ports.RecordOverhead{
		BusinessID:    business.ID(77),
		Electricity:   120000,
		Allocation:    0.5,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrMissingOvenHours) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingOvenHours)
	}
	if len(overheads.written) != 0 {
		t.Errorf("the use case wrote %d overheads, want none", len(overheads.written))
	}
}

// TestTheOverheadWithAnAllocationThatIsNotAShareIsRefused proves that the
// use case refuses an allocation that is not a share of the electricity.
func TestTheOverheadWithAnAllocationThatIsNotAShareIsRefused(t *testing.T) {
	t.Parallel()

	overheads := &fakeOverheads{}
	service := application.NewOverheadService(overheads, theClock())

	_, err := service.Record(t.Context(), ports.RecordOverhead{
		BusinessID:    business.ID(77),
		Electricity:   120000,
		OvenHours:     40,
		Allocation:    1.5,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrInvalidAllocation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInvalidAllocation)
	}
	if len(overheads.written) != 0 {
		t.Errorf("the use case wrote %d overheads, want none", len(overheads.written))
	}
}

// TestTheOverheadWithAnInferredElectricityIsRefused proves that the use
// case refuses an overhead whose electricity was filled by the agent: the
// money of the house is never inferred, and a refused overhead leaves
// nothing behind.
func TestTheOverheadWithAnInferredElectricityIsRefused(t *testing.T) {
	t.Parallel()

	overheads := &fakeOverheads{}
	service := application.NewOverheadService(overheads, theClock())

	_, err := service.Record(t.Context(), ports.RecordOverhead{
		BusinessID:          business.ID(77),
		Electricity:         120000,
		ElectricityInferred: true,
		OvenHours:           40,
		Allocation:          0.5,
		ConfirmedWith:       "sí",
	})
	if !errors.Is(err, recipe.ErrInferredElectricity) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrInferredElectricity)
	}
	if len(overheads.written) != 0 {
		t.Errorf("the use case wrote %d overheads, want none", len(overheads.written))
	}
}

// TestTheOverheadOfTheDayOfToday proves that the day of an overhead the
// agent names none is the day of today: the day is one of the three things
// Paca may fill by herself.
func TestTheOverheadOfTheDayOfToday(t *testing.T) {
	t.Parallel()

	overheads := &fakeOverheads{}
	service := application.NewOverheadService(overheads, theClock())

	written, err := service.Record(t.Context(), ports.RecordOverhead{
		BusinessID:    business.ID(77),
		Electricity:   120000,
		OvenHours:     40,
		Allocation:    0.5,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the overhead was refused: %v", err)
	}

	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day of the overhead is %q, want the day of today", written.Day.String())
	}
}

// TestTheOverheadWithoutADatabaseSaysSo proves that a service without a
// database says so, and it says it in the only capability it has.
func TestTheOverheadWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewOverheadService(nil, theClock())

	_, err := service.Record(t.Context(), ports.RecordOverhead{
		BusinessID:    business.ID(77),
		Electricity:   120000,
		OvenHours:     40,
		Allocation:    0.5,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// theOverheadOfTheBusiness is the overhead the business declared: the rate
// of the house the overhead of a batch is valued with.
func theOverheadOfTheBusiness(t *testing.T) *fakeOverheads {
	t.Helper()

	return &fakeOverheads{held: []recipe.Overhead{theOverheadOfTheTest(t)}}
}
