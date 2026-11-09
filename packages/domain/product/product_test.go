package product

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// theDayOfTheTest is a day of the book of the bakery.
func theDayOfTheTest(day string) ledger.Date {
	parsed, err := ledger.ParseDate(day)
	if err != nil {
		panic(err)
	}

	return parsed
}

// TestAllChannelsNamesTheTwoChannelsOfTheBook proves that the list of the
// channels names the two of them, in the order they are presented.
func TestAllChannelsNamesTheTwoChannelsOfTheBook(t *testing.T) {
	t.Parallel()

	all := AllChannels()
	if len(all) != 2 {
		t.Fatalf("the list holds %d channels, want 2", len(all))
	}

	if all[0] != ChannelRetail {
		t.Errorf("the first channel is %q, want %q", all[0], ChannelRetail)
	}
	if all[1] != ChannelWholesale {
		t.Errorf("the second channel is %q, want %q", all[1], ChannelWholesale)
	}
}

// TestTheCallerOfTheListCannotChangeIt proves that the caller of the list
// receives a copy: the vocabulary of the domain never changes under it.
func TestTheCallerOfTheListCannotChangeIt(t *testing.T) {
	t.Parallel()

	all := AllChannels()
	all[0] = ChannelWholesale

	if AllChannels()[0] != ChannelRetail {
		t.Error("the list of the channels changed under the caller")
	}
}

// TestParseChannelReadsTheWaysTheOwnerSaysThem proves that the domain
// reads the channel the way the owner says it: the retail and the
// wholesale, in the language of the book and in hers.
func TestParseChannelReadsTheWaysTheOwnerSaysThem(t *testing.T) {
	t.Parallel()

	cases := []struct {
		said string
		want Channel
	}{
		{said: "retail", want: ChannelRetail},
		{said: "  Retail  ", want: ChannelRetail},
		{said: "minorista", want: ChannelRetail},
		{said: "Minorista", want: ChannelRetail},
		{said: "wholesale", want: ChannelWholesale},
		{said: "mayorista", want: ChannelWholesale},
		{said: "  MAYORISTA  ", want: ChannelWholesale},
	}

	for _, one := range cases {
		t.Run(one.said, func(t *testing.T) {
			t.Parallel()

			parsed, err := ParseChannel(one.said)
			if err != nil {
				t.Fatalf("the channel %q was refused: %v", one.said, err)
			}
			if parsed != one.want {
				t.Errorf("the channel %q is %q, want %q", one.said, parsed, one.want)
			}
		})
	}
}

// TestParseChannelRefusesAChannelTheBookDoesNotKnow proves that the
// domain refuses a channel that is not one of the two, and that it wraps
// the error of the catalogue.
func TestParseChannelRefusesAChannelTheBookDoesNotKnow(t *testing.T) {
	t.Parallel()

	for _, said := range []string{"", "  ", "store", "mercadol"} {
		t.Run(said, func(t *testing.T) {
			t.Parallel()

			_, err := ParseChannel(said)
			if !errors.Is(err, ErrUnknownChannel) {
				t.Fatalf("the error is %v, want %v", err, ErrUnknownChannel)
			}
		})
	}
}

// TestTheStringOfTheChannelIsTheNameTheStorageHolds proves that the
// channel is stored with no translation in between.
func TestTheStringOfTheChannelIsTheNameTheStorageHolds(t *testing.T) {
	t.Parallel()

	if ChannelRetail.String() != "retail" {
		t.Errorf("the retail is %q, want %q", ChannelRetail.String(), "retail")
	}
	if ChannelWholesale.String() != "wholesale" {
		t.Errorf("the wholesale is %q, want %q", ChannelWholesale.String(), "wholesale")
	}
}

// TestThePriceCarriesTheChannelItWasSetFor proves that every price carries
// the channel it was set for: there is no price without a channel and no
// default channel.
func TestThePriceCarriesTheChannelItWasSetFor(t *testing.T) {
	t.Parallel()

	built, err := NewPrice(business.ID(77), ID(10), ChannelWholesale, ledger.Amount(2500),
		theDayOfTheTest("2026-03-01"))
	if err != nil {
		t.Fatalf("the price was refused: %v", err)
	}

	if built.Channel != ChannelWholesale {
		t.Errorf("the channel of the price is %q, want %q", built.Channel, ChannelWholesale)
	}
	if built.ProductID != ID(10) {
		t.Errorf("the product of the price is %d, want 10", built.ProductID)
	}
	if built.BusinessID != business.ID(77) {
		t.Errorf("the business of the price is %d, want 77", built.BusinessID)
	}
}

