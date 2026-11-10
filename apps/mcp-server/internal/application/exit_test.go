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

// The fake of the storage of the exits.
//
// It remembers the exit it wrote.
type fakeExits struct {
	written []recipe.Exit
	err     error
}

// Record writes the exit and remembers it.
func (f *fakeExits) Record(_ context.Context, written recipe.Exit) (recipe.Exit, error) {
	if f.err != nil {
		return recipe.Exit{}, f.err
	}

	written.ID = recipe.ExitID(6)
	f.written = append(f.written, written)

	return written, nil
}

// The fake of the transaction of the storage of the exit.
//
// It keeps what the work wrote, and it gives it back when the work fails:
// a transaction that rolls back leaves the storage as it was.
type fakeExitWork struct {
	exits    *fakeExits
	book     *fakeBook
	accounts *theBooksOfTheExit
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakeExitWork) Do(ctx context.Context, work func(context.Context) error) error {
	exits := len(f.exits.written)
	movements := len(f.book.recorded)

	opened := 0
	if f.accounts != nil {
		opened = len(f.accounts.created)
	}

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.exits.written = f.exits.written[:exits]
	f.book.recorded = f.book.recorded[:movements]

	if f.accounts != nil {
		f.accounts.created = f.accounts.created[:opened]
		f.accounts.held = f.accounts.held[:opened]
	}

	return err
}

// The fake of the storage of the accounts of the tests of the exit.
//
// It opens the account the book of the business does not carry, the way
// the storage opens it.
type theBooksOfTheExit struct {
	held    []account.Account
	created []account.Account
	next    account.ID
	err     error
}

// Create writes the account and remembers it.
func (b *theBooksOfTheExit) Create(_ context.Context, written account.Account) (account.Account, error) {
	if b.err != nil {
		return account.Account{}, b.err
	}

	for _, one := range b.held {
		if one.Name == written.Name {
			return account.Account{}, account.ErrAlreadyExists
		}
	}

	b.next++
	written.ID = b.next
	b.created = append(b.created, written)
	b.held = append(b.held, written)

	return written, nil
}

// CreateMany writes the accounts of the book.
func (b *theBooksOfTheExit) CreateMany(_ context.Context, accounts []account.Account) ([]account.Account, error) {
	return nil, errors.New("the test does not write the accounts of the book")
}

// Find answers the account of that identity.
func (b *theBooksOfTheExit) Find(_ context.Context, id account.ID) (account.Account, error) {
	for _, one := range b.held {
		if one.ID == id {
			return one, nil
		}
	}

	return account.Account{}, account.ErrNotFound
}

// List answers the accounts of the business.
func (b *theBooksOfTheExit) List(_ context.Context, _ business.ID) ([]account.Account, error) {
	if b.err != nil {
		return nil, b.err
	}

	return b.held, nil
}

// theProductionOfTheExit is the production the owner made: ten units of
// bread that cost ten thousand pesos.
func theProductionOfTheExit() recipe.Production {
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

// theUnitsOfTheExit are the units of the production that are still in the
// book.
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
			ProducedOn: ledger.NewDate(time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)),
		})
	}

	return units
}

// theAccountsOfTheExit are the accounts of the book of the bakery: the
// cash, the inventory, the sales, the cost of the sales and the donation.
func theAccountsOfTheExit() []account.Account {
	return []account.Account{
		{ID: account.ID(1), BusinessID: business.ID(77), Kind: account.KindAsset, Name: "Efectivo"},
		{ID: account.ID(3), BusinessID: business.ID(77), Kind: account.KindAsset, Name: "Inventario"}, //nolint:misspell // "Inventario" is the name the owner reads in their book.
		{ID: account.ID(4), BusinessID: business.ID(77), Kind: account.KindIncome, Name: "Ventas"},
		{ID: account.ID(5), BusinessID: business.ID(77), Kind: account.KindExpense, Name: "Costo de ventas"},
		{ID: account.ID(6), BusinessID: business.ID(77), Kind: account.KindExpense, Name: "Donación"}, //nolint:misspell // "Donación" is the name the owner reads in their book.
	}
}

// theExitService builds the use case of the exit with the fakes of the
// test.
func theExitService(
	t *testing.T,
	productions *fakeProductionsRead,
	units *fakeUnitsInTheBook,
	exits *fakeExits,
	book *fakeBook,
) *application.ExitService {
	t.Helper()

	return application.NewExitService(
		productions,
		units,
		&fakeAccounts{existing: theAccountsOfTheExit()},
		book,
		exits,
		&fakeExitWork{exits: exits, book: book},
		theClock(),
	)
}

