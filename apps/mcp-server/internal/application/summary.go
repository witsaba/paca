package application

import (
	"context"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// SummaryService is the use case that closes a month.
//
// The closing of a month is what the person earned, what they spent, and
// what they still owe: the three numbers a person asks for when they say
// "how did this month go?".
//
// The service speaks in the model of the domain and it reads through the
// ports of the storage, and it knows nothing about the protocol that
// reaches it.
type SummaryService struct {
	accounts account.Repository
	book     ledger.Repository
	clock    ports.Clock
}

// NewSummaryService builds the use case of the closing of a month.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The clock is the
// port of the time: a month the caller does not name is the month that just
// ended, and the service does not read the clock of the machine by itself.
func NewSummaryService(
	accounts account.Repository,
	book ledger.Repository,
	clock ports.Clock,
) *SummaryService {
	return &SummaryService{accounts: accounts, book: book, clock: clock}
}

// Month reads the closing of a month: the income, the expenses, the net,
// and the debts that were open at the end of it.
//
// The month is the one the caller names, and the month that just ended when
// the caller names none: a person that says "close the month" means the
// month that is over.
func (s *SummaryService) Month(ctx context.Context, businessID business.ID, year, month int) (ports.MonthSummary, error) {
	if s == nil || s.accounts == nil || s.book == nil || s.clock == nil {
		return ports.MonthSummary{}, ports.ErrNoDatabase
	}

	period, err := s.periodOf(year, month)
	if err != nil {
		return ports.MonthSummary{}, err
	}

	summary, err := s.book.Summary(ctx, businessID, period.From, period.To)
	if err != nil {
		return ports.MonthSummary{}, err
	}

	accounts, err := s.accounts.List(ctx, businessID)
	if err != nil {
		return ports.MonthSummary{}, err
	}

	balances, err := s.book.Balances(ctx, businessID, period.To)
	if err != nil {
		return ports.MonthSummary{}, err
	}

	return ports.MonthSummary{
		From:    period.From,
		To:      period.To,
		Summary: summary,
		Debts:   ledger.Debts(accounts, balances),
	}, nil
}

// periodOf returns the month the caller names, and the month that just
// ended when the caller names none.
//
// The month of the calendar holds the days it really has, and both of its
// ends are inside it: the movement of the first day and the movement of
// the last one are counted in the closing.
func (s *SummaryService) periodOf(year, month int) (ledger.Period, error) {
	if year == 0 && month == 0 {
		now := s.clock.Now().UTC()
		last := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		return ledger.Month(last.Year(), last.Month()), nil
	}

	if year < 0 || month < 1 || month > 12 {
		return ledger.Period{}, ledger.ErrInvalidDate
	}

	return ledger.Month(year, time.Month(month)), nil
}
