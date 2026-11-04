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
