package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/product"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// fakeProducts is the fake of the storage of the products.
//
// It remembers the products it holds and the ones it wrote, and it answers
// what the test tells it to answer.
type fakeProducts struct {
	held      []product.Product
	written   []product.Product
	stated    []product.Product
	err       error
	createErr error
	stateErr  error
	listed    business.ID
}

// Create writes the product and remembers it.
func (f *fakeProducts) Create(_ context.Context, p product.Product) (product.Product, error) {
	if f.createErr != nil {
		return product.Product{}, f.createErr
	}

	if f.err != nil {
		return product.Product{}, f.err
	}

	p.ID = product.ID(len(f.written) + 100)
	f.written = append(f.written, p)

	return p, nil
}

// List answers the products of the business, and remembers it.
func (f *fakeProducts) List(_ context.Context, own business.ID) ([]product.Product, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]product.Product, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// State writes the tax treatment of the product and remembers it.
func (f *fakeProducts) State(_ context.Context, p product.Product) (product.Product, error) {
	if f.stateErr != nil {
		return product.Product{}, f.stateErr
	}

	if f.err != nil {
		return product.Product{}, f.err
	}

	f.stated = append(f.stated, p)

	for index, one := range f.held {
		if one.ID == p.ID {
			f.held[index] = p
		}
	}

	return p, nil
}

// fakePrices is the fake of the storage of the price list.
//
// It remembers the prices it holds and the ones it wrote, and it answers
// what the test tells it to answer.
type fakePrices struct {
	held    []product.Price
	written []product.Price
	err     error
	listed  business.ID
}

// Set writes the price and remembers it.
func (f *fakePrices) Set(_ context.Context, price product.Price) (product.Price, error) {
	if f.err != nil {
		return product.Price{}, f.err
	}

	price.ID = product.PriceID(len(f.written) + 100)
	f.written = append(f.written, price)

	return price, nil
}

// List answers the prices of the business, and remembers it.
func (f *fakePrices) List(_ context.Context, own business.ID) ([]product.Price, error) {
	f.listed = own

	if f.err != nil {
		return nil, f.err
	}

	held := make([]product.Price, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == own {
			held = append(held, one)
		}
	}

	return held, nil
}

// fakePriceWork is the fake of the transaction of the storage.
//
// The work that fails leaves the storage as it was: the product and the
// price it wrote are forgotten.
type fakePriceWork struct {
	products *fakeProducts
	prices   *fakePrices
}

// Do runs the work, and it forgets everything the work wrote when the work
// fails.
func (f *fakePriceWork) Do(ctx context.Context, work func(context.Context) error) error {
	products := len(f.products.written)
	prices := len(f.prices.written)

	err := work(ctx)
	if err == nil {
		return nil
	}

	f.products.written = f.products.written[:products]
	f.prices.written = f.prices.written[:prices]

	return err
}

// theBakeryOfThePrices is the business that owns the price list of the
// tests.
func theBakeryOfThePrices() business.Business {
	return business.Business{
		ID:       business.ID(77),
		Kind:     business.KindBusiness,
		Name:     "La Panadería",
		Currency: "COP",
	}
}

// theRecipeOfThePrices is the recipe the product of the tests comes from.
func theRecipeOfThePrices() recipe.Recipe {
	return recipe.Recipe{
		ID:         recipe.ID(5),
		BusinessID: business.ID(77),
		Name:       "Pan de molde",
	}
}

// thePriceService builds the use case of the price list with the fakes of
// the test.
func thePriceService(
	products *fakeProducts,
	prices *fakePrices,
	recipes *fakeRecipes,
	bakery *business.Business,
) *application.PriceService {
	return application.NewPriceService(
		products,
		prices,
		recipes,
		&fakeBusinesses{existing: bakery},
		&fakePriceWork{products: products, prices: prices},
		&fakePriceClock{moment: time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC)},
	)
}

// fakePriceClock is the fake of the clock of the service.
type fakePriceClock struct {
	moment time.Time
}

// Now answers the moment of the test.
func (f *fakePriceClock) Now() time.Time {
	return f.moment
}