// TestTheExitIsWrittenWithTheProductTheQuantityAndTheKind proves that the
// use case writes the exit the owner confirmed: the product, the quantity,
// the unit, the kind of the exit and her word.
func TestTheExitIsWrittenWithTheProductTheQuantityAndTheKind(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := theExitService(t, productions, units, exits, book)

	exit, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "consumption",
		Quantity:      3,
		Unit:          "unidad",
		Utterance:     "me comí tres panes",
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if len(exits.written) != 1 {
		t.Fatalf("the use case wrote %d exits, want 1", len(exits.written))
	}

	written := exits.written[0]
	if written.Kind != recipe.ExitConsumption {
		t.Errorf("the kind of the exit is %q, want %q", written.Kind, recipe.ExitConsumption)
	}

	if written.RecipeID != recipe.ID(9) {
		t.Errorf("the product of the exit is %d, want %d", written.RecipeID, recipe.ID(9))
	}

	if written.Quantity != 3 {
		t.Errorf("the quantity of the exit is %v, want 3", written.Quantity)
	}

	if written.Unit != "unidad" {
		t.Errorf("the unit of the exit is %q, want %q", written.Unit, "unidad")
	}

	if written.Cost != 3000 {
		t.Errorf("the cost of the exit is %d, want 3000", written.Cost)
	}

	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day of the exit is %q, want the day of today", written.Day.String())
	}

	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the owner is %q, want %q", written.ConfirmedWith, "sí")
	}

	if written.TaxTreatment != recipe.TaxPendingTheProfessional {
		t.Errorf("the tax treatment of the exit is %q, want %q",
			written.TaxTreatment, recipe.TaxPendingTheProfessional)
	}

	if exit.ID != recipe.ExitID(6) {
		t.Errorf("the identity of the exit is %d, want 6", exit.ID)
	}
}

// TestTheExitMovesNoMoneyAndReleasesTheCost proves that the use case
// writes one movement only: the cost of the units that left the book
// leaves the inventory and lands in the account of the cost of the sales,
// and no money moves in.
func TestTheExitMovesNoMoneyAndReleasesTheCost(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := theExitService(t, productions, units, exits, book)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "consumption",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if len(book.recorded) != 1 {
		t.Fatalf("the use case wrote %d movements, want 1", len(book.recorded))
	}

	cost := book.recorded[0]
	if cost.DebitAccountID != account.ID(5) {
		t.Errorf("the cost lands in the account %d, want the cost of the sales (5)",
			cost.DebitAccountID)
	}

	if cost.CreditAccountID != account.ID(3) {
		t.Errorf("the cost comes from the account %d, want the inventory (3)",
			cost.CreditAccountID)
	}

	if cost.Amount != 3000 {
		t.Errorf("the movement carries %d pesos, want 3000", cost.Amount)
	}
}

// TestTheDonationLandsInTheSocialExpenseOfTheBusiness proves that the use
// case writes the cost of the donation in the account of the donation: the
// donation is a social expense of the business.
func TestTheDonationLandsInTheSocialExpenseOfTheBusiness(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := theExitService(t, productions, units, exits, book)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "donation",
		Quantity:      2,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if len(book.recorded) != 1 {
		t.Fatalf("the use case wrote %d movements, want 1", len(book.recorded))
	}

	if book.recorded[0].DebitAccountID != account.ID(6) {
		t.Errorf("the cost lands in the account %d, want the donation (6)",
			book.recorded[0].DebitAccountID)
	}

	if book.recorded[0].Amount != 2000 {
		t.Errorf("the movement carries %d pesos, want 2000", book.recorded[0].Amount)
	}
}

// TestTheExitAndItsMovementAreWrittenInOneTransaction proves that a
// failure inside the transaction leaves nothing behind: the exit and the
// movement of its cost land together or not at all.
func TestTheExitAndItsMovementAreWrittenInOneTransaction(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{err: errors.New("the book of the business is closed")}
	service := theExitService(t, productions, units, exits, book)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "consumption",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if err == nil {
		t.Fatal("the exit was written, want the failure of the book")
	}

	if len(exits.written) != 0 {
		t.Errorf("the use case wrote %d exits, want none", len(exits.written))
	}

	if len(book.recorded) != 0 {
		t.Errorf("the use case wrote %d movements, want none", len(book.recorded))
	}
}

