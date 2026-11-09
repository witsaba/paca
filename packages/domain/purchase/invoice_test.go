package purchase

import (
	"errors"
	"math"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/party"
)

// theProviderOfTheTest builds the provider that issues the invoices of the
// test: a party that holds the role of the provider.
func theProviderOfTheTest() party.Party {
	return party.Party{
		ID:         party.ID(10),
		BusinessID: business.ID(77),
		Kind:       party.KindFarm,
		Name:       "Finca La Esperanza",
		Roles:      []party.Role{party.RoleProvider},
	}
}

// theDayOfTheInvoice is the day the invoices of the test were issued.
func theDayOfTheInvoice() ledger.Date {
	day, err := ledger.ParseDate("2026-03-01")
	if err != nil {
		panic(err)
	}

	return day
}

// TestNewInvoiceBuildsTheInvoiceOfThePurchase proves that the domain builds
// the invoice of the purchase it came from: the provider that issued it,
// its number, its day, its base, its rate and its total.
func TestNewInvoiceBuildsTheInvoiceOfThePurchase(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	built, err := NewInvoice(business.ID(77), theProviderOfTheTest(), InvoiceDefinition{
		PurchaseID: ID(5),
		Number:     "  FV-001  ",
		Day:        theDayOfTheInvoice(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Total:      ledger.Amount(11900),
		Reference:  "CUFE-1234",
	})
	if err != nil {
		t.Fatalf("the invoice was refused: %v", err)
	}

	if built.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", built.BusinessID)
	}
	if built.ProviderID != party.ID(10) {
		t.Errorf("the provider is %d, want 10", built.ProviderID)
	}
	if built.PurchaseID != ID(5) {
		t.Errorf("the purchase is %d, want 5", built.PurchaseID)
	}
	if built.Number != "FV-001" {
		t.Errorf("the number is %q, want %q", built.Number, "FV-001")
	}
	if built.Day != theDayOfTheInvoice() {
		t.Errorf("the day is %v, want %v", built.Day, theDayOfTheInvoice())
	}
	if built.Base != ledger.Amount(10000) {
		t.Errorf("the base is %d, want 10000", built.Base)
	}
	if built.Rate != rate {
		t.Errorf("the rate is %d, want %d", built.Rate, rate)
	}
	if built.Tax != ledger.Amount(1900) {
		t.Errorf("the tax is %d, want 1900", built.Tax)
	}
	if built.Total != ledger.Amount(11900) {
		t.Errorf("the total is %d, want 11900", built.Total)
	}
	if built.Reference != "CUFE-1234" {
		t.Errorf("the reference is %q, want %q", built.Reference, "CUFE-1234")
	}
	if built.ID != 0 {
		t.Errorf("the identity is %d, want 0: the storage generates it", built.ID)
	}
}

// TestTheTaxOfTheInvoiceComesFromTheInvoiceAndIsNeverAssumed proves that
// the tax of an invoice is the one the invoice states: the rate it was
// issued with and the base it applies to. The book never assumes a rate,
// and the rate of the country is never read here.
func TestTheTaxOfTheInvoiceComesFromTheInvoiceAndIsNeverAssumed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		rate float64
		base ledger.Amount
		want ledger.Amount
	}{
		{name: "the rate of nineteen percent", rate: 19, base: 10000, want: 1900},
		{name: "the rate of nineteen and a half", rate: 19.5, base: 10000, want: 1950},
		{name: "the rate of five percent", rate: 5, base: 10000, want: 500},
		{name: "the rate of zero, the one the invoice states", rate: 0, base: 10000, want: 0},
		{name: "a base of one peso", rate: 19, base: 1, want: 0},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rate, err := NewRate(testCase.rate)
			if err != nil {
				t.Fatalf("the rate was refused: %v", err)
			}

			if got := TheTax(testCase.base, rate); got != testCase.want {
				t.Errorf("the tax is %d, want %d", got, testCase.want)
			}
		})
	}
}

// TestTheTaxOfTheInvoiceIsRoundedToTheNearestWholePeso proves the rounding
// rule of the book: the tax is rounded to the nearest whole peso, and a
// half peso goes UP.
//
// The rule bites: a base of fifty pesos at nineteen percent is nine pesos
// and a half, and the tax of that invoice is ten pesos and not nine.
func TestTheTaxOfTheInvoiceIsRoundedToTheNearestWholePeso(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	if got := TheTax(ledger.Amount(50), rate); got != ledger.Amount(10) {
		t.Errorf("the tax of fifty pesos is %d, want 10: a half peso goes up", got)
	}

	onePercent, err := NewRate(1)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	if got := TheTax(ledger.Amount(50), onePercent); got != ledger.Amount(1) {
		t.Errorf("the tax of fifty pesos at one percent is %d, want 1: a half peso goes up", got)
	}

	if got := TheTax(ledger.Amount(49), onePercent); got != ledger.Amount(0) {
		t.Errorf("the tax of forty-nine pesos at one percent is %d, want 0", got)
	}
}