// TestThePriceIsSetForTheChannelOfTheOwner proves that the use case writes
// the price of the product for the channel, with the money and the day the
// owner named.
func TestThePriceIsSetForTheChannelOfTheOwner(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	written, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "mayorista",
		Amount:     2500,
		Day:        "2026-03-10",
	})
	if err != nil {
		t.Fatalf("the price was refused: %v", err)
	}

	if len(prices.written) != 1 {
		t.Fatalf("the storage holds %d prices, want 1", len(prices.written))
	}

	price := prices.written[0]
	if price.Channel != product.ChannelWholesale {
		t.Errorf("the channel of the price is %q, want %q", price.Channel, product.ChannelWholesale)
	}
	if price.Amount != ledger.Amount(2500) {
		t.Errorf("the money of the price is %d, want 2500", price.Amount)
	}
	if price.Day.String() != "2026-03-10" {
		t.Errorf("the day of the price is %q, want %q", price.Day.String(), "2026-03-10")
	}
	if price.BusinessID != business.ID(77) {
		t.Errorf("the business of the price is %d, want 77", price.BusinessID)
	}
	if written.Price.ID == 0 {
		t.Error("the price has no identity, want the one the storage generated")
	}
}

// TestTheProductIsOpenedTheFirstTimeItIsNamed proves that the product of
// the bakery comes into being the first time it is named, and that it
// carries the recipe it comes from and the name the owner reads.
func TestTheProductIsOpenedTheFirstTimeItIsNamed(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Name:       "Pan de molde",
		Channel:    "retail",
		Amount:     3000,
	})
	if err != nil {
		t.Fatalf("the price was refused: %v", err)
	}

	if len(products.written) != 1 {
		t.Fatalf("the storage holds %d products, want 1", len(products.written))
	}

	opened := products.written[0]
	if opened.RecipeID != recipe.ID(5) {
		t.Errorf("the recipe of the product is %d, want 5", opened.RecipeID)
	}
	if opened.Name != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", opened.Name, "Pan de molde")
	}
	if opened.BusinessID != business.ID(77) {
		t.Errorf("the business of the product is %d, want 77", opened.BusinessID)
	}
}

// TestTheProductThatIsAlreadyInTheBookIsNotOpenedAgain proves that the
// product comes into being the first time it is named, and that the price
// of a product the business already holds is the price of that product.
func TestTheProductThatIsAlreadyInTheBookIsNotOpenedAgain(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{held: []product.Product{{
		ID:         product.ID(10),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(5),
		Name:       "Pan de molde",
	}}}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     3000,
	})
	if err != nil {
		t.Fatalf("the price was refused: %v", err)
	}

	if len(products.written) != 0 {
		t.Errorf("the storage holds %d products, want none", len(products.written))
	}
	if len(prices.written) != 1 {
		t.Fatalf("the storage holds %d prices, want 1", len(prices.written))
	}
	if prices.written[0].ProductID != product.ID(10) {
		t.Errorf("the product of the price is %d, want 10", prices.written[0].ProductID)
	}
}

// TestTheProductIsNamedByTheNameTheOwnerReads proves that the owner may
// name the product with the name she reads, and that the recipe that
// carries that name is the one the product comes from.
func TestTheProductIsNamedByTheNameTheOwnerReads(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		Name:       "Pan de molde",
		Channel:    "retail",
		Amount:     3000,
	})
	if err != nil {
		t.Fatalf("the price was refused: %v", err)
	}

	if len(products.written) != 1 {
		t.Fatalf("the storage holds %d products, want 1", len(products.written))
	}
	if products.written[0].RecipeID != recipe.ID(5) {
		t.Errorf("the recipe of the product is %d, want 5", products.written[0].RecipeID)
	}
	if products.written[0].Name != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", products.written[0].Name, "Pan de molde")
	}
}

// TestTheNameThatIsNeitherAProductNorARecipeIsRefused proves that the
// name of a product is never guessed: the book refuses a name it does not
// know.
func TestTheNameThatIsNeitherAProductNorARecipeIsRefused(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		Name:       "Biznaga",
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, product.ErrUnknownRecipe) {
		t.Fatalf("the error is %v, want %v", err, product.ErrUnknownRecipe)
	}
	if len(products.written) != 0 {
		t.Errorf("the storage holds %d products, want none", len(products.written))
	}
	if len(prices.written) != 0 {
		t.Errorf("the storage holds %d prices, want none", len(prices.written))
	}
}

// TestThePriceIsRefusedWhenTheMoneyIsNotPositive proves that the use case
// asks the domain to build the price, and that a price that is not a
// positive number of pesos leaves nothing behind.
func TestThePriceIsRefusedWhenTheMoneyIsNotPositive(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     0,
	})
	if !errors.Is(err, product.ErrInvalidPrice) {
		t.Fatalf("the error is %v, want %v", err, product.ErrInvalidPrice)
	}
	if len(products.written) != 0 {
		t.Errorf("the storage holds %d products, want none", len(products.written))
	}
	if len(prices.written) != 0 {
		t.Errorf("the storage holds %d prices, want none", len(prices.written))
	}
}