// TestThePriceIsRefusedWithoutItsChannel proves that a price without a
// channel is refused: the channel is already in the price from the first
// row.
func TestThePriceIsRefusedWithoutItsChannel(t *testing.T) {
	t.Parallel()

	_, err := NewPrice(business.ID(77), ID(10), Channel(""), ledger.Amount(2500),
		theDayOfTheTest("2026-03-01"))
	if !errors.Is(err, ErrUnknownChannel) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownChannel)
	}
}

// TestThePriceIsWholePesosAndNeverNegative proves the rule of the money
// of the book: the price is a positive number of whole pesos, and a price
// of zero pesos is not a price the owner charges.
func TestThePriceIsWholePesosAndNeverNegative(t *testing.T) {
	t.Parallel()

	for _, money := range []ledger.Amount{0, -1, -3000} {
		_, err := NewPrice(business.ID(77), ID(10), ChannelRetail, money,
			theDayOfTheTest("2026-03-01"))
		if !errors.Is(err, ErrInvalidPrice) {
			t.Fatalf("the price of %d pesos was accepted, want the error %v", money, ErrInvalidPrice)
		}
	}
}

// TestThePriceIsRefusedWithoutItsProduct proves that a price without a
// product is refused: a price of nobody is a price of nobody.
func TestThePriceIsRefusedWithoutItsProduct(t *testing.T) {
	t.Parallel()

	_, err := NewPrice(business.ID(77), ID(0), ChannelRetail, ledger.Amount(2500),
		theDayOfTheTest("2026-03-01"))
	if !errors.Is(err, ErrMissingProduct) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingProduct)
	}
}

// TestThePriceIsRefusedWithoutItsDay proves that a price without a day is
// refused: the price is a fact with the day it was set.
func TestThePriceIsRefusedWithoutItsDay(t *testing.T) {
	t.Parallel()

	_, err := NewPrice(business.ID(77), ID(10), ChannelRetail, ledger.Amount(2500), ledger.Date{})
	if !errors.Is(err, ErrMissingDay) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingDay)
	}
}

// TestThePriceThatHoldsIsTheLastOneSetOnOrBeforeTheDay proves the rule of
// the price list: the price that holds on a day is the last one set on or
// before that day.
func TestThePriceThatHoldsIsTheLastOneSetOnOrBeforeTheDay(t *testing.T) {
	t.Parallel()

	first := thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-01")
	second := thePriceOfTheTest(2, ChannelRetail, 3500, "2026-03-05")
	held := []Price{first, second}

	onTheFirstDay, ok := ThePriceOn(held, ID(10), ChannelRetail, theDayOfTheTest("2026-03-01"))
	if !ok {
		t.Fatal("the price list holds no price on the first day")
	}
	if onTheFirstDay.Amount != ledger.Amount(3000) {
		t.Errorf("the price of the first day is %d, want 3000", onTheFirstDay.Amount)
	}

	onTheSecondDay, ok := ThePriceOn(held, ID(10), ChannelRetail, theDayOfTheTest("2026-03-05"))
	if !ok {
		t.Fatal("the price list holds no price on the second day")
	}
	if onTheSecondDay.Amount != ledger.Amount(3500) {
		t.Errorf("the price of the second day is %d, want 3500", onTheSecondDay.Amount)
	}

	today, ok := ThePriceOn(held, ID(10), ChannelRetail, theDayOfTheTest("2026-03-31"))
	if !ok {
		t.Fatal("the price list holds no price today")
	}
	if today.Amount != ledger.Amount(3500) {
		t.Errorf("the price of today is %d, want 3500", today.Amount)
	}
}

