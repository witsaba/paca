package ledger

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/account"
)

func TestTheDebtIsTheAccountAndWhatItStillHolds(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)

	debt := Debt{
		Account: account.Account{
			ID:           account.ID(1),
			Kind:         account.KindLiability,
			Name:         "Préstamo: Ana",
			Counterparty: "Ana",
			DueDate:      &due,
		},
		Amount: 500,
	}

	if debt.Account.Name != "Préstamo: Ana" {
		t.Errorf("the name of the account is %q, want %q", debt.Account.Name, "Préstamo: Ana")
	}
	if debt.Amount != 500 {
		t.Errorf("the amount is %d, want 500", debt.Amount)
	}
}

func TestOnlyTheAccountsThatCarryADebtAreDebts(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)

	accounts := []account.Account{
		{ID: account.ID(1), Kind: account.KindLiability, Name: "Préstamo: Ana", Counterparty: "Ana", DueDate: &due},
		{ID: account.ID(2), Kind: account.KindLiability, Name: "Deuda sin nombre"},
		{ID: account.ID(3), Kind: account.KindAsset, Name: "Efectivo"},
		{ID: account.ID(4), Kind: account.KindIncome, Name: "Salario"},
		{ID: account.ID(5), Kind: account.KindExpense, Name: "Alimentación"},
	}

	balances := []Balance{
		{AccountID: account.ID(1), Kind: account.KindLiability, Credits: 500},
		{AccountID: account.ID(2), Kind: account.KindLiability, Credits: 300},
		{AccountID: account.ID(3), Kind: account.KindAsset, Debits: 700},
		{AccountID: account.ID(4), Kind: account.KindIncome, Credits: 900},
		{AccountID: account.ID(5), Kind: account.KindExpense, Debits: 100},
	}

	debts := Debts(accounts, balances)

	if len(debts) != 1 {
		t.Fatalf("the answer holds %d debts, want 1", len(debts))
	}
	if debts[0].Account.Name != "Préstamo: Ana" {
		t.Errorf("the debt is %q, want %q", debts[0].Account.Name, "Préstamo: Ana")
	}
}

func TestTheDebtHoldsWhatThePersonStillOwesOrIsOwed(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)

	accounts := []account.Account{
		{ID: account.ID(1), Kind: account.KindLiability, Name: "Préstamo: Ana", Counterparty: "Ana", DueDate: &due},
		{ID: account.ID(2), Kind: account.KindAsset, Name: "Deuda de Beto", Counterparty: "Beto"},
	}

	balances := []Balance{
		{AccountID: account.ID(1), Kind: account.KindLiability, Debits: 100, Credits: 500},
		{AccountID: account.ID(2), Kind: account.KindAsset, Debits: 200},
	}

	debts := Debts(accounts, balances)

	if len(debts) != 2 {
		t.Fatalf("the answer holds %d debts, want 2", len(debts))
	}
	if debts[0].Amount != 400 {
		t.Errorf("the debt of Ana is %d, want 400", debts[0].Amount)
	}
	if debts[1].Amount != 200 {
		t.Errorf("the debt of Beto is %d, want 200", debts[1].Amount)
	}
}

func TestOnlyTheOpenDebtsAreInTheAnswer(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)

	accounts := []account.Account{
		{ID: account.ID(1), Kind: account.KindLiability, Name: "Préstamo pagado", Counterparty: "Ana", DueDate: &due},
		{ID: account.ID(2), Kind: account.KindLiability, Name: "Préstamo abierto", Counterparty: "Beto", DueDate: &due},
		{ID: account.ID(3), Kind: account.KindLiability, Name: "Préstamo sin saldo", Counterparty: "Carla", DueDate: &due},
	}

	balances := []Balance{
		{AccountID: account.ID(1), Kind: account.KindLiability, Debits: 500, Credits: 500},
		{AccountID: account.ID(2), Kind: account.KindLiability, Credits: 250},
	}

	debts := Debts(accounts, balances)

	if len(debts) != 1 {
		t.Fatalf("the answer holds %d debts, want 1", len(debts))
	}
	if debts[0].Account.Name != "Préstamo abierto" {
		t.Errorf("the debt is %q, want %q", debts[0].Account.Name, "Préstamo abierto")
	}
}

func TestTheDebtsComeByTheDayTheyFallDue(t *testing.T) {
	t.Parallel()

	first := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)
	second := time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC)

	accounts := []account.Account{
		{ID: account.ID(1), Kind: account.KindLiability, Name: "Préstamo: Carla", Counterparty: "Carla", DueDate: &second},
		{ID: account.ID(2), Kind: account.KindLiability, Name: "Deuda sin fecha", Counterparty: "Diego"},
		{ID: account.ID(3), Kind: account.KindLiability, Name: "Préstamo: Beto", Counterparty: "Beto", DueDate: &second},
		{ID: account.ID(4), Kind: account.KindLiability, Name: "Préstamo: Ana", Counterparty: "Ana", DueDate: &first},
		{ID: account.ID(5), Kind: account.KindLiability, Name: "Otra deuda sin fecha", Counterparty: "Elena"},
	}

	balances := []Balance{
		{AccountID: account.ID(1), Kind: account.KindLiability, Credits: 100},
		{AccountID: account.ID(2), Kind: account.KindLiability, Credits: 100},
		{AccountID: account.ID(3), Kind: account.KindLiability, Credits: 100},
		{AccountID: account.ID(4), Kind: account.KindLiability, Credits: 100},
		{AccountID: account.ID(5), Kind: account.KindLiability, Credits: 100},
	}

	debts := Debts(accounts, balances)

	want := []string{"Préstamo: Ana", "Préstamo: Beto", "Préstamo: Carla", "Deuda sin fecha", "Otra deuda sin fecha"}
	if len(debts) != len(want) {
		t.Fatalf("the answer holds %d debts, want %d", len(debts), len(want))
	}

	for index, name := range want {
		if debts[index].Account.Name != name {
			t.Errorf("the debt %d is %q, want %q", index, debts[index].Account.Name, name)
		}
	}
}

func TestTheDebtAcceptsThePaymentItStillHolds(t *testing.T) {
	t.Parallel()

	debt := Debt{Account: account.Account{Kind: account.KindLiability, Name: "Préstamo: Ana"}, Amount: 500}

	if err := debt.AcceptsPayment(100); err != nil {
		t.Fatalf("the payment was refused: %v", err)
	}
	if err := debt.AcceptsPayment(500); err != nil {
		t.Fatalf("the payment of the whole debt was refused: %v", err)
	}
}

func TestTheDebtRefusesThePaymentItCannotHold(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payment Amount
		want    error
	}{
		{name: "a payment of nothing", payment: 0, want: ErrInvalidAmount},
		{name: "a payment of a negative amount", payment: -100, want: ErrInvalidAmount},
		{name: "a payment that passes the debt", payment: 501, want: ErrPaymentTooLarge},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			debt := Debt{Account: account.Account{Kind: account.KindLiability, Name: "Préstamo: Ana"}, Amount: 500}

			err := debt.AcceptsPayment(testCase.payment)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}
