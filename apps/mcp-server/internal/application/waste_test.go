package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The fake of the storage of the productions, read back.
//
// It holds the productions of the business.
type fakeProductionsRead struct {
	held []recipe.Production
	err  error
}

// List answers the productions of the business.
func (f *fakeProductionsRead) List(_ context.Context, _ business.ID) ([]recipe.Production, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.held, nil
}

// The fake of the storage of the units that are still in the book.
//
// It holds the units of the business.
type fakeUnitsInTheBook struct {
	held []recipe.ExpiringUnit
	err  error
}

// ListInTheBook answers the units of the business that are still in it.
func (f *fakeUnitsInTheBook) ListInTheBook(_ context.Context, _ business.ID) ([]recipe.ExpiringUnit, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.held, nil
}

// The fake of the storage of the wastes.
//
// It remembers the waste it wrote.
type fakeWastes struct {
	written []recipe.Waste
	err     error
}

// Record writes the waste and remembers it.
func (f *fakeWastes) Record(_ context.Context, written recipe.Waste) (recipe.Waste, error) {
	if f.err != nil {
		return recipe.Waste{}, f.err
	}

	written.ID = recipe.WasteID(4)
	f.written = append(f.written, written)

	return written, nil
}

// The fake of the transaction of the storage of the waste.
//
// It keeps what the work wrote, and it gives it back when the work fails:
// a transaction that rolls back leaves the storage as it was.
type fakeWasteWork struct {
	wastes *fakeWastes
	book   *fakeBook
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakeWasteWork) Do(ctx context.Context, work func(context.Context) error) error {
	wastes := len(f.wastes.written)
	movements := len(f.book.recorded)

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.wastes.written = f.wastes.written[:wastes]
	f.book.recorded = f.book.recorded[:movements]

	return err
}

// theProductionOfTheWaste is the production the owner made: ten units of
// bread that cost ten thousand pesos.
func theProductionOfTheWaste() recipe.Production {
	units := make([]recipe.ProductionUnit, 0, 10)

	for position := 1; position <= 10; position++ {
		units = append(units, recipe.ProductionUnit{
			ID:        recipe.UnitID(int64(position)),
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
		Day:        ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
		GramsIn:    900,
		Money:      10000,
		Units:      units,
	}
}

// theUnitsOfTheWaste are the units of the production that are still in the
// book.
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
			ProducedOn: ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
		})
	}

	return units
}

// theAccountsOfTheWaste are the accounts of the book of the bakery: the
// inventory and the waste.
func theAccountsOfTheWaste() []account.Account {
	return []account.Account{
		{ID: account.ID(3), BusinessID: business.ID(77), Kind: account.KindAsset, Name: "Inventario"}, //nolint:misspell // "Inventario" is the name the owner reads in their book.
		{ID: account.ID(5), BusinessID: business.ID(77), Kind: account.KindExpense, Name: "Merma"},
	}
}

// theWasteService builds the use case of the waste with the fakes of the
// test.
func theWasteService(
	t *testing.T,
	productions *fakeProductionsRead,
	units *fakeUnitsInTheBook,
	wastes *fakeWastes,
	book *fakeBook,
) *application.WasteService {
	t.Helper()

	return application.NewWasteService(
		productions,
		units,
		&fakeAccounts{existing: theAccountsOfTheWaste()},
		book,
		wastes,
		&fakeWasteWork{wastes: wastes, book: book},
		theClock(),
	)
}

// TestTheWasteIsWrittenWithTheProductTheQuantityAndTheReason proves that
// the use case writes the waste the owner confirmed: the product, the
// quantity, the unit, the reason and her word.
func TestTheWasteIsWrittenWithTheProductTheQuantityAndTheReason(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheWaste()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheWaste()}
	wastes := &fakeWastes{}
	book := &fakeBook{}
	service := theWasteService(t, productions, units, wastes, book)

	waste, err := service.Record(t.Context(), ports.RecordWaste{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Reason:        "burnt",
		Utterance:     "se me quemaron tres panes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the waste was refused: %v", err)
	}

	if len(wastes.written) != 1 {
		t.Fatalf("the use case wrote %d wastes, want 1", len(wastes.written))
	}

	written := wastes.written[0]
	if written.RecipeID != recipe.ID(9) {
		t.Errorf("the product of the waste is %d, want %d", written.RecipeID, recipe.ID(9))
	}

	if written.Quantity != 3 {
		t.Errorf("the quantity of the waste is %v, want 3", written.Quantity)
	}

	if written.Unit != "unidad" {
		t.Errorf("the unit of the waste is %q, want %q", written.Unit, "unidad")
	}

	if written.Reason != recipe.WasteBurnt {
		t.Errorf("the reason of the waste is %q, want %q", written.Reason, recipe.WasteBurnt)
	}

	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day of the waste is %q, want the day of today", written.Day.String())
	}

	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the owner is %q, want %q", written.ConfirmedWith, "sí")
	}

	if waste.ID != recipe.WasteID(4) {
		t.Errorf("the identity of the waste is %d, want 4", waste.ID)
	}
}

