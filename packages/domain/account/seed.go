package account

// Seed is an account of the default set of a personal business: its kind and
// its name, without the business that will own it.
type Seed struct {
	// Kind is the kind of the account.
	Kind Kind

	// Name is the name the person reads in their book.
	Name string
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
