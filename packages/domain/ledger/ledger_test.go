package ledger

import (
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
)

func TestNewDateKeepsTheDayAndDropsTheTime(t *testing.T) {
	t.Parallel()

	moment := time.Date(2026, time.March, 1, 15, 30, 45, 0, time.UTC)

	day := NewDate(moment)

	if day.String() != "2026-03-01" {
		t.Errorf("the day is %q, want %q", day.String(), "2026-03-01")
	}
	if day.Time().Hour() != 0 {
		t.Errorf("the hour of the day is %d, want 0", day.Time().Hour())
	}
}

func TestParseDateReadsTheDayAPersonWrites(t *testing.T) {
	t.Parallel()

	day, err := ParseDate("2026-03-01")
	if err != nil {
		t.Fatalf("the day was refused: %v", err)
	}
	if day.String() != "2026-03-01" {
		t.Errorf("the day is %q, want %q", day.String(), "2026-03-01")
	}
}

func TestParseDateRefusesWhatIsNotADay(t *testing.T) {
	t.Parallel()

	cases := []string{"", "01/03/2026", "2026-13-01", "the first of March"}

	for _, day := range cases {
		t.Run(day, func(t *testing.T) {
			t.Parallel()

			if _, err := ParseDate(day); !errors.Is(err, ErrInvalidDate) {
				t.Fatalf("the error is %v, want %v", err, ErrInvalidDate)
			}
		})
	}
}

func TestTheDaysAreOrdered(t *testing.T) {
	t.Parallel()

	first := NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	second := NewDate(time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC))

	if !first.Before(second) {
		t.Error("the first day is not before the second one")
	}
	if !second.After(first) {
		t.Error("the second day is not after the first one")
	}
	if first.Before(first) {
		t.Error("a day is before itself")
	}
}

func TestNewEntryBuildsAMovement(t *testing.T) {
	t.Parallel()

	day := NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))

	entry, err := NewEntry(business.ID(77), account.ID(1), account.ID(2), 500, day, "  Salary of March  ")
	if err != nil {
		t.Fatalf("the movement was refused: %v", err)
	}

	if entry.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", entry.BusinessID)
	}
	if entry.DebitAccountID != account.ID(1) {
		t.Errorf("the debit account is %d, want 1", entry.DebitAccountID)
	}
	if entry.CreditAccountID != account.ID(2) {
		t.Errorf("the credit account is %d, want 2", entry.CreditAccountID)
	}
	if entry.Amount != 500 {
		t.Errorf("the amount is %d, want 500", entry.Amount)
	}
	if entry.OccurredOn != day {
		t.Errorf("the day is %v, want %v", entry.OccurredOn, day)
	}
	if entry.Description != "Salary of March" {
		t.Errorf("the description is %q, want %q", entry.Description, "Salary of March")
	}
	if entry.ID != 0 {
		t.Errorf("the identity is %d, want 0: the storage generates it", entry.ID)
	}
}

func TestNewEntryRefusesWhatTheDomainRefuses(t *testing.T) {
	t.Parallel()

	day := NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))

	cases := []struct {
		name        string
		debit       account.ID
		credit      account.ID
		amount      Amount
		description string
		want        error
	}{
		{
			name:        "a movement without money",
			debit:       1,
			credit:      2,
			amount:      0,
			description: "Salary",
			want:        ErrInvalidAmount,
		},
		{
			name:        "a movement of a negative amount",
			debit:       1,
			credit:      2,
			amount:      -500,
			description: "Salary",
			want:        ErrInvalidAmount,
		},
		{
			name:        "money that goes nowhere",
			debit:       1,
			credit:      1,
			amount:      500,
			description: "Salary",
			want:        ErrSameAccount,
		},
		{
			name:        "a movement without a description",
			debit:       1,
			credit:      2,
			amount:      500,
			description: "   ",
			want:        ErrInvalidDescription,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewEntry(business.ID(77), testCase.debit, testCase.credit, testCase.amount, day, testCase.description)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestTheBalanceReadsWhatItMeansToThePerson(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		kind    account.Kind
		debits  Amount
		credits Amount
		want    Amount
	}{
		{
			name:   "the cash the person has",
			kind:   account.KindAsset,
			debits: 500,
			want:   500,
		},
		{
			name:    "the food the person spent",
			kind:    account.KindExpense,
			debits:  50,
			credits: 10,
			want:    40,
		},
		{
			name:    "the money the person owes",
			kind:    account.KindLiability,
			debits:  100,
			credits: 500,
			want:    400,
		},
		{
			name:    "the money the person earned",
			kind:    account.KindIncome,
			credits: 500,
			want:    500,
		},
		{
			name:    "the money the person put into the book",
			kind:    account.KindEquity,
			credits: 500,
			want:    500,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			balance := Balance{
				AccountID: 1,
				Kind:      testCase.kind,
				Debits:    testCase.debits,
				Credits:   testCase.credits,
			}

			if got := balance.Normal(); got != testCase.want {
				t.Fatalf("the balance is %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestBookOfReadsTheMoneyOfTheBusiness(t *testing.T) {
	t.Parallel()

	balances := []Balance{
		{AccountID: 1, Kind: account.KindAsset, Debits: 500},
		{AccountID: 2, Kind: account.KindAsset, Debits: 200, Credits: 50},
		{AccountID: 3, Kind: account.KindLiability, Credits: 300},
		{AccountID: 4, Kind: account.KindIncome, Credits: 700},
		{AccountID: 5, Kind: account.KindExpense, Debits: 200},
	}

	book := BookOf(business.ID(77), balances)

	if book.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", book.BusinessID)
	}
	if book.Assets != 650 {
		t.Errorf("the assets are %d, want 650", book.Assets)
	}
	if book.Liabilities != 300 {
		t.Errorf("the liabilities are %d, want 300", book.Liabilities)
	}
	if book.Income != 700 {
		t.Errorf("the income is %d, want 700", book.Income)
	}
	if book.Expenses != 200 {
		t.Errorf("the expenses are %d, want 200", book.Expenses)
	}
	if book.NetWorth() != 350 {
		t.Errorf("the net worth is %d, want 350", book.NetWorth())
	}
}

func TestErrorsNamesEveryError(t *testing.T) {
	t.Parallel()

	all := Errors()
	if len(all) != 6 {
		t.Fatalf("the catalogue holds %d errors, want 6", len(all))
	}

	for _, err := range all {
		if err == nil {
			t.Error("the catalogue holds an empty error")
		}
	}
}