// TestThePriceIsRefusedWhenTheChannelIsNotOneTheBookKnows proves that the
// use case asks the domain to read the channel, and that there is no
// default channel.
func TestThePriceIsRefusedWhenTheChannelIsNotOneTheBookKnows(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "store",
		Amount:     3000,
	})
	if !errors.Is(err, product.ErrUnknownChannel) {
		t.Fatalf("the error is %v, want %v", err, product.ErrUnknownChannel)
	}
	if len(prices.written) != 0 {
		t.Errorf("the storage holds %d prices, want none", len(prices.written))
	}
}

// TestTheTaxTreatmentTheOwnerStatesIsWritten proves that the tax treatment
// of a product is a fact the owner states, and that the book never derives
// it from the price.
func TestTheTaxTreatmentTheOwnerStatesIsWritten(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID:   77,
		RecipeID:     5,
		Channel:      "retail",
		Amount:       3000,
		TaxTreatment: "Grammy 19",
	})
	if err != nil {
		t.Fatalf("the price was refused: %v", err)
	}

	if len(products.written) != 1 {
		t.Fatalf("the storage holds %d products, want 1", len(products.written))
	}
	if products.written[0].TaxTreatment != "Grammy 19" {
		t.Errorf("the tax treatment of the product is %q, want %q",
			products.written[0].TaxTreatment, "Grammy 19")
	}
}

// TestThePriceListIsRead proves that the use case reads the price list of
// the business: the price that holds for every product and every channel.
func TestThePriceListIsRead(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{held: []product.Product{{
		ID:         product.ID(10),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(5),
		Name:       "Pan de molde",
	}}}
	prices := &fakePrices{held: []product.Price{
		{
			ID:         product.PriceID(1),
			BusinessID: business.ID(77),
			ProductID:  product.ID(10),
			Channel:    product.ChannelRetail,
			Amount:     ledger.Amount(3000),
			Day:        theDayOfThePrices(t, "2026-03-01"),
		},
		{
			ID:         product.PriceID(2),
			BusinessID: business.ID(77),
			ProductID:  product.ID(10),
			Channel:    product.ChannelWholesale,
			Amount:     ledger.Amount(2500),
			Day:        theDayOfThePrices(t, "2026-03-02"),
		},
	}}
	recipes := &fakeRecipes{}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	held, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77})
	if err != nil {
		t.Fatalf("the price list was refused: %v", err)
	}

	if len(held) != 2 {
		t.Fatalf("the answer holds %d prices, want 2", len(held))
	}
	if held[0].Price.Channel != product.ChannelRetail || held[0].Price.Amount != ledger.Amount(3000) {
		t.Errorf("the first price is %q of %d pesos, want the retail of 3000 pesos",
			held[0].Price.Channel, held[0].Price.Amount)
	}
	if held[1].Price.Channel != product.ChannelWholesale || held[1].Price.Amount != ledger.Amount(2500) {
		t.Errorf("the second price is %q of %d pesos, want the wholesale of 2500 pesos",
			held[1].Price.Channel, held[1].Price.Amount)
	}
	if prices.listed != business.ID(77) {
		t.Errorf("the prices that were read are the ones of %d, want 77", prices.listed)
	}
}

// TestThePriceListIsReadForTheDayThatIsAskedFor proves that the price that
// holds is the last one set on or before the day that is asked for.
func TestThePriceListIsReadForTheDayThatIsAskedFor(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{held: []product.Product{{
		ID:         product.ID(10),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(5),
		Name:       "Pan de molde",
	}}}
	prices := &fakePrices{held: []product.Price{
		{
			ID:         product.PriceID(1),
			BusinessID: business.ID(77),
			ProductID:  product.ID(10),
			Channel:    product.ChannelRetail,
			Amount:     ledger.Amount(3000),
			Day:        theDayOfThePrices(t, "2026-03-01"),
		},
		{
			ID:         product.PriceID(2),
			BusinessID: business.ID(77),
			ProductID:  product.ID(10),
			Channel:    product.ChannelRetail,
			Amount:     ledger.Amount(3500),
			Day:        theDayOfThePrices(t, "2026-03-05"),
		},
	}}
	recipes := &fakeRecipes{}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	held, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77, Day: "2026-03-02"})
	if err != nil {
		t.Fatalf("the price list was refused: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the answer holds %d prices, want 1", len(held))
	}
	if held[0].Price.Amount != ledger.Amount(3000) {
		t.Errorf("the price that holds is %d pesos, want 3000", held[0].Price.Amount)
	}
}

