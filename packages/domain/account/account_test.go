package account

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
)

func TestNewBuildsAnAccount(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	built, err := New(7, KindAsset, "  Efectivo  ", "", nil)
	if err != nil {
		t.Fatalf("the account was refused: %v", err)
	}

	if built.BusinessID != business.ID(7) {
		t.Errorf("the business is %d, want 7", built.BusinessID)
	}
	if built.Kind != KindAsset {
		t.Errorf("the kind is %q, want %q", built.Kind, KindAsset)
	}
	if built.Name != "Efectivo" {
		t.Errorf("the name is %q, want %q", built.Name, "Efectivo")
	}
	if built.Counterparty != "" {
		t.Errorf("the counterparty is %q, want it empty", built.Counterparty)
	}
	if built.DueDate != nil {
		t.Errorf("the due date is %v, want it absent", built.DueDate)
	}
	if built.ID != 0 {
		t.Errorf("the identity is %d, want 0: the storage generates it", built.ID)
	}

	debt, err := New(7, KindLiability, "Préstamo: Juan", "  Juan  ", &due)
	if err != nil {
		t.Fatalf("the debt was refused: %v", err)
	}
	if debt.Counterparty != "Juan" {
		t.Errorf("the counterparty is %q, want %q", debt.Counterparty, "Juan")
	}
	if debt.DueDate == nil || !debt.DueDate.Equal(due) {
		t.Errorf("the due date is %v, want %v", debt.DueDate, due)
	}
}

func TestNewRefusesWhatTheDomainRefuses(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name         string
		kind         Kind
		account      string
		counterparty string
		dueDate      *time.Time
		want         error
	}{
		{
			name:    "an unknown kind",
			kind:    Kind("savings"),
			account: "Efectivo",
			want:    ErrUnknownKind,
		},
		{
			name:    "an empty name",
			kind:    KindAsset,
			account: "   ",
			want:    ErrInvalidName,
		},
		{
			name:         "a counterparty in an expense",
			kind:         KindExpense,
			account:      "Alimentación",
			counterparty: "Juan",
			want:         ErrDebtNotAllowed,
		},
		{
			name:    "a due date in an income",
			kind:    KindIncome,
			account: "Salario",
			dueDate: &due,
			want:    ErrDebtNotAllowed,
		},
		{
			name:         "a counterparty in an equity",
			kind:         KindEquity,
			account:      "Apertura",
			counterparty: "Juan",
			want:         ErrDebtNotAllowed,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(7, testCase.kind, testCase.account, testCase.counterparty, testCase.dueDate)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestKindCarriesDebt(t *testing.T) {
	t.Parallel()

	if !KindLiability.CarriesDebt() {
		t.Error("a liability carries no debt")
	}
	if !KindAsset.CarriesDebt() {
		t.Error("an asset carries no debt")
	}
	if KindIncome.CarriesDebt() {
		t.Error("an income carries a debt")
	}
	if KindExpense.CarriesDebt() {
		t.Error("an expense carries a debt")
	}
	if KindEquity.CarriesDebt() {
		t.Error("an equity carries a debt")
	}
}

func TestAnAccountCarriesADebtWhenItOwesOrIsOwed(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	debt, err := New(7, KindLiability, "Préstamo: Juan", "Juan", &due)
	if err != nil {
		t.Fatalf("the debt was refused: %v", err)
	}
	if !debt.IsDebt() {
		t.Error("a liability with a counterparty carries no debt")
	}

	owed, err := New(7, KindAsset, "Deuda de Pedro", "Pedro", nil)
	if err != nil {
		t.Fatalf("the account was refused: %v", err)
	}
	if !owed.IsDebt() {
		t.Error("an asset with a counterparty carries no debt")
	}

	cash, err := New(7, KindAsset, "Efectivo", "", nil)
	if err != nil {
		t.Fatalf("the account was refused: %v", err)
	}
	if cash.IsDebt() {
		t.Error("an account without a counterparty carries a debt")
	}

	salary, err := New(7, KindIncome, "Salario", "", nil)
	if err != nil {
		t.Fatalf("the account was refused: %v", err)
	}
	if salary.IsDebt() {
		t.Error("an income carries a debt")
	}
}

func TestParseKind(t *testing.T) {
	t.Parallel()

	kind, err := ParseKind("liability")
	if err != nil {
		t.Fatalf("the kind was refused: %v", err)
	}
	if kind != KindLiability {
		t.Errorf("the kind is %q, want %q", kind, KindLiability)
	}

	if _, err := ParseKind("savings"); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownKind)
	}
}

