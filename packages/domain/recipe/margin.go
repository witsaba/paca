package recipe

import (
	"fmt"
	"math"
	"sort"

	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The margin of a product: what the owner charges above the direct cost of
// the batch, and what the market said about the same ingredients.
//
// The margin lives here, next to the cost of the batch and the report of the
// real price it is built upon: the margin needs the yield of the recipe
// and the cost of the batch, and the direction of the dependencies forbids
// the package of the purchase from reaching for them.
//
// The price of the owner is hers: the margin reads it and never moves it.
// The reference of the market is a REFERENCE: the margin reads it and never
// turns it into the price of the ingredient, and no money of the book ever
// moves with it.

// ReferenceExpectation is what the owner expects of the age of a reference.
//
// The owner says how old a reference may be, and the answer shows the age
// of every reference and says which ones are older than she expects.
type ReferenceExpectation struct {
	// MaxAgeDays is the number of days a reference may be old.
	//
	// It is zero when the owner expects no age: every reference then
	// shows its age, and none of them is old.
	MaxAgeDays int
}

// Profit is what a price leaves above a cost.
//
// It is a rate and never a money: the price of the owner and the direct
// cost of the batch are two money facts of their own, and their
// difference is what they weigh against, per unit, per gram and per
// kilogram.
//
// The profit of what nobody declared is not known: the money of what the
// owner sells is never inferred, and the money of the book is never
// inferred either.
type Profit struct {
	// PerUnit is the profit per unit of what the recipe makes.
	PerUnit float64

	// PerGram is the profit per gram of what the recipe makes.
	PerGram float64

	// PerKilogram is the profit per kilogram of what the recipe makes.
	PerKilogram float64

	// Unknown says that the profit is not derivable, and the reason says
	// why.
	Unknown bool

	// Reason says why the profit is not derivable, in the words of the
	// book.
	Reason string
}

// ReferenceView is a reference of the market as the margin shows it.
//
// Every reference carries its source and its day, and the age it has on
// the day of the reading: a reference older than the owner expects is
// shown with its age.
type ReferenceView struct {
	// ID is the identity of the reference.
	ID purchase.ReferenceID

	// IngredientID is the ingredient of the catalog the reference is about.
	IngredientID ingredient.ID

	// Price is the price that was seen, in whole pesos. It is the price
	// of the market and never the price of the ingredient.
	Price ledger.Amount

	// Unit is the unit of that price.
	Unit string

	// Market is the market where the price was seen. It is empty when the
	// research named none.
	Market string

	// Source is the place the price was read. It is never empty.
	Source string

	// SeenOn is the day the price was seen.
	SeenOn ledger.Date

	// AgeDays is how many days old the reference is on the day of the
	// reading.
	AgeDays int

	// Old says that the reference is older than the owner expects.
	Old bool
}

// MarketLine is what the market said about one ingredient of the batch.
//
// The line carries the two truths side by side: what the owner paid for
// the grams of the recipe, and what the market says they cost.
type MarketLine struct {
	// IngredientID is the ingredient of the catalog the line is of.
	IngredientID ingredient.ID

	// Name is the name the owner reads of the ingredient.
	Name string

	// Grams is how many grams of it the recipe needs.
	Grams float64

	// Paid is what the owner paid for those grams: the direct cost of the
	// line, in whole pesos.
	//
	// It comes from the layers of the stock: a reference never moves the
	// money of the book.
	Paid ledger.Amount

	// Money is what the market says that many grams cost, in whole pesos.
	Money ledger.Amount

	// Unknown says that the money of the market is not derivable, and the
	// reason says why.
	Unknown bool

	// Reason says why the money of the market is not derivable, in the
	// words of the book.
	Reason string

	// Reference is the reference that compares: the newest one of the
	// ingredient.
	Reference ReferenceView

	// References are every reference of the ingredient, the newest one
	// first, each one with its source, its day and its age.
	References []ReferenceView
}

// Market is what the market said about the ingredients of the batch.
//
// The money of the market is what the market says the ingredients of the
// batch cost, and the money the owner paid is what she really paid: the
// difference between the two is what a later analysis reads.
type Market struct {
	// Money is what the market says the ingredients of the batch cost, in
	// whole pesos.
	//
	// It is zero when the market is not known: a sum that hides a missing
	// part is a lie.
	Money ledger.Amount

	// Paid is what the owner paid for the ingredients of the batch: the
	// direct cost of the batch, in whole pesos.
	Paid ledger.Amount

	// Difference is what the owner paid above the market. It is positive
	// when she pays more than the market says.
	Difference ledger.Amount

	// PerUnit is the money of the market per unit of what the recipe
	// makes.
	PerUnit float64

	// PerGram is the money of the market per gram of what the recipe
	// makes.
	PerGram float64

	// PerKilogram is the money of the market per kilogram of what the
	// recipe makes.
	PerKilogram float64

	// Unknown says that the money of the market is not derivable, and the
	// reason says why.
	Unknown bool

	// Reason says why the money of the market is not derivable, in the
	// words of the book.
	Reason string

	// Lines are the lines of the market: one for every ingredient of the
	// recipe, in the order the owner said them.
	Lines []MarketLine
}

// Margin is the margin of a product.
//
// It compares the price of the owner with the direct cost of the batch,
// and with the reference of the market. The two comparisons travel apart:
// the profit of the product and the profit over the market are two
// numbers of their own, and neither of them hides the other.
type Margin struct {
	// RecipeID is the recipe the margin is of.
	RecipeID ID

	// Name is the name the owner reads of the recipe.
	Name string

	// Yield is what the recipe makes: the units it makes and the weight
	// of each one.
	Yield Yield

	// Price is the price of the owner: what she charges, as she declared
	// it. The book never moves it.
	Price Rate

	// PriceBasis is the way she charges: per unit, per gram or per
	// kilogram. It is empty when she declared no price.
	PriceBasis purchase.PriceBasis

	// Direct is the direct cost of the batch, with the rates it weighs
	// against.
	Direct Rate

	// Profit is what the price of the owner leaves above the direct cost
	// of the batch.
	Profit Profit

	// Percent is the profit over the direct cost, in percent.
	Percent float64

	// Market is what the market said about the ingredients of the batch,
	// with the references that say it and their age.
	Market Market

	// MarketProfit is what the price of the owner leaves above what the
	// market said.
	MarketProfit Profit

	// Missing is what the margin still needs: what the stock does not
	// hold, the price of the owner when she declared none, and what the
	// market still does not say.
	Missing []string
}

// MarginOf builds the margin of a product.
//
// The rule of the domain: the margin compares the price of the owner with
// the direct cost of the batch, and with the reference of the market. The
// profit of the product and the profit over the market are two numbers of
// their own, and neither of them hides the other.
//
// The rule of the domain: the price of the owner is hers, and the book
// never moves it. The margin carries the money she declared and the rates
// it weighs against, and a price nobody declared says that it is not known.
//
// The rule of the domain: a reference is a reference, and it never moves
// the money of the book. The direct cost of the batch comes from the
// layers of the stock, and the reference is read beside it: what the owner
// paid and what the market says are two truths that never mix.
//
// The rule of the domain: every reference carries its source and its day,
// and a reference older than the owner expects is shown with its age.
//
// The profit is unknown when the price of the owner is unknown, and when
// the stock does not hold everything the recipe needs: the margin of a
// batch that cannot be baked is not the margin of the product.
//
// The money of the market is unknown when one of the ingredients has no
// reference, when the reference is in a unit the book never converts, and
// when the weight of the package is not declared: a sum that hides a
// missing part is a lie, and the answer says what is missing.
func MarginOf(
	cost Cost,
	price purchase.OwnerPrice,
	references []purchase.Reference,
	catalog ingredient.Catalog,
	expectation ReferenceExpectation,
	day ledger.Date,
) Margin {
	margin := Margin{
		RecipeID: cost.RecipeID,
		Name:     cost.Name,
		Yield:    cost.Yield,
		Missing:  make([]string, 0, len(cost.Missing)+2),
	}

	margin.Missing = append(margin.Missing, cost.Missing...)

	margin.Price, margin.PriceBasis = rateOfThePriceOfTheOwner(price, cost.Yield)
	margin.Direct = rateOfTheLayer(cost.Direct, cost.Yield, false, "")
	margin.Profit = profitOfThePrice(cost, margin.Price)
	margin.Percent = percentOfTheProfit(margin.Profit, cost)
	margin.Market = marketOfTheIngredients(cost, references, catalog, expectation, day)
	margin.MarketProfit = profitOfTheMarket(margin.Price, margin.Market)

	if margin.Price.Unknown {
		margin.Missing = append(margin.Missing, margin.Price.Reason)
	}

	if margin.Market.Unknown {
		margin.Missing = append(margin.Missing, margin.Market.Reason)
	}

	return margin
}

// profitOfThePrice answers what the price of the owner leaves above the
// direct cost of the batch.
//
// The rule of the domain: the money of what the owner sells is never
// inferred, and the profit of a price nobody declared is not known.
//
// The rule of the domain: the sum of a batch that cannot be baked is not
// the money of the batch: the profit of a batch the stock does not cover
// is not known either.
func profitOfThePrice(cost Cost, price Rate) Profit {
	switch {
	case price.Unknown:
		return Profit{Unknown: true, Reason: price.Reason}
	case !cost.Complete:
		return Profit{Unknown: true, Reason: "the stock does not hold everything the recipe needs"}
	}

	return Profit{
		PerUnit:     price.PerUnit - cost.PerUnit,
		PerGram:     price.PerGram - cost.PerGram,
		PerKilogram: price.PerKilogram - cost.PerKilogram,
	}
}

// percentOfTheProfit answers the profit over the direct cost, in percent.
//
// The percent is zero when the profit is not known, and when the direct
// cost is nothing: the percent of nothing is not a number.
func percentOfTheProfit(profit Profit, cost Cost) float64 {
	if profit.Unknown || cost.PerUnit <= 0 {
		return 0
	}

	return profit.PerUnit / cost.PerUnit * 100
}

// profitOfTheMarket answers what the price of the owner leaves above what
// the market said.
//
// The rule of the domain: the price of the owner and the money of the
// market are two truths of their own, and their difference is a number of
// its own.
//
// The profit over the market is unknown when the price of the owner is
// unknown, and when the money of the market is: the money of the book is
// never inferred.
func profitOfTheMarket(price Rate, market Market) Profit {
	switch {
	case price.Unknown:
		return Profit{Unknown: true, Reason: price.Reason}
	case market.Unknown:
		return Profit{Unknown: true, Reason: market.Reason}
	}

	return Profit{
		PerUnit:     price.PerUnit - market.PerUnit,
		PerGram:     price.PerGram - market.PerGram,
		PerKilogram: price.PerKilogram - market.PerKilogram,
	}
}

// marketOfTheIngredients answers what the market said about the ingredients
// of the batch.
//
// The rule of the domain: the reference is a reference, and it never moves
// the money of the book. The money the owner paid comes from the layers of
// the stock, and the money of the market comes from the references, and the
// two travel side by side.
//
// The rule of the domain: the newest reference of an ingredient is the one
// that compares - the market of today is the last one that was seen - and
// the older ones travel with it as the history they are.
//
// The money of the market is unknown when one of the lines is: the lines
// carry the truth of every ingredient, and the sum never hides the one
// that is missing.
func marketOfTheIngredients(
	cost Cost,
	references []purchase.Reference,
	catalog ingredient.Catalog,
	expectation ReferenceExpectation,
	day ledger.Date,
) Market {
	market := Market{Lines: make([]MarketLine, 0, len(cost.Lines))}
	money := ledger.Amount(0)
	paid := ledger.Amount(0)
	unknown := false
	reason := ""

	for _, line := range cost.Lines {
		entry := MarketLine{
			IngredientID: line.IngredientID,
			Name:         line.Name,
			Grams:        line.Grams,
			Paid:         line.Money,
		}

		found := referencesOfTheIngredient(references, line.IngredientID, expectation, day)
		entry.References = found

		switch {
		case len(found) == 0:
			entry.Unknown = true
			entry.Reason = fmt.Sprintf("the market has no reference of %s", nameOfTheLine(line))
		case line.Unknown:
			// The money the owner paid for the line is not derivable,
			// and the difference between what she paid and what the
			// market says needs it: the line is not whole.
			entry.Unknown = true
			entry.Reason = fmt.Sprintf("the money of %s is not derivable", nameOfTheLine(line))
		default:
			entry.Reference = found[0]

			item, held := catalog.Find(line.IngredientID)

			value, why := moneyOfTheMarket(found[0], line.Grams, item, held, nameOfTheLine(line))
			entry.Money = value

			if why != "" {
				entry.Unknown = true
				entry.Reason = why
			}
		}

		if entry.Unknown {
			unknown = true

			if reason == "" {
				reason = entry.Reason
			}
		} else {
			money += entry.Money
		}

		paid += entry.Paid

		market.Lines = append(market.Lines, entry)
	}

	market.Paid = paid

	if unknown {
		market.Unknown = true
		market.Reason = reason

		return market
	}

	market.Money = money
	market.Difference = paid - money
	market.PerUnit, market.PerGram, market.PerKilogram = ratesOf(money, cost.Yield)

	return market
}

// referencesOfTheIngredient answers the references of an ingredient, the
// newest one first.
//
// The rule of the domain: the market of today is the last one that was
// seen, and the newest reference is the one that compares. The older ones
// travel with it, with their source, their day and their age: the history
// is what a later analysis reads.
func referencesOfTheIngredient(
	references []purchase.Reference,
	id ingredient.ID,
	expectation ReferenceExpectation,
	day ledger.Date,
) []ReferenceView {
	found := make([]purchase.Reference, 0)

	for _, one := range references {
		if one.IngredientID == id {
			found = append(found, one)
		}
	}

	sort.SliceStable(found, func(i, j int) bool {
		if found[i].SeenOn.Time().Equal(found[j].SeenOn.Time()) {
			return found[i].ID > found[j].ID
		}

		return found[i].SeenOn.After(found[j].SeenOn)
	})

	views := make([]ReferenceView, 0, len(found))

	for _, one := range found {
		views = append(views, ReferenceView{
			ID:           one.ID,
			IngredientID: one.IngredientID,
			Price:        one.Price,
			Unit:         one.Unit,
			Market:       one.Market,
			Source:       one.Source,
			SeenOn:       one.SeenOn,
			AgeDays:      one.AgeOn(day),
			Old:          one.IsOlderThan(day, expectation.MaxAgeDays),
		})
	}

	return views
}

// moneyOfTheMarket answers what the market says that many grams cost.
//
// The rule of the domain: the money of the book counts whole pesos, and
// the money of the market is rounded to the nearest peso.
//
// The rule of the domain: the unit of a reference is the unit the book
// knows, and only the gram and the kilogram are converted into the grams
// of the recipe. The package is converted only when the owner declared
// the weight of it, and the piece is never converted: the weight of a
// piece is not known until she says it.
func moneyOfTheMarket(
	reference ReferenceView,
	grams float64,
	item ingredient.Ingredient,
	held bool,
	name string,
) (ledger.Amount, string) {
	switch ingredient.Unit(reference.Unit) {
	case ingredient.UnitGram:
		return ledger.Amount(math.Round(float64(reference.Price) * grams)), ""
	case ingredient.UnitKilogram:
		return ledger.Amount(math.Round(float64(reference.Price) * grams / ingredient.GramsPerKilogram)), ""
	case ingredient.UnitPackage:
		if !held || item.PackageGrams <= 0 {
			return 0, fmt.Sprintf("the weight of the package of %s is not declared", name)
		}

		return ledger.Amount(math.Round(float64(reference.Price) * grams / item.PackageGrams)), ""
	default:
		return 0, fmt.Sprintf("the weight of a piece of %s is never assumed", name)
	}
}

// nameOfTheLine answers the name the owner reads of the line of the cost.
//
// The name is empty when the catalog does not know the ingredient, and the
// identity of the ingredient answers in its place: a line without a name is
// a line nobody can read.
func nameOfTheLine(line CostLine) string {
	if line.Name != "" {
		return line.Name
	}

	return fmt.Sprintf("the ingredient %d", line.IngredientID)
}
