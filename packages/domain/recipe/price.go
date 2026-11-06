package recipe

import (
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// Rate is a money of the book with the rates it weighs against.
//
// The money is whole pesos, and the rates are not money in the book: they
// are what the money weighs per unit, per gram and per kilogram. The three
// answers travel together, because a comparison between two products that
// knows only one of them is a lie.
type Rate struct {
	// Money is the money of the layer, in whole pesos.
	Money ledger.Amount

	// PerUnit is the money per unit of what the recipe makes.
	PerUnit float64

	// PerGram is the money per gram of what the recipe makes.
	PerGram float64

	// PerKilogram is the money per kilogram of what the recipe makes.
	PerKilogram float64

	// Unknown says that the money is not derivable: the owner declared
	// none of it, and the money of the book is never inferred.
	Unknown bool

	// Reason says why the money is not derivable, in the words of the
	// book.
	Reason string
}

// RealPrice is the report of the real price of a product.
//
// The three layers of the cost of the batch travel apart: the direct cost,
// the labour and the overhead, and their sum apart. No layer is ever
// hidden inside another one, and the price of the owner travels beside
// them, as she declared it.
//
// The report answers per unit, per gram and per kilogram, because the
// bakery sells in two ways and a comparison that knows only one of them is
// a lie.
type RealPrice struct {
	// RecipeID is the recipe the report is of.
	RecipeID ID

	// Name is the name the owner reads of the recipe.
	Name string

	// Yield is what the recipe makes: the units it makes and the weight
	// of each one.
	Yield Yield

	// Direct is the direct cost of the batch: the money of the layers of
	// the stock the recipe consumes.
	//
	// It is a layer of the cost of its own: it is never added to the
	// labour nor to the overhead.
	Direct Rate

	// Labor is the labour of the batch: the money of the hours of the
	// steps of the recipe.
	//
	// It is a layer of the cost of its own: it is never added to the
	// direct cost nor to the overhead.
	Labor Rate

	// Overhead is the overhead of the batch: the money of the hours of
	// the oven of the steps of the recipe.
	//
	// It is a layer of the cost of its own: it is never added to the
	// direct cost nor to the labour.
	Overhead Rate

	// Total is the sum of the three layers, shown apart.
	//
	// It is a number of its own and never a layer: the layers keep their
	// own money beside it, and the sum never hides one of them.
	Total Rate

	// Price is the price of the owner: what she charges, as she declared
	// it. The book never moves it.
	Price Rate

	// PriceBasis is the way she charges: per unit, per gram or per
	// kilogram. It is empty when she declared no price.
	PriceBasis purchase.PriceBasis

	// Complete says that the stock holds everything the recipe needs: the
	// report of a batch that cannot be baked is not the report of the
	// batch.
	Complete bool

	// Missing is what the report still needs: what the stock does not
	// hold, and the price of the owner when she declared none.
	Missing []string
}

// RealPriceOf builds the report of the real price of a recipe.
//
// The rule of the domain: the report shows the three layers of the cost
// apart — the direct cost, the labour and the overhead — per unit, per gram
// and per kilogram, and the price of the owner beside them.
//
// The rule of the domain: the report never hides a layer inside another
// one. The three layers keep their own money and their own rates, and their
// sum travels apart, beside them.
//
// The rule of the domain: the price of the owner is hers, and the book
// never moves it. The report carries the money she declared and the rates
// it weighs against, and a price nobody declared says that it is not known:
// the money of what she sells is never inferred.
//
// The sum of the three layers is the sum of a batch that can be baked: a
// layer that is not known, or a stock that does not hold everything the
// recipe needs, leaves the total unknown, and the answer says why.
func RealPriceOf(cost Cost, price purchase.OwnerPrice) RealPrice {
	report := RealPrice{
		RecipeID: cost.RecipeID,
		Name:     cost.Name,
		Yield:    cost.Yield,
		Complete: cost.Complete,
		Missing:  make([]string, 0, len(cost.Missing)+1),
	}

	report.Missing = append(report.Missing, cost.Missing...)

	report.Direct = rateOfTheLayer(cost.Direct, cost.Yield, false, "")
	report.Labor = rateOfTheLayer(cost.Labor.Money, cost.Yield, cost.Labor.Unknown, cost.Labor.Reason)
	report.Overhead = rateOfTheLayer(cost.Overhead.Money, cost.Yield, cost.Overhead.Unknown, cost.Overhead.Reason)
	report.Total = totalOfTheLayers(report, cost)
	report.Price, report.PriceBasis = rateOfThePriceOfTheOwner(price, cost.Yield)

	if report.Price.Unknown {
		report.Missing = append(report.Missing, report.Price.Reason)
	}

	return report
}

// rateOfTheLayer answers the money of a layer of the cost with the rates
// it weighs against.
//
// The rule of the domain: a layer that is not known keeps its money at
// nothing and says why: the money of the labour and the money of the
// overhead are never inferred.
func rateOfTheLayer(money ledger.Amount, made Yield, unknown bool, reason string) Rate {
	rate := Rate{Money: money, Unknown: unknown, Reason: reason}

	if unknown {
		return rate
	}

	rate.PerUnit, rate.PerGram, rate.PerKilogram = ratesOf(money, made)

	return rate
}

// totalOfTheLayers answers the sum of the three layers of the cost.
//
// The rule of the domain: the sum is a number of its own and never a layer:
// the three layers keep their money beside it, and the sum never hides one
// of them.
//
// The sum is unknown when one of the three layers is unknown, and when the
// stock does not hold everything the recipe needs: the sum of a batch that
// cannot be baked is not the money of the batch.
func totalOfTheLayers(report RealPrice, cost Cost) Rate {
	switch {
	case cost.Labor.Unknown:
		return Rate{Unknown: true, Reason: cost.Labor.Reason}
	case cost.Overhead.Unknown:
		return Rate{Unknown: true, Reason: cost.Overhead.Reason}
	case !cost.Complete:
		return Rate{Unknown: true, Reason: "the stock does not hold everything the recipe needs"}
	}

	money := report.Direct.Money + report.Labor.Money + report.Overhead.Money
	total := Rate{Money: money}
	total.PerUnit, total.PerGram, total.PerKilogram = ratesOf(money, cost.Yield)

	return total
}

// rateOfThePriceOfTheOwner answers the price of the owner with the rates
// it weighs against.
//
// The rule of the domain: the price of the owner is hers, and the book
// never moves it. The report carries the money she declared, and the rates
// it weighs against, so that the comparison with the layers of the cost is
// a comparison of two numbers that mean the same thing.
//
// The price of the unit is the price of one piece of what the recipe makes,
// and the gram of it is the weight the owner declared. The price of the
// gram and the price of the kilogram are the same price said in two ways,
// and the price of the unit is derived from the weight of the piece.
//
// The price that the owner did not declare says that it is not known: the
// money of what she sells is never inferred.
func rateOfThePriceOfTheOwner(price purchase.OwnerPrice, made Yield) (Rate, purchase.PriceBasis) {
	if price.Money <= 0 {
		return Rate{
			Unknown: true,
			Reason:  "the owner has not declared the price of the product",
		}, price.Basis
	}

	rate := Rate{Money: price.Money}

	if made.Units <= 0 || made.Grams <= 0 {
		rate.Unknown = true
		rate.Reason = "the recipe declares no yield, and the price of the owner has no rate"

		return rate, price.Basis
	}

	grams := made.Grams

	switch price.Basis {
	case purchase.PricePerUnit:
		rate.PerUnit = float64(price.Money)
		rate.PerGram = float64(price.Money) / grams
		rate.PerKilogram = rate.PerGram * ingredient.GramsPerKilogram
	case purchase.PricePerGram:
		rate.PerGram = float64(price.Money)
		rate.PerKilogram = float64(price.Money) * ingredient.GramsPerKilogram
		rate.PerUnit = float64(price.Money) * grams
	case purchase.PricePerKilogram:
		rate.PerKilogram = float64(price.Money)
		rate.PerGram = float64(price.Money) / ingredient.GramsPerKilogram
		rate.PerUnit = float64(price.Money) * grams / ingredient.GramsPerKilogram
	default:
		rate.Unknown = true
		rate.Reason = "the owner has not declared the way she charges"
	}

	return rate, price.Basis
}