func TestKindValid(t *testing.T) {
	t.Parallel()

	for _, kind := range AllKinds() {
		if !kind.Valid() {
			t.Errorf("the kind %q is not valid", kind)
		}
	}

	if Kind("savings").Valid() {
		t.Error("an unknown kind is valid")
	}
}

func TestKindString(t *testing.T) {
	t.Parallel()

	if KindExpense.String() != "expense" {
		t.Errorf("the name of the kind is %q, want %q", KindExpense.String(), "expense")
	}
}

func TestAllKindsReturnsACopy(t *testing.T) {
	t.Parallel()

	all := AllKinds()
	if len(all) != 5 {
		t.Fatalf("the list holds %d kinds, want 5", len(all))
	}
	if all[0] != KindAsset {
		t.Errorf("the first kind is %q, want %q", all[0], KindAsset)
	}

	all[0] = Kind("changed")

	if AllKinds()[0] != KindAsset {
		t.Error("the list changed under the caller")
	}
}

func TestDefaultPersonalAccounts(t *testing.T) {
	t.Parallel()

	seed := DefaultPersonalAccounts()
	if len(seed) != 12 {
		t.Fatalf("the seed holds %d accounts, want 12", len(seed))
	}

	names := make(map[string]bool, len(seed))
	for _, account := range seed {
		if account.Name == "" {
			t.Error("the seed holds an account without a name")
		}
		if names[account.Name] {
			t.Errorf("the seed holds %q twice", account.Name)
		}
		names[account.Name] = true

		if !account.Kind.Valid() {
			t.Errorf("the account %q holds the unknown kind %q", account.Name, account.Kind)
		}
	}

	for _, want := range []string{"Efectivo", "Banco", "Salario", "Otros ingresos", "Vivienda",
		"Alimentación", "Transporte", "Servicios", "Salud", "Educación", "Ocio", "Otros gastos"} {
		if !names[want] {
			t.Errorf("the seed does not hold %q", want)
		}
	}
}

func TestDefaultBusinessAccounts(t *testing.T) {
	t.Parallel()

	seed := DefaultBusinessAccounts()
	if len(seed) != 10 {
		t.Fatalf("the seed holds %d accounts, want 10", len(seed))
	}

	names := make(map[string]bool, len(seed))
	for _, one := range seed {
		if one.Name == "" {
			t.Error("the seed holds an account without a name")
		}
		if names[one.Name] {
			t.Errorf("the seed holds %q twice", one.Name)
		}
		names[one.Name] = true

		if !one.Kind.Valid() {
			t.Errorf("the account %q holds the unknown kind %q", one.Name, one.Kind)
		}
	}

	for _, want := range []string{"Efectivo", "Banco", "Inventario", "Ventas", "Costo de ventas", //nolint:misspell // "Inventario" is the name the person reads in their book.
		"Mano de obra", "Gastos indirectos", "Merma", "Donación", "Capital"} { //nolint:misspell // "Donación" is the name the person reads in their book.
		if !names[want] {
			t.Errorf("the seed does not hold %q", want)
		}
	}

	kinds := make(map[string]Kind, len(seed))
	for _, one := range seed {
		kinds[one.Name] = one.Kind
	}

	for name, want := range map[string]Kind{
		"Efectivo":          KindAsset,
		"Banco":             KindAsset,
		"Inventario":        KindAsset, //nolint:misspell // "Inventario" is the name the person reads in their book.
		"Ventas":            KindIncome,
		"Costo de ventas":   KindExpense,
		"Mano de obra":      KindExpense,
		"Gastos indirectos": KindExpense,
		"Merma":             KindExpense,
		"Donación":          KindExpense, //nolint:misspell // "Donación" is the name the person reads in their book.
		"Capital":           KindEquity,
	} {
		if kinds[name] != want {
			t.Errorf("the account %q is of the kind %q, want %q", name, kinds[name], want)
		}
	}
}