// TestTheExitIsRefusedWithoutTheWordOfTheOwner proves that the use case
// writes nothing when the exit carries no word: the exit is written with
// the confirmation of the owner.
func TestTheExitIsRefusedWithoutTheWordOfTheOwner(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := theExitService(t, productions, units, exits, book)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID: business.ID(77),
		RecipeName: "Pan de molde",
		Kind:       "consumption",
		Quantity:   3,
	})
	if !errors.Is(err, recipe.ErrMissingExitConfirmation) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrMissingExitConfirmation)
	}

	if len(exits.written) != 0 {
		t.Errorf("the use case wrote %d exits, want none", len(exits.written))
	}
}

// TestTheExitOfAnotherKindIsRefused proves that the use case writes
// nothing when the kind of the exit is not one of the two of the book.
func TestTheExitOfAnotherKindIsRefused(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := theExitService(t, productions, units, exits, book)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "sale",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, recipe.ErrUnknownExitKind) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrUnknownExitKind)
	}

	if len(exits.written) != 0 {
		t.Errorf("the use case wrote %d exits, want none", len(exits.written))
	}
}

// TestTheExitWithoutADatabaseSaysSo proves that the use case refuses to
// write anything when it has no collaborators: a service with no database
// is not a service that writes.
func TestTheExitWithoutADatabaseSaysSo(t *testing.T) {
	t.Parallel()

	service := application.NewExitService(nil, nil, nil, nil, nil, nil, nil)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "consumption",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// TestTheExitOfADayThatIsNotADayIsRefused proves that the use case writes
// nothing when the day the agent names is not a day of the book.
func TestTheExitOfADayThatIsNotADayIsRefused(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := theExitService(t, productions, units, exits, book)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "consumption",
		Quantity:      3,
		Day:           "el día que no existe",
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, ledger.ErrInvalidDate) {
		t.Fatalf("the error is %v, want %v", err, ledger.ErrInvalidDate)
	}

	if len(exits.written) != 0 {
		t.Errorf("the use case wrote %d exits, want none", len(exits.written))
	}
}

// TestTheExitIsRefusedWhenTheProductionsCannotBeRead proves that the use
// case writes nothing when the productions of the business cannot be read:
// the cost of a unit is the money of the production it came out of.
func TestTheExitIsRefusedWhenTheProductionsCannotBeRead(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{err: errors.New("the productions cannot be read")}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := theExitService(t, productions, units, exits, book)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "consumption",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if err == nil {
		t.Fatal("the exit was written, want the failure of the productions")
	}

	if len(exits.written) != 0 {
		t.Errorf("the use case wrote %d exits, want none", len(exits.written))
	}
}

// TestTheExitIsRefusedWhenTheUnitsCannotBeRead proves that the use case
// writes nothing when the units of the business cannot be read: only the
// units that are still in the book can leave it.
func TestTheExitIsRefusedWhenTheUnitsCannotBeRead(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{err: errors.New("the units cannot be read")}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := theExitService(t, productions, units, exits, book)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "consumption",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if err == nil {
		t.Fatal("the exit was written, want the failure of the units")
	}

	if len(exits.written) != 0 {
		t.Errorf("the use case wrote %d exits, want none", len(exits.written))
	}
}

