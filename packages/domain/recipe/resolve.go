package recipe

import (
	"strconv"
	"strings"

	"github.com/witsaba/paca/packages/domain/ingredient"
)

// resolvedIngredient is an ingredient of a recipe resolved against the
// catalog of the business.
//
// The recipe names the ingredient the way the owner says, and the catalog
// is asked when the money or the stock is needed. The resolution is never
// stored in the recipe: it is derived every time it is needed, and it
// answers the ingredient the recipe names even when the recipe was written
// before the business bought it.
type resolvedIngredient struct {
	// ID is the identity of the ingredient of the catalog. It is empty
	// when the catalog does not hold the ingredient.
	ID ingredient.ID

	// Name is the name of the ingredient: the name of the catalog when it
	// holds it, and the name the owner said when it does not.
	Name string

	// Grams is how many grams of it the recipe needs.
	Grams float64

	// Item is the ingredient of the catalog. It is empty when the catalog
	// does not hold the ingredient.
	Item ingredient.Ingredient

	// Held says that the catalog holds the ingredient.
	Held bool
}

// resolveTheIngredients answers the ingredients of the recipe resolved
// against the catalog of the business.
//
// The rules of the domain:
//
//   - the name the owner said is the answer, and the catalog is asked when
//     the money or the stock is needed: a recipe written before the first
//     purchase is therefore not a recipe without an ingredient, it is a
//     recipe whose ingredient the catalog did not know yet, and the catalog
//     answers it as soon as the owner buys it;
//   - the link the recipe carries is the memory of what the owner meant
//     when her wording was not enough, and it decides only when the name
//     does not answer: a renamed ingredient never breaks the recipe;
//   - two ingredients the recipe names twice are one ingredient of the
//     batch: the grams of the two are added up, and the cost of the batch
//     never pays twice for the same gram.
//
// The answer keeps the order the owner said the ingredients in.
func resolveTheIngredients(baked Recipe, catalog ingredient.Catalog) []resolvedIngredient {
	resolved := make([]resolvedIngredient, 0, len(baked.Ingredients))
	indexes := make(map[string]int, len(baked.Ingredients))

	for _, one := range baked.Ingredients {
		item, held := resolveTheIngredient(one, catalog)

		key := keyOfTheIngredient(one, item, held)
		if index, seen := indexes[key]; seen {
			resolved[index].Grams += one.Grams

			continue
		}

		indexes[key] = len(resolved)
		resolved = append(resolved, resolvedIngredient{
			ID:    item.ID,
			Name:  nameOfTheIngredient(one, item, held),
			Grams: one.Grams,
			Item:  item,
			Held:  held,
		})
	}

	return resolved
}

// resolveTheIngredient answers the ingredient of the catalog that one
// ingredient of a recipe names.
//
// The rule of the domain: the name the owner said decides, and the catalog
// is the one that answers. A name the catalog knows exactly is the
// ingredient. A name the catalog does not know is answered by the link the
// recipe carries, when the catalog still holds it: the owner named it the
// way she says it, and the catalog does not know that wording yet. A name
// several ingredients answer is answered by the link when it names one of
// them, and by nothing at all when it does not: the money of an ambiguous
// name is never invented.
func resolveTheIngredient(one Ingredient, catalog ingredient.Catalog) (ingredient.Ingredient, bool) {
	matched, err := catalog.Match(one.Name)
	if err != nil {
		return theLinked(one, catalog)
	}

	switch matched.Status {
	case ingredient.StatusExact:
		return matched.Ingredient, true
	case ingredient.StatusAmbiguous:
		for _, option := range matched.Options {
			if option.ID == one.IngredientID {
				return theLinked(one, catalog)
			}
		}

		return ingredient.Ingredient{}, false
	default:
		return theLinked(one, catalog)
	}
}

// theLinked answers the ingredient the link of a recipe names.
//
// The rule of the domain: the link is the memory of what the owner meant
// when her wording was not enough, and it is never a requirement: a recipe
// without a link is resolved by its name alone.
func theLinked(one Ingredient, catalog ingredient.Catalog) (ingredient.Ingredient, bool) {
	if one.IngredientID == 0 {
		return ingredient.Ingredient{}, false
	}

	item, held := catalog.Find(one.IngredientID)

	return item, held
}

// nameOfTheIngredient answers the name of the line of an ingredient.
//
// The rule of the domain: the name of the catalog is the name of the line
// when the catalog holds the ingredient, and the name the owner said is the
// name of the line when it does not.
func nameOfTheIngredient(one Ingredient, item ingredient.Ingredient, held bool) string {
	if held {
		return item.Name
	}

	return strings.TrimSpace(one.Name)
}

// keyOfTheIngredient answers the key that tells two ingredients apart.
//
// The rule of the domain: an ingredient the catalog holds is the one of its
// identity, and an ingredient the catalog does not hold is the one of the
// name the owner said - the name is read the way a person reads it, trimmed
// and without case.
func keyOfTheIngredient(one Ingredient, item ingredient.Ingredient, held bool) string {
	if held {
		return "id:" + strconv.FormatInt(int64(item.ID), 10)
	}

	return "name:" + strings.ToLower(strings.TrimSpace(one.Name))
}
