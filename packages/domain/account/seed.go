package account

// Seed is an account of the default set of a personal business: its kind and
// its name, without the business that will own it.
type Seed struct {
	// Kind is the kind of the account.
	Kind Kind

	// Name is the name the person reads in their book.
	Name string
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
		{Kind: KindAsset, Name: "Efectivo"},
		{Kind: KindAsset, Name: "Banco"},
		// The name is in Spanish, and the linter of the English words
		// reads it as a typo: the name is the one the person reads.
		{Kind: KindAsset, Name: "Inventario"}, //nolint:misspell // "Inventario" is the name the person reads in their book.
		{Kind: KindIncome, Name: "Ventas"},
		{Kind: KindExpense, Name: "Costo de ventas"},
		{Kind: KindExpense, Name: "Mano de obra"},
		{Kind: KindExpense, Name: "Gastos indirectos"},
		{Kind: KindExpense, Name: "Merma"},
		{Kind: KindEquity, Name: "Capital"},
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
		{Kind: KindAsset, Name: "Efectivo"},
		{Kind: KindAsset, Name: "Banco"},
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