// TestTheBookThatLacksTheAccountOfTheExpenseOpensIt proves that the use
// case writes the exit when the book of the business was opened before the
// role of the expense existed: the book catches up with the seed of the
// book of a business, and the cost of the exit lands in the account it
// opens.
func TestTheBookThatLacksTheAccountOfTheExpenseOpensIt(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	accounts := &theBooksOfTheExit{held: []account.Account{
		{ID: account.ID(3), BusinessID: business.ID(77), Kind: account.KindAsset, Name: "Inventario"}, //nolint:misspell // "Inventario" is the name the owner reads in their book.
	}}
	service := application.NewExitService(
		productions,
		units,
		accounts,
		book,
		exits,
		&fakeExitWork{exits: exits, book: book, accounts: accounts},
		theClock(),
	)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "donation",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if len(accounts.created) != 1 {
		t.Fatalf("the book opened %d accounts, want 1", len(accounts.created))
	}

	opened := accounts.created[0]
	if opened.Name != "Donación" { //nolint:misspell // "Donación" is the name the person reads in their book.
		t.Errorf("the name of the account is %q, want %q", opened.Name, "Donación") //nolint:misspell // "Donación" is the name the person reads in their book.
	}

	if opened.Kind != account.KindExpense {
		t.Errorf("the kind of the account is %q, want %q", opened.Kind, account.KindExpense)
	}

	if len(exits.written) != 1 {
		t.Fatalf("the use case wrote %d exits, want 1", len(exits.written))
	}

	if len(book.recorded) != 1 {
		t.Fatalf("the use case wrote %d movements, want 1", len(book.recorded))
	}

	if book.recorded[0].DebitAccountID != opened.ID {
		t.Errorf("the cost lands in the account %d, want %d", book.recorded[0].DebitAccountID, opened.ID)
	}

	if book.recorded[0].CreditAccountID != account.ID(3) {
		t.Errorf("the cost comes from the account %d, want 3", book.recorded[0].CreditAccountID)
	}
}

// TestTheExitIsRefusedWhenTheAccountOfTheExpenseCannotBeOpened proves
// that the use case writes nothing when the book cannot open the account
// of the expense of the exit: the cost of the exit has nowhere to land.
func TestTheExitIsRefusedWhenTheAccountOfTheExpenseCannotBeOpened(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	accounts := &theBooksOfTheExit{
		held: []account.Account{
			{ID: account.ID(3), BusinessID: business.ID(77), Kind: account.KindAsset, Name: "Inventario"}, //nolint:misspell // "Inventario" is the name the person reads in their book.
		},
		err: account.ErrNoStorage,
	}
	service := application.NewExitService(
		productions,
		units,
		accounts,
		book,
		exits,
		&fakeExitWork{exits: exits, book: book, accounts: accounts},
		theClock(),
	)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "donation",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, account.ErrNoStorage) {
		t.Fatalf("the error is %v, want %v", err, account.ErrNoStorage)
	}

	if len(exits.written) != 0 {
		t.Errorf("the use case wrote %d exits, want none", len(exits.written))
	}

	if len(book.recorded) != 0 {
		t.Errorf("the use case wrote %d movements, want none", len(book.recorded))
	}
}

// TestTheExitIsRefusedWhenTheBookHoldsNoInventory proves that the use case
// writes nothing when the book of the business holds no inventory: the
// cost of the exit comes from the inventory.
func TestTheExitIsRefusedWhenTheBookHoldsNoInventory(t *testing.T) {
	t.Parallel()

	productions := &fakeProductionsRead{held: []recipe.Production{theProductionOfTheExit()}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := application.NewExitService(
		productions,
		units,
		&fakeAccounts{existing: []account.Account{
			{ID: account.ID(5), BusinessID: business.ID(77), Kind: account.KindExpense, Name: "Costo de ventas"},
		}},
		book,
		exits,
		&fakeExitWork{exits: exits, book: book},
		theClock(),
	)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "consumption",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, account.ErrNotFound)
	}

	if len(exits.written) != 0 {
		t.Errorf("the use case wrote %d exits, want none", len(exits.written))
	}
}

// TestTheExitOfNoCostMovesNoMoney proves that the use case writes no
// movement when the exit costs nothing: there is no money to move, and the
// book never writes a movement of nothing.
func TestTheExitOfNoCostMovesNoMoney(t *testing.T) {
	t.Parallel()

	production := theProductionOfTheExit()
	production.Money = 0

	productions := &fakeProductionsRead{held: []recipe.Production{production}}
	units := &fakeUnitsInTheBook{held: theUnitsOfTheExit()}
	exits := &fakeExits{}
	book := &fakeBook{}
	service := theExitService(t, productions, units, exits, book)

	_, err := service.Record(t.Context(), ports.RecordExit{
		BusinessID:    business.ID(77),
		RecipeName:    "Pan de molde",
		Kind:          "consumption",
		Quantity:      3,
		ConfirmedWith: "sí",
	})
	if err != nil {
		t.Fatalf("the exit was refused: %v", err)
	}

	if len(exits.written) != 1 {
		t.Fatalf("the use case wrote %d exits, want 1", len(exits.written))
	}

	if len(book.recorded) != 0 {
		t.Errorf("the use case wrote %d movements, want none", len(book.recorded))
	}
}
