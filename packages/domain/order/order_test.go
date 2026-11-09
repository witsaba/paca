package order

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
)

// theDayOfTheTest is a day of the book of the bakery.
func theDayOfTheTest(day string) ledger.Date {
	parsed, err := ledger.ParseDate(day)
	if err != nil {
		panic(err)
	}

	return parsed
}

// theClientOfTheTest is the party that orders: a person that holds the
// role of the client of the business of the test.
func theClientOfTheTest(t *testing.T) party.Party {
	t.Helper()

	built, err := party.New(
		business.ID(77),
		party.KindPerson,
		"Doña Rosalba",
		[]party.Role{party.RoleClient},
		"",
		"rosalba@example.com",
		"Calle 5 # 12-30",
		party.LegalHuman{},
	)
	if err != nil {
		t.Fatalf("the client was refused: %v", err)
	}

	built.ID = party.ID(10)

	return built
}

// theProviderOfTheTest is the party that does not order: it holds the role
// of the provider and not the role of the client.
func theProviderOfTheTest(t *testing.T) party.Party {
	t.Helper()

	built, err := party.New(
		business.ID(77),
		party.KindFarm,
		"Finca Laesperanza",
		[]party.Role{party.RoleProvider},
		"",
		"",
		"Vereda El Carmen",
		party.LegalHuman{},
	)
	if err != nil {
		t.Fatalf("the provider was refused: %v", err)
	}

	built.ID = party.ID(11)

	return built
}

// theProductOfTheTest is the product of the price list that the client
// orders.
func theProductOfTheTest(t *testing.T, own business.ID) product.Product {
	t.Helper()

	built, err := product.NewFromFacts(product.Facts{
		ID:         product.ID(5),
		BusinessID: own,
		RecipeID:   3,
		Name:       "Pan de molde",
	})
	if err != nil {
		t.Fatalf("the product was refused: %v", err)
	}

	return built
}

// theLinesOfTheTest are the lines of the order of the test.
func theLinesOfTheTest(t *testing.T, own business.ID) []LineDefinition {
	t.Helper()

	return []LineDefinition{
		{Product: theProductOfTheTest(t, own), Quantity: 12, Unit: "pan"},
	}
}

// TestNewBuildsTheOrderOfTheClient proves that the domain builds the order
// of the business: the party that ordered, the products it ordered with
// their quantities and their units, and the day it is due.
func TestNewBuildsTheOrderOfTheClient(t *testing.T) {
	t.Parallel()

	client := theClientOfTheTest(t)

	built, err := New(
		business.ID(77),
		client,
		theDayOfTheTest("2026-03-14"),
		theLinesOfTheTest(t, business.ID(77)),
	)
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	if built.ID != 0 {
		t.Errorf("the identity is %d, want 0: the storage generates it", built.ID)
	}
	if built.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", built.BusinessID)
	}
	if built.PartyID != party.ID(10) {
		t.Errorf("the party is %d, want 10", built.PartyID)
	}
	if built.PartyName != "Doña Rosalba" {
		t.Errorf("the name of the party is %q, want %q", built.PartyName, "Doña Rosalba")
	}
	if built.DueOn.String() != "2026-03-14" {
		t.Errorf("the day it is due is %q, want %q", built.DueOn.String(), "2026-03-14")
	}
	if len(built.Lines) != 1 {
		t.Fatalf("the order carries %d lines, want 1", len(built.Lines))
	}
	if built.Lines[0].ProductID != product.ID(5) {
		t.Errorf("the product of the line is %d, want 5", built.Lines[0].ProductID)
	}
	if built.Lines[0].ProductName != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", built.Lines[0].ProductName, "Pan de molde")
	}
	if built.Lines[0].Quantity != 12 {
		t.Errorf("the quantity is %d, want 12", built.Lines[0].Quantity)
	}
	if built.Lines[0].Unit != "pan" {
		t.Errorf("the unit is %q, want %q", built.Lines[0].Unit, "pan")
	}
}

// TestTheOrderThatWasNotDeliveredStaysOpen proves the rule of the domain:
// an order that was not delivered stays open, and it is the domain that
// writes it that way.
func TestTheOrderThatWasNotDeliveredStaysOpen(t *testing.T) {
	t.Parallel()

	built, err := New(
		business.ID(77),
		theClientOfTheTest(t),
		theDayOfTheTest("2026-03-14"),
		theLinesOfTheTest(t, business.ID(77)),
	)
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	if built.State != StateOpen {
		t.Errorf("the state of the order is %q, want %q", built.State, StateOpen)
	}
	if !built.IsOpen() {
		t.Error("the order that was not delivered is not open")
	}
}

