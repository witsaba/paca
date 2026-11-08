package postgres

// The names the database gives to the constraints of the table of the exits
// of the book.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another. The table of the exits is shared
// by every exit of the book: the waste and the sale, and the ones to come.
const (
	unitExitsPrimaryKey       = "unit_exits_pkey"
	unitExitsOnePositionIndex = "unit_exits_one_position"
)

// The kinds of the exits of the book.
//
// A unit leaves the book once, and the table of the exits says which exit
// took it.
const (
	// exitKindWaste is the exit of a unit that the bakery lost.
	exitKindWaste = "waste"

	// exitKindSale is the exit of a unit that the bakery sold.
	exitKindSale = "sale"
)
