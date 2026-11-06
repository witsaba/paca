package recipe

import (
	"fmt"
	"strings"

	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// CostLine is the cost of one ingredient of a batch.
//
// The line carries what the recipe needs, what the stock holds and what
// the money of it was: the cost of a batch is derived from the layers that
// the recipe consumes, and never typed.
type CostLine struct {
	// IngredientID is the ingredient of the catalog the line is of.
	IngredientID ingredient.ID

	// Name is the name the owner reads of the ingredient.
	Name string

	// Grams is how many grams of it the recipe needs.
	Grams float64

	// Available is how many grams of it the stock gave to the batch.
	Available float64

	// Missing is how many grams of it the stock does not hold.
	Missing float64

	// Money is the money of the grams of the batch, in whole pesos.
	Money ledger.Amount

	// Unknown says that the money of the ingredient is not derivable: the
	// stock holds it in a unit the book never converts into grams, and the
	// weight of a piece is never assumed.
	Unknown bool

	// Layers are the layers the batch took from, in the order they leave
	// the stock: the oldest one is the first.
	Layers []purchase.Take
}

// Cost is the cost of a batch of a recipe.
//
// The cost is the direct cost of the batch: the money of the ingredients it
// consumes. The labor and the overhead are layers of their own, and they
// are never hidden inside this number: the labor of the batch is carried
// beside the direct cost, and the overhead will be carried beside it too.
type Cost struct {
	// RecipeID is the recipe the cost is of.
	RecipeID ID

	// Name is the name the owner reads of the recipe.
	Name string

	// Lines are the lines of the cost: one for every ingredient of the
	// recipe, in the order the owner said them.
	Lines []CostLine

	// Direct is the money of the batch, in whole pesos.
	Direct ledger.Amount

	// Labor is the labor of the batch: the hours of the steps of the
	// recipe, in the three kinds of the labor, and the money of them.
	//
	// It is a layer of the cost of its own: it is never added to the
	// direct cost, and the report of the real price shows the two
	// apart.
	Labor LaborCost

	// Yield is what the recipe makes: the units it makes and the weight of
	// each one.
	Yield Yield

	// PerUnit is the money of the batch per unit it makes.
	PerUnit float64

	// PerGram is the money of the batch per gram it makes.
	PerGram float64

	// PerKilogram is the money of the batch per kilogram it makes.
	PerKilogram float64

	// Complete says that the stock holds everything the recipe needs: the
	// cost of a batch that cannot be baked is not the cost of the batch.
	Complete bool

	// Missing is what the cost still needs, in the words of the book.
	Missing []string
}

// CostOf answers the cost of a batch of the recipe.
//
// The rule of the domain: the cost of a batch is derived from the layers
// that the recipe consumes, and never typed. The recipe consumes the layers
// in the order of the FIFO - the oldest layer is the first one that leaves
// - and the money of what it takes is the money of the layer, pro rata the
// quantity that was taken.
//
// The labor of the batch is derived from the hours of its steps, and it is
// carried beside the direct cost: the two are never added into one number.
//
// The cost answers per unit, per gram and per kilogram: the money of the
// book counts whole pesos, and the money of the batch is whole pesos, and
// the three answers are rates.
//
// The answer is never a promise: the cost is complete when the stock holds
// everything the recipe needs, and it says what it still needs when it does
// not.
//
// The error is the error of the domain: the recipe that needs no grams of
// an ingredient is not a recipe.
func CostOf(
	baked Recipe,
	layers []purchase.Layer,
	catalog ingredient.Catalog,
	hourValue ledger.Amount,
) (Cost, error) {
	lines := make([]CostLine, 0, len(baked.Ingredients))
	direct := ledger.Amount(0)
	missing := []string{}
	complete := true

	labor, err := LaborOf(baked, hourValue)
	if err != nil {
		return Cost{}, err
	}

	for _, one := range gramsOfTheRecipe(baked) {
		line := CostLine{
			IngredientID: one.IngredientID,
			Grams:        one.Grams,
			Layers:       []purchase.Take{},
		}

		item, held := catalog.Find(one.IngredientID)
		if !held {
			line.Unknown = true
			missing = append(missing, fmt.Sprintf("the ingredient %d is not one of the catalog", one.IngredientID))
			complete = false
			lines = append(lines, line)

			continue
		}

		line.Name = item.Name

		consumption, err := purchase.Consume(layers, one.IngredientID, one.Grams, item)
		if err != nil {
			return Cost{}, err
		}

		line.Available = consumption.Taken
		line.Missing = consumption.Missing
		line.Money = consumption.Money
		line.Layers = consumption.Takes
		line.Unknown = consumption.Unknown

		if consumption.Unknown {
			missing = append(missing, consumption.Reason)
			complete = false
		}

		if consumption.Missing > 0 {
			missing = append(missing, fmt.Sprintf("the stock does not hold %v grams of %s",
				consumption.Missing, item.Name))
			complete = false
		}

		direct += consumption.Money
		lines = append(lines, line)
	}

	cost := Cost{
		RecipeID: baked.ID,
		Name:     baked.Name,
		Lines:    lines,
		Direct:   direct,
		Labor:    labor,
		Yield:    baked.Yield,
		Complete: complete,
		Missing:  missing,
	}

	cost.PerUnit, cost.PerGram, cost.PerKilogram = ratesOf(direct, baked.Yield)

	return cost, nil
}

// gramsOfTheRecipe answers the grams every ingredient of the recipe needs.
//
// The rule of the domain: an ingredient the recipe names twice is one
// ingredient of the batch: the grams of the two are added up, and the cost
// of the batch never pays twice for the same gram.
//
// The answer keeps the order the owner said the ingredients in.
func gramsOfTheRecipe(baked Recipe) []Ingredient {
	grams := make([]Ingredient, 0, len(baked.Ingredients))
	indexes := make(map[ingredient.ID]int, len(baked.Ingredients))

	for _, one := range baked.Ingredients {
		if index, seen := indexes[one.IngredientID]; seen {
			grams[index].Grams += one.Grams

			continue
		}

		indexes[one.IngredientID] = len(grams)
		grams = append(grams, Ingredient{IngredientID: one.IngredientID, Grams: one.Grams})
	}

	return grams
}

// ratesOf answers the money of the batch per unit, per gram and per
// kilogram.
//
// The money of the book counts whole pesos, and the money of the batch is
// whole pesos: the three answers are rates, and a rate is not money in the
// book.
//
// The yield is a declared constant: the batch makes the units the recipe
// declares, and the weight of them is the weight the owner declared.
func ratesOf(direct ledger.Amount, made Yield) (float64, float64, float64) {
	if made.Units <= 0 || made.Grams <= 0 {
		return 0, 0, 0
	}

	grams := float64(made.Units) * made.Grams
	perUnit := float64(direct) / float64(made.Units)
	perGram := float64(direct) / grams

	return perUnit, perGram, perGram * ingredient.GramsPerKilogram
}

// Named answers the recipes that carry that name.
//
// The rule of the domain: the name is compared the way a person reads it,
// trimmed and without case: "Pan de molde" and "pan de molde " are the
// same name.
//
// The name of a recipe is not unique inside the business, and the answer
// carries every recipe that carries it: the owner chooses with one word.
func Named(recipes []Recipe, name string) []Recipe {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return nil
	}

	found := make([]Recipe, 0, len(recipes))

	for _, one := range recipes {
		if strings.ToLower(strings.TrimSpace(one.Name)) == key {
			found = append(found, one)
		}
	}

	return found
}