// TestTheRoundOfTheDayCarriesTheOrdersThatAreDueOnIt proves the rule of
// the domain: the round of a day carries the orders that are due on it.
func TestTheRoundOfTheDayCarriesTheOrdersThatAreDueOnIt(t *testing.T) {
	t.Parallel()

	today := theDayOfTheTest("2026-03-14")
	tomorrow := theDayOfTheTest("2026-03-15")

	due, err := New(business.ID(77), theClientOfTheTest(t), today, theLinesOfTheTest(t, business.ID(77)))
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	later, err := New(business.ID(77), theClientOfTheTest(t), tomorrow, theLinesOfTheTest(t, business.ID(77)))
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	round := TheRound(today, []Order{due, later})

	if len(round.Orders) != 1 {
		t.Fatalf("the round carries %d orders, want 1", len(round.Orders))
	}
	if round.Orders[0].DueOn.String() != "2026-03-14" {
		t.Errorf("the order of the round is due on %q, want %q", round.Orders[0].DueOn.String(), "2026-03-14")
	}
	if round.Day.String() != "2026-03-14" {
		t.Errorf("the day of the round is %q, want %q", round.Day.String(), "2026-03-14")
	}
}

// TestTheRoundWithoutADayCarriesEveryOrder proves the rule of the domain:
// the book of the orders is read whole when the owner names no day.
func TestTheRoundWithoutADayCarriesEveryOrder(t *testing.T) {
	t.Parallel()

	today := theDayOfTheTest("2026-03-14")
	tomorrow := theDayOfTheTest("2026-03-15")

	due, err := New(business.ID(77), theClientOfTheTest(t), today, theLinesOfTheTest(t, business.ID(77)))
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	later, err := New(business.ID(77), theClientOfTheTest(t), tomorrow, theLinesOfTheTest(t, business.ID(77)))
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	round := TheRound(ledger.Date{}, []Order{due, later})

	if len(round.Orders) != 2 {
		t.Fatalf("the round carries %d orders, want 2", len(round.Orders))
	}
}

// TestTheRoundAnswersWhatIsStillOwed proves the rule of the domain: the
// round of the day answers what is still owed, and an order that was
// delivered is not owed.
func TestTheRoundAnswersWhatIsStillOwed(t *testing.T) {
	t.Parallel()

	today := theDayOfTheTest("2026-03-14")

	open, err := New(business.ID(77), theClientOfTheTest(t), today, theLinesOfTheTest(t, business.ID(77)))
	if err != nil {
		t.Fatalf("the order was refused: %v", err)
	}

	delivered, err := NewFromFacts(Facts{
		ID:         ID(2),
		BusinessID: business.ID(77),
		PartyID:    party.ID(10),
		PartyName:  "Doña Rosalba",
		DueOn:      today,
		Lines: []Line{
			{ProductID: product.ID(5), ProductName: "Pan de molde", Quantity: 12, Unit: "pan"},
		},
		State: StateDelivered,
	})
	if err != nil {
		t.Fatalf("the delivered order was refused: %v", err)
	}

	round := TheRound(today, []Order{open, delivered})

	owed := round.Owed()
	if len(owed) != 1 {
		t.Fatalf("the round owes %d orders, want 1", len(owed))
	}
	if owed[0].ID != open.ID {
		t.Errorf("the order that is owed is %d, want %d", owed[0].ID, open.ID)
	}
}

// TestNewRefusesThePartyThatIsNotAClient proves the rule of the domain:
// the order is the bread the bakery owes, and the bakery owes it to a
// client.
func TestNewRefusesThePartyThatIsNotAClient(t *testing.T) {
	t.Parallel()

	_, err := New(
		business.ID(77),
		theProviderOfTheTest(t),
		theDayOfTheTest("2026-03-14"),
		theLinesOfTheTest(t, business.ID(77)),
	)
	if err == nil {
		t.Fatal("the order of a party that is not a client was built")
	}
	if !errors.Is(err, ErrNotAClient) {
		t.Fatalf("the error is %v, want %v", err, ErrNotAClient)
	}
}