// TestTheRateOfTheTaxIsCountedInHundredthsOfAPercent proves that the rate
// of the book keeps the hundredths a person reads: nineteen and a half is
// nineteen hundred and fifty, and it is never rounded away.
func TestTheRateOfTheTaxIsCountedInHundredthsOfAPercent(t *testing.T) {
	t.Parallel()

	nineteen, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}
	if nineteen.Hundredths() != 1900 {
		t.Errorf("the rate is %d hundredths, want 1900", nineteen.Hundredths())
	}
	if nineteen.String() != "19" {
		t.Errorf("the rate reads %q, want %q", nineteen.String(), "19")
	}

	half, err := NewRate(19.5)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}
	if half.Hundredths() != 1950 {
		t.Errorf("the rate is %d hundredths, want 1950", half.Hundredths())
	}
	if half.String() != "19.5" {
		t.Errorf("the rate reads %q, want %q", half.String(), "19.5")
	}
	if half.Percent() != 19.5 {
		t.Errorf("the rate is %v percent, want 19.5", half.Percent())
	}
}

// TestTheRateOfTheTaxIsRefusedWhenItIsNotARate proves that the book
// refuses a rate that is not a rate: a rate is between zero and a hundred
// percent, and a rate outside of it is a number nobody can charge.
func TestTheRateOfTheTaxIsRefusedWhenItIsNotARate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		percent float64
	}{
		{name: "a negative rate", percent: -1},
		{name: "a rate above a hundred percent", percent: 100.5},
		{name: "a rate that is not a number", percent: math.NaN()},
		{name: "a rate that is infinite", percent: math.Inf(1)},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewRate(testCase.percent); !errors.Is(err, ErrInvalidRate) {
				t.Fatalf("the error is %v, want %v", err, ErrInvalidRate)
			}
		})
	}
}