// TestThePriceThatHoldsBeforeAnyPriceWasSetIsNotThere proves that the
// price list answers no price on a day that is before every price of the
// product and the channel.
func TestThePriceThatHoldsBeforeAnyPriceWasSetIsNotThere(t *testing.T) {
	t.Parallel()

	held := []Price{thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-05")}

	_, ok := ThePriceOn(held, ID(10), ChannelRetail, theDayOfTheTest("2026-03-01"))
	if ok {
		t.Error("the price list answered a price before it was set")
	}
}

// TestThePriceOfAnotherChannelIsNeverThePriceOfThisOne proves that the
// channel is already in the price: the price of the retail is not the
// price of the wholesale.
func TestThePriceOfAnotherChannelIsNeverThePriceOfThisOne(t *testing.T) {
	t.Parallel()

	held := []Price{
		thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-01"),
		thePriceOfTheTest(2, ChannelWholesale, 2500, "2026-03-02"),
	}

	wholesale, ok := ThePriceOn(held, ID(10), ChannelWholesale, theDayOfTheTest("2026-03-05"))
	if !ok {
		t.Fatal("the price list holds no price of the wholesale")
	}
	if wholesale.Amount != ledger.Amount(2500) {
		t.Errorf("the price of the wholesale is %d, want 2500", wholesale.Amount)
	}
}

// TestTheHistoryOfThePricesIsNeverRewritten proves that the price list is
// a history: the price that held on a day is still the price of that day
// after the owner set another one.
func TestTheHistoryOfThePricesIsNeverRewritten(t *testing.T) {
	t.Parallel()

	first := thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-01")
	second := thePriceOfTheTest(2, ChannelRetail, 3500, "2026-03-05")
	held := []Price{first, second}

	if len(held) != 2 {
		t.Fatalf("the price list holds %d prices, want 2", len(held))
	}

	onTheFirstDay, ok := ThePriceOn(held, ID(10), ChannelRetail, theDayOfTheTest("2026-03-02"))
	if !ok {
		t.Fatal("the price list holds no price on the second day")
	}
	if onTheFirstDay.Amount != ledger.Amount(3000) {
		t.Errorf("the price of the second day is %d, want 3000", onTheFirstDay.Amount)
	}
	if onTheFirstDay.ID != first.ID {
		t.Errorf("the price of the second day is the price %d, want %d", onTheFirstDay.ID, first.ID)
	}
}

// TestTheLastPriceSetOnTheSameDayIsTheOneThatHolds proves that the price
// that holds is the LAST one that was set, even when two prices of the
// same day share it.
func TestTheLastPriceSetOnTheSameDayIsTheOneThatHolds(t *testing.T) {
	t.Parallel()

	first := thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-01")
	second := thePriceOfTheTest(2, ChannelRetail, 3500, "2026-03-01")
	held := []Price{first, second}

	holds, ok := ThePriceOn(held, ID(10), ChannelRetail, theDayOfTheTest("2026-03-01"))
	if !ok {
		t.Fatal("the price list holds no price on that day")
	}
	if holds.Amount != ledger.Amount(3500) {
		t.Errorf("the price that holds is %d, want 3500", holds.Amount)
	}
}

// TestThePriceThatWasSetBeforeTheOneThatHoldsDoesNotTakeItsPlace proves
// that the price list is read in the order the prices were set: a price
// that was set before another one never takes its place.
func TestThePriceThatWasSetBeforeTheOneThatHoldsDoesNotTakeItsPlace(t *testing.T) {
	t.Parallel()

	older := thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-01")
	newer := thePriceOfTheTest(2, ChannelRetail, 3500, "2026-03-05")
	held := []Price{newer, older}

	holds, ok := ThePriceOn(held, ID(10), ChannelRetail, theDayOfTheTest("2026-03-31"))
	if !ok {
		t.Fatal("the price list holds no price today")
	}
	if holds.Amount != ledger.Amount(3500) {
		t.Errorf("the price that holds is %d, want 3500", holds.Amount)
	}
}

// TestTheProductIsOpenedTheFirstTimeItIsNamed proves that the product
// comes into being the first time it is named, and that it comes with the
// recipe it comes from and the name the owner reads.
func TestTheProductIsOpenedTheFirstTimeItIsNamed(t *testing.T) {
	t.Parallel()

	opened, err := TheProduct(business.ID(77), nil, recipe.ID(5), "Pan de molde")
	if err != nil {
		t.Fatalf("the product was refused: %v", err)
	}

	if opened.ID != 0 {
		t.Errorf("the identity of the product is %d, want none until the storage writes it", opened.ID)
	}
	if opened.BusinessID != business.ID(77) {
		t.Errorf("the business of the product is %d, want 77", opened.BusinessID)
	}
	if opened.RecipeID != recipe.ID(5) {
		t.Errorf("the recipe of the product is %d, want 5", opened.RecipeID)
	}
	if opened.Name != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", opened.Name, "Pan de molde")
	}
}