// TestThePriceListOfOneChannelIsRead proves that the price list can be read
// for one channel alone.
func TestThePriceListOfOneChannelIsRead(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{held: []product.Product{{
		ID:         product.ID(10),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(5),
		Name:       "Pan de molde",
	}}}
	prices := &fakePrices{held: []product.Price{
		{
			ID:         product.PriceID(1),
			BusinessID: business.ID(77),
			ProductID:  product.ID(10),
			Channel:    product.ChannelRetail,
			Amount:     ledger.Amount(3000),
			Day:        theDayOfThePrices(t, "2026-03-01"),
		},
		{
			ID:         product.PriceID(2),
			BusinessID: business.ID(77),
			ProductID:  product.ID(10),
			Channel:    product.ChannelWholesale,
			Amount:     ledger.Amount(2500),
			Day:        theDayOfThePrices(t, "2026-03-02"),
		},
	}}
	recipes := &fakeRecipes{}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	held, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77, Channel: "mayorista"})
	if err != nil {
		t.Fatalf("the price list was refused: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the answer holds %d prices, want 1", len(held))
	}
	if held[0].Price.Channel != product.ChannelWholesale {
		t.Errorf("the channel of the price is %q, want %q", held[0].Price.Channel, product.ChannelWholesale)
	}
}

// TestThePriceListIsRefusedForThePersonalBusiness proves the rule of the
// domain: the price list belongs to the business of the bakery, and the
// Personal Business of a person holds the money of that person.
func TestThePriceListIsRefusedForThePersonalBusiness(t *testing.T) {
	t.Parallel()

	personal := theBakeryOfThePrices()
	personal.Kind = business.KindPersonal

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{}
	service := thePriceService(products, prices, recipes, &personal)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, product.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, product.ErrWrongBusiness)
	}

	if _, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77}); !errors.Is(err, product.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, product.ErrWrongBusiness)
	}
}

// TestThePriceListIsRefusedWhenTheBusinessIsNotThere proves that the use
// case reads the business that owns the price list, and that it writes
// nothing when there is none.
func TestThePriceListIsRefusedWhenTheBusinessIsNotThere(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{}
	service := application.NewPriceService(products, prices, recipes, &fakeBusinesses{},
		&fakePriceWork{products: products, prices: prices}, &fakePriceClock{})

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, business.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, business.ErrNotFound)
	}
	if len(prices.written) != 0 {
		t.Errorf("the storage holds %d prices, want none", len(prices.written))
	}
}

// TestTheUseCaseOfThePriceListHasNoStorage proves that the service without
// a database says so, and that it writes nothing.
func TestTheUseCaseOfThePriceListHasNoStorage(t *testing.T) {
	t.Parallel()

	service := application.NewPriceService(nil, nil, nil, nil, nil, nil)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}

	if _, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77}); !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}

// theDayOfThePrices is a day of the book of the bakery.
func theDayOfThePrices(t *testing.T, day string) ledger.Date {
	t.Helper()

	parsed, err := ledger.ParseDate(day)
	if err != nil {
		t.Fatalf("the day %q could not be read: %v", day, err)
	}

	return parsed
}

// TestThePriceIsRefusedWhenTheDayIsNotADayOfTheBook proves that the day is
// a day of the book, and that a day that is not one is refused.
func TestThePriceIsRefusedWhenTheDayIsNotADayOfTheBook(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     3000,
		Day:        "the day of today",
	})
	if err == nil {
		t.Fatal("the price of a day that is not a day of the book was written")
	}

	if _, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77, Day: "the day of today"}); err == nil {
		t.Fatal("the price list of a day that is not a day of the book was read")
	}
}

// TestThePriceIsNotWrittenWhenTheStorageRefusesIt proves that the use case
// answers the error of the storage, and that a refused price leaves nothing
// behind.
func TestThePriceIsNotWrittenWhenTheStorageRefusesIt(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{err: product.ErrInvalidValue}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, product.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, product.ErrInvalidValue)
	}
	if len(products.written) != 0 {
		t.Errorf("the storage holds %d products, want none", len(products.written))
	}
	if len(prices.written) != 0 {
		t.Errorf("the storage holds %d prices, want none", len(prices.written))
	}
}