// TestNewRefusesThePartyOfAnotherBusiness proves the rule of the domain:
// the party of an order is a party of the business that owns the order.
func TestNewRefusesThePartyOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	other := theClientOfTheTest(t)
	other.BusinessID = business.ID(88)

	_, err := New(
		business.ID(77),
		other,
		theDayOfTheTest("2026-03-14"),
		theLinesOfTheTest(t, business.ID(77)),
	)
	if err == nil {
		t.Fatal("the order of a party of another business was built")
	}
	if !errors.Is(err, ErrPartyOfAnotherBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrPartyOfAnotherBusiness)
	}
}

// TestNewRefusesTheProductOfAnotherBusiness proves the rule of the domain:
// the products of an order are the products of the price list of the
// business that owns the order.
func TestNewRefusesTheProductOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	lines := []LineDefinition{
		{Product: theProductOfTheTest(t, business.ID(88)), Quantity: 3, Unit: "pan"},
	}

	_, err := New(
		business.ID(77),
		theClientOfTheTest(t),
		theDayOfTheTest("2026-03-14"),
		lines,
	)
	if err == nil {
		t.Fatal("the order of a product of another business was built")
	}
	if !errors.Is(err, ErrProductOfAnotherBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrProductOfAnotherBusiness)
	}
}

// TestNewRefusesAQuantityThatIsNotAWholeNumberOfUnits proves the rule of
// the domain: the order counts whole units of a product, and it never
// splits a unit.
func TestNewRefusesAQuantityThatIsNotAWholeNumberOfUnits(t *testing.T) {
	t.Parallel()

	for _, quantity := range []int{0, -3} {
		lines := []LineDefinition{
			{Product: theProductOfTheTest(t, business.ID(77)), Quantity: quantity, Unit: "pan"},
		}

		_, err := New(
			business.ID(77),
			theClientOfTheTest(t),
			theDayOfTheTest("2026-03-14"),
			lines,
		)
		if err == nil {
			t.Fatalf("the order of %d units was built", quantity)
		}
		if !errors.Is(err, ErrInvalidQuantity) {
			t.Fatalf("the error is %v, want %v", err, ErrInvalidQuantity)
		}
	}
}

// TestNewRefusesTheLineWithoutItsUnit proves the rule of the domain: the
// line of an order carries the unit the owner said.
func TestNewRefusesTheLineWithoutItsUnit(t *testing.T) {
	t.Parallel()

	lines := []LineDefinition{
		{Product: theProductOfTheTest(t, business.ID(77)), Quantity: 3, Unit: "  "},
	}

	_, err := New(
		business.ID(77),
		theClientOfTheTest(t),
		theDayOfTheTest("2026-03-14"),
		lines,
	)
	if err == nil {
		t.Fatal("the order of a line without its unit was built")
	}
	if !errors.Is(err, ErrMissingUnit) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingUnit)
	}
}

// TestNewRefusesTheOrderWithoutLines proves the rule of the domain: an
// order without a line orders nothing.
func TestNewRefusesTheOrderWithoutLines(t *testing.T) {
	t.Parallel()

	_, err := New(
		business.ID(77),
		theClientOfTheTest(t),
		theDayOfTheTest("2026-03-14"),
		nil,
	)
	if err == nil {
		t.Fatal("the order without lines was built")
	}
	if !errors.Is(err, ErrNoLines) {
		t.Fatalf("the error is %v, want %v", err, ErrNoLines)
	}
}

// TestNewRefusesTheOrderWithoutItsDay proves the rule of the domain: the
// order carries the day it is due.
func TestNewRefusesTheOrderWithoutItsDay(t *testing.T) {
	t.Parallel()

	_, err := New(
		business.ID(77),
		theClientOfTheTest(t),
		ledger.Date{},
		theLinesOfTheTest(t, business.ID(77)),
	)
	if err == nil {
		t.Fatal("the order without its day was built")
	}
	if !errors.Is(err, ErrMissingDay) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingDay)
	}
}

// TestNewRefusesTheOrderWithoutABusiness proves the rule of the domain:
// the order belongs to a business, and the book never invents one.
func TestNewRefusesTheOrderWithoutABusiness(t *testing.T) {
	t.Parallel()

	_, err := New(
		business.ID(0),
		theClientOfTheTest(t),
		theDayOfTheTest("2026-03-14"),
		theLinesOfTheTest(t, business.ID(77)),
	)
	if err == nil {
		t.Fatal("the order of no business was built")
	}
	if !errors.Is(err, ErrMissingBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingBusiness)
	}
}

