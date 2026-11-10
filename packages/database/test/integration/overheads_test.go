//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/user"
)

// overheadsRepository returns the repository of the overhead, bound to the
// database of the suite.
func overheadsRepository(t *testing.T) *postgres.OverheadRepository {
	t.Helper()

	repo, err := postgres.NewOverheadRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the overhead could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheOverhead opens the business that owns the overhead of the
// test.
func theBakeryOfTheOverhead(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Overhead")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theOverheadOfTheTest builds the overhead the tests of the table write:
// the electricity of the house, the hours the oven ran and the share of the
// electricity that belongs to the oven.
func theOverheadOfTheTest(bakery business.Business) recipe.Overhead {
	built, err := recipe.NewOverhead(bakery.ID, recipe.OverheadDefinition{
		Electricity:   ledger.Amount(120000),
		OvenHours:     40,
		Allocation:    0.5,
		Day:           ledger.NewDate(time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC)),
		Utterance:     "Pagué la luz del mes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		panic("the overhead of the test could not be built: " + err.Error())
	}

	return built
}

// TestTheOverheadIsKeptWithItsElectricityAndItsAllocation proves the rules
// of the table against a real database: the overhead is written with the
// electricity of the house, the hours of the oven, the allocation, the day
// it was declared and the word of the owner, and it is read back as it was
// written.
func TestTheOverheadIsKeptWithItsElectricityAndItsAllocation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := overheadsRepository(t)
	bakery := theBakeryOfTheOverhead(t, ctx)

	written, err := repo.Record(ctx, theOverheadOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the overhead could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the overhead has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the overhead is %d, want %d", written.BusinessID, bakery.ID)
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
	if written.Utterance != "Pagué la luz del mes" {
		t.Errorf("the sentence of the overhead is %q, want the one of the owner", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if written.CreatedAt.IsZero() {
		t.Error("the overhead carries no moment of creation, want the one the database filled")
	}
}

// TestTheOverheadIsReadBackWithItsElectricity proves that the overhead of a
// business is read back whole: the electricity of the house, the hours of
// the oven, the allocation and the money of the overhead.
func TestTheOverheadIsReadBackWithItsElectricity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := overheadsRepository(t)
	bakery := theBakeryOfTheOverhead(t, ctx)

	written, err := repo.Record(ctx, theOverheadOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the overhead could not be written: %v", err)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the overhead could not be read: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the bakery holds %d overheads, want 1", len(held))
	}
	if held[0].ID != written.ID {
		t.Errorf("the overhead of the list is %d, want %d", held[0].ID, written.ID)
	}
	if held[0].Electricity != written.Electricity {
		t.Errorf("the electricity of the house is %d, want %d", held[0].Electricity, written.Electricity)
	}
	if held[0].OvenHours != written.OvenHours {
		t.Errorf("the hours of the oven are %v, want %v", held[0].OvenHours, written.OvenHours)
	}
	if held[0].Allocation != written.Allocation {
		t.Errorf("the allocation is %v, want %v", held[0].Allocation, written.Allocation)
	}
	if held[0].Money != written.Money {
		t.Errorf("the money of the overhead is %d, want %d", held[0].Money, written.Money)
	}
}

// TestTheTableRefusesTheOverheadWithoutElectricity proves the rule of the
// domain against a real database: the money of the house is declared by the
// owner, and it is positive.
func TestTheTableRefusesTheOverheadWithoutElectricity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheOverhead(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO overheads (business_id, day, electricity, oven_hours, allocation, confirmed_with)
		 VALUES ($1, '2026-03-15', 0, 40, 0.5, 'sí')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the database stored an overhead without electricity, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "overheads_electricity_positive_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "overheads_electricity_positive_check")
	}
}

// TestTheTableRefusesTheOverheadWithoutTheHoursOfTheOven proves the rule
// of the domain against a real database: without the hours of the oven the
// money of the oven has no rate.
func TestTheTableRefusesTheOverheadWithoutTheHoursOfTheOven(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheOverhead(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO overheads (business_id, day, electricity, oven_hours, allocation, confirmed_with)
		 VALUES ($1, '2026-03-15', 120000, 0, 0.5, 'sí')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the database stored an overhead without the hours of the oven, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "overheads_oven_hours_positive_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "overheads_oven_hours_positive_check")
	}
}

// TestTheTableRefusesTheAllocationThatIsNotAShare proves the rule of the
// domain against a real database: the allocation is the share of the
// electricity that belongs to the oven, and it is a share.
func TestTheTableRefusesTheAllocationThatIsNotAShare(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheOverhead(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO overheads (business_id, day, electricity, oven_hours, allocation, confirmed_with)
		 VALUES ($1, '2026-03-15', 120000, 40, 1.5, 'sí')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the database stored an overhead with an allocation that is not a share, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "overheads_allocation_share_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "overheads_allocation_share_check")
	}
}

// TestTheTableRefusesTheOverheadWithoutTheWordOfTheOwner proves the rule
// of the domain against a real database: the overhead is written with the
// word of the owner, and the word is kept with the record.
func TestTheTableRefusesTheOverheadWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheOverhead(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO overheads (business_id, day, electricity, oven_hours, allocation, confirmed_with)
		 VALUES ($1, '2026-03-15', 120000, 40, 0.5, '')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the database stored an overhead without the word of the owner, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "overheads_confirmed_with_not_empty_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "overheads_confirmed_with_not_empty_check")
	}
}

// TestTheOverheadOfABusinessThatIsNotThereIsRefused proves that the
// storage refuses an overhead of a business that does not exist, and that
// it never answers an overhead of nothing.
func TestTheOverheadOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := overheadsRepository(t)

	_, err := repo.Record(ctx, theOverheadOfTheTest(business.Business{ID: business.ID(999999)}))
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}
}