// TestThePriceOfTheProductThatIsNotWrittenLeavesNothingBehind proves that
// the product and the price are written whole or not at all.
func TestThePriceOfTheProductThatIsNotWrittenLeavesNothingBehind(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{err: product.ErrInvalidValue}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, product.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, product.ErrInvalidValue)
	}
	if len(products.written) != 0 {
		t.Errorf("the storage holds %d products, want none", len(products.written))
	}
}

// TestTheTaxTreatmentOfAProductThatIsAlreadyInTheBookIsWritten proves that
// the tax treatment is a fact the owner states, and that the statement of
// an existing product is written.
func TestTheTaxTreatmentOfAProductThatIsAlreadyInTheBookIsWritten(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{held: []product.Product{{
		ID:         product.ID(10),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(5),
		Name:       "Pan de molde",
	}}}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID:   77,
		RecipeID:     5,
		Channel:      "retail",
		Amount:       3000,
		TaxTreatment: "Grammy 19",
	})
	if err != nil {
		t.Fatalf("the price was refused: %v", err)
	}

	if len(products.stated) != 1 {
		t.Fatalf("the storage holds %d statements, want 1", len(products.stated))
	}
	if products.stated[0].TaxTreatment != "Grammy 19" {
		t.Errorf("the tax treatment of the product is %q, want %q",
			products.stated[0].TaxTreatment, "Grammy 19")
	}
}

// TestTheTaxTreatmentThatTheStorageRefusesLeavesNothingBehind proves that a
// refused statement of the tax treatment leaves the price list as it was.
func TestTheTaxTreatmentThatTheStorageRefusesLeavesNothingBehind(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{
		held: []product.Product{{
			ID:         product.ID(10),
			BusinessID: business.ID(77),
			RecipeID:   recipe.ID(5),
			Name:       "Pan de molde",
		}},
		stateErr: product.ErrInvalidValue,
	}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID:   77,
		RecipeID:     5,
		Channel:      "retail",
		Amount:       3000,
		TaxTreatment: "Grammy 19",
	})
	if !errors.Is(err, product.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, product.ErrInvalidValue)
	}
	if len(prices.written) != 0 {
		t.Errorf("the storage holds %d prices, want none", len(prices.written))
	}
}

// TestThePriceListIsReadIsRefusedWhenTheBusinessIsNotThere proves that the
// use case reads the business that owns the price list, and that it reads
// nothing when there is none.
func TestThePriceListIsReadIsRefusedWhenTheBusinessIsNotThere(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{}
	service := application.NewPriceService(products, prices, recipes, &fakeBusinesses{},
		&fakePriceWork{products: products, prices: prices}, &fakePriceClock{})

	_, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77})
	if !errors.Is(err, business.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, business.ErrNotFound)
	}
}

// TestThePriceListIsRefusedWhenTheChannelIsNotOneTheBookKnows proves that
// the use case asks the domain to read the channel, and that the reading
// refuses a channel the book does not know.
func TestThePriceListIsRefusedWhenTheChannelIsNotOneTheBookKnows(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77, Channel: "store"})
	if !errors.Is(err, product.ErrUnknownChannel) {
		t.Fatalf("the error is %v, want %v", err, product.ErrUnknownChannel)
	}
}

// TestThePriceListIsRefusedWhenTheStorageRefusesIt proves that the use
// case answers the error of the storage, and that it does not hide it.
func TestThePriceListIsRefusedWhenTheStorageRefusesIt(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{err: product.ErrInvalidValue}
	prices := &fakePrices{}
	recipes := &fakeRecipes{}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	if _, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77}); !errors.Is(err, product.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, product.ErrInvalidValue)
	}

	prices = &fakePrices{err: product.ErrInvalidValue}
	service = thePriceService(&fakeProducts{}, prices, recipes, &bakery)

	if _, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77}); !errors.Is(err, product.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, product.ErrInvalidValue)
	}
}

// TestThePriceListOfAProductThatIsNotThereIsRefused proves that the price
// list of a product the business does not hold is the price list of
// nobody.
func TestThePriceListOfAProductThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77, RecipeID: 5})
	if !errors.Is(err, product.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, product.ErrNotFound)
	}

	_, err = service.Read(t.Context(), ports.ReadPrices{BusinessID: 77, Name: "Biznaga"})
	if !errors.Is(err, product.ErrUnknownRecipe) {
		t.Fatalf("the error is %v, want %v", err, product.ErrUnknownRecipe)
	}
}