// TestTheProductTheDomainBuildsIsRefusedWithoutItsName proves that the
// constructor of the product refuses a product without a name: the name the
// owner reads is never invented, and the spaces around it are not part of
// it.
func TestTheProductTheDomainBuildsIsRefusedWithoutItsName(t *testing.T) {
	t.Parallel()

	_, err := New(business.ID(77), recipe.ID(5), "   ", "")
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidName)
	}

	built, err := New(business.ID(77), recipe.ID(5), "  Pan de molde  ", "")
	if err != nil {
		t.Fatalf("the product was refused: %v", err)
	}
	if built.Name != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", built.Name, "Pan de molde")
	}
}

// TestTheNameOfTheProductIsNeverGuessed proves that the domain refuses a
// product without a name: the name is never invented.
func TestTheNameOfTheProductIsNeverGuessed(t *testing.T) {
	t.Parallel()

	_, err := TheProduct(business.ID(77), nil, recipe.ID(5), "   ")
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidName)
	}
}

// TestTheProductIsRefusedWithoutItsRecipe proves that the product of the
// bakery comes from a recipe: the bakery sells what it bakes.
func TestTheProductIsRefusedWithoutItsRecipe(t *testing.T) {
	t.Parallel()

	_, err := TheProduct(business.ID(77), nil, recipe.ID(0), "Pan de molde")
	if !errors.Is(err, ErrMissingRecipe) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingRecipe)
	}

	if _, err := New(business.ID(77), recipe.ID(0), "Pan de molde", ""); !errors.Is(err, ErrMissingRecipe) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingRecipe)
	}
}

// TestTheTaxTreatmentTheOwnerStatesIsTheOneTheProductCarries proves that
// the tax treatment of a product is the fact the owner states: the last
// statement is the one that counts, and an empty one leaves the product as
// it was.
func TestTheTaxTreatmentTheOwnerStatesIsTheOneTheProductCarries(t *testing.T) {
	t.Parallel()

	built, err := New(business.ID(77), recipe.ID(5), "Pan de molde", "")
	if err != nil {
		t.Fatalf("the product was refused: %v", err)
	}

	stated := built.WithTaxTreatment("  Grammy 19  ")
	if stated.TaxTreatment != "Grammy 19" {
		t.Errorf("the tax treatment of the product is %q, want %q", stated.TaxTreatment, "Grammy 19")
	}
	if built.TaxTreatment != "" {
		t.Errorf("the tax treatment of the product that was built is %q, want none", built.TaxTreatment)
	}

	kept := stated.WithTaxTreatment("")
	if kept.TaxTreatment != "Grammy 19" {
		t.Errorf("the tax treatment of the product is %q, want %q", kept.TaxTreatment, "Grammy 19")
	}
}

// TestTheNamedProductsAreTheOnesThatCarryTheName proves that the name of a
// product is compared the way a person reads it: trimmed and without case.
func TestTheNamedProductsAreTheOnesThatCarryTheName(t *testing.T) {
	t.Parallel()

	products := []Product{
		{ID: ID(10), BusinessID: business.ID(77), RecipeID: recipe.ID(5), Name: "Pan de molde"},
		{ID: ID(20), BusinessID: business.ID(77), RecipeID: recipe.ID(6), Name: "Biznaga"},
	}

	found := Named(products, "  pan de molde ")
	if len(found) != 1 {
		t.Fatalf("the list holds %d products, want 1", len(found))
	}
	if found[0].ID != ID(10) {
		t.Errorf("the product is %d, want 10", found[0].ID)
	}

	if len(Named(products, "")) != 0 {
		t.Error("the empty name found a product")
	}
	if len(Named(products, "Biznaga")) != 1 {
		t.Error("the name of the product was not found")
	}
}

// TestTheProductThatIsAlreadyInTheBookIsTheOneThatIsAnswered proves that
// the product is opened the first time it is named, and that the book
// answers the one it already holds afterwards.
func TestTheProductThatIsAlreadyInTheBookIsTheOneThatIsAnswered(t *testing.T) {
	t.Parallel()

	held := []Product{{
		ID:         ID(10),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(5),
		Name:       "Pan de molde",
	}}

	answered, err := TheProduct(business.ID(77), held, recipe.ID(5), "pan de molde")
	if err != nil {
		t.Fatalf("the product was refused: %v", err)
	}
	if answered.ID != ID(10) {
		t.Errorf("the identity of the product is %d, want 10", answered.ID)
	}
}

