package purchase

import (
	"errors"
	"fmt"
	"math"

	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// Take is what a batch took from one layer of the stock.
//
// The layer is the part of a purchase the business still holds, and a take
// is the part of it that a batch consumed: the money of the take is the
// money of the layer, pro rata the quantity that was taken.
type Take struct {
	// LayerID is the layer the batch took from.
	LayerID LayerID

	// PurchaseID is the purchase the layer comes from: the purchase is the
	// fact, and the layer is derived from it.
	PurchaseID ID

	// Day is the day the layer was bought.
	Day ledger.Date

	// Quantity is what was taken, in the unit of the layer.
	Quantity float64

	// Unit is the unit of that quantity.
	Unit string

	// Grams is what was taken, in grams.
	Grams float64

	// Money is the money of what was taken, in whole pesos.
	Money ledger.Amount
}

// Consumption is what a batch took from the stock of one ingredient.
//
// The recipe consumes the layers in the order of the FIFO: the oldest
// layer is the first one that leaves.
type Consumption struct {
	// IngredientID is the ingredient the batch consumed.
	IngredientID ingredient.ID

	// Grams is how many grams the batch needs.
	Grams float64

	// Taken is how many grams the stock could give.
	Taken float64

	// Missing is how many grams the stock does not hold.
	Missing float64

	// Money is the money of the grams taken, in whole pesos.
	Money ledger.Amount

	// Takes are the layers the batch took from, in the order they leave the
	// stock: the oldest one is the first.
	Takes []Take

	// Unknown says that the money of the ingredient is not derivable: the
	// stock holds it in a unit the book never converts into grams, and the
	// weight of a piece is never assumed.
	Unknown bool

	// Reason says why the money is not derivable, in the words of the book.
	Reason string
}

// MoneyLeft answers the money of what is left of the layer.
//
// The rule of the domain: the money of what is left of a layer is the money
// of what it bought, pro rata the quantity that is left.
//
// A layer that bought twenty-five kilos of flour for twelve thousand pesos
// and that holds five kilos left is worth two thousand four hundred pesos:
// the money follows the quantity, and the quantity and its unit travel
// together.
//
// The money of the book counts whole pesos: the answer is rounded to the
// nearest peso.
func (l Layer) MoneyLeft() ledger.Amount {
	return l.moneyOf(l.Left.Quantity)
}

// moneyOf answers the money of a quantity of the layer.
//
// The money of a quantity is the money of what the layer bought, pro rata
// the quantity: the money of what is left of a layer is the money of what
// it bought, and the money of what left it is the difference of the two.
//
// The money of the book counts whole pesos: the answer is rounded to the
// nearest peso.
func (l Layer) moneyOf(quantity float64) ledger.Amount {
	if l.Bought.Quantity <= 0 || quantity <= 0 {
		return 0
	}

	if quantity > l.Bought.Quantity {
		quantity = l.Bought.Quantity
	}

	return ledger.Amount(math.Round(float64(l.Cost) * quantity / l.Bought.Quantity))
}

// Consume takes the grams of an ingredient from the layers that hold it.
//
// The rule of the domain: the recipe consumes the layers in the order of
// the FIFO - the oldest layer is the first one that leaves - and the money
// of what it takes is the money of the layer, pro rata the quantity that
// was taken.
//
// The quantity and its unit travel together: the grams the batch needs are
// taken from every layer in the unit of that layer, and the conversion of
// a unit into the gram is declared, never assumed. The gram is one, the
// kilogram is a thousand, the package is the weight the owner declares,
// and the piece is one unit of the ingredient, whose weight the book never
// assumes: a layer the book cannot convert is not consumed, and the money
// of that ingredient is not derivable.
//
// The answer never invents what the stock does not hold: it says how many
// grams are missing.
//
// The error is ErrInvalidQuantity when the batch needs no grams.
func Consume(layers []Layer, of ingredient.ID, grams float64, item ingredient.Ingredient) (Consumption, error) {
	if grams <= 0 {
		return Consumption{}, fmt.Errorf("%w: the batch needs %v grams of the ingredient %d",
			ErrInvalidQuantity, grams, of)
	}

	consumption := Consumption{
		IngredientID: of,
		Grams:        grams,
		Takes:        []Take{},
	}

	needed := grams

	for _, one := range byFIFO(OfIngredient(layers, of)) {
		if needed <= 0 {
			break
		}

		if !one.Open() {
			continue
		}

		perUnit, err := gramsPerUnit(one.Bought.Unit, item)
		if err != nil {
			consumption.Unknown = true
			consumption.Reason = reasonOf(err, item)

			continue
		}

		available := one.Left.Quantity * perUnit
		if available <= 0 {
			continue
		}

		taken := available
		if taken > needed {
			taken = needed
		}

		takenInUnits := taken / perUnit
		left := one.Left.Quantity - takenInUnits
		money := one.Cost - one.moneyOf(left)

		consumption.Takes = append(consumption.Takes, Take{
			LayerID:    one.ID,
			PurchaseID: one.PurchaseID,
			Day:        one.Day,
			Quantity:   takenInUnits,
			Unit:       one.Bought.Unit,
			Grams:      taken,
			Money:      money,
		})

		consumption.Taken += taken
		consumption.Money += money
		needed -= taken
	}

	consumption.Missing = needed

	return consumption, nil
}

// gramsPerUnit answers how many grams one unit of the layer weighs.
//
// The conversion is declared, never assumed: the gram is one, the kilogram
// is a thousand, the package is the weight the owner declares, and the
// piece is one unit of the ingredient, whose weight the book never assumes.
//
// The errors are ingredient.ErrMissingPackageWeight when the owner has not
// declared the weight of a package, ingredient.ErrUnknownConversion when
// the piece is never converted into grams, and ingredient.ErrUnknownUnit
// when the unit is not one of the units of the book.
func gramsPerUnit(unit string, item ingredient.Ingredient) (float64, error) {
	switch ingredient.Unit(unit) {
	case ingredient.UnitGram:
		return 1, nil
	case ingredient.UnitKilogram:
		return ingredient.GramsPerKilogram, nil
	case ingredient.UnitPackage:
		if item.PackageGrams <= 0 {
			return 0, fmt.Errorf("%w: the ingredient %q", ingredient.ErrMissingPackageWeight, item.Name)
		}

		return item.PackageGrams, nil
	case ingredient.UnitPiece:
		return 0, fmt.Errorf("%w: the piece is one unit of the ingredient %q",
			ingredient.ErrUnknownConversion, item.Name)
	default:
		return 0, fmt.Errorf("%w: the ingredient %q carries the unit %q",
			ingredient.ErrUnknownUnit, item.Name, unit)
	}
}

// reasonOf answers why the money of an ingredient is not derivable, in the
// words of the book.
//
// The reason is what the agent reads to know what to ask the owner: the
// weight of a package she has not declared, or the weight of a piece the
// book never assumes.
func reasonOf(err error, item ingredient.Ingredient) string {
	switch {
	case errors.Is(err, ingredient.ErrMissingPackageWeight):
		return "the weight of a package of " + item.Name + " is not declared"
	case errors.Is(err, ingredient.ErrUnknownConversion):
		return "the weight of a piece of " + item.Name + " is never assumed"
	default:
		return "the unit of " + item.Name + " is not one of the units of the book"
	}
}

// CheckedAgainstTheLine answers whether the layer is the layer of the line
// of the purchase it comes from.
//
// The rule of the domain: the money of the layer is the money of the line
// of the purchase, and the layer comes from that line and from no other
// one. The layer is derived from the purchase: a row that another writer
// broke with another money is refused, and it is never read as a layer.
//
// The error is ErrLayerNotTied when the layer carries another money, or
// another quantity, or another ingredient than the line it comes from.
func (l Layer) CheckedAgainstTheLine(bought Purchase, line Line, position int) error {
	derived, err := newLayer(bought, line, position)
	if err != nil {
		return err
	}

	if l.PurchaseID != derived.PurchaseID ||
		l.IngredientID != derived.IngredientID ||
		l.Bought != derived.Bought ||
		l.Cost != derived.Cost {
		return fmt.Errorf("%w: the layer of the line %d of the purchase %d carries %v %s of the ingredient %d for %d pesos, and the line costs %v %s for %d pesos",
			ErrLayerNotTied, position, bought.ID,
			l.Bought.Quantity, l.Bought.Unit, l.IngredientID, l.Cost,
			derived.Bought.Quantity, derived.Bought.Unit, derived.Cost)
	}

	return nil
}