// TestThePriceListOfOneProductIsRead proves that the price list can be read
// for one product alone.
func TestThePriceListOfOneProductIsRead(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{held: []product.Product{{
		ID:         product.ID(10),
		BusinessID: business.ID(77),
		RecipeID:   recipe.ID(5),
		Name:       "Pan de molde",
	}}}
	prices := &fakePrices{held: []product.Price{{
		ID:         product.PriceID(1),
		BusinessID: business.ID(77),
		ProductID:  product.ID(10),
		Channel:    product.ChannelRetail,
		Amount:     ledger.Amount(3000),
		Day:        theDayOfThePrices(t, "2026-03-01"),
	}}}
	recipes := &fakeRecipes{}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	held, err := service.Read(t.Context(), ports.ReadPrices{BusinessID: 77, Name: "Pan de molde"})
	if err != nil {
		t.Fatalf("the price list was refused: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the answer holds %d prices, want 1", len(held))
	}
	if held[0].Product.ID != product.ID(10) {
		t.Errorf("the product of the line is %d, want 10", held[0].Product.ID)
	}
}

// TestTheProductIsRefusedWhenTheRecipeIsNotThere proves that the recipe
// the product comes from is the one the business holds, and that a recipe
// that is not there is refused.
func TestTheProductIsRefusedWhenTheRecipeIsNotThere(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, recipe.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrNotFound)
	}
}

// TestTheProductIsRefusedWhenTheRecipeIsAmbiguous proves that the book
// never guesses which recipe the owner meant.
func TestTheProductIsRefusedWhenTheRecipeIsAmbiguous(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{
		theRecipeOfThePrices(),
		{ID: recipe.ID(6), BusinessID: business.ID(77), Name: "Pan de molde"},
	}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		Name:       "Pan de molde",
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, recipe.ErrAmbiguousRecipe) {
		t.Fatalf("the error is %v, want %v", err, recipe.ErrAmbiguousRecipe)
	}
}

// TestTheProductIsNamedByTheNameOfItsRecipe proves that the owner may name
// the recipe the product comes from, and that the product opens with it.
func TestTheProductIsNamedByTheNameOfItsRecipe(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeName: "Pan de molde",
		Channel:    "retail",
		Amount:     3000,
	})
	if err != nil {
		t.Fatalf("the price was refused: %v", err)
	}

	if len(products.written) != 1 {
		t.Fatalf("the storage holds %d products, want 1", len(products.written))
	}
	if products.written[0].RecipeID != recipe.ID(5) {
		t.Errorf("the recipe of the product is %d, want 5", products.written[0].RecipeID)
	}
	if products.written[0].Name != "Pan de molde" {
		t.Errorf("the name of the product is %q, want %q", products.written[0].Name, "Pan de molde")
	}
}

// TestTheProductThatCannotBeOpenedLeavesNothingBehind proves that a
// product the storage refuses leaves the price list as it was.
func TestTheProductThatCannotBeOpenedLeavesNothingBehind(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{createErr: product.ErrInvalidValue}
	prices := &fakePrices{}
	recipes := &fakeRecipes{held: []recipe.Recipe{theRecipeOfThePrices()}}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeID:   5,
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, product.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, product.ErrInvalidValue)
	}
	if len(prices.written) != 0 {
		t.Errorf("the storage holds %d prices, want none", len(prices.written))
	}
}

// TestThePriceIsRefusedWhenTheRecipesCannotBeRead proves that the use case
// answers the error of the storage of the recipes, and that it writes
// nothing when the recipes cannot be read.
func TestThePriceIsRefusedWhenTheRecipesCannotBeRead(t *testing.T) {
	t.Parallel()

	products := &fakeProducts{}
	prices := &fakePrices{}
	recipes := &fakeRecipes{err: product.ErrInvalidValue}
	bakery := theBakeryOfThePrices()
	service := thePriceService(products, prices, recipes, &bakery)

	_, err := service.Set(t.Context(), ports.SetPrice{
		BusinessID: 77,
		RecipeName: "Pan de molde",
		Channel:    "retail",
		Amount:     3000,
	})
	if !errors.Is(err, product.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, product.ErrInvalidValue)
	}
	if len(products.written) != 0 {
		t.Errorf("the storage holds %d products, want none", len(products.written))
	}
}