// TestTheNameOfTheProductIsTakenInsideTheBusiness proves that two
// products of a business never share the name the owner reads.
func TestTheNameOfTheProductIsTakenInsideTheBusiness(t *testing.T) {
	t.Parallel()

	held := []Product{{
		ID:         ID(10),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(5),
		Name:       "Pan de molde",
	}}

	_, err := TheProduct(business.ID(77), held, recipe.ID(6), "Pan de molde")
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("the error is %v, want %v", err, ErrDuplicateName)
	}
}

// TestOneRecipeBacksOneProduct proves that a recipe backs one product
// inside the business: the product of a recipe is never a second one.
func TestOneRecipeBacksOneProduct(t *testing.T) {
	t.Parallel()

	held := []Product{{
		ID:         ID(10),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(5),
		Name:       "Pan de molde",
	}}

	_, err := TheProduct(business.ID(77), held, recipe.ID(5), "Pan de molde grande")
	if !errors.Is(err, ErrDuplicateRecipe) {
		t.Fatalf("the error is %v, want %v", err, ErrDuplicateRecipe)
	}
}

// TestTheTaxTreatmentOfTheProductIsOptional proves that the tax treatment
// of a product is a fact the owner states or nothing: the book never
// derives it and never assumes one.
func TestTheTaxTreatmentOfTheProductIsOptional(t *testing.T) {
	t.Parallel()

	without, err := New(business.ID(77), recipe.ID(5), "Pan de molde", "")
	if err != nil {
		t.Fatalf("the product was refused: %v", err)
	}
	if without.TaxTreatment != "" {
		t.Errorf("the tax treatment of the product is %q, want none", without.TaxTreatment)
	}

	with, err := New(business.ID(77), recipe.ID(5), "Pan de molde", "  Grammy 19  ")
	if err != nil {
		t.Fatalf("the product was refused: %v", err)
	}
	if with.TaxTreatment != "Grammy 19" {
		t.Errorf("the tax treatment of the product is %q, want %q", with.TaxTreatment, "Grammy 19")
	}
}

// TestThePriceListBelongsToABusinessOfItsOwn proves the rule of the
// domain: the price list belongs to the business of the bakery, and the
// Personal Business of a person holds the money of that person and not the
// price list of the bakery.
func TestThePriceListBelongsToABusinessOfItsOwn(t *testing.T) {
	t.Parallel()

	if err := OwnedBy(business.KindBusiness); err != nil {
		t.Fatalf("the business of the bakery was refused: %v", err)
	}

	err := OwnedBy(business.KindPersonal)
	if !errors.Is(err, ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrWrongBusiness)
	}
}

// TestThePriceListHoldsThePriceThatHoldsForEveryProductAndChannel proves
// that the price list is read for every product of the business and for
// both of the channels.
func TestThePriceListHoldsThePriceThatHoldsForEveryProductAndChannel(t *testing.T) {
	t.Parallel()

	products := []Product{
		{ID: ID(10), BusinessID: business.ID(77), RecipeID: recipe.ID(5), Name: "Pan de molde"},
		{ID: ID(20), BusinessID: business.ID(77), RecipeID: recipe.ID(6), Name: "Biznaga"},
	}
	held := []Price{
		thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-01"),
		thePriceOfTheTest(2, ChannelWholesale, 2500, "2026-03-02"),
	}

	list := ThePriceList(held, products, theDayOfTheTest("2026-03-05"), ID(0), Channel(""))
	if len(list) != 2 {
		t.Fatalf("the price list holds %d lines, want 2", len(list))
	}
	if list[0].Product.ID != ID(10) || list[0].Price.Channel != ChannelRetail {
		t.Errorf("the first line is the product %d of the channel %q, want 10 of the retail",
			list[0].Product.ID, list[0].Price.Channel)
	}
	if list[1].Product.ID != ID(10) || list[1].Price.Channel != ChannelWholesale {
		t.Errorf("the second line is the product %d of the channel %q, want 10 of the wholesale",
			list[1].Product.ID, list[1].Price.Channel)
	}
	if list[1].Price.Amount != ledger.Amount(2500) {
		t.Errorf("the price of the wholesale is %d, want 2500", list[1].Price.Amount)
	}
}