// TestNewInvoiceRefusesTheInvoiceThatDoesNotAddUp proves the rule of the
// domain: an invoice that does not add up is never read as an invoice. The
// total of an invoice is the base plus the tax the rate gives it.
func TestNewInvoiceRefusesTheInvoiceThatDoesNotAddUp(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	_, err = NewInvoice(business.ID(77), theProviderOfTheTest(), InvoiceDefinition{
		PurchaseID: ID(5),
		Number:     "FV-002",
		Day:        theDayOfTheInvoice(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Total:      ledger.Amount(12000),
	})
	if !errors.Is(err, ErrInvoiceDoesNotAddUp) {
		t.Fatalf("the error is %v, want %v", err, ErrInvoiceDoesNotAddUp)
	}
}

// TestNewInvoiceRefusesAPartyThatIsNotAProvider proves the rule of the
// domain: the provider of an invoice holds the role of the provider, and a
// party that holds only the role of the client is refused.
func TestNewInvoiceRefusesAPartyThatIsNotAProvider(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	client := party.Party{
		ID:         party.ID(20),
		BusinessID: business.ID(77),
		Kind:       party.KindPerson,
		Name:       "Doña Rosalba",
		Roles:      []party.Role{party.RoleClient},
	}

	_, err = NewInvoice(business.ID(77), client, InvoiceDefinition{
		PurchaseID: ID(5),
		Number:     "FV-003",
		Day:        theDayOfTheInvoice(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Total:      ledger.Amount(11900),
	})
	if !errors.Is(err, ErrNotAProvider) {
		t.Fatalf("the error is %v, want %v", err, ErrNotAProvider)
	}
}

// TestNewInvoiceRefusesTheProviderOfAnotherBusiness proves the rule of the
// domain: the invoice is a fact of the business that bought, and the
// provider of another business is not the provider of that invoice.
func TestNewInvoiceRefusesTheProviderOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	other := theProviderOfTheTest()
	other.BusinessID = business.ID(88)

	_, err = NewInvoice(business.ID(77), other, InvoiceDefinition{
		PurchaseID: ID(5),
		Number:     "FV-004",
		Day:        theDayOfTheInvoice(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Total:      ledger.Amount(11900),
	})
	if !errors.Is(err, ErrInvoiceOfAnotherBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrInvoiceOfAnotherBusiness)
	}
}

// TestNewInvoiceRefusesWhatIsMissing proves that the domain refuses the
// invoice that is not whole: the number, the day, the purchase it came
// from and the money it carries are required.
func TestNewInvoiceRefusesWhatIsMissing(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	cases := []struct {
		name       string
		definition InvoiceDefinition
		want       error
	}{
		{
			name: "an invoice without its number",
			definition: InvoiceDefinition{
				PurchaseID: ID(5),
				Number:     "   ",
				Day:        theDayOfTheInvoice(),
				Base:       ledger.Amount(10000),
				Rate:       rate,
				Total:      ledger.Amount(11900),
			},
			want: ErrMissingInvoiceNumber,
		},
		{
			name: "an invoice without its day",
			definition: InvoiceDefinition{
				PurchaseID: ID(5),
				Number:     "FV-005",
				Day:        ledger.Date{},
				Base:       ledger.Amount(10000),
				Rate:       rate,
				Total:      ledger.Amount(11900),
			},
			want: ErrMissingInvoiceDay,
		},
		{
			name: "an invoice without the purchase it came from",
			definition: InvoiceDefinition{
				PurchaseID: 0,
				Number:     "FV-006",
				Day:        theDayOfTheInvoice(),
				Base:       ledger.Amount(10000),
				Rate:       rate,
				Total:      ledger.Amount(11900),
			},
			want: ErrMissingPurchase,
		},
		{
			name: "an invoice with a negative base",
			definition: InvoiceDefinition{
				PurchaseID: ID(5),
				Number:     "FV-007",
				Day:        theDayOfTheInvoice(),
				Base:       ledger.Amount(-1),
				Rate:       rate,
				Total:      ledger.Amount(0),
			},
			want: ErrInvalidInvoiceMoney,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewInvoice(business.ID(77), theProviderOfTheTest(), testCase.definition)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}

// TestTheInvoiceIsAttachedToThePurchaseOfItsBusiness proves the rule of the
// domain: a purchase of a business cannot point at an invoice of another
// business, and an invoice of one business cannot be attached to the
// purchase of another.
func TestTheInvoiceIsAttachedToThePurchaseOfItsBusiness(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	built, err := NewInvoice(business.ID(77), theProviderOfTheTest(), InvoiceDefinition{
		PurchaseID: ID(5),
		Number:     "FV-008",
		Day:        theDayOfTheInvoice(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Total:      ledger.Amount(11900),
	})
	if err != nil {
		t.Fatalf("the invoice was refused: %v", err)
	}

	own := Purchase{ID: ID(5), BusinessID: business.ID(77)}
	if err := built.AttachesTo(own); err != nil {
		t.Fatalf("the invoice was refused by the purchase of its business: %v", err)
	}

	other := Purchase{ID: ID(5), BusinessID: business.ID(88)}
	if err := built.AttachesTo(other); !errors.Is(err, ErrInvoiceOfAnotherBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrInvoiceOfAnotherBusiness)
	}
}

// TestTheInvoiceOfTheStorageIsBuiltByTheDomain proves that the domain
// builds the invoice the storage holds, and that it refuses the row that
// another writer broke: an invoice that does not add up is never read as
// an invoice.
func TestTheInvoiceOfTheStorageIsBuiltByTheDomain(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	facts := InvoiceFacts{
		ID:         InvoiceID(3),
		BusinessID: business.ID(77),
		ProviderID: party.ID(10),
		PurchaseID: ID(5),
		Number:     "FV-001",
		Day:        theDayOfTheInvoice(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Tax:        ledger.Amount(1900),
		Total:      ledger.Amount(11900),
		Reference:  "CUFE-1234",
	}

	built, err := NewInvoiceFromFacts(facts)
	if err != nil {
		t.Fatalf("the invoice was refused: %v", err)
	}

	if built.ID != InvoiceID(3) {
		t.Errorf("the identity is %d, want 3", built.ID)
	}
	if built.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", built.BusinessID)
	}
	if built.ProviderID != party.ID(10) {
		t.Errorf("the provider is %d, want 10", built.ProviderID)
	}
	if built.PurchaseID != ID(5) {
		t.Errorf("the purchase is %d, want 5", built.PurchaseID)
	}
	if built.Number != "FV-001" {
		t.Errorf("the number is %q, want %q", built.Number, "FV-001")
	}
	if built.Tax != ledger.Amount(1900) {
		t.Errorf("the tax is %d, want 1900", built.Tax)
	}
	if built.Total != ledger.Amount(11900) {
		t.Errorf("the total is %d, want 11900", built.Total)
	}
	if built.Reference != "CUFE-1234" {
		t.Errorf("the reference is %q, want %q", built.Reference, "CUFE-1234")
	}
}

// TestTheInvoiceOfTheStorageThatDoesNotAddUpIsRefused proves that the
// domain refuses the row of another writer that breaks the rules of the
// invoice: a row that does not add up is never read as an invoice.
func TestTheInvoiceOfTheStorageThatDoesNotAddUpIsRefused(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	facts := InvoiceFacts{
		BusinessID: business.ID(77),
		ProviderID: party.ID(10),
		PurchaseID: ID(5),
		Number:     "FV-001",
		Day:        theDayOfTheInvoice(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Tax:        ledger.Amount(1900),
		Total:      ledger.Amount(11900),
	}

	cases := []struct {
		name   string
		broken func(InvoiceFacts) InvoiceFacts
		want   error
	}{
		{
			name: "a row without its number",
			broken: func(f InvoiceFacts) InvoiceFacts {
				f.Number = "  "
				return f
			},
			want: ErrMissingInvoiceNumber,
		},
		{
			name: "a row without its day",
			broken: func(f InvoiceFacts) InvoiceFacts {
				f.Day = ledger.Date{}
				return f
			},
			want: ErrMissingInvoiceDay,
		},
		{
			name: "a row without the purchase it came from",
			broken: func(f InvoiceFacts) InvoiceFacts {
				f.PurchaseID = 0
				return f
			},
			want: ErrMissingPurchase,
		},
		{
			name: "a row with a negative total",
			broken: func(f InvoiceFacts) InvoiceFacts {
				f.Total = ledger.Amount(-1)
				return f
			},
			want: ErrInvalidInvoiceMoney,
		},
		{
			name: "a row whose tax is not the tax of its rate",
			broken: func(f InvoiceFacts) InvoiceFacts {
				f.Tax = ledger.Amount(1000)
				f.Total = ledger.Amount(11000)
				return f
			},
			want: ErrInvoiceDoesNotAddUp,
		},
		{
			name: "a row whose total does not add up",
			broken: func(f InvoiceFacts) InvoiceFacts {
				f.Total = ledger.Amount(12000)
				return f
			},
			want: ErrInvoiceDoesNotAddUp,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewInvoiceFromFacts(testCase.broken(facts))
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}

// TestNewInvoiceRefusesTheProviderThatIsNotThere proves that the domain
// refuses the invoice that names no provider: the provider of an invoice
// is a party, and an invoice without one is a fact of nobody.
func TestNewInvoiceRefusesTheProviderThatIsNotThere(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	_, err = NewInvoice(business.ID(77), party.Party{}, InvoiceDefinition{
		PurchaseID: ID(5),
		Number:     "FV-010",
		Day:        theDayOfTheInvoice(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Total:      ledger.Amount(11900),
	})
	if !errors.Is(err, ErrMissingProvider) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingProvider)
	}
}

// TestTheInvoiceIsAttachedToThePurchaseItCameFrom proves that the domain
// refuses the invoice that names no purchase when it is attached to one.
func TestTheInvoiceIsAttachedToThePurchaseItCameFrom(t *testing.T) {
	t.Parallel()

	rate, err := NewRate(19)
	if err != nil {
		t.Fatalf("the rate was refused: %v", err)
	}

	built, err := NewInvoice(business.ID(77), theProviderOfTheTest(), InvoiceDefinition{
		PurchaseID: ID(5),
		Number:     "FV-011",
		Day:        theDayOfTheInvoice(),
		Base:       ledger.Amount(10000),
		Rate:       rate,
		Total:      ledger.Amount(11900),
	})
	if err != nil {
		t.Fatalf("the invoice was refused: %v", err)
	}

	if err := built.AttachesTo(Purchase{}); !errors.Is(err, ErrMissingPurchase) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingPurchase)
	}
}

// TestTheNumberOfTheInvoiceIsUniqueInsideTheBusiness proves the rule of the
// domain: the number of an invoice is what identifies it, and it is unique
// inside the business that holds it.
func TestTheNumberOfTheInvoiceIsUniqueInsideTheBusiness(t *testing.T) {
	t.Parallel()

	held := []Invoice{
		{ID: InvoiceID(1), BusinessID: business.ID(77), Number: "FV-001"},
		{ID: InvoiceID(2), BusinessID: business.ID(88), Number: "FV-001"},
	}

	candidate := Invoice{BusinessID: business.ID(77), Number: "FV-001"}
	if err := UniqueInvoiceNumber(held, candidate); !errors.Is(err, ErrDuplicateInvoiceNumber) {
		t.Fatalf("the error is %v, want %v", err, ErrDuplicateInvoiceNumber)
	}

	free := Invoice{BusinessID: business.ID(77), Number: "FV-009"}
	if err := UniqueInvoiceNumber(held, free); err != nil {
		t.Fatalf("the number of the invoice was refused: %v", err)
	}

	itself := Invoice{ID: InvoiceID(1), BusinessID: business.ID(77), Number: "FV-001"}
	if err := UniqueInvoiceNumber(held, itself); err != nil {
		t.Fatalf("the invoice was refused by its own number: %v", err)
	}
}