// TestNewRefusesTheOrderWithoutAParty proves the rule of the domain: the
// order names the party that ordered.
func TestNewRefusesTheOrderWithoutAParty(t *testing.T) {
	t.Parallel()

	client := theClientOfTheTest(t)
	client.ID = 0

	_, err := New(
		business.ID(77),
		client,
		theDayOfTheTest("2026-03-14"),
		theLinesOfTheTest(t, business.ID(77)),
	)
	if err == nil {
		t.Fatal("the order without a party was built")
	}
	if !errors.Is(err, ErrMissingParty) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingParty)
	}
}

// TestOwnedByRefusesABusinessOfAnotherKind proves the rule of the domain:
// the orders belong to a business of its own, the one that sells the
// bread.
func TestOwnedByRefusesABusinessOfAnotherKind(t *testing.T) {
	t.Parallel()

	if err := OwnedBy(business.KindBusiness); err != nil {
		t.Fatalf("the business of the bakery was refused: %v", err)
	}

	err := OwnedBy(business.KindPersonal)
	if err == nil {
		t.Fatal("the business of a person was accepted")
	}
	if !errors.Is(err, ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrWrongBusiness)
	}
}

// TestParseStateReadsTheStatesOfTheBook proves that the domain reads the
// state of an order, and that it refuses the one it does not know.
func TestParseStateReadsTheStatesOfTheBook(t *testing.T) {
	t.Parallel()

	parsed, err := ParseState("open")
	if err != nil {
		t.Fatalf("the state was refused: %v", err)
	}
	if parsed != StateOpen {
		t.Errorf("the state is %q, want %q", parsed, StateOpen)
	}

	parsed, err = ParseState("delivered")
	if err != nil {
		t.Fatalf("the state was refused: %v", err)
	}
	if parsed != StateDelivered {
		t.Errorf("the state is %q, want %q", parsed, StateDelivered)
	}

	_, err = ParseState("cancelled")
	if err == nil {
		t.Fatal("the state that the book does not know was accepted")
	}
	if !errors.Is(err, ErrUnknownState) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownState)
	}
}

// TestAllStatesNamesTheStatesOfTheBook proves that the list of the states
// names the two of them, in the order they are presented.
func TestAllStatesNamesTheStatesOfTheBook(t *testing.T) {
	t.Parallel()

	all := AllStates()
	if len(all) != 2 {
		t.Fatalf("the list holds %d states, want 2", len(all))
	}
	if all[0] != StateOpen {
		t.Errorf("the first state is %q, want %q", all[0], StateOpen)
	}
	if all[1] != StateDelivered {
		t.Errorf("the second state is %q, want %q", all[1], StateDelivered)
	}
}

// TestTheCallerOfTheListCannotChangeIt proves that the caller of the list
// receives a copy: the vocabulary of the domain never changes under it.
func TestTheCallerOfTheListCannotChangeIt(t *testing.T) {
	t.Parallel()

	all := AllStates()
	all[0] = StateDelivered

	if AllStates()[0] != StateOpen {
		t.Error("the list of the states changed under the caller")
	}
}

// TestNewFromFactsRefusesTheRowThatBreaksARule proves the rule of the
// domain: the row is read back through the domain, and the row never
// decides what an order is.
func TestNewFromFactsRefusesTheRowThatBreaksARule(t *testing.T) {
	t.Parallel()

	_, err := NewFromFacts(Facts{
		BusinessID: business.ID(77),
		PartyID:    party.ID(10),
		PartyName:  "Doña Rosalba",
		DueOn:      theDayOfTheTest("2026-03-14"),
		Lines: []Line{
			{ProductID: product.ID(5), ProductName: "Pan de molde", Quantity: 12, Unit: "pan"},
		},
		State: "cancelled",
	})
	if err == nil {
		t.Fatal("the row with a state the book does not know was read")
	}
	if !errors.Is(err, ErrUnknownState) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownState)
	}
}

// TestNewRefusesTheLineWithoutItsProduct proves the rule of the domain: the
// line of an order names a product of the price list of the business.
func TestNewRefusesTheLineWithoutItsProduct(t *testing.T) {
	t.Parallel()

	lines := []LineDefinition{
		{Product: product.Product{}, Quantity: 3, Unit: "pan"},
	}

	_, err := New(
		business.ID(77),
		theClientOfTheTest(t),
		theDayOfTheTest("2026-03-14"),
		lines,
	)
	if err == nil {
		t.Fatal("the order of a line without its product was built")
	}
	if !errors.Is(err, ErrMissingProduct) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingProduct)
	}
}