func TestDefaultBusinessAccountsReturnsACopy(t *testing.T) {
	t.Parallel()

	seed := DefaultBusinessAccounts()
	seed[0].Name = "changed"

	if DefaultBusinessAccounts()[0].Name != "Efectivo" {
		t.Error("the seed changed under the caller")
	}
}

func TestDefaultPersonalAccountsReturnsACopy(t *testing.T) {
	t.Parallel()

	seed := DefaultPersonalAccounts()
	seed[0].Name = "changed"

	if DefaultPersonalAccounts()[0].Name != "Efectivo" {
		t.Error("the seed changed under the caller")
	}
}

func TestErrorsNamesEveryError(t *testing.T) {
	t.Parallel()

	all := Errors()
	if len(all) != 7 {
		t.Fatalf("the catalogue holds %d errors, want 7", len(all))
	}

	for _, err := range all {
		if err == nil {
			t.Error("the catalogue holds an empty error")
		}
	}
}

func TestKindNormalSign(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind Kind
		want int
	}{
		{kind: KindAsset, want: 1},
		{kind: KindExpense, want: 1},
		{kind: KindLiability, want: -1},
		{kind: KindIncome, want: -1},
		{kind: KindEquity, want: -1},
	}

	for _, testCase := range cases {
		t.Run(string(testCase.kind), func(t *testing.T) {
			t.Parallel()

			if got := testCase.kind.NormalSign(); got != testCase.want {
				t.Fatalf("the sign is %d, want %d", got, testCase.want)
			}
		})
	}

	if Kind("savings").NormalSign() != 0 {
		t.Error("an unknown kind carries a sign")
	}
}

// TestTheSeedOfTheRoleIsTheAccountTheBookNames proves that the role of an
// account of the book of a business is declared by the seed: the seed owns
// the names, and the role is the way of asking for one of them.
func TestTheSeedOfTheRoleIsTheAccountTheBookNames(t *testing.T) {
	t.Parallel()

	for role, want := range map[Role]string{
		RoleCash:        "Efectivo",
		RoleBank:        "Banco",
		RoleInventory:   "Inventario", //nolint:misspell // "Inventario" is the name the person reads in their book.
		RoleSales:       "Ventas",
		RoleCostOfSales: "Costo de ventas",
		RoleLabor:       "Mano de obra",
		RoleOverhead:    "Gastos indirectos",
		RoleWaste:       "Merma",
		RoleDonation:    "Donación", //nolint:misspell // "Donación" is the name the person reads in their book.
		RoleCapital:     "Capital",
	} {
		seed, ok := SeedOfRole(role)
		if !ok {
			t.Errorf("the book holds no account of the role %q", role)
			continue
		}
		if seed.Name != want {
			t.Errorf("the account of the role %q is %q, want %q", role, seed.Name, want)
		}
	}
}

// TestTheAccountOfTheRoleIsTheAccountOfTheBook proves that the account of
// a role is found among the accounts of the book.
func TestTheAccountOfTheRoleIsTheAccountOfTheBook(t *testing.T) {
	t.Parallel()

	seed := DefaultBusinessAccounts()
	accounts := make([]Account, 0, len(seed))

	for index, one := range seed {
		created, err := New(business.ID(77), one.Kind, one.Name, "", nil)
		if err != nil {
			t.Fatalf("the account was refused: %v", err)
		}
		created.ID = ID(index + 1)
		accounts = append(accounts, created)
	}

	found, err := OfRole(accounts, RoleInventory)
	if err != nil {
		t.Fatalf("the account of the inventory was refused: %v", err)
	}
	if found.Name != "Inventario" { //nolint:misspell // "Inventario" is the name the person reads in their book.
		t.Errorf("the account of the inventory is %q, want %q", found.Name, "Inventario") //nolint:misspell // "Inventario" is the name the person reads in their book.
	}
}

// TestTheAccountOfTheRoleThatIsNotThereIsRefused proves that a book
// without the account of a role says so: the account is not there.
func TestTheAccountOfTheRoleThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := OfRole(nil, RoleInventory); !errors.Is(err, ErrNotFound) {
		t.Errorf("the error is %v, want %v", err, ErrNotFound)
	}

	if _, err := OfRole(nil, Role("nothing")); !errors.Is(err, ErrNotFound) {
		t.Errorf("the error is %v, want %v", err, ErrNotFound)
	}
}
