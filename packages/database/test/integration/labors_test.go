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

// laborsRepository returns the repository of the labor, bound to the
// database of the suite.
func laborsRepository(t *testing.T) *postgres.LaborRepository {
	t.Helper()

	repo, err := postgres.NewLaborRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the labor could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheLabor opens the business that owns the labor of the test.
func theBakeryOfTheLabor(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Labor")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theLaborOfTheTest builds the labor the tests of the table write: two
// hours with the hands, one in the oven and half an hour of waiting, at
// twenty thousand pesos the hour.
func theLaborOfTheTest(bakery business.Business) recipe.Labor {
	hours, err := recipe.NewHours(2, 1, 0.5)
	if err != nil {
		panic("the hours of the test could not be built: " + err.Error())
	}

	built, err := recipe.NewLabor(bakery.ID, recipe.LaborDefinition{
		Hours:         hours,
		HourValue:     ledger.Amount(20000),
		Day:           ledger.NewDate(time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC)),
		Utterance:     "Hoy trabajé tres horas y media en la panadería",
		ConfirmedWith: "sí",
	})
	if err != nil {
		panic("the labor of the test could not be built: " + err.Error())
	}

	return built
}

// TestTheLaborIsKeptWithItsHoursAndItsValueOfTheHour proves the rules of
// the table against a real database: the labor is written with its hours
// in the three kinds, the value of the hour, the day it was worked and the
// word of the owner, and it is read back as it was written.
func TestTheLaborIsKeptWithItsHoursAndItsValueOfTheHour(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := laborsRepository(t)
	bakery := theBakeryOfTheLabor(t, ctx)

	written, err := repo.Record(ctx, theLaborOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the labor could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the labor has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the labor is %d, want %d", written.BusinessID, bakery.ID)
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
	if written.Utterance != "Hoy trabajé tres horas y media en la panadería" {
		t.Errorf("the sentence of the labor is %q, want the one of the owner", written.Utterance)
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if written.CreatedAt.IsZero() {
		t.Error("the labor carries no moment of creation, want the one the database filled")
	}
}

// TestTheLaborIsReadBackWithItsHours proves that the labor of a business
// is read back whole: the hours in the three kinds, the value of the hour
// and the money of the labor.
func TestTheLaborIsReadBackWithItsHours(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := laborsRepository(t)
	bakery := theBakeryOfTheLabor(t, ctx)

	written, err := repo.Record(ctx, theLaborOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the labor could not be written: %v", err)
	}

	held, err := repo.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the labor could not be read: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the bakery holds %d labors, want 1", len(held))
	}
	if held[0].ID != written.ID {
		t.Errorf("the labor of the list is %d, want %d", held[0].ID, written.ID)
	}
	if held[0].Hours != written.Hours {
		t.Errorf("the hours of the labor are %+v, want %+v", held[0].Hours, written.Hours)
	}
	if held[0].HourValue != written.HourValue {
		t.Errorf("the value of the hour is %d, want %d", held[0].HourValue, written.HourValue)
	}
}

// TestTheTableRefusesTheLaborWithoutHours proves the rule of the domain
// against a real database: the hours of the owner are the fact, and a labor
// without hours is nothing.
func TestTheTableRefusesTheLaborWithoutHours(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLabor(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO labors (business_id, day, hands_on_hours, oven_hours, waiting_hours, hour_value, confirmed_with)
		 VALUES ($1, '2026-03-15', 0, 0, 0, 20000, 'sí')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the database stored a labor without hours, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "labors_hours_present_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "labors_hours_present_check")
	}
}

// TestTheTableRefusesTheLaborWithNegativeHours proves the rule of the
// domain against a real database: the hours of the labor are not negative,
// because a negative number of hours is a number nobody worked.
func TestTheTableRefusesTheLaborWithNegativeHours(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLabor(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO labors (business_id, day, hands_on_hours, oven_hours, waiting_hours, hour_value, confirmed_with)
		 VALUES ($1, '2026-03-15', -1, 0, 0, 20000, 'sí')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the database stored a labor with negative hours, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "labors_hours_not_negative_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "labors_hours_not_negative_check")
	}
}

// TestTheTableRefusesTheLaborWithoutTheValueOfTheHour proves the rule of
// the domain against a real database: the value of the hour is declared by
// the owner, and it is positive.
func TestTheTableRefusesTheLaborWithoutTheValueOfTheHour(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLabor(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO labors (business_id, day, hands_on_hours, oven_hours, waiting_hours, hour_value, confirmed_with)
		 VALUES ($1, '2026-03-15', 2, 0, 0, 0, 'sí')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the database stored a labor without the value of the hour, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "labors_hour_value_positive_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "labors_hour_value_positive_check")
	}
}

// TestTheTableRefusesTheLaborWithoutTheWordOfTheOwner proves the rule of
// the domain against a real database: the labor is written with the word of
// the owner, and the word is kept with the record.
func TestTheTableRefusesTheLaborWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheLabor(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO labors (business_id, day, hands_on_hours, oven_hours, waiting_hours, hour_value, confirmed_with)
		 VALUES ($1, '2026-03-15', 2, 0, 0, 20000, '')`,
		int64(bakery.ID),
	)
	if err == nil {
		t.Fatal("the database stored a labor without the word of the owner, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "labors_confirmed_with_not_empty_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "labors_confirmed_with_not_empty_check")
	}
}

// TestTheLaborOfABusinessThatIsNotThereIsRefused proves that the storage
// refuses a labor of a business that does not exist, and that it never
// answers a labor of nothing.
func TestTheLaborOfABusinessThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := laborsRepository(t)

	_, err := repo.Record(ctx, theLaborOfTheTest(business.Business{ID: business.ID(999999)}))
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}
}
