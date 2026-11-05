package account

import "fmt"

// Role is what an account of the book of a business is for.
//
// The role is declared by the seed of the book: the account that holds the
// cash of the business is the one of the role cash, and the account of the
// inventory is the one of the role inventory.
//
// The role is what lets a use case ask for an account without knowing its
// name: the name is what the person reads, and the name may be renamed by
// her, while the role is what the book means.
type Role string

const (
	// RoleCash is the role of the account that holds the cash of the
	// business.
	RoleCash Role = "cash"

	// RoleBank is the role of the account that holds the money of the
	// business in the bank.
	RoleBank Role = "bank"

	// RoleInventory is the role of the account that holds what the
	// business bought and what it has.
	RoleInventory Role = "inventory"

	// RoleSales is the role of the account that holds what the business
	// earned.
	RoleSales Role = "sales"

	// RoleCostOfSales is the role of the account that holds the cost of
	// what the business sold.
	RoleCostOfSales Role = "cost_of_sales"

	// RoleLabor is the role of the account that holds the labor of the
	// business.
	RoleLabor Role = "labor"

	// RoleOverhead is the role of the account that holds the overhead of
	// the business.
	RoleOverhead Role = "overhead"

	// RoleWaste is the role of the account that holds what the business
	// lost.
	RoleWaste Role = "waste"

	// RoleCapital is the role of the account that holds what the owner put
	// into the book of the business.
	RoleCapital Role = "capital"
)

// Seed is an account of the default set of a personal business: its kind and
// its name, without the business that will own it.
type Seed struct {
	// Kind is the kind of the account.
	Kind Kind

	// Name is the name the person reads in their book.
	Name string

	// Role is what the account is for. It is empty when the account is
	// not one of the roles of the book of a business.
	Role Role
}

// DefaultBusinessAccounts returns the accounts a business of kind business
// starts with.
//
// The domain owns the seed: the contract the agent reads is derived from
// this list, and it never restates it. The names are in Spanish, because the
// person reads them in their book, and the identifiers of the code are in
// English.
//
// The accounts are the three layers of the real price of the owner - the
// cost of the sales, the labor and the overhead - and the money that buys
// them: the cash, the bank and the inventory. The sales are the income of
// the book, the capital is what the owner put into it, and the waste is the
// product that never reached a client.
//
// A debt is not in the seed, and neither is an account that is not in this
// list: they are created when they happen.
func DefaultBusinessAccounts() []Seed {
	return []Seed{
		{Kind: KindAsset, Name: "Efectivo", Role: RoleCash},
		{Kind: KindAsset, Name: "Banco", Role: RoleBank},
		// The name is in Spanish, and the linter of the English words
		// reads it as a typo: the name is the one the person reads.
		{Kind: KindAsset, Name: "Inventario", Role: RoleInventory}, //nolint:misspell // "Inventario" is the name the person reads in their book.
		{Kind: KindIncome, Name: "Ventas", Role: RoleSales},
		{Kind: KindExpense, Name: "Costo de ventas", Role: RoleCostOfSales},
		{Kind: KindExpense, Name: "Mano de obra", Role: RoleLabor},
		{Kind: KindExpense, Name: "Gastos indirectos", Role: RoleOverhead},
		{Kind: KindExpense, Name: "Merma", Role: RoleWaste},
		{Kind: KindEquity, Name: "Capital", Role: RoleCapital},
	}
}

// DefaultPersonalAccounts returns the accounts the Personal Business of a
// person starts with.
//
// The domain owns the seed: the contract the agent reads is derived from
// this list, and it never restates it. The names are in Spanish, because the
// person reads them in their book, and the identifiers of the code are in
// English.
//
// A debt is not in the seed: a loan with a friend and a product to pay
// later are created when they happen, and they are accounts like any other.
func DefaultPersonalAccounts() []Seed {
	return []Seed{
		{Kind: KindAsset, Name: "Efectivo", Role: RoleCash},
		{Kind: KindAsset, Name: "Banco", Role: RoleBank},
		{Kind: KindIncome, Name: "Salario"},
		{Kind: KindIncome, Name: "Otros ingresos"},
		{Kind: KindExpense, Name: "Vivienda"},
		{Kind: KindExpense, Name: "Alimentación"},
		{Kind: KindExpense, Name: "Transporte"},
		{Kind: KindExpense, Name: "Servicios"},
		{Kind: KindExpense, Name: "Salud"},
		{Kind: KindExpense, Name: "Educación"},
		{Kind: KindExpense, Name: "Ocio"},
		{Kind: KindExpense, Name: "Otros gastos"},
	}
}

// SeedOfRole returns the seed of the default book of a business that
// carries that role.
//
// The seed owns the names of the book: the role is the way of asking for
// one of them without writing its name down twice.
//
// The second answer is false when the role is not one of the book.
func SeedOfRole(role Role) (Seed, bool) {
	for _, one := range DefaultBusinessAccounts() {
		if one.Role == role {
			return one, true
		}
	}

	return Seed{}, false
}

// OfRole returns the account of the list that carries that role.
//
// The error is ErrNotFound when no account of the list carries that role:
// a book without the account of the inventory cannot hold what the
// business buys.
func OfRole(accounts []Account, role Role) (Account, error) {
	seed, ok := SeedOfRole(role)
	if !ok {
		return Account{}, fmt.Errorf("%w: %q is not a role of the book of a business", ErrNotFound, role)
	}

	for _, one := range accounts {
		if one.Name == seed.Name {
			return one, nil
		}
	}

	return Account{}, fmt.Errorf("%w: the book holds no account of the role %q", ErrNotFound, role)
}
