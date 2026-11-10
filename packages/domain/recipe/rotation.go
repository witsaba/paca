package recipe

import (
	"sort"

	"github.com/witsaba/paca/packages/domain/ledger"
)

// The rotation of the perishable output: the order in which the units of
// the book leave it.
//
// The rotation follows the EARLIEST EXPIRY: the unit that expires first is
// the first one that leaves the book. The valuation of the stock follows
// the FIFO: the oldest layer leaves first. The two rules are apart, and the
// model keeps them apart.
//
// The rotation carries no money: the FIFO money of a layer is not the
// rotation, and one answer that carried both would break the rule that
// keeps them apart. The money of a unit is the money of the production it
// came out of, and that is the rule of the sale, of the waste and of the
// exit.

// RotationUnit is a unit of the book, in the order it leaves it.
//
// It carries the day the unit expires, the days that are left of it and the
// product it belongs to. It carries no money: the money of a unit belongs
// to the movement that takes it out of the book.
type RotationUnit struct {
	// UnitID is the identity of the unit.
	UnitID UnitID

	// Order is the place the unit takes in the rotation: the first one
	// that leaves is the first one.
	Order int

	// ProductionID is the production the unit came out of.
	ProductionID ProductionID

	// Position is the order the unit was made in.
	Position int

	// RecipeID is the recipe the unit was made of.
	RecipeID ID

	// RecipeName is the name the owner reads of the recipe.
	RecipeName string

	// ProducedOn is the day the production was made.
	ProducedOn ledger.Date

	// Grams is the weight of the unit.
	Grams float64

	// ExpiresOn is the day the unit expires.
	ExpiresOn ledger.Date

	// DaysLeft are the days that are left of the unit.
	DaysLeft int
}

// Rotation is the order in which the units of the book leave it.
//
// It carries the day the answer was read on, because the days that are left
// of a unit are counted from that day.
type Rotation struct {
	// Day is the day the answer was read on.
	Day ledger.Date

	// RecipeID is the product the rotation is of. It is zero when the
	// rotation is of the whole book of the business.
	RecipeID ID

	// RecipeName is the name the owner reads of the product. It is empty
	// when the rotation is of the whole book of the business.
	RecipeName string

	// Units are the units of the book, in the order they leave it.
	Units []RotationUnit
}

// RotationOf answers the rotation of the units that are still in the book.
//
// The rules of the domain:
//
//   - the rotation follows the earliest expiry: the unit that expires
//     first is the first one that leaves the book;
//   - the units that expire on the same day keep a stable order: the
//     production first, and the position of the unit in it;
//   - the rotation may be of the whole book of a business or of one
//     product of it: the zero identity is the whole book, and the identity
//     of a product is that product alone;
//   - the answer carries the day every unit expires, the days that are
//     left of it, the product every unit belongs to and the place the unit
//     takes in the order;
//   - the answer carries no money: the FIFO money of a layer is the rule
//     of the stock, and it is not the rotation.
//
// The units are the ones that are still in the book: the caller reads them
// through the reader of the book, and a unit that left it - the one that
// was sold, wasted, consumed or given away - is never offered again.
//
// The day is the day the answer was read on, and the days that are left of
// a unit are counted from it: a rotation read with no day carries no days
// left.
func RotationOf(held []ExpiringUnit, of ID, name string, day ledger.Date) Rotation {
	rotation := Rotation{
		Day:        day,
		RecipeID:   of,
		RecipeName: name,
		Units:      make([]RotationUnit, 0, len(held)),
	}

	for index, one := range theUnitsOfTheProduct(held, of) {
		days := 0

		if !day.Time().IsZero() {
			days = one.DaysLeft(day)
		}

		rotation.Units = append(rotation.Units, RotationUnit{
			UnitID:       one.ID,
			Order:        index + 1,
			ProductionID: one.ProductionID,
			Position:     one.Position,
			RecipeID:     one.RecipeID,
			RecipeName:   one.RecipeName,
			ProducedOn:   one.ProducedOn,
			Grams:        one.Grams,
			ExpiresOn:    one.ExpiresOn,
			DaysLeft:     days,
		})
	}

	return rotation
}

// theUnitsOfTheProduct answers the units of the product that are still in
// the book, in the order they leave it.
//
// The rule of the domain: the rotation of the output follows the earliest
// expiry, and the unit that expires first is the first one that leaves the
// book. The valuation of the stock by the FIFO is another rule, and it does
// not live here.
//
// The units of the other products are not in the answer when the caller
// asks for one product, and the whole book is in the answer when the
// caller asks for no product at all.
func theUnitsOfTheProduct(held []ExpiringUnit, of ID) []ExpiringUnit {
	units := make([]ExpiringUnit, 0, len(held))

	for _, one := range held {
		if of > 0 && one.RecipeID != of {
			continue
		}

		units = append(units, one)
	}

	return orderedByTheRotation(units)
}

// orderedByTheRotation answers the units in the order they leave the book.
//
// The rule of the domain: the unit that expires first is the first one
// that leaves, and the units that expire on the same day keep a stable
// order: the production first, and the position of the unit in it.
func orderedByTheRotation(units []ExpiringUnit) []ExpiringUnit {
	ordered := make([]ExpiringUnit, len(units))
	copy(ordered, units)

	sort.SliceStable(ordered, func(first, second int) bool {
		if !ordered[first].ExpiresOn.Time().Equal(ordered[second].ExpiresOn.Time()) {
			return ordered[first].ExpiresOn.Before(ordered[second].ExpiresOn)
		}

		if ordered[first].ProductionID != ordered[second].ProductionID {
			return ordered[first].ProductionID < ordered[second].ProductionID
		}

		if ordered[first].Position != ordered[second].Position {
			return ordered[first].Position < ordered[second].Position
		}

		return ordered[first].ID < ordered[second].ID
	})

	return ordered
}