// TestTheWasteMovesTheMoneyOfTheWaste proves that the use case writes the
// movement of the money of the waste: the money lands in the account of the
// waste and it comes from the inventory, and it is never lost from the book.
func TestTheWasteMovesTheMoneyOfTheWaste(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheWaste()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheWaste()}
	wastes := &fakeWastes{}
	book := &fakeBook{}
	service := theWasteService(t, productions, units, wastes, book)

	_, err := service.Record(t.Context(), ports.RecordWaste{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Reason:        "burnt",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the waste was refused: %v", err)
	}

	if len(book.recorded) != 1 {
		t.Fatalf("the use case wrote %d movements, want 1", len(book.recorded))
	}

	movement := book.recorded[0]
	if movement.DebitAccountID != account.ID(5) {
		t.Errorf("the money lands in the account %d, want the waste", movement.DebitAccountID)
	}

	if movement.CreditAccountID != account.ID(3) {
		t.Errorf("the money comes from the account %d, want the inventory", movement.CreditAccountID)
	}

	if movement.Amount != 3000 {
		t.Errorf("the movement carries %d pesos, want %d", movement.Amount, 3000)
	}
}

// TestTheWasteIsRefusedWithoutAReason proves that the use case writes
// nothing when the reason is not one of the four of the book: the reason is
// what a later analysis reads.
func TestTheWasteIsRefusedWithoutAReason(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheWaste()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheWaste()}
	wastes := &fakeWastes{}
	book := &fakeBook{}
	service := theWasteService(t, productions, units, wastes, book)

	_, err := service.Record(t.Context(), ports.RecordWaste{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Reason:        "roto",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrUnknownWasteReason) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownWasteReason)
	}

	if len(wastes.written) != 0 {
		t.Errorf("the use case wrote %d wastes, want none", len(wastes.written))
	}

	if len(book.recorded) != 0 {
		t.Errorf("the use case wrote %d movements, want none", len(book.recorded))
	}
}

// TestTheWasteIsRefusedWithoutTheWordOfTheOwner proves that the use case
// writes nothing when the waste carries no word: the waste is written with
// the confirmation of the owner.
func TestTheWasteIsRefusedWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheWaste()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheWaste()}
	wastes := &fakeWastes{}
	book := &fakeBook{}
	service := theWasteService(t, productions, units, wastes, book)

	_, err := service.Record(t.Context(), ports.RecordWaste{
		BusinessID: business.ID(77),
		RecipeName: "Pan de molde",
		Quantity:   3,
		Reason:     "burnt",
	})
	if !errors.Is(err, recipe.ErrMissingWasteConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingWasteConfirmation)
	}

	if len(wastes.written) != 0 {
		t.Errorf("the use case wrote %d wastes, want none", len(wastes.written))
	}
}

// TestTheWasteIsRefusedWhenTheBookHoldsNoUnit proves that the use case
// writes nothing when the book holds no unit of the product: the book never
// writes a waste of nothing.
func TestTheWasteIsRefusedWhenTheBookHoldsNoUnit(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheWaste()}}
	units := &fakeUnitsInTheBook{}
	wastes := &fakeWastes{}
	book := &fakeBook{}
	service := theWasteService(t, productions, units, wastes, book)

	_, err := service.Record(t.Context(), ports.RecordWaste{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Reason:        "burnt",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrNotEnoughUnits) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotEnoughUnits)
	}

	if len(wastes.written) != 0 {
		t.Errorf("the use case wrote %d wastes, want none", len(wastes.written))
	}
}

// TestTheWasteIsRefusedWhenTheDayIsNotADay proves that the use case writes
// nothing when the day the owner named is not a day of the book.
func TestTheWasteIsRefusedWhenTheDayIsNotADay(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheWaste()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheWaste()}
	wastes := &fakeWastes{}
	book := &fakeBook{}
	service := theWasteService(t, productions, units, wastes, book)

	_, err := service.Record(t.Context(), ports.RecordWaste{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Quantity:      3,
		Reason:        "burnt",
		Day:           "ayer",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, ledger.ErrInvalidDate) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidDate)
	}

	if len(wastes.written) != 0 {
		t.Errorf("the use case wrote %d wastes, want none", len(wastes.written))
	}
}