// TestThePriceListOfOneChannelHoldsOnlyThatChannel proves that the price
// list can be read for one channel alone.
func TestThePriceListOfOneChannelHoldsOnlyThatChannel(t *testing.T) {
	t.Parallel()

	products := []Product{
		{ID: ID(10), BusinessID: business.ID(77), RecipeID: recipe.ID(5), Name: "Pan de molde"},
	}
	held := []Price{
		thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-01"),
		thePriceOfTheTest(2, ChannelWholesale, 2500, "2026-03-02"),
	}

	list := ThePriceList(held, products, theDayOfTheTest("2026-03-05"), ID(0), ChannelWholesale)
	if len(list) != 1 {
		t.Fatalf("the price list holds %d lines, want 1", len(list))
	}
	if list[0].Price.Channel != ChannelWholesale {
		t.Errorf("the channel of the line is %q, want %q", list[0].Price.Channel, ChannelWholesale)
	}
}

// TestThePriceListOfOneProductHoldsOnlyThatProduct proves that the price
// list can be read for one product alone: the lines of the other products
// are not in the answer.
func TestThePriceListOfOneProductHoldsOnlyThatProduct(t *testing.T) {
	t.Parallel()

	products := []Product{
		{ID: ID(10), BusinessID: business.ID(77), RecipeID: recipe.ID(5), Name: "Pan de molde"},
		{ID: ID(20), BusinessID: business.ID(77), RecipeID: recipe.ID(6), Name: "Biznaga"},
	}
	held := []Price{
		thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-01"),
		thePriceOfTheProductOfTheTest(2, ID(20), ChannelRetail, 3500, "2026-03-05"),
	}

	list := ThePriceList(held, products, theDayOfTheTest("2026-03-05"), ID(20), Channel(""))
	if len(list) != 1 {
		t.Fatalf("the price list holds %d lines, want 1", len(list))
	}
	if list[0].Product.ID != ID(20) {
		t.Errorf("the line is the product %d, want 20", list[0].Product.ID)
	}
	if list[0].Price.Amount != ledger.Amount(3500) {
		t.Errorf("the money of the line is %d, want 3500", list[0].Price.Amount)
	}
}

// TestThePriceListHoldsNoLineForAProductThatWasNeverPriced proves that a
// product the business has not priced yet is not a line of the price list.
func TestThePriceListHoldsNoLineForAProductThatWasNeverPriced(t *testing.T) {
	t.Parallel()

	products := []Product{
		{ID: ID(10), BusinessID: business.ID(77), RecipeID: recipe.ID(5), Name: "Pan de molde"},
		{ID: ID(20), BusinessID: business.ID(77), RecipeID: recipe.ID(6), Name: "Biznaga"},
	}
	held := []Price{
		thePriceOfTheTest(1, ChannelRetail, 3000, "2026-03-01"),
	}

	list := ThePriceList(held, products, theDayOfTheTest("2026-03-05"), ID(20), Channel(""))
	if len(list) != 0 {
		t.Fatalf("the price list holds %d lines, want none", len(list))
	}
}

// TestTheCatalogueOfTheErrorsNamesEveryErrorOfTheDomain proves that the
// catalogue holds every error of the package, so that a caller can prove
// that it maps every one of them.
func TestTheCatalogueOfTheErrorsNamesEveryErrorOfTheDomain(t *testing.T) {
	t.Parallel()

	all := Errors()
	if len(all) == 0 {
		t.Fatal("the catalogue of the errors is empty")
	}

	seen := make(map[error]bool, len(all))
	for _, one := range all {
		if one == nil {
			t.Error("the catalogue holds an error that is not there")
		}
		if seen[one] {
			t.Errorf("the catalogue holds the error %v twice", one)
		}
		seen[one] = true
	}
}

// thePriceOfTheTest builds one price of the price list of the test.
func thePriceOfTheTest(id PriceID, channel Channel, amount ledger.Amount, day string) Price {
	return thePriceOfTheProductOfTheTest(id, ID(10), channel, amount, day)
}

// thePriceOfTheProductOfTheTest is a price of the product it names.
func thePriceOfTheProductOfTheTest(id PriceID, of ID, channel Channel, amount ledger.Amount, day string) Price {
	built, err := NewPrice(business.ID(77), of, channel, amount, theDayOfTheTest(day))
	if err != nil {
		panic(err)
	}

	built.ID = id

	return built
}
