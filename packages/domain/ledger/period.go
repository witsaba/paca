package ledger

import (
	"fmt"
	"time"
)

// Period is the stretch of the book that a reading covers.
//
// The period has two ends and it includes both of them: the movement of the
// first day and the movement of the last one are inside the period. The
// period is how a reading names the days it takes: the month of the person
// and the quarter of the business.
//
// The period holds days and not moments: the time of the day is not a fact
// of the money.
type Period struct {
	// From is the first day of the period. It is inside the period.
	From Date

	// To is the last day of the period. It is inside the period.
	To Date
}

// NewPeriod returns the period between two days.
//
// The error is ErrInvalidPeriod when the first day is after the last one: a
// period that ends before it begins is not a period, and the caller that
// built one holds the days in the wrong order.
func NewPeriod(from, to Date) (Period, error) {
	if from.After(to) {
		return Period{}, fmt.Errorf("%w: %s is after %s", ErrInvalidPeriod, from, to)
	}

	return Period{From: from, To: to}, nil
}

// Month returns the period of a month of the calendar.
//
// The period holds the first day and the last day of the month, and it
// includes both of them. The days are in UTC, because the book counts days
// and the day of a moment does not depend on the time zone of the reader.
//
// The month is the one the calendar says: a month of twenty eight, twenty
// nine, thirty or thirty one days holds the days it really has.
func Month(year int, month time.Month) Period {
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC)

	return Period{From: NewDate(from), To: NewDate(to)}
}

// Summary is the money of a period of the book.
//
// The two numbers are what the person earned and what they spent during the
// period, and both of them are positive magnitudes: the income is what came
// in and the expenses are what went out, never what is left of them.
type Summary struct {
	// Income is what the person earned during the period.
	Income Amount

	// Expenses is what the person spent during the period.
	Expenses Amount
}

// NewSummary returns the money of a period.
//
// The income and the expenses are the magnitudes of what the person earned
// and of what they spent: the money of a period never carries a negative
// number, because a negative income reads like a loss of money that never
// happened.
func NewSummary(income, expenses Amount) Summary {
	return Summary{Income: income, Expenses: expenses}
}

// Net returns what is left of the income of the period once its expenses
// are taken out of it.
//
// The net is negative when the person spent more than they earned, and that
// is a fact of the period and not an error of the book.
func (s Summary) Net() Amount {
	return s.Income - s.Expenses
}
