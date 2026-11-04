package ledger

import (
	"errors"
	"testing"
	"time"
)

func TestThePeriodHoldsTheTwoEndsOfTheTimeItCovers(t *testing.T) {
	t.Parallel()

	from := NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	to := NewDate(time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC))

	period, err := NewPeriod(from, to)
	if err != nil {
		t.Fatalf("the period was refused: %v", err)
	}

	if period.From != from {
		t.Errorf("the first day is %v, want %v", period.From, from)
	}
	if period.To != to {
		t.Errorf("the last day is %v, want %v", period.To, to)
	}
}

func TestThePeriodRefusesAnEndThatComesBeforeItsBeginning(t *testing.T) {
	t.Parallel()

	from := NewDate(time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC))
	to := NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))

	_, err := NewPeriod(from, to)
	if !errors.Is(err, ErrInvalidPeriod) {
		t.Fatalf("the error is %v, want %v", err, ErrInvalidPeriod)
	}
}

func TestThePeriodOfAMonthHoldsItsFirstAndItsLastDay(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		year  int
		month time.Month
		from  string
		to    string
	}{
		{name: "a month of thirty one days", year: 2026, month: time.March, from: "2026-03-01", to: "2026-03-31"},
		{name: "a month of thirty days", year: 2026, month: time.June, from: "2026-06-01", to: "2026-06-30"},
		{name: "a month of twenty eight days", year: 2026, month: time.February, from: "2026-02-01", to: "2026-02-28"},
		{name: "a leap month", year: 2024, month: time.February, from: "2024-02-01", to: "2024-02-29"},
		{name: "the last month of the year", year: 2026, month: time.December, from: "2026-12-01", to: "2026-12-31"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			period := Month(testCase.year, testCase.month)

			if period.From.String() != testCase.from {
				t.Errorf("the first day is %q, want %q", period.From.String(), testCase.from)
			}
			if period.To.String() != testCase.to {
				t.Errorf("the last day is %q, want %q", period.To.String(), testCase.to)
			}
		})
	}
}

func TestTheMoneyOfAPeriodReadsWhatThePersonEarnedAndSpent(t *testing.T) {
	t.Parallel()

	summary := NewSummary(2_000_000, 1_500_000)

	if summary.Income != 2_000_000 {
		t.Errorf("the income is %d, want 2000000", summary.Income)
	}
	if summary.Expenses != 1_500_000 {
		t.Errorf("the expenses are %d, want 1500000", summary.Expenses)
	}
	if summary.Net() != 500_000 {
		t.Errorf("the net is %d, want 500000", summary.Net())
	}
}

func TestTheMoneyOfAPeriodWithMoreExpensesThanIncomeIsNegative(t *testing.T) {
	t.Parallel()

	summary := NewSummary(1_000_000, 1_500_000)

	if summary.Net() != -500_000 {
		t.Errorf("the net is %d, want -500000", summary.Net())
	}
}