// TestNewFromFactsRefusesTheRowThatIsNotWhole proves the rule of the
// domain: the row is read back through the domain, and a row that breaks a
// rule of it is refused.
func TestNewFromFactsRefusesTheRowThatIsNotWhole(t *testing.T) {
	t.Parallel()

	whole := Facts{
		BusinessID: business.ID(77),
		PartyID:    party.ID(10),
		PartyName:  "Doña Rosalba",
		DueOn:      theDayOfTheTest("2026-03-14"),
		Lines: []Line{
			{ProductID: product.ID(5), ProductName: "Pan de molde", Quantity: 12, Unit: "pan"},
		},
		State: StateOpen,
	}

	cases := []struct {
		name  string
		facts Facts
		want  error
	}{
		{
			name:  "the order of no business",
			facts: Facts{PartyID: 10, DueOn: whole.DueOn, Lines: whole.Lines, State: StateOpen},
			want:  ErrMissingBusiness,
		},
		{
			name:  "the order that names no party",
			facts: Facts{BusinessID: 77, DueOn: whole.DueOn, Lines: whole.Lines, State: StateOpen},
			want:  ErrMissingParty,
		},
		{
			name:  "the order that carries no day",
			facts: Facts{BusinessID: 77, PartyID: 10, Lines: whole.Lines, State: StateOpen},
			want:  ErrMissingDay,
		},
		{
			name:  "the order that carries no line",
			facts: Facts{BusinessID: 77, PartyID: 10, DueOn: whole.DueOn, State: StateOpen},
			want:  ErrNoLines,
		},
		{
			name: "the line that names no product",
			facts: Facts{
				BusinessID: 77,
				PartyID:    10,
				DueOn:      whole.DueOn,
				Lines:      []Line{{Quantity: 12, Unit: "pan"}},
				State:      StateOpen,
			},
			want: ErrMissingProduct,
		},
		{
			name: "the line that carries no units",
			facts: Facts{
				BusinessID: 77,
				PartyID:    10,
				DueOn:      whole.DueOn,
				Lines:      []Line{{ProductID: 5, Quantity: 0, Unit: "pan"}},
				State:      StateOpen,
			},
			want: ErrInvalidQuantity,
		},
		{
			name: "the line that carries no unit",
			facts: Facts{
				BusinessID: 77,
				PartyID:    10,
				DueOn:      whole.DueOn,
				Lines:      []Line{{ProductID: 5, Quantity: 12}},
				State:      StateOpen,
			},
			want: ErrMissingUnit,
		},
	}

	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewFromFacts(one.facts)
			if err == nil {
				t.Fatal("the row that is not whole was read")
			}
			if !errors.Is(err, one.want) {
				t.Fatalf("the error is %v, want %v", err, one.want)
			}
		})
	}
}

// TestNewFromFactsBuildsTheOrderOfTheRow proves that the domain builds the
// order the storage holds, with its lines and its state.
func TestNewFromFactsBuildsTheOrderOfTheRow(t *testing.T) {
	t.Parallel()

	built, err := NewFromFacts(Facts{
		ID:         ID(9),
		BusinessID: business.ID(77),
		PartyID:    party.ID(10),
		PartyName:  "  Doña Rosalba  ",
		DueOn:      theDayOfTheTest("2026-03-14"),
		Lines: []Line{
			{ProductID: product.ID(5), ProductName: "Pan de molde", Quantity: 12, Unit: "pan"},
		},
		State: StateOpen,
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if built.ID != ID(9) {
		t.Errorf("the identity is %d, want 9", built.ID)
	}
	if built.PartyName != "Doña Rosalba" {
		t.Errorf("the name of the party is %q, want %q", built.PartyName, "Doña Rosalba")
	}
	if built.State != StateOpen {
		t.Errorf("the state of the order is %q, want %q", built.State, StateOpen)
	}
	if len(built.Lines) != 1 || built.Lines[0].Unit != "pan" {
		t.Errorf("the lines of the order are %v, want the line of the pan", built.Lines)
	}
}

// TestErrorsNamesEveryError proves that the catalogue of the domain names
// every error it owns.
func TestErrorsNamesEveryError(t *testing.T) {
	t.Parallel()

	all := Errors()
	if len(all) != 15 {
		t.Fatalf("the catalogue holds %d errors, want 15", len(all))
	}

	for _, err := range all {
		if err == nil {
			t.Error("the catalogue holds an empty error")
		}
	}
}
